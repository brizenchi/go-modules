// Package email is a portable, provider-agnostic transactional email module.
//
// Layering (mirrors pkg/billing):
//
//	domain/    pure types: Message, Address, Attachment, errors
//	port/      interfaces (Sender, Renderer)
//	adapter/   concrete implementations
//	  brevo/      Brevo API
//	  smtp/       net/smtp
//	  log/        no-op for dev/tests
//	  gotemplate/ Renderer using html/text template
//	app/       use cases (SendService, SendTemplate)
//	email.go   Module + multi-tenant Manager
//
// The module never persists user data and never imports project-specific
// types. Hosts inject a configured Sender (and optionally a Renderer)
// and use SendService to send.
//
// Multi-tenancy: a single Manager can hold multiple SendServices keyed by
// project (or environment, or workspace). Use New() for a single-tenant
// app and NewManager() when you need per-project configuration.
package email

import (
	"context"
	"sync"

	"github.com/brizenchi/go-modules/email/app"
	"github.com/brizenchi/go-modules/email/domain"
	"github.com/brizenchi/go-modules/email/port"
)

// Module bundles a Sender + Renderer + SendService for one tenant.
type Module struct {
	Sender   port.Sender
	Renderer port.Renderer
	Service  *app.SendService
}

// New wires a Module from a Sender and (optional) Renderer.
func New(sender port.Sender, renderer port.Renderer) *Module {
	return &Module{
		Sender:   sender,
		Renderer: renderer,
		Service:  app.NewSendService(sender, renderer),
	}
}

// Send is a shortcut to Module.Service.Send.
func (m *Module) Send(ctx context.Context, msg *domain.Message) (*domain.Receipt, error) {
	return m.Service.Send(ctx, msg)
}

// SendTemplate is a shortcut to Module.Service.SendTemplate.
func (m *Module) SendTemplate(ctx context.Context, in app.TemplateMessage) (*domain.Receipt, error) {
	return m.Service.SendTemplate(ctx, in)
}

// SendProviderTemplate is a shortcut for provider-side templates.
func (m *Module) SendProviderTemplate(ctx context.Context, templateRef string, to []domain.Address, vars map[string]any) (*domain.Receipt, error) {
	return m.Service.SendProviderTemplate(ctx, templateRef, to, vars)
}

// Manager holds one Module per project key. Useful for multi-tenant
// apps where each tenant has its own provider credentials.
type Manager struct {
	mu      sync.RWMutex
	modules map[string]*Module
}

func NewManager() *Manager {
	return &Manager{modules: make(map[string]*Module)}
}

// Register stores a Module under the given key.
func (m *Manager) Register(key string, mod *Module) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modules[key] = mod
}

// Get returns the Module for a key. Returns ErrSenderUnavailable when missing.
func (m *Manager) Get(key string) (*Module, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if mod, ok := m.modules[key]; ok {
		return mod, nil
	}
	return nil, domain.ErrSenderUnavailable
}

// Send sends via the Module registered under key.
func (m *Manager) Send(ctx context.Context, key string, msg *domain.Message) (*domain.Receipt, error) {
	mod, err := m.Get(key)
	if err != nil {
		return nil, err
	}
	return mod.Send(ctx, msg)
}
