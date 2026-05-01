// Package http exposes Gin handlers and middleware for the auth module.
package http

import (
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
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
		token := bearerToken(c)
		if token == "" {
			httpresp.Unauthorized(c, "missing bearer token")
			return
		}
		id, err := session.VerifyToken(token)
		if err != nil {
			httpresp.Unauthorized(c, err.Error())
			return
		}
		SetIdentity(c, id)
		c.Next()
	}
}

// RequireAdmin extends RequireUser with a role check.
func RequireAdmin(session *app.SessionService) gin.HandlerFunc {
	authMW := RequireUser(session)
	return func(c *gin.Context) {
		authMW(c)
		if c.IsAborted() {
			return
		}
		id := Authenticated(c)
		if id == nil || id.Role != domain.RoleAdmin {
			httpresp.Forbidden(c, "admin role required")
			return
		}
		c.Next()
	}
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
