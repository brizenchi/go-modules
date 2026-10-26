package content

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

// Real thinking-model responses carry a thought part before the answer part.
func geminiThinkingCandidate(thought, answer, finishReason string) string {
	payload := map[string]any{
		"candidates": []map[string]any{{
			"content":      map[string]any{"parts": []map[string]any{{"text": thought, "thought": true}, {"text": answer}}},
			"finishReason": finishReason,
		}},
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func geminiCandidate(text, finishReason string) string {
	payload := struct {
		Candidates []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		} `json:"candidates"`
	}{
		Candidates: []struct {
			Content      geminiContent `json:"content"`
			FinishReason string        `json:"finishReason"`
		}{{
			Content:      geminiContent{Parts: []geminiPart{{Text: "```json\n" + text + "\n```"}}},
			FinishReason: finishReason,
		}},
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func testGeminiConfig() GeminiConfig {
	return GeminiConfig{
		BaseURL:         "https://api.aigocode.dev",
		APIKey:          "aistudio_test_not-a-real-key",
		Model:           "gemini-3.8-flash",
		MaxOutputTokens: 16384,
		Timeout:         90 * time.Second,
	}
}

func testSource() Translation {
	return Translation{Title: "快速开始", Summary: "五分钟跑起来", Body: "# 快速开始\n\n安装 [SDK](https://example.com)。", SEOTitle: "", SEODescription: ""}
}

const validTranslationJSON = `{"title":"Quick Start","summary":"Run it in five minutes","body":"# Quick Start\n\nInstall the [SDK](https://example.com).","seo_title":"","seo_description":""}`

func TestGeminiConfigRejectsInvalidTransportSettings(t *testing.T) {
	for _, base := range []string{"", "http://api.example.com", "https://user:pass@api.example.com", "https://api.example.com?key=value", "https://api.example.com#fragment", "://invalid"} {
		cfg := testGeminiConfig()
		cfg.BaseURL = base
		if err := cfg.Validate(); err == nil {
			t.Fatalf("base %q accepted", base)
		}
	}
	for _, configure := range []func(*GeminiConfig){
		func(c *GeminiConfig) { c.APIKey = "" },
		func(c *GeminiConfig) { c.APIKey = "key\r\nheader" },
		func(c *GeminiConfig) { c.Model = "" },
		func(c *GeminiConfig) { c.Model = strings.Repeat("m", 201) },
		func(c *GeminiConfig) { c.MaxOutputTokens = 0 },
		func(c *GeminiConfig) { c.MaxOutputTokens = 65537 },
		func(c *GeminiConfig) { c.Timeout = 0 },
	} {
		cfg := testGeminiConfig()
		configure(&cfg)
		if err := cfg.Validate(); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	if _, err := NewGeminiClient(testGeminiConfig(), nil); err == nil {
		t.Fatal("nil HTTP client accepted")
	}
}

func TestGeminiGenerateBuildsBoundedStructuredRequest(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "trace-example")
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method=%s", req.Method)
		}
		if req.URL.String() != "https://api.aigocode.dev/v1beta/models/gemini-3.8-flash:generateContent" {
			t.Fatalf("url=%s", req.URL.String())
		}
		if strings.Contains(req.URL.String(), "aistudio_test") {
			t.Fatal("api key leaked into URL")
		}
		if req.Header.Get("x-goog-api-key") != "aistudio_test_not-a-real-key" {
			t.Fatal("missing x-goog-api-key header")
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing content type")
		}
		if req.Context().Value(contextKey{}) != "trace-example" {
			t.Fatal("caller context lost")
		}
		var payload struct {
			SystemInstruction *struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"systemInstruction"`
			Contents []struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
			GenerationConfig struct {
				ResponseMIMEType string `json:"responseMimeType"`
				MaxOutputTokens  int    `json:"maxOutputTokens"`
				ResponseSchema   struct {
					Type       string         `json:"type"`
					Properties map[string]any `json:"properties"`
					Required   []string       `json:"required"`
				} `json:"responseSchema"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(io.LimitReader(req.Body, 4<<20)).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.SystemInstruction == nil || payload.SystemInstruction.Parts[0].Text == "" {
			t.Fatal("missing system instruction")
		}
		if len(payload.Contents) != 1 || payload.Contents[0].Role != "user" {
			t.Fatalf("contents=%+v", payload.Contents)
		}
		var input struct {
			SourceLocale string      `json:"source_locale"`
			TargetLocale string      `json:"target_locale"`
			Source       Translation `json:"source"`
		}
		if err := json.Unmarshal([]byte(payload.Contents[0].Parts[0].Text), &input); err != nil {
			t.Fatalf("user part is not structured JSON: %v", err)
		}
		if input.SourceLocale != "zh" || input.TargetLocale != "en" || input.Source.Title != "快速开始" || input.Source.Body == "" {
			t.Fatalf("input=%+v", input)
		}
		if payload.GenerationConfig.ResponseMIMEType != "application/json" || payload.GenerationConfig.MaxOutputTokens != 16384 {
			t.Fatalf("generation config=%+v", payload.GenerationConfig)
		}
		schema := payload.GenerationConfig.ResponseSchema
		if schema.Type != "OBJECT" || len(schema.Properties) != 5 || len(schema.Required) != 5 {
			t.Fatalf("schema=%+v", schema)
		}
		return testResponse(200, geminiCandidate(validTranslationJSON, "STOP")), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Generate(ctx, testSource(), "zh", "en")
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Quick Start" || result.Summary == "" || !strings.Contains(result.Body, "# Quick Start") || result.SEOTitle != "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestGeminiIgnoresThinkingPartsInResponses(t *testing.T) {
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := geminiThinkingCandidate("Let me plan the translation strategy in English first.\n\n```json\nthis is not the answer\n```", validTranslationJSON, "STOP")
		return testResponse(200, body), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Generate(context.Background(), testSource(), "zh", "en")
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Quick Start" || result.SEOTitle != "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestGeminiRejectsInvalidLocalePairsBeforeCallingProvider(t *testing.T) {
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected provider call")
		return nil, errors.New("unreachable")
	})})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"ja", "en"}, {"zh", "zh"}, {"", "en"}, {"zh", ""}} {
		if _, err := client.Generate(context.Background(), testSource(), pair[0], pair[1]); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s→%s: err=%v", pair[0], pair[1], err)
		}
	}
}

