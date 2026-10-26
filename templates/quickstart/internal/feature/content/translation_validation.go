package content

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode"
)

// HashTranslation hashes the canonical JSON encoding of a translation. Struct
// fields marshal in a fixed order, so identical content always hashes equally.
func HashTranslation(t Translation) string {
	raw, _ := json.Marshal(t)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// translationTextFields lists the fields contributing to length limits. Empty
// optional SEO fields stay empty and do not count.
func translationTextFields(t Translation) []string {
	fields := []string{t.Title, t.Summary, t.Body}
	if strings.TrimSpace(t.SEOTitle) != "" {
		fields = append(fields, t.SEOTitle)
	}
	if strings.TrimSpace(t.SEODescription) != "" {
		fields = append(fields, t.SEODescription)
	}
	return fields
}

// CountTranslationCharacters counts non-whitespace Unicode characters,
// punctuation and Markdown included, for the zh product limit.
func CountTranslationCharacters(t Translation) int {
	count := 0
	for _, field := range translationTextFields(t) {
		for _, r := range field {
			if !unicode.IsSpace(r) {
				count++
			}
		}
	}
	return count
}

// CountTranslationWords counts whitespace-separated non-empty tokens for the
// en product limit.
func CountTranslationWords(t Translation) int {
	count := 0
	for _, field := range translationTextFields(t) {
		count += len(strings.Fields(field))
	}
	return count
}

// MarkSourceStaleness reports which target locales were checked against a
// source that no longer applies: the confirmed source language changed, was
// cleared, or its draft content was edited. Presentation-only, like
// ActiveTranslationJobID; public reads never expose it.
func MarkSourceStaleness(a Article) Article {
	if len(a.TranslationMeta) == 0 {
		return a
	}
	updated := make(map[string]bool, len(a.TranslationMeta))
	for locale, meta := range a.TranslationMeta {
		if a.SourceLocale == "" || a.SourceLocale == locale {
			updated[locale] = true
			continue
		}
		source, ok := a.Translations[meta.SourceLocale]
		updated[locale] = !ok || HashTranslation(source) != meta.SourceHash
	}
	a.TranslationSourceUpdated = updated
	return a
}

// TranslationWithinLimits applies the source-locale product bound: zh counts
// characters, en counts words. The backend verdict is authoritative.
func TranslationWithinLimits(t Translation, locale string, maxZhCharacters, maxEnWords int) bool {
	switch locale {
	case "zh":
		return CountTranslationCharacters(t) <= maxZhCharacters
	case "en":
		return CountTranslationWords(t) <= maxEnWords
	default:
		return false
	}
}
