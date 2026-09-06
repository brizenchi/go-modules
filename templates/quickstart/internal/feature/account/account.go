// Package account exposes host-owned account profile endpoints.
//
// Authentication remains in the reusable auth module, while editable product
// profile fields remain on this SaaS's internal/user.User model.
package account

import (
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/user"
)

// Module bundles account profile HTTP wiring.
type Module struct {
	handler *Handler
}

// New builds the account feature from the host user repository.
func New(users *user.Repository) *Module {
	return &Module{handler: newHandler(newService(users))}
}

// Register mounts profile routes on the authenticated user group.
func (m *Module) Register(groups hostapi.Groups) {
	if m == nil || m.handler == nil || groups.User == nil {
		return
	}
	groups.User.GET("/account/profile", m.handler.GetProfile)
	groups.User.PATCH("/account/profile", m.handler.UpdateProfile)
}
