// Package gotemplate is a Renderer backed by Go's standard html/template
// and text/template packages.
//
// Templates are registered up-front; lookup at render time is by name.
// Each template defines three blocks ({{define "subject"}}, "html",
// "text") so a single template file produces all three outputs:
//
//	{{define "subject"}}Welcome, {{.Name}}{{end}}
//	{{define "text"}}Hi {{.Name}}, your code is {{.Code}}.{{end}}
//	{{define "html"}}<p>Hi {{.Name}}, your code is <b>{{.Code}}</b>.</p>{{end}}
package gotemplate

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"sync"
	texttemplate "text/template"

	"github.com/brizenchi/go-modules/modules/email/domain"
	"github.com/brizenchi/go-modules/modules/email/port"
)

// Renderer holds the registered templates.
type Renderer struct {
	mu       sync.RWMutex
	htmlTpls map[string]*template.Template     // name -> tpl with blocks
	textTpls map[string]*texttemplate.Template // name -> tpl with subject+text blocks
}

func New() *Renderer {
	return &Renderer{
		htmlTpls: make(map[string]*template.Template),
		textTpls: make(map[string]*texttemplate.Template),
	}
}

// Register parses and stores a template under the given name.
//
// The source must define at least one of {"subject", "html", "text"} via
// {{define "..."}}...{{end}} blocks. Missing blocks render to empty.
func (r *Renderer) Register(name, source string) error {
	htmlTpl, err := template.New(name).Parse(source)
	if err != nil {
		return fmt.Errorf("gotemplate: parse html %q: %w", name, err)
	}
	textTpl, err := texttemplate.New(name).Parse(source)
	if err != nil {
		return fmt.Errorf("gotemplate: parse text %q: %w", name, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.htmlTpls[name] = htmlTpl
	r.textTpls[name] = textTpl
	return nil
}

func (r *Renderer) Render(ctx context.Context, name string, vars map[string]any) (subject, html, text string, err error) {
	r.mu.RLock()
	htmlTpl := r.htmlTpls[name]
	textTpl := r.textTpls[name]
	r.mu.RUnlock()
	if htmlTpl == nil || textTpl == nil {
		return "", "", "", fmt.Errorf("%w: %q", domain.ErrTemplateNotFound, name)
	}

	subject, err = execText(textTpl, "subject", vars)
	if err != nil {
		return "", "", "", err
	}
	text, err = execText(textTpl, "text", vars)
	if err != nil {
		return "", "", "", err
	}
	html, err = execHTML(htmlTpl, "html", vars)
	if err != nil {
		return "", "", "", err
	}
	return subject, html, text, nil
}

func execText(tpl *texttemplate.Template, block string, vars map[string]any) (string, error) {
	if tpl.Lookup(block) == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, block, vars); err != nil {
		return "", fmt.Errorf("gotemplate: execute %q: %w", block, err)
	}
	return buf.String(), nil
}

func execHTML(tpl *template.Template, block string, vars map[string]any) (string, error) {
	if tpl.Lookup(block) == nil {
		return "", nil
	}
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, block, vars); err != nil {
		return "", fmt.Errorf("gotemplate: execute %q: %w", block, err)
	}
	return buf.String(), nil
}

var _ port.Renderer = (*Renderer)(nil)
