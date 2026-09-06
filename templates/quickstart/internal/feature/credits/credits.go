// Package credits exposes the host-owned credit ledger and a paid note export
// example. It has no external effects: export output and charging commit together.
package credits

import (
	"context"
	"time"

	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/user"
)

const DefaultExportCost int64 = 1

type NoteExport struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	UserID         string    `gorm:"type:varchar(36);not null;uniqueIndex:uniq_note_export_request"`
	IdempotencyKey string    `gorm:"type:varchar(128);not null;uniqueIndex:uniq_note_export_request"`
	NoteID         uint64    `gorm:"not null;index"`
	TransactionID  uint64    `gorm:"not null;uniqueIndex"`
	Filename       string    `gorm:"type:varchar(255);not null"`
	Content        string    `gorm:"type:text;not null"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (NoteExport) TableName() string { return "note_exports" }
func Models() []any                  { return []any{&NoteExport{}} }

type Module struct {
	users      *user.Repository
	exportCost func(context.Context) (int64, error)
}
type Option func(*Module)

func WithExportCost(resolve func(context.Context) (int64, error)) Option {
	return func(m *Module) {
		if resolve != nil {
			m.exportCost = resolve
		}
	}
}
func New(users *user.Repository, options ...Option) *Module {
	m := &Module{users: users, exportCost: func(context.Context) (int64, error) { return DefaultExportCost, nil }}
	for _, option := range options {
		option(m)
	}
	return m
}
func (m *Module) Register(groups hostapi.Groups) {
	if m == nil || m.users == nil {
		return
	}
	if groups.User != nil {
		groups.User.GET("/credits", m.summary)
		groups.User.GET("/credits/transactions", m.transactions)
		groups.User.POST("/notes/:id/export", m.exportNote)
	}
	if groups.Admin != nil {
		groups.Admin.GET("/admin/credits/transactions", m.adminTransactions)
		groups.Admin.POST("/admin/credits/grants", m.grant)
		groups.Admin.POST("/admin/credits/refunds", m.refund)
	}
}
