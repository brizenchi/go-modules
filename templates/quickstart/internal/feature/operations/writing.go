package operations

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/ai"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
	"github.com/brizenchi/quickstart-template/internal/feature/writer"
	"github.com/gin-gonic/gin"
)

// AI drafting and translation for administrators. Results return to the editor
// and are saved through the ordinary article endpoints, so versions and audit
// stay in one place and nothing reaches the public site before review.

// articleWriter builds the writer from deployment configuration on first use.
func (m *Module) articleWriter(ctx context.Context) *writer.Service {
	m.writerOnce.Do(func() {
		if m.writer != nil {
			return
		}
		var client *http.Client
		if m.deps.Modules != nil {
			client = m.deps.Modules.Config.HTTPClient
		}
		model, err := ai.New(m.deps.Config.AI, client)
		if err != nil {
			if !errors.Is(err, ai.ErrDisabled) {
				slog.ErrorContext(ctx, "AI writer unavailable", "error", err)
			}
			m.writer = writer.New(nil)
			return
		}
		m.writer = writer.New(model)
	})
	return m.writer
}

// allowSlowResponse lets a model request outlive the server's write timeout.
func (m *Module) allowSlowResponse(c *gin.Context, requests int) {
	timeout := time.Duration(m.deps.Config.AI.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	deadline := time.Now().Add(time.Duration(requests)*timeout + 15*time.Second)
	if err := http.NewResponseController(c.Writer).SetWriteDeadline(deadline); err != nil {
		slog.WarnContext(c.Request.Context(), "cannot extend AI response deadline", "error", err)
	}
}

func writerFailed(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	ctx := c.Request.Context()
	switch {
	case errors.Is(err, writer.ErrUnavailable):
		httpresp.Custom(c, 503, 503, "AI writing is not configured", gin.H{"reason": "AI_NOT_CONFIGURED"})
	case errors.Is(err, writer.ErrInvalidBrief):
		httpresp.BadRequest(c, "invalid writing request")
	case errors.Is(err, writer.ErrInvalidOutput):
		slog.WarnContext(ctx, "AI reply rejected", "error", err)
		httpresp.Custom(c, 502, 502, "the model reply could not be used", gin.H{"reason": "AI_OUTPUT_INVALID"})
	default:
		slog.ErrorContext(ctx, "AI writing failed", "error", err)
		httpresp.Custom(c, 502, 502, "the AI provider did not respond", gin.H{"reason": "AI_PROVIDER_FAILED"})
	}
	return true
}

// publishedPages lists pages live in locale, the only internal links a draft may use.
func (m *Module) publishedPages(ctx context.Context, locale string) ([]writer.Page, error) {
	rows, err := m.contentService().List(ctx)
	if err != nil {
		return nil, err
	}
	pages := []writer.Page{}
	for _, row := range rows {
		if live, ok := row.Published[locale]; ok {
			pages = append(pages, writer.Page{Path: row.Path, Title: live.Title})
		}
	}
	return pages, nil
}

func (m *Module) draftArticle(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	var brief writer.Brief
	if !decodeBody(c, &brief) {
		return
	}
	ctx := c.Request.Context()
	pages, err := m.publishedPages(ctx, brief.Locale)
	if contentError(c, err) {
		return
	}
	brief.Pages = pages
	m.allowSlowResponse(c, 1)
	draft, err := m.articleWriter(ctx).Draft(ctx, brief)
	if writerFailed(c, err) {
		return
	}
	httpresp.OK(c, draft)
}

func (m *Module) translateArticle(c *gin.Context) {
	var in struct {
		From        string                   `json:"from"`
		To          []string                 `json:"to"`
		Translation content.Translation      `json:"translation"`
		Images      []writer.ImageSuggestion `json:"images"`
	}
	if !decodeBodyLimit(c, &in, 400000) {
		return
	}
	if len(in.Images) > writer.MaxImages {
		httpresp.BadRequest(c, "too many image suggestions")
		return
	}
	targets := []string{}
	for _, locale := range in.To {
		if _, known := writer.Languages[locale]; !known || locale == in.From || slices.Contains(targets, locale) {
			httpresp.BadRequest(c, "choose languages other than the source, each once")
			return
		}
		targets = append(targets, locale)
	}
	if len(targets) == 0 {
		httpresp.BadRequest(c, "choose at least one language")
		return
	}
	ctx := c.Request.Context()
	m.allowSlowResponse(c, len(targets))
	translations := map[string]content.Translation{}
	images := map[string][]writer.ImageSuggestion{}
	for _, locale := range targets {
		translation, translated, err := m.articleWriter(ctx).TranslateWithImages(ctx, in.From, locale, in.Translation, in.Images)
		if writerFailed(c, err) {
			return
		}
		translations[locale] = translation
		images[locale] = translated
	}
	if len(in.Images) == 0 {
		httpresp.OK(c, gin.H{"translations": translations})
		return
	}
	httpresp.OK(c, gin.H{"translations": translations, "images": images})
}
