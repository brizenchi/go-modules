// Package writer drafts and translates website articles with the host's AI model.
// It stores nothing: an administrator reviews the result and saves it as a draft
// through the content service, which keeps validation, versions and audit.
package writer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/brizenchi/go-modules/foundation/aix"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
)

var (
	// ErrUnavailable means AI writing is not configured for this deployment.
	ErrUnavailable = errors.New("writer: AI model not configured")
	// ErrInvalidBrief identifies a request the writer cannot act on.
	ErrInvalidBrief = errors.New("writer: invalid brief")
	// ErrInvalidOutput means the model reply was cut off or broke the article rules.
	ErrInvalidOutput = errors.New("writer: invalid model output")
)

// Languages the content service can store, by locale code. Adding a site
// language adds it here and to the content service.
var Languages = map[string]string{"en": "English", "zh": "Simplified Chinese", "ja": "Japanese", "ko": "Korean", "es": "Spanish", "fr": "French", "de": "German"}

// Kinds describes each article section to the model.
var Kinds = map[string]string{
	"features":  "a product feature page: what the capability does, how customers use it and its limits",
	"use-cases": "a use-case page: how a kind of business applies the product to reach an outcome",
	"compare":   "a comparison page: how the options differ and when each is the better choice",
	"docs":      "a documentation page: step-by-step instructions to configure or use the product",
	"blog":      "an article answering a question people search for",
}

// Page is a published page the draft may link to.
type Page struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

// Brief is what the administrator asks for.
type Brief struct {
	Kind    string `json:"kind"`
	Keyword string `json:"keyword"`
	Notes   string `json:"notes"`
	Locale  string `json:"locale"`
	// Pages are the published pages internal links may use; others are removed.
	Pages []Page `json:"-"`
}

// ImageSuggestion tells the editor where an uploaded image would help.
type ImageSuggestion struct {
	Section     string `json:"section"`
	Alt         string `json:"alt"`
	Description string `json:"description"`
}

// Draft is a reviewed-before-saving article proposal.
type Draft struct {
	Path        string              `json:"path"`
	Translation content.Translation `json:"translation"`
	Images      []ImageSuggestion   `json:"images"`
}

// Service turns briefs into drafts and drafts into other languages.
type Service struct{ model aix.Generator }

// New requires a configured model; nil means writing is unavailable.
func New(model aix.Generator) *Service { return &Service{model: model} }

const (
	maxKeyword = 120
	maxNotes   = 4000
	maxPages   = 80
)

// MaxImages bounds the image suggestions of one article.
const MaxImages = 6

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Draft writes one article in the brief's language.
func (s *Service) Draft(ctx context.Context, brief Brief) (Draft, error) {
	if s == nil || s.model == nil {
		return Draft{}, ErrUnavailable
	}
	brief.Keyword, brief.Notes = strings.TrimSpace(brief.Keyword), strings.TrimSpace(brief.Notes)
	language, known := Languages[brief.Locale]
	section, kind := Kinds[brief.Kind]
	if !known || !kind || brief.Keyword == "" || utf8.RuneCountInString(brief.Keyword) > maxKeyword || utf8.RuneCountInString(brief.Notes) > maxNotes || !utf8.ValidString(brief.Notes) {
		return Draft{}, ErrInvalidBrief
	}
	pages := brief.Pages
	if len(pages) > maxPages {
		pages = pages[:maxPages]
	}
	links, _ := json.Marshal(pages)
	prompt := fmt.Sprintf("Section: %s\nTarget search keyword: %s\nFacts and notes from the editor (the only product facts you may state):\n%s\nPublished pages you may link to (JSON):\n%s",
		section, brief.Keyword, orNone(brief.Notes), links)
	reply, err := s.complete(ctx, draftInstructions(language), prompt)
	if err != nil {
		return Draft{}, err
	}
	var out struct {
		Slug           string            `json:"slug"`
		Title          string            `json:"title"`
		Summary        string            `json:"summary"`
		SEOTitle       string            `json:"seo_title"`
		SEODescription string            `json:"seo_description"`
		Body           string            `json:"body"`
		Images         []ImageSuggestion `json:"images"`
	}
	if err := decode(reply, &out); err != nil {
		return Draft{}, err
	}
	allowed := map[string]bool{}
	for _, page := range brief.Pages {
		allowed[page.Path] = true
	}
	// Images come from uploads only, so an image the model writes is dropped.
	body := keepLinks(imagePattern.ReplaceAllString(out.Body, ""), allowed)
	translation := content.Translation{Title: out.Title, Summary: out.Summary, SEOTitle: out.SEOTitle, SEODescription: out.SEODescription, Body: body}
	path := "/" + brief.Kind + "/" + strings.TrimSpace(out.Slug)
	if !slugPattern.MatchString(strings.TrimSpace(out.Slug)) || !content.ValidArticlePath(path) || !validTranslation(&translation) {
		return Draft{}, fmt.Errorf("%w: draft fields", ErrInvalidOutput)
	}
	images := []ImageSuggestion{}
	for _, image := range out.Images {
		if image, ok := validImage(image); ok && len(images) < MaxImages {
			images = append(images, image)
		}
	}
	return Draft{Path: path, Translation: translation, Images: images}, nil
}

