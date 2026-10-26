package operations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/brizenchi/quickstart-template/internal/feature/content"
	"github.com/gin-gonic/gin"
)

type translationCreateRequest struct {
	Version      int64  `json:"version"`
	SourceLocale string `json:"source_locale"`
	TargetLocale string `json:"target_locale"`
	Reason       string `json:"reason"`
}

// translationError extends the shared contentError mapping with the
// translation-specific 409 reasons; response shapes stay identical.
func translationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, content.ErrTranslationInProgress):
		httpresp.Custom(c, http.StatusConflict, http.StatusConflict, "translation already in progress", gin.H{"reason": content.ReasonTranslationInProgressConflict})
	case errors.Is(err, audit.ErrConflict):
		httpresp.Custom(c, http.StatusConflict, http.StatusConflict, "idempotency key reused for a different request", gin.H{"reason": content.ReasonTranslationIdempotencyConflict})
	case errors.Is(err, content.ErrInvalid):
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "source locale does not match the saved article", gin.H{"reason": content.ReasonTranslationSourceMismatch})
	default:
		return contentError(c, err)
	}
	return true
}

// createTranslation only enqueues work; it never waits for the model. The
// source snapshot always comes from the saved article, never the request.
func (m *Module) createTranslation(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	cfg := m.deps.Config.ContentTranslation
	if !cfg.Enabled {
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "content translation is not enabled", gin.H{"reason": content.ReasonTranslationDisabledConflict})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var in translationCreateRequest
	if !decodeBody(c, &in) {
		return
	}
	key, ok := operationMetadata(c, in.Reason)
	if !ok {
		return
	}
	service := m.contentService()
	article, err := service.Read(c.Request.Context(), c.Param("id"))
	if contentError(c, err) {
		return
	}
	switch {
	case article.SourceLocale == "":
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "source locale not confirmed yet", gin.H{"reason": content.ReasonTranslationSourceRequired})
		return
	case article.SourceLocale != in.SourceLocale:
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "source locale does not match the saved article", gin.H{"reason": content.ReasonTranslationSourceMismatch})
		return
	}
	if (in.TargetLocale != "zh" && in.TargetLocale != "en") || in.TargetLocale == in.SourceLocale {
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "unsupported target language", gin.H{"reason": content.ReasonTranslationLanguageInvalid})
		return
	}
	source, exists := article.Translations[in.SourceLocale]
	if !exists {
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "source translation missing", gin.H{"reason": content.ReasonTranslationSourceMismatch})
		return
	}
	if !content.TranslationWithinLimits(source, in.SourceLocale, cfg.MaxZhCharacters, cfg.MaxEnWords) {
		httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "source exceeds the translation length limit", gin.H{"reason": content.ReasonTranslationTooLong})
		return
	}
	job, err := service.CreateTranslationJob(c.Request.Context(), content.TranslationJobRequest{
		ArticleID:      article.ID,
		ActorID:        userID(c),
		IdempotencyKey: key,
		Reason:         in.Reason,
		SourceLocale:   in.SourceLocale,
		TargetLocale:   in.TargetLocale,
		ArticleVersion: in.Version,
		Source:         source,
	})
	if translationError(c, err) {
		return
	}
	httpresp.Custom(c, http.StatusAccepted, http.StatusAccepted, "accepted", gin.H{
		"id": job.ID, "article_id": job.ArticleID, "status": job.Status, "phase": job.Phase,
	})
}

// translationJob reports progress; querying a job successfully does not mean
// the translation itself succeeded.
func (m *Module) translationJob(c *gin.Context) {
	if m.db(c) == nil {
		return
	}
	job, err := m.contentService().TranslationJob(c.Request.Context(), c.Param("id"))
	if translationError(c, err) {
		return
	}
	response := gin.H{
		"id": job.ID, "article_id": job.ArticleID, "status": job.Status, "phase": job.Phase,
		"article_version": job.ArticleVersion, "created_at": job.CreatedAt, "updated_at": job.UpdatedAt,
	}
	if job.Status == "failed" {
		response["reason"] = job.ErrorReason
	}
	if job.Status == "succeeded" && strings.TrimSpace(job.ResultMeta) != "" {
		var meta content.TranslationMeta
		if json.Unmarshal([]byte(job.ResultMeta), &meta) == nil {
			response["check_status"] = meta.CheckStatus
			if len(meta.Issues) > 0 {
				response["issues"] = meta.Issues
			}
		}
	}
	httpresp.OK(c, response)
}

// translationSettings exposes only non-sensitive client hints.
func (m *Module) translationSettings(c *gin.Context) {
	cfg := m.deps.Config.ContentTranslation
	httpresp.OK(c, gin.H{
		"enabled": cfg.Enabled, "locales": []string{"en", "zh"},
		"max_zh_characters": cfg.MaxZhCharacters, "max_en_words": cfg.MaxEnWords,
	})
}
