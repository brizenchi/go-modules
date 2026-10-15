package operations

import (
	"log/slog"
	"strings"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Failed administrator mutations are recorded without reading request bodies or
// query strings. Successful business writes record their own transactional event.
func (m *Module) auditFailures(c *gin.Context) {
	c.Next()
	if m.deps.DB == nil || c.Writer.Status() < 400 || c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
		return
	}
	actor := userID(c)
	if actor == "" {
		return
	}
	route := c.FullPath()
	if route == "" {
		return
	}
	action := "admin.request.failed"
	switch {
	case strings.Contains(route, "membership"):
		action = "membership.update.failed"
	case strings.Contains(route, "credits"):
		action = "credits.change.failed"
	case strings.Contains(route, "articles"):
		action = "content.change.failed"
	case strings.Contains(route, "settings"):
		action = "settings.update.failed"
	case strings.Contains(route, "integrations"):
		action = "integration.write.denied"
	}
	entry := audit.Operation{ActorID: actor, IdempotencyKey: "failed:" + uuid.NewString(), Action: action, TargetID: route, Reason: "Administrative request rejected", RequestHash: audit.Hash(c.Writer.Status()), Status: "pending"}
	ctx := c.Request.Context()
	err := m.deps.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := m.deps.Audit.Claim(ctx, tx, &entry); err != nil {
			return err
		}
		return m.deps.Audit.Finish(ctx, tx, &entry, "failed", stringJSON(map[string]any{"status": c.Writer.Status(), "method": c.Request.Method, "resource_id": c.Param("id")}))
	})
	if err != nil {
		slog.ErrorContext(ctx, "administrative failure audit unavailable", "error", err)
	}
}