// Translate rewrites a draft in another language, keeping every link and image.
func (s *Service) Translate(ctx context.Context, from, to string, source content.Translation) (content.Translation, error) {
	translation, _, err := s.TranslateWithImages(ctx, from, to, source, nil)
	return translation, err
}

// TranslateWithImages also translates the draft's image suggestions, so an
// image uploaded once gets alt text and a heading in every language. The
// suggestions come back in the same order; none in means none out.
func (s *Service) TranslateWithImages(ctx context.Context, from, to string, source content.Translation, images []ImageSuggestion) (content.Translation, []ImageSuggestion, error) {
	if s == nil || s.model == nil {
		return content.Translation{}, nil, ErrUnavailable
	}
	fromName, okFrom := Languages[from]
	toName, okTo := Languages[to]
	if !okFrom || !okTo || from == to || strings.TrimSpace(source.Title) == "" || strings.TrimSpace(source.Body) == "" || len(images) > MaxImages {
		return content.Translation{}, nil, ErrInvalidBrief
	}
	request := map[string]any{"title": source.Title, "summary": source.Summary, "seo_title": source.SEOTitle, "seo_description": source.SEODescription, "body": source.Body}
	if len(images) > 0 {
		request["images"] = images
	}
	input, _ := json.Marshal(request)
	reply, err := s.complete(ctx, translateInstructions(fromName, toName, len(images) > 0), string(input))
	if err != nil {
		return content.Translation{}, nil, err
	}
	var out struct {
		Title          string            `json:"title"`
		Summary        string            `json:"summary"`
		SEOTitle       string            `json:"seo_title"`
		SEODescription string            `json:"seo_description"`
		Body           string            `json:"body"`
		Images         []ImageSuggestion `json:"images"`
	}
	if err := decode(reply, &out); err != nil {
		return content.Translation{}, nil, err
	}
	// Images must survive unchanged; links may only point where the source does.
	if !sameImages(source.Body, out.Body) {
		return content.Translation{}, nil, fmt.Errorf("%w: images changed", ErrInvalidOutput)
	}
	var translatedImages []ImageSuggestion
	if len(images) > 0 {
		if len(out.Images) != len(images) {
			return content.Translation{}, nil, fmt.Errorf("%w: image suggestions changed", ErrInvalidOutput)
		}
		for _, image := range out.Images {
			if image, ok := validImage(image); ok {
				translatedImages = append(translatedImages, image)
			} else {
				return content.Translation{}, nil, fmt.Errorf("%w: image suggestion fields", ErrInvalidOutput)
			}
		}
	}
	allowed := map[string]bool{}
	for _, target := range linkTargets(source.Body) {
		allowed[target] = true
	}
	translation := content.Translation{Title: out.Title, Summary: out.Summary, SEOTitle: out.SEOTitle, SEODescription: out.SEODescription, Body: keepLinks(out.Body, allowed)}
	if !validTranslation(&translation) {
		return content.Translation{}, nil, fmt.Errorf("%w: translation fields", ErrInvalidOutput)
	}
	return translation, translatedImages, nil
}

// validImage trims a suggestion and checks the limits the editor relies on.
func validImage(image ImageSuggestion) (ImageSuggestion, bool) {
	image.Section, image.Alt, image.Description = strings.TrimSpace(image.Section), strings.TrimSpace(image.Alt), strings.TrimSpace(image.Description)
	return image, image.Alt != "" && utf8.RuneCountInString(image.Alt) <= 125 && utf8.RuneCountInString(image.Section) <= 160 && utf8.RuneCountInString(image.Description) <= 300
}

func (s *Service) complete(ctx context.Context, instructions, input string) (string, error) {
	result, err := s.model.Complete(ctx, []aix.Message{{Role: "system", Content: instructions}, {Role: "user", Content: input}})
	if err != nil {
		return "", fmt.Errorf("writer: complete: %w", err)
	}
	if result.FinishReason == "length" {
		return "", fmt.Errorf("%w: output cut off at the token limit", ErrInvalidOutput)
	}
	return result.Content, nil
}

func orNone(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}

// decode accepts a JSON object optionally wrapped in a Markdown code fence.
func decode(reply string, target any) error {
	start, end := strings.Index(reply, "{"), strings.LastIndex(reply, "}")
	if start < 0 || end < start || json.Unmarshal([]byte(reply[start:end+1]), target) != nil {
		return fmt.Errorf("%w: not a JSON object", ErrInvalidOutput)
	}
	return nil
}

