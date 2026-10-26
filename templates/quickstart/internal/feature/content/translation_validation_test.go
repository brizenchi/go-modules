package content

import (
	"testing"
)

func limitsFixture() (Translation, Translation) {
	zh := Translation{Title: "快速开始", Summary: "五分钟", Body: "安装 SDK。", SEOTitle: "", SEODescription: ""}
	en := Translation{Title: "Quick start", Summary: "Five minutes", Body: "Install the SDK.", SEOTitle: "", SEODescription: ""}
	return zh, en
}

func TestHashTranslationIsCanonicalAndSensitive(t *testing.T) {
	zh, _ := limitsFixture()
	if HashTranslation(zh) != HashTranslation(zh) {
		t.Fatal("hash is not deterministic")
	}
	changed := zh
	changed.Summary = "六分钟"
	if HashTranslation(changed) == HashTranslation(zh) {
		t.Fatal("hash ignores content changes")
	}
	if len(HashTranslation(zh)) != 64 {
		t.Fatalf("hash length=%d", len(HashTranslation(zh)))
	}
}

func TestTranslationLengthCountsChineseCharactersWithoutWhitespace(t *testing.T) {
	zh, _ := limitsFixture()
	// 快速开始(4) + 五分钟(3) + 安装SDK。(6: 安装=2, SDK=3, 。=1) = 13
	if got := CountTranslationCharacters(zh); got != 13 {
		t.Fatalf("characters=%d want 13", got)
	}
	withSpaces := Translation{Title: "标 题", Summary: "摘 要", Body: "正 文\n\n第二段", SEOTitle: "SEO 标题", SEODescription: ""}
	if got := CountTranslationCharacters(withSpaces); got != 14 {
		t.Fatalf("characters with whitespace=%d want 14", got)
	}
}

func TestTranslationLengthCountsEnglishWords(t *testing.T) {
	_, en := limitsFixture()
	// Quick start(2) + Five minutes(2) + Install the SDK.(3) = 7
	if got := CountTranslationWords(en); got != 7 {
		t.Fatalf("words=%d want 7", got)
	}
	multi := Translation{Title: "a  b\tc", Summary: "d", Body: "e\nf", SEOTitle: "g h", SEODescription: ""}
	if got := CountTranslationWords(multi); got != 8 {
		t.Fatalf("words=%d want 8", got)
	}
}

func TestTranslationWithinLimitsAtExactBoundary(t *testing.T) {
	atLimit := Translation{Title: "", Summary: "", Body: repeatRunes("文", 5000), SEOTitle: "", SEODescription: ""}
	if !TranslationWithinLimits(atLimit, "zh", 5000, 3000) {
		t.Fatal("exact zh limit rejected")
	}
	over := Translation{Body: repeatRunes("文", 5001)}
	if TranslationWithinLimits(over, "zh", 5000, 3000) {
		t.Fatal("zh limit+1 accepted")
	}
	words := Translation{Body: repeatWords("word", 3000)}
	if !TranslationWithinLimits(words, "en", 5000, 3000) {
		t.Fatal("exact en limit rejected")
	}
	wordsOver := Translation{Body: repeatWords("word", 3001)}
	if TranslationWithinLimits(wordsOver, "en", 5000, 3000) {
		t.Fatal("en limit+1 accepted")
	}
}

func repeatRunes(char string, count int) string {
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = []rune(char)[0]
	}
	return string(runes)
}

func repeatWords(word string, count int) string {
	result := ""
	for i := range count {
		if i > 0 {
			result += " "
		}
		result += word
	}
	return result
}

func TestMarkSourceStalenessDetectsSourceDriftAndSwitch(t *testing.T) {
	zh, en := limitsFixture()
	base := Article{
		SourceLocale: "zh",
		Translations: map[string]Translation{"zh": zh, "en": en},
		TranslationMeta: map[string]TranslationMeta{
			"en": {SourceLocale: "zh", SourceHash: HashTranslation(zh), CheckStatus: "passed"},
		},
	}
	if got := MarkSourceStaleness(base); got.TranslationSourceUpdated["en"] {
		t.Fatal("unchanged source reported as updated")
	}
	edited := base
	edited.Translations["zh"] = Translation{Title: "改过的标题", Summary: zh.Summary, Body: zh.Body}
	if !MarkSourceStaleness(edited).TranslationSourceUpdated["en"] {
		t.Fatal("changed source draft not reported")
	}
	switched := base
	switched.SourceLocale = "en"
	if !MarkSourceStaleness(switched).TranslationSourceUpdated["en"] {
		t.Fatal("switching the confirmed source must invalidate the check")
	}
	unconfirmed := base
	unconfirmed.SourceLocale = ""
	if !MarkSourceStaleness(unconfirmed).TranslationSourceUpdated["en"] {
		t.Fatal("unconfirmed source must invalidate the check")
	}
	missing := base
	missing.Translations = map[string]Translation{"en": en}
	if !MarkSourceStaleness(missing).TranslationSourceUpdated["en"] {
		t.Fatal("missing source draft must count as updated")
	}
	empty := Article{}
	if got := MarkSourceStaleness(empty); got.TranslationSourceUpdated != nil {
		t.Fatal("articles without meta must not carry staleness data")
	}
}
