package serviceconfig

import (
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var pricePattern = regexp.MustCompile(`^price_[A-Za-z0-9]+$`)

func invalid(message string) error { return &ValidationError{Message: message} }

func knownProvider(provider string) bool { return provider == "resend" || provider == "stripe" }

func validatePatch(provider string, patch Patch, value *configuration) error {
	if patch.Version < 0 {
		return invalid("version must be a nonnegative integer")
	}
	if patch.Source != "database" && patch.Source != "environment" {
		return invalid("source must be database or environment")
	}
	if patch.Source == "environment" {
		if patch.Enabled != nil || len(patch.Fields) != 0 || len(patch.Secrets) != 0 {
			return invalid("environment reset must not include provider fields")
		}
		return nil
	}
	if patch.Enabled != nil {
		value.Enabled = *patch.Enabled
	}
	for key, entry := range patch.Fields {
		if _, ok := value.Fields[key]; !ok {
			return invalid("unknown integration field")
		}
		value.Fields[key] = strings.TrimSpace(entry)
	}
	for key, entry := range patch.Secrets {
		if _, ok := value.Secrets[key]; !ok {
			return invalid("unknown integration secret")
		}
		if entry == "" && value.Enabled {
			return invalid("disable the integration before clearing a credential")
		}
		value.Secrets[key] = entry
	}
	// A disabled mail provider cannot deliver email login codes.
	if provider == "resend" && !value.Enabled {
		value.Fields["email_auth_enabled"] = "false"
	}
	return validateConfiguration(provider, value)
}

func validateConfiguration(provider string, value *configuration) error {
	for _, entry := range value.Fields {
		if !utf8.ValidString(entry) || len(entry) > 2048 || strings.ContainsAny(entry, "\r\n\x00") {
			return invalid("integration fields contain invalid characters or are too long")
		}
	}
	for _, entry := range value.Secrets {
		if !utf8.ValidString(entry) || len(entry) > 4096 || strings.IndexFunc(entry, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			return invalid("credentials must be at most 4096 bytes without whitespace or control characters")
		}
	}
	if provider == "resend" {
		return validateResend(value)
	}
	return validateStripe(value)
}

func validateResend(value *configuration) error {
	if value.Fields["email_auth_enabled"] != "true" && value.Fields["email_auth_enabled"] != "false" {
		return invalid("email_auth_enabled must be true or false")
	}
	if utf8.RuneCountInString(value.Fields["sender_name"]) > 100 {
		return invalid("sender_name must contain at most 100 characters")
	}
	email := value.Fields["sender_email"]
	if email != "" {
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || address.Name != "" || len(email) > 254 {
			return invalid("sender_email must be an email address without a display name")
		}
	}
	key := value.Secrets["api_key"]
	if key != "" && (!strings.HasPrefix(key, "re_") || len(key) <= len("re_")) {
		return invalid("Resend API key must start with re_")
	}
	if value.Enabled && (key == "" || email == "") {
		return invalid("enabled Resend requires api_key and sender_email")
	}
	if !value.Enabled && value.Fields["email_auth_enabled"] == "true" {
		return invalid("email login requires an enabled mail provider")
	}
	return nil
}

func validateStripe(value *configuration) error {
	mode := value.Fields["mode"]
	if mode != "test" && mode != "live" {
		return invalid("Stripe mode must be test or live")
	}
	secret := value.Secrets["secret_key"]
	if secret != "" && !hasKeyPrefix(secret, "sk_"+mode+"_", "rk_"+mode+"_") {
		return invalid("Stripe secret key must match the selected test or live mode")
	}
	if key := value.Fields["publishable_key"]; key != "" && !hasKeyPrefix(key, "pk_"+mode+"_") {
		return invalid("Stripe publishable key must match the selected test or live mode")
	}
	webhook := value.Secrets["webhook_secret"]
	if webhook != "" && !hasKeyPrefix(webhook, "whsec_") {
		return invalid("Stripe webhook secret must start with whsec_")
	}
	trial, err := strconv.ParseInt(value.Fields["trial_days"], 10, 64)
	if err != nil || trial < 0 || trial > 730 {
		return invalid("trial_days must be an integer from 0 to 730")
	}
	credits, err := strconv.ParseInt(value.Fields["credits_per_package"], 10, 64)
	if err != nil || credits < 1 || credits > 1000000 {
		return invalid("credits_per_package must be an integer from 1 to 1000000")
	}
	value.Fields["trial_days"] = strconv.FormatInt(trial, 10)
	value.Fields["credits_per_package"] = strconv.FormatInt(credits, 10)
	seen := map[string]bool{}
	addPrice := func(id string) error {
		if !pricePattern.MatchString(id) {
			return invalid("Stripe price IDs must start with price_ and contain only letters and digits")
		}
		if seen[id] {
			return invalid("each Stripe price ID must belong to only one offer")
		}
		seen[id] = true
		return nil
	}
	for _, field := range priceFields {
		if id := value.Fields[field]; id != "" {
			if err := addPrice(id); err != nil {
				return err
			}
		}
	}
	if raw := value.Fields["credit_price_ids"]; raw != "" {
		ids := strings.Split(raw, ",")
		if len(ids) > 20 {
			return invalid("at most 20 credit price IDs are supported")
		}
		for index := range ids {
			ids[index] = strings.TrimSpace(ids[index])
			if err := addPrice(ids[index]); err != nil {
				return err
			}
		}
		value.Fields["credit_price_ids"] = strings.Join(ids, ",")
	}
	if value.Enabled && (secret == "" || webhook == "" || len(seen) == 0) {
		return invalid("enabled Stripe requires secret_key, webhook_secret, and at least one price ID")
	}
	// Stripe price IDs and webhook signing secrets do not encode test/live mode.
	// This local validation deliberately makes no network or account claim.
	return nil
}

func hasKeyPrefix(key string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) && len(key) > len(prefix) {
			return true
		}
	}
	return false
}
