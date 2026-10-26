package content

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// TranslationPipeline runs one claimed job through the bounded
// generate → program check → review → (repair → recheck) flow. All model HTTP
// calls happen outside any database transaction; the repository commits the
// final short transaction.
type TranslationPipeline struct {
	service  *Service
	provider TranslationProvider
	model    string
}

func NewTranslationPipeline(service *Service, provider TranslationProvider, model string) *TranslationPipeline {
	return &TranslationPipeline{service: service, provider: provider, model: model}
}

func (p *TranslationPipeline) review(ctx context.Context, job TranslationJob, source, candidate Translation) (ReviewResult, error) {
	if err := p.service.UpdateTranslationJobPhase(ctx, job, "reviewing"); err != nil {
		return ReviewResult{}, err
	}
	return p.provider.Review(ctx, source, candidate, job.SourceLocale, job.TargetLocale)
}

func (p *TranslationPipeline) fail(ctx context.Context, job TranslationJob, err error) error {
	reason := TranslationFailureReason(err)
	// The job row must reach its terminal state even though the pipeline
	// context is being cancelled; WithoutCancel keeps the trace linkage.
	if failErr := p.service.FailTranslationJob(context.WithoutCancel(ctx), job, reason); failErr != nil {
		return failErr
	}
	return err
}

// Run executes the whole bounded flow for one leased job. Quality warnings
// complete the job with a warning check status; only technical failures
// return with the job marked failed.
func (p *TranslationPipeline) Run(ctx context.Context, job TranslationJob) error {
	source := job.SourceTranslation()
	candidate, err := p.provider.Generate(ctx, source, job.SourceLocale, job.TargetLocale)
	if err != nil {
		return p.fail(ctx, job, err)
	}
	review, err := p.review(ctx, job, source, candidate)
	if err != nil {
		return p.fail(ctx, job, err)
	}
	issues := append(programmaticIssues(source, candidate), review.Issues...)
	if review.Passed && len(issues) == 0 {
		_, err := p.service.CompleteTranslationJob(ctx, job, candidate, "passed", nil, p.model)
		return p.completeOutcome(job, err)
	}
	if err := p.service.UpdateTranslationJobPhase(ctx, job, "repairing"); err != nil {
		return err
	}
	repaired, err := p.provider.Repair(ctx, source, candidate, ReviewResult{Passed: false, Issues: issues}, job.SourceLocale, job.TargetLocale)
	if err != nil {
		return p.fail(ctx, job, err)
	}
	recheck, err := p.review(ctx, job, source, repaired)
	if err != nil {
		return p.fail(ctx, job, err)
	}
	finalIssues := append(programmaticIssues(source, repaired), recheck.Issues...)
	if recheck.Passed && len(finalIssues) == 0 {
		_, err := p.service.CompleteTranslationJob(ctx, job, repaired, "passed", nil, p.model)
		return p.completeOutcome(job, err)
	}
	_, err = p.service.CompleteTranslationJob(ctx, job, repaired, "warning", boundIssues(finalIssues), p.model)
	return p.completeOutcome(job, err)
}

func (p *TranslationPipeline) completeOutcome(job TranslationJob, err error) error {
	// Version conflicts and lost leases already put the job in a terminal
	// state inside the repository; surfacing the error is for worker logs.
	if errors.Is(err, ErrVersion) || errors.Is(err, ErrTranslationLease) {
		slog.Warn("translation job could not apply its result", "job_id", job.ID, "error", err)
		return err
	}
	return err
}

func boundIssues(issues []ReviewIssue) []ReviewIssue {
	if len(issues) > 20 {
		return issues[:20]
	}
	return issues
}

