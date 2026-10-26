package content

import (
	"context"
	"errors"
)

// Stable failure reasons stored on translation jobs and shown to admins.
const (
	ReasonTranslationUnavailable   = "CONTENT_TRANSLATION_PROVIDER_UNAVAILABLE"
	ReasonTranslationRateLimited   = "CONTENT_TRANSLATION_RATE_LIMITED"
	ReasonTranslationTimeout       = "CONTENT_TRANSLATION_TIMEOUT"
	ReasonTranslationInterrupted   = "CONTENT_TRANSLATION_INTERRUPTED"
	ReasonTranslationIncomplete    = "CONTENT_TRANSLATION_INCOMPLETE"
	ReasonTranslationInvalidOutput = "CONTENT_TRANSLATION_INVALID_OUTPUT"
)

// Stable API error reasons for the translation endpoints.
const (
	ReasonTranslationSourceRequired      = "CONTENT_SOURCE_REQUIRED"
	ReasonTranslationSourceMismatch      = "CONTENT_SOURCE_MISMATCH"
	ReasonTranslationLanguageInvalid     = "CONTENT_TRANSLATION_LANGUAGE_INVALID"
	ReasonTranslationTooLong             = "CONTENT_TRANSLATION_TOO_LONG"
	ReasonTranslationVersionConflict     = "CONTENT_VERSION_CONFLICT"
	ReasonTranslationInProgressConflict  = "CONTENT_TRANSLATION_IN_PROGRESS"
	ReasonTranslationIdempotencyConflict = "CONTENT_TRANSLATION_IDEMPOTENCY_CONFLICT"
	ReasonTranslationDisabledConflict    = "CONTENT_TRANSLATION_DISABLED"
)

// TranslationMeta snapshots how one target draft was produced. It is written
// only by the server and compared by hash to derive staleness; check statuses
// are "passed" and "warning".
type TranslationMeta struct {
	SourceLocale string        `json:"source_locale"`
	SourceHash   string        `json:"source_hash"`
	DraftHash    string        `json:"draft_hash"`
	CheckStatus  string        `json:"check_status"`
	Issues       []ReviewIssue `json:"issues,omitempty"`
	Model        string        `json:"model"`
	CompletedAt  string        `json:"completed_at"`
	JobID        string        `json:"job_id"`
}

var (
	ErrProviderUnavailable   = errors.New("content: translation provider unavailable")
	ErrProviderRateLimited   = errors.New("content: translation provider rate limited")
	ErrProviderTimeout       = errors.New("content: translation provider timeout")
	ErrProviderInterrupted   = errors.New("content: translation interrupted")
	ErrProviderIncomplete    = errors.New("content: translation incomplete")
	ErrProviderInvalidOutput = errors.New("content: translation invalid output")
)

// TranslationFailureReason maps a provider error onto the stable job reason.
func TranslationFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrProviderRateLimited):
		return ReasonTranslationRateLimited
	case errors.Is(err, ErrProviderTimeout):
		return ReasonTranslationTimeout
	case errors.Is(err, ErrProviderInterrupted):
		return ReasonTranslationInterrupted
	case errors.Is(err, ErrProviderIncomplete):
		return ReasonTranslationIncomplete
	case errors.Is(err, ErrProviderInvalidOutput):
		return ReasonTranslationInvalidOutput
	default:
		return ReasonTranslationUnavailable
	}
}

// ReviewIssue is one quality finding. MessageZH is untrusted model output and
// is only rendered as plain text.
type ReviewIssue struct {
	Code      string `json:"code"`
	Field     string `json:"field"`
	MessageZH string `json:"message_zh"`
}

// ReviewResult is the structured outcome of one AI review request.
type ReviewResult struct {
	Passed bool          `json:"passed"`
	Issues []ReviewIssue `json:"issues"`
}

// TranslationProvider performs the bounded generate/review/repair model calls.
// Implementations never touch drafts, published snapshots or task storage.
type TranslationProvider interface {
	Generate(ctx context.Context, source Translation, sourceLocale, targetLocale string) (Translation, error)
	Review(ctx context.Context, source, candidate Translation, sourceLocale, targetLocale string) (ReviewResult, error)
	Repair(ctx context.Context, source, candidate Translation, review ReviewResult, sourceLocale, targetLocale string) (Translation, error)
}

func validTranslationLocales(source, target string) bool {
	return (source == "zh" || source == "en") && (target == "zh" || target == "en") && source != target
}
