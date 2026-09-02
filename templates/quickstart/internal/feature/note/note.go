// Package note is the reference feature for this template.
//
// It exists to show the layering every host feature should follow:
//
//	note.go        entity + module wiring (New / Models / Register)
//	repository.go  data access, the only layer that touches *gorm.DB
//	service.go     business rules, no gin and no SQL
//	handler.go     HTTP in/out only, no business rules
//
// Copy this folder to start a new feature, or delete it outright:
//
//	rm -rf internal/feature/note
//
// then drop its two references in internal/bootstrap/host_migrate.go and
// internal/http/host_routes.go.
package note

import (
	"time"

	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"gorm.io/gorm"
)

// Note is a user-owned row. UserID matches this SaaS's internal/user.User.ID.
type Note struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Title     string    `gorm:"type:varchar(200);not null" json:"title"`
	Body      string    `gorm:"type:text" json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Note) TableName() string { return "notes" }

// Module bundles the feature's wiring so bootstrap only sees one symbol.
type Module struct {
	handler *Handler
}

// New builds the feature from its dependencies.
func New(db *gorm.DB) *Module {
	return &Module{handler: newHandler(newService(newRepository(db)))}
}

// Register mounts the feature's routes. Pick the group that matches the
// access level; never mount a user-scoped route on Groups.Public.
func (m *Module) Register(g hostapi.Groups) {
	g.User.POST("/notes", m.handler.Create)
	g.User.GET("/notes", m.handler.ListMine)
	g.Admin.GET("/admin/notes/count", m.handler.CountAll)
}