var (
	fenceLinePattern   = regexp.MustCompile("(?m)^```(.*)$")
	inlineCodePattern  = regexp.MustCompile("`[^`\n]+`")
	linkTargetPattern  = regexp.MustCompile(`\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	placeholderPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)
	numberPattern      = regexp.MustCompile(`\d+(?:[.,]\d+)*`)
	translationFields  = []string{"title", "summary", "body", "seo_title", "seo_description"}
)

// fenceStructure reports how many ``` lines a text has and the info strings
// (language tags) of the opening fences, so translated comments inside a
// fence still count as structurally preserved.
func fenceStructure(text string) (int, []string) {
	lines := fenceLinePattern.FindAllStringSubmatch(text, -1)
	count := len(lines)
	info := make([]string, 0, count/2+1)
	for index, match := range lines {
		if index%2 == 0 {
			info = append(info, strings.TrimSpace(match[1]))
		}
	}
	return count, info
}

func translationFieldValue(t Translation, field string) string {
	switch field {
	case "title":
		return t.Title
	case "summary":
		return t.Summary
	case "body":
		return t.Body
	case "seo_title":
		return t.SEOTitle
	case "seo_description":
		return t.SEODescription
	}
	return ""
}

// programmaticIssues compares structural invariants the translation must
// preserve: fence structure (count and language tags — fence content may be
// translated for comments), inline code, link destinations, explicit
// placeholders and numeric values (matched as numbers, not strings, so
// 1,024 and 1024 or locale formatting do not create false alarms).
func programmaticIssues(source, candidate Translation) []ReviewIssue {
	var issues []ReviewIssue
	for _, field := range translationFields {
		src := translationFieldValue(source, field)
		dst := translationFieldValue(candidate, field)
		issues = append(issues, structuralFieldIssues(field, src, dst)...)
	}
	return boundIssues(issues)
}

func structuralFieldIssues(field, source, candidate string) []ReviewIssue {
	var issues []ReviewIssue
	add := func(code, message string) {
		issues = append(issues, ReviewIssue{Code: code, Field: field, MessageZH: truncateRunes(message, 300)})
	}
	sourceCount, sourceInfo := fenceStructure(source)
	candidateCount, candidateInfo := fenceStructure(candidate)
	if sourceCount != candidateCount || !slices.Equal(sourceInfo, candidateInfo) {
		add("format", "代码块结构缺失或被改写")
	}
	for _, value := range inlineCodePattern.FindAllString(source, -1) {
		if !strings.Contains(candidate, value) {
			add("format", "行内代码缺失: "+value)
		}
	}
	for _, match := range linkTargetPattern.FindAllStringSubmatch(source, -1) {
		if !strings.Contains(candidate, match[1]) {
			add("format", "链接目的地址缺失: "+match[1])
		}
	}
	for _, value := range placeholderPattern.FindAllString(source, -1) {
		if !strings.Contains(candidate, value) {
			add("format", "占位符缺失: "+value)
		}
	}
	sourceNumbers := numericValues(source)
	candidateNumbers := numericValues(candidate)
	for _, missing := range unmatchedNumbers(sourceNumbers, candidateNumbers) {
		add("meaning", "数字缺失: "+strconv.FormatFloat(missing, 'f', -1, 64))
	}
	for _, extra := range unmatchedNumbers(candidateNumbers, sourceNumbers) {
		add("addition", "新增数字: "+strconv.FormatFloat(extra, 'f', -1, 64))
	}
	return issues
}

func numericValues(text string) []float64 {
	matches := numberPattern.FindAllString(text, -1)
	values := make([]float64, 0, len(matches))
	for _, match := range matches {
		normalized := strings.ReplaceAll(match, ",", "")
		if value, err := strconv.ParseFloat(normalized, 64); err == nil {
			values = append(values, value)
		}
	}
	return values
}

// unmatchedNumbers returns values from left without an equal partner in
// right; equality tolerates floating point rounding only.
func unmatchedNumbers(left, right []float64) []float64 {
	remaining := append([]float64(nil), right...)
	var missing []float64
	for _, value := range left {
		found := -1
		for i, candidate := range remaining {
			if math.Abs(value-candidate) < 1e-9 {
				found = i
				break
			}
		}
		if found >= 0 {
			remaining = append(remaining[:found], remaining[found+1:]...)
			continue
		}
		missing = append(missing, value)
	}
	return missing
}
