// Package http exposes Gin handlers and middleware for the auth module.
package http

import (
	"context"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	fslog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/gin-gonic/gin"
)

// contextKey is the Gin context key under which the parsed Identity is stored.
const contextKey = "auth.identity"

// Authenticated returns the parsed Identity from a Gin context, or nil if absent.
func Authenticated(c *gin.Context) *domain.Identity {
	v, _ := c.Get(contextKey)
	if id, ok := v.(*domain.Identity); ok {
		return id
	}
	return nil
}

// SetIdentity stores an Identity in the Gin context (used by tests + middleware).
func SetIdentity(c *gin.Context, id *domain.Identity) {
	c.Set(contextKey, id)
}

// RequireUser is a Gin middleware that parses the Bearer token and
// attaches the Identity. Aborts with 401 on missing/invalid token.
func RequireUser(session *app.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := authenticateRequest(c, session); !ok {
			return
		}
		c.Next()
	}
}

// RequireAdmin extends RequireUser with a role check.
func RequireAdmin(session *app.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := authenticateRequest(c, session)
		if !ok {
			return
		}
		if id.Role != domain.RoleAdmin {
			httpresp.Forbidden(c, "admin role required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// authenticateRequest performs authentication without advancing Gin's handler
// chain. This is shared by RequireUser and RequireAdmin so the admin role check
// always happens before the protected handler is allowed to run.
func authenticateRequest(c *gin.Context, session *app.SessionService) (*domain.Identity, bool) {
	token := bearerToken(c)
	if token == "" {
		httpresp.Unauthorized(c, "missing bearer token")
		c.Abort()
		return nil, false
	}
	if session == nil {
		httpresp.Unauthorized(c, "authentication unavailable")
		c.Abort()
		return nil, false
	}
	id, err := session.VerifyToken(token)
	if err != nil || id == nil {
		message := "invalid bearer token"
		if err != nil {
			message = err.Error()
		}
		httpresp.Unauthorized(c, message)
		c.Abort()
		return nil, false
	}
	SetIdentity(c, id)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), fslog.UserIDKey, id.UserID))
	return id, true
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return ""
}