func TestGeminiProviderFailuresAreClassifiedAndSanitized(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"rate limited", 429, `{"error":{"code":429}}`, ErrProviderRateLimited},
		{"upstream error", 500, "server detail with aistudio_test_not-a-real-key", ErrProviderUnavailable},
		{"client error", 400, `{"error":{"code":400}}`, ErrProviderUnavailable},
		{"malformed json", 200, "not json", ErrProviderInvalidOutput},
		{"blocked prompt", 200, `{"candidates":[],"promptFeedback":{"blockReason":"SAFETY"}}`, ErrProviderInvalidOutput},
		{"no candidates", 200, `{"candidates":[]}`, ErrProviderIncomplete},
		{"truncated output", 200, geminiCandidate(validTranslationJSON, "MAX_TOKENS"), ErrProviderIncomplete},
		{"safety finish", 200, geminiCandidate(validTranslationJSON, "SAFETY"), ErrProviderInvalidOutput},
		{"empty text", 200, geminiCandidate("  ", "STOP"), ErrProviderInvalidOutput},
		{"missing required field", 200, geminiCandidate(`{"title":"Only a title"}`, "STOP"), ErrProviderInvalidOutput},
		{"oversize body", 200, strings.Repeat("x", 2*1024*1024+1), ErrProviderInvalidOutput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				result := testResponse(tt.status, tt.body)
				result.Header.Set("Location", "https://other.example.com")
				return result, nil
			})}
			client, err := NewGeminiClient(testGeminiConfig(), httpClient)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Generate(context.Background(), testSource(), "zh", "en")
			if !errors.Is(err, tt.want) || calls != 1 {
				t.Fatalf("calls=%d err=%v want=%v", calls, err, tt.want)
			}
			if err != nil && (strings.Contains(err.Error(), "not-a-real-key") || strings.Contains(err.Error(), "private")) {
				t.Fatalf("error leaks provider detail: %v", err)
			}
			if httpClient.CheckRedirect != nil {
				t.Fatal("the injected client must not be mutated")
			}
		})
	}
}

