// Package http exposes Gin handlers and middleware for the auth module.
package http

import (
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/auth/app"
	"github.com/brizenchi/go-modules/auth/domain"
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
			respondJSON(c, http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			c.Abort()
			return
		}
		id, err := session.VerifyToken(token)
		if err != nil {
			respondJSON(c, http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
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
			respondJSON(c, http.StatusForbidden, gin.H{"error": "admin role required"})
			c.Abort()
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

func respondJSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}
