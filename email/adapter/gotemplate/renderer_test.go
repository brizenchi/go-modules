package gotemplate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/email/domain"
)

func TestRenderer_RegisterAndRender(t *testing.T) {
	r := New()
	src := `
{{define "subject"}}Welcome, {{.Name}}{{end}}
{{define "text"}}Hi {{.Name}}, code is {{.Code}}.{{end}}
{{define "html"}}<p>Hi {{.Name}}, code is <b>{{.Code}}</b>.</p>{{end}}`
	if err := r.Register("welcome", src); err != nil {
		t.Fatalf("register: %v", err)
	}
	subject, html, text, err := r.Render(context.Background(), "welcome", map[string]any{
		"Name": "Alice",
		"Code": "12345",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subject, "Welcome, Alice") {
		t.Errorf("subject = %q", subject)
	}
	if !strings.Contains(text, "code is 12345") {
		t.Errorf("text = %q", text)
	}
	if !strings.Contains(html, "<b>12345</b>") {
		t.Errorf("html = %q", html)
	}
}

func TestRenderer_HTMLOnlyTemplate(t *testing.T) {
	r := New()
	src := `{{define "subject"}}S{{end}}{{define "html"}}<p>{{.X}}</p>{{end}}`
	if err := r.Register("html_only", src); err != nil {
		t.Fatal(err)
	}
	_, html, text, err := r.Render(context.Background(), "html_only", map[string]any{"X": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if html == "" {
		t.Error("expected html body")
	}
	if text != "" {
		t.Errorf("expected empty text body, got %q", text)
	}
}

func TestRenderer_UnknownTemplate(t *testing.T) {
	r := New()
	_, _, _, err := r.Render(context.Background(), "missing", nil)
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestRenderer_HTMLEscapesAutomatically(t *testing.T) {
	r := New()
	src := `{{define "html"}}<p>{{.X}}</p>{{end}}{{define "subject"}}S{{end}}`
	if err := r.Register("esc", src); err != nil {
		t.Fatal(err)
	}
	_, html, _, _ := r.Render(context.Background(), "esc", map[string]any{
		"X": "<script>alert(1)</script>",
	})
	if strings.Contains(html, "<script>") {
		t.Errorf("html template did not escape: %q", html)
	}
}