// Limits match the content service so a draft always saves.
func validTranslation(t *content.Translation) bool {
	for _, field := range []*string{&t.Title, &t.Summary, &t.SEOTitle, &t.SEODescription, &t.Body} {
		*field = strings.TrimSpace(*field)
		if !utf8.ValidString(*field) {
			return false
		}
	}
	return t.Title != "" && t.Summary != "" && t.Body != "" &&
		utf8.RuneCountInString(t.Title) <= 160 && utf8.RuneCountInString(t.Summary) <= 500 &&
		utf8.RuneCountInString(t.SEOTitle) <= 160 && utf8.RuneCountInString(t.SEODescription) <= 500 && len(t.Body) <= 150000 &&
		!strings.ContainsAny(t.Title+t.SEOTitle, "\r\n")
}

var (
	linkPattern  = regexp.MustCompile(`(!?)\[([^\]\n]*)\]\(([^)\s]+)\)`)
	imagePattern = regexp.MustCompile(`!\[[^\]\n]*\]\(([^)\s]+)\)`)
)

// keepLinks turns links outside allowed into their plain text and leaves images alone.
func keepLinks(body string, allowed map[string]bool) string {
	return linkPattern.ReplaceAllStringFunc(body, func(match string) string {
		parts := linkPattern.FindStringSubmatch(match)
		if parts[1] == "!" || allowed[strings.SplitN(strings.SplitN(parts[3], "#", 2)[0], "?", 2)[0]] {
			return match
		}
		return parts[2]
	})
}

func linkTargets(body string) []string {
	targets := []string{}
	for _, parts := range linkPattern.FindAllStringSubmatch(body, -1) {
		if parts[1] == "" {
			targets = append(targets, strings.SplitN(strings.SplitN(parts[3], "#", 2)[0], "?", 2)[0])
		}
	}
	return targets
}

func sameImages(source, translated string) bool {
	count := func(body string) map[string]int {
		seen := map[string]int{}
		for _, parts := range imagePattern.FindAllStringSubmatch(body, -1) {
			seen[parts[1]]++
		}
		return seen
	}
	a, b := count(source), count(translated)
	if len(a) != len(b) {
		return false
	}
	for url, n := range a {
		if b[url] != n {
			return false
		}
	}
	return true
}

func draftInstructions(language string) string {
	return `You are the SEO editor of a SaaS website. Write one article in ` + language + `.
Reply with one JSON object only, no commentary, with these string keys: slug, title, summary, seo_title, seo_description, body, and "images" (an array).
Rules:
- slug: lowercase English words joined by hyphens that describe the keyword, at most 60 characters.
- title: the page heading. Put the target keyword near the start. At most 60 characters (30 for Chinese).
- seo_title: the search result title, 50-60 characters (25-30 for Chinese), keyword first, different wording from title.
- seo_description: 150-160 characters (70-80 for Chinese) with the keyword and a reason to click.
- summary: one or two sentences shown under the heading.
- body: Markdown using only "## " and "### " headings, paragraphs, "- " or "1. " lists, "> " quotes, **bold** and [links](/path). No "# " heading, tables, HTML or images.
  The first paragraph answers what the reader searched for within its first 100 words and contains the keyword.
  Use 3-6 "## " sections whose headings match how people search. End with a "## " FAQ section (in the article language) of 3-5 "### " questions, each followed by a short answer.
  Write at least 800 words (at least 1500 characters for Chinese). Short paragraphs.
  Add 2-4 links to pages from the supplied published pages list, with descriptive anchor text. Do not link anywhere else.
- Only state product facts that appear in the editor's notes or the page titles. Never invent statistics, prices, customers, testimonials or capabilities.
- images: 2-4 objects {"section": the exact text of the "## " heading the image belongs under, "alt": concise alt text in the article language, "description": what the image should show}.`
}

func translateInstructions(from, to string, images bool) string {
	keys := `title, summary, seo_title, seo_description and body`
	imageRule := ""
	if images {
		keys += `, and "images": suggestions for pictures the editor will upload`
		imageRule = "\n- images: the same number of objects in the same order. Translate alt and description; set section to your translation of that \"## \" heading exactly as it appears in your body (keep \"\" when empty)."
	}
	return `Translate a website article from ` + from + ` to ` + to + ` for native readers searching in ` + to + `.
The input is a JSON object with keys ` + keys + `. Reply with one JSON object only with the same keys.
Rules:
- Keep the Markdown structure: the same headings, lists, quotes and paragraphs in the same order. Do not add or remove information.
- Keep every link and image URL exactly as written. Translate link text and image alt text.
- Keep product and brand names unchanged.
- Write natural, fluent ` + to + `, not word-for-word. Use the terms people actually search for in ` + to + `.
- seo_title: 50-60 characters (25-30 for Chinese, Japanese and Korean). seo_description: 150-160 characters (70-80 for Chinese, Japanese and Korean).` + imageRule
}
