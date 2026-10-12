// Command billing-sandbox exercises Stripe test mode using isolated Test Clock objects.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	adapter "github.com/brizenchi/go-modules/modules/billing/adapter/stripe"
	"github.com/brizenchi/go-modules/modules/billing/domain"
)

var secret string
var client = &http.Client{Timeout: 25 * time.Second}

func call(ctx context.Context, method, path string, form url.Values) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, method, "https://api.stripe.com/v1"+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(secret, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", "2023-10-16")
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed")
	}
	defer res.Body.Close()
	var value map[string]any
	if err := json.NewDecoder(res.Body).Decode(&value); err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		detail, _ := value["error"].(map[string]any)
		return nil, fmt.Errorf("HTTP %d type=%v code=%v param=%v", res.StatusCode, detail["type"], detail["code"], detail["param"])
	}
	return value, nil
}
func run(parent context.Context, envFile string) (resultErr error) {
	raw, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			env[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
		}
	}
	secret = env["APP_BILLING_STRIPE_SECRET_KEY"]
	if !strings.HasPrefix(secret, "sk_test_") {
		return fmt.Errorf("test key required")
	}
	ctx, cancel := context.WithTimeout(parent, 6*time.Minute)
	defer cancel()
	prices := map[domain.PlanType]map[domain.BillingInterval]string{}
	for _, plan := range []domain.PlanType{domain.PlanStarter, domain.PlanPro, domain.PlanPremium} {
		prices[plan] = map[domain.BillingInterval]string{}
		for _, interval := range []domain.BillingInterval{domain.IntervalMonthly, domain.IntervalYearly} {
			prices[plan][interval] = env["APP_BILLING_STRIPE_PRICES_"+strings.ToUpper(string(plan))+"_"+strings.ToUpper(string(interval))]
		}
	}
	for _, plan := range []domain.PlanType{domain.PlanStarter, domain.PlanPro, domain.PlanPremium} {
		v, err := call(ctx, "GET", "/prices/"+prices[plan][domain.IntervalMonthly], nil)
		if err != nil {
			return fmt.Errorf("verify test price: %w", err)
		}
		if v["livemode"] != false {
			return fmt.Errorf("non-test price rejected")
		}
	}
	clock, err := call(ctx, "POST", "/test_helpers/test_clocks", url.Values{"frozen_time": {fmt.Sprint(time.Now().Unix())}, "name": {"quickstart billing lifecycle verification"}})
	if err != nil {
		return err
	}
	clockID := clock["id"].(string)
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := call(cleanupCtx, "DELETE", "/test_helpers/test_clocks/"+clockID, nil); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("cleanup failed for test clock %s: %w", clockID, err))
		} else {
			fmt.Println("PASS cleanup isolated test clock and customer")
		}
	}()
	customer, err := call(ctx, "POST", "/customers", url.Values{"test_clock": {clockID}, "name": {"Quickstart automated sandbox verification"}})
	if err != nil {
		return err
	}
	customerID := customer["id"].(string)
	pm, err := call(ctx, "POST", "/payment_methods/pm_card_visa/attach", url.Values{"customer": {customerID}})
	if err != nil {
		return err
	}
	_, err = call(ctx, "POST", "/customers/"+customerID, url.Values{"invoice_settings[default_payment_method]": {pm["id"].(string)}})
	if err != nil {
		return err
	}
	sub, err := call(ctx, "POST", "/subscriptions", url.Values{"customer": {customerID}, "items[0][price]": {prices[domain.PlanPro][domain.IntervalMonthly]}})
	if err != nil {
		return err
	}
	subID := sub["id"].(string)
	provider := adapter.NewProvider(adapter.Config{Enabled: true, SecretKey: secret, SubscriptionPrices: prices, HTTPClient: client})
	snap, err := provider.GetSubscription(ctx, subID)
	if err != nil {
		return fmt.Errorf("read subscription: %w", err)
	}
	if snap.Plan != domain.PlanPro || snap.PeriodEnd == nil {
		return fmt.Errorf("initial snapshot invalid")
	}
	fmt.Println("PASS initial subscription and period")
	_, err = provider.PreviewSubscriptionChange(ctx, customerID, subID, domain.SubscriptionPreviewInput{Plan: domain.PlanPremium, Interval: domain.IntervalMonthly, Mode: domain.ChangeModeImmediateProrated})
	if err != nil {
		return fmt.Errorf("preview upgrade: %w", err)
	}
	fmt.Println("PASS upgrade preview")
	snap, err = provider.ChangeSubscription(ctx, subID, domain.SubscriptionChangeInput{Plan: domain.PlanPremium, Interval: domain.IntervalMonthly, Mode: domain.ChangeModeImmediateProrated})
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}
	if snap.Plan != domain.PlanPremium || snap.PendingPayment != nil {
		return fmt.Errorf("paid upgrade not applied")
	}
	fmt.Println("PASS paid upgrade")
	snap, err = provider.ScheduleSubscriptionChange(ctx, subID, domain.SubscriptionChangeInput{Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, Mode: domain.ChangeModePeriodEnd})
	if err != nil {
		return fmt.Errorf("schedule downgrade: %w", err)
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.ScheduledChange == nil || snap.ScheduledChange.Plan != domain.PlanStarter || snap.Plan != domain.PlanPremium {
		return fmt.Errorf("scheduled downgrade snapshot invalid")
	}
	fmt.Println("PASS schedule and read future downgrade")
	if err := provider.CancelScheduledChange(ctx, subID); err != nil {
		return fmt.Errorf("remove schedule: %w", err)
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.ScheduledChange != nil || snap.Plan != domain.PlanPremium {
		return fmt.Errorf("schedule release changed membership")
	}
	fmt.Println("PASS remove scheduled change")
	if err := provider.CancelSubscription(ctx, subID, domain.CancelAtPeriodEnd); err != nil {
		return err
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if !snap.CancelAtPeriodEnd || snap.CancelEffectiveAt == nil {
		return fmt.Errorf("cancellation missing deadline")
	}
	fmt.Println("PASS end-of-period cancellation")
	if err := provider.ReactivateSubscription(ctx, subID); err != nil {
		return err
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.CancelAtPeriodEnd || snap.CancelEffectiveAt != nil {
		return fmt.Errorf("reactivation retained cancel flag")
	}
	fmt.Println("PASS reactivate")

	// Move a real Test Clock through the pending downgrade and a later cancellation.
	snap, err = provider.ScheduleSubscriptionChange(ctx, subID, domain.SubscriptionChangeInput{Plan: domain.PlanStarter, Interval: domain.IntervalMonthly, Mode: domain.ChangeModePeriodEnd})
	if err != nil {
		return err
	}
	advance := func(target int64) error {
		if _, err := call(ctx, "POST", "/test_helpers/test_clocks/"+clockID+"/advance", url.Values{"frozen_time": {fmt.Sprint(target)}}); err != nil {
			return err
		}
		for i := 0; i < 40; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
			state, err := call(ctx, "GET", "/test_helpers/test_clocks/"+clockID, nil)
			if err != nil {
				return err
			}
			if state["status"] == "ready" {
				return nil
			}
		}
		return fmt.Errorf("test clock not ready")
	}
	if err := advance(snap.ScheduledChange.EffectiveAt.Unix() + 60); err != nil {
		return err
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.Plan != domain.PlanStarter {
		return fmt.Errorf("scheduled downgrade not applied at next cycle")
	}
	fmt.Println("PASS Test Clock applies downgrade at next cycle")
	declined, err := call(ctx, "POST", "/payment_methods/pm_card_chargeCustomerFail/attach", url.Values{"customer": {customerID}})
	if err != nil {
		return err
	}
	if _, err := call(ctx, "POST", "/customers/"+customerID, url.Values{"invoice_settings[default_payment_method]": {declined["id"].(string)}}); err != nil {
		return err
	}
	snap, err = provider.ChangeSubscription(ctx, subID, domain.SubscriptionChangeInput{Plan: domain.PlanPro, Interval: domain.IntervalMonthly, Mode: domain.ChangeModeImmediateProrated})
	if err != nil {
		return err
	}
	if snap.Plan != domain.PlanStarter || snap.PendingPayment == nil || snap.PendingPayment.InvoiceURL == "" {
		return fmt.Errorf("failed upgrade must retain current plan and expose invoice")
	}
	fmt.Println("PASS failed upgrade preserves plan and exposes payment action")
	current, err := call(ctx, "GET", "/subscriptions/"+subID, nil)
	if err != nil {
		return err
	}
	if _, err := call(ctx, "POST", "/customers/"+customerID, url.Values{"invoice_settings[default_payment_method]": {pm["id"].(string)}}); err != nil {
		return err
	}
	if _, err := call(ctx, "POST", "/invoices/"+current["latest_invoice"].(string)+"/pay", url.Values{"payment_method": {pm["id"].(string)}}); err != nil {
		return err
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.Plan != domain.PlanPro || snap.PendingPayment != nil {
		return fmt.Errorf("recovered payment did not apply upgrade")
	}
	fmt.Println("PASS payment recovery applies pending upgrade")
	if err := provider.CancelSubscription(ctx, subID, domain.CancelAtPeriodEnd); err != nil {
		return err
	}
	if err := advance(snap.PeriodEnd.Unix() + 60); err != nil {
		return err
	}
	snap, err = provider.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}
	if snap.Status != domain.StatusCanceled {
		return fmt.Errorf("subscription did not cancel at deadline")
	}
	fmt.Println("PASS Test Clock cancels at period end")
	return nil
}
func main() {
	envFile := flag.String("env-file", "templates/quickstart/.env", "test environment file (never printed)")
	execute := flag.Bool("run", false, "create isolated Stripe test objects and execute checks")
	flag.Parse()
	if !*execute {
		fmt.Println("Use -run to execute Stripe test-mode checks; no requests sent.")
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *envFile); err != nil {
		fmt.Println("FAIL", err)
		os.Exit(1)
	}
}
