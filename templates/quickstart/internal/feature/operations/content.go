package operations

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
	"github.com/gin-gonic/gin"
)

func (m *Module) contentService() *content.Service { return content.New(m.deps.DB, m.deps.Audit) }
func contentError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, content.ErrNotFound):
		httpresp.NotFound(c, "article not found")
	case errors.Is(err, content.ErrInvalid):
		httpresp.BadRequest(c, "invalid article or publication")
	case errors.Is(err, content.ErrVersion), errors.Is(err, audit.ErrConflict):
		httpresp.Custom(c, 409, 409, "article changed", gin.H{"reason": "CONTENT_VERSION_CONFLICT"})
	default:
		queryFailed(c, err)
	}
	return true
}
func (m *Module) articles(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	if m.db(c) == nil {
		return
	}
	rows, err := m.contentService().List(c.Request.Context())
	if contentError(c, err) {
		return
	}
	items := []content.Article{}
	for _, row := range rows {
		raw := strings.ToLower(stringJSON(row.Translations))
		if p.query == "" || strings.Contains(strings.ToLower(row.Path)+raw, strings.ToLower(p.query)) {
			items = append(items, row)
		}
	}
	total := len(items)
	start := min((p.page-1)*p.limit, total)
	end := min(start+p.limit, total)
	pageResponse(c, p, items[start:end], int64(total))
}
func (m *Module) article(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	service := m.contentService()
	row, err := service.Read(c.Request.Context(), c.Param("id"))
	if contentError(c, err) {
		return
	}
	if jobID, err := service.ActiveTranslationJobID(c.Request.Context(), row.ID); err == nil {
		row.ActiveTranslationJobID = jobID
	}
	httpresp.OK(c, content.MarkSourceStaleness(row))
}
func (m *Module) saveArticle(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 400000)
	var in content.Save
	if !decodeContentBody(c, &in) {
		return
	}
	key, ok := operationMetadata(c, in.Reason)
	if !ok {
		return
	}
	row, err := m.contentService().Save(c.Request.Context(), c.Param("id"), userID(c), key, in)
	if !contentError(c, err) {
		httpresp.OK(c, content.MarkSourceStaleness(row))
	}
}
func (m *Module) publishArticle(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	var in content.Publish
	if !decodeBody(c, &in) {
		return
	}
	key, ok := operationMetadata(c, in.Reason)
	if !ok {
		return
	}
	row, err := m.contentService().Publish(c.Request.Context(), c.Param("id"), userID(c), key, in)
	if !contentError(c, err) {
		httpresp.OK(c, content.MarkSourceStaleness(row))
	}
}

func (m *Module) importArticles(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	var in struct {
		Apply  bool   `json:"apply"`
		Reason string `json:"reason"`
	}
	if !decodeBody(c, &in) {
		return
	}
	key := ""
	if in.Apply {
		var ok bool
		key, ok = operationMetadata(c, in.Reason)
		if !ok {
			return
		}
	}
	result, err := m.contentService().ImportStarter(c.Request.Context(), userID(c), key, in.Reason, in.Apply)
	if !contentError(c, err) {
		httpresp.OK(c, result)
	}
}

// Public views contain published snapshots only. An absent translation never
// falls back to a draft, starter file or another language.
var markdownReference = regexp.MustCompile(`\]\((/[^\s)]+)\)`)

func publicArticle(row content.Article, locale string, body bool) gin.H {
	live := row.Published[locale]
	translation := live.Translation
	references := []string{}
	seen := map[string]bool{}
	for _, match := range markdownReference.FindAllStringSubmatch(translation.Body, -1) {
		path := strings.SplitN(strings.SplitN(match[1], "#", 2)[0], "?", 2)[0]
		if !seen[path] && !strings.HasPrefix(path, "//") {
			references = append(references, path)
			seen[path] = true
		}
	}
	if !body {
		translation.Body = ""
	}
	locales := []string{}
	for _, l := range []string{"en", "zh"} {
		if _, ok := row.Published[l]; ok {
			locales = append(locales, l)
		}
	}
	return gin.H{"references": references, "id": row.ID, "path": row.Path, "kind": row.Kind, "position": row.Position, "version": row.Version, "translation": translation, "published_at": live.PublishedAt, "updated_at": row.UpdatedAt, "locales": locales}
}
func (m *Module) publicArticles(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	if m.db(c) == nil {
		return
	}
	locale := c.DefaultQuery("locale", "en")
	if locale != "en" && locale != "zh" {
		httpresp.BadRequest(c, "unsupported locale")
		return
	}
	rows, err := m.contentService().List(c.Request.Context())
	if contentError(c, err) {
		return
	}
	items := []gin.H{}
	for _, row := range rows {
		if _, ok := row.Published[locale]; !ok {
			continue
		}
		if kind := c.Query("kind"); kind != "" && kind != row.Kind {
			continue
		}
		items = append(items, publicArticle(row, locale, false))
	}
	total := len(items)
	start := min((p.page-1)*p.limit, total)
	end := min(start+p.limit, total)
	pageResponse(c, p, items[start:end], int64(total))
}
func (m *Module) resolveArticle(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	locale := c.DefaultQuery("locale", "en")
	row, err := m.contentService().Resolve(c.Request.Context(), c.Query("path"), locale)
	if contentError(c, err) {
		return
	}
	if row == nil {
		httpresp.OK(c, gin.H{"article": nil})
		return
	}
	httpresp.OK(c, gin.H{"article": publicArticle(*row, locale, true)})
}