func TestGeminiTransportErrorsKeepCancellationSemantics(t *testing.T) {
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), testSource(), "zh", "en"); !errors.Is(err, ErrProviderTimeout) {
		t.Fatalf("err=%v want timeout", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err = NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(ctx, testSource(), "zh", "en"); !errors.Is(err, ErrProviderInterrupted) {
		t.Fatalf("err=%v want interrupted", err)
	}
	client, err = NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private transport detail")
	})})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(context.Background(), testSource(), "zh", "en")
	if !errors.Is(err, ErrProviderUnavailable) || strings.Contains(err.Error(), "private transport") {
		t.Fatalf("err=%v", err)
	}
}

func TestGeminiReviewValidatesStructureAndBoundsUntrustedIssues(t *testing.T) {
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		issues := make([]string, 0, 25)
		long := strings.Repeat("长", 400)
		for i := range 25 {
			code, field := "omission", "body"
			if i == 1 {
				code, field = "format", "seo_title"
			}
			issues = append(issues, `{"code":"`+code+`","field":"`+field+`","message_zh":"`+long+`"}`)
		}
		body := `{"passed":false,"issues":[` + strings.Join(issues, ",") + `]}`
		return testResponse(200, geminiCandidate(body, "STOP")), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Review(context.Background(), testSource(), Translation{Title: "Quick Start", Summary: "s", Body: "b"}, "zh", "en")
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Fatal("passed must be false")
	}
	if len(result.Issues) != 20 {
		t.Fatalf("issues=%d want bounded to 20", len(result.Issues))
	}
	for _, issue := range result.Issues {
		if len([]rune(issue.MessageZH)) != 300 {
			t.Fatalf("message length=%d want 300", len([]rune(issue.MessageZH)))
		}
	}
}

func TestGeminiReviewRejectsStructuralViolations(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"passed with issues", `{"passed":true,"issues":[{"code":"omission","field":"body","message_zh":"矛盾"}]}`},
		{"unknown code", `{"passed":false,"issues":[{"code":"style","field":"body","message_zh":"问题"}]}`},
		{"unknown field", `{"passed":false,"issues":[{"code":"omission","field":"tags","message_zh":"问题"}]}`},
		{"missing issues", `{"passed":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return testResponse(200, geminiCandidate(tt.body, "STOP")), nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Review(context.Background(), testSource(), Translation{Title: "t", Summary: "s", Body: "b"}, "zh", "en")
			if !errors.Is(err, ErrProviderInvalidOutput) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestGeminiRepairSendsCandidateAndIssues(t *testing.T) {
	client, err := NewGeminiClient(testGeminiConfig(), &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(io.LimitReader(req.Body, 4<<20))
		var payload struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatal(err)
		}
		var input struct {
			Candidate Translation   `json:"candidate"`
			Issues    []ReviewIssue `json:"issues"`
		}
		if err := json.Unmarshal([]byte(payload.Contents[0].Parts[0].Text), &input); err != nil {
			t.Fatalf("repair input is not structured JSON: %v", err)
		}
		if input.Candidate.Title != "Quick Start" || len(input.Issues) != 1 || input.Issues[0].Code != "omission" {
			t.Fatalf("input=%+v", input)
		}
		return testResponse(200, geminiCandidate(validTranslationJSON, "STOP")), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	review := ReviewResult{Passed: false, Issues: []ReviewIssue{{Code: "omission", Field: "body", MessageZH: "末段缺失"}}}
	result, err := client.Repair(context.Background(), testSource(), Translation{Title: "Quick Start", Summary: "s", Body: "b"}, review, "zh", "en")
	if err != nil || result.Title != "Quick Start" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
