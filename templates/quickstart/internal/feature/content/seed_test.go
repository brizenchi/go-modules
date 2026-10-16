package content

import (
	"regexp"
	"testing"
)

func TestStarterContentUsesProductCopyWithoutArchitecturePromotion(t *testing.T) {
	architecture := regexp.MustCompile(`(?i)Next\.js|\bGo (API|modules)\b|技术栈|技术架构`)
	for _, row := range Seed() {
		for locale, text := range row.Translations {
			for _, value := range []string{text.Title, text.Summary, text.Body, text.SEOTitle, text.SEODescription} {
				if architecture.MatchString(value) {
					t.Errorf("%s %s still promotes the implementation stack", row.Path, locale)
				}
			}
		}
	}
}
