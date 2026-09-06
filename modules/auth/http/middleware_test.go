package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/gin-gonic/gin"
)

type middlewareSigner struct{}

func (middlewareSigner) Issue(identity domain.Identity, ttl time.Duration) (*domain.Token, error) {
	return &domain.Token{Value: string(identity.Role), ExpiresAt: time.Now().Add(ttl)}, nil
}

func (middlewareSigner) Parse(value string) (*domain.Identity, error) {
	switch value {
	case "user-token":
		return &domain.Identity{UserID: "user-1", Role: domain.RoleUser}, nil
	case "admin-token":
		return &domain.Identity{UserID: "admin-1", Role: domain.RoleAdmin}, nil
	default:
		return nil, domain.ErrInvalidToken
	}
}

func middlewareSession() *app.SessionService {
	return app.NewSessionService(app.SessionDeps{Signer: middlewareSigner{}})
}

func performProtectedRequest(t *testing.T, middleware gin.HandlerFunc, token string) (*httptest.ResponseRecorder, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	called := 0
	router.GET("/protected", middleware, func(c *gin.Context) {
		called++
		c.JSON(http.StatusOK, gin.H{"protected": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response, called
}

func TestRequireUserRejectsWithoutRunningProtectedHandler(t *testing.T) {
	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "missing token"},
		{name: "invalid token", token: "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, calls := performProtectedRequest(t, RequireUser(middlewareSession()), test.token)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
			}
			if calls != 0 {
				t.Fatal("protected handler ran after authentication rejection")
			}
		})
	}
}

func TestRequireAdminChecksRoleBeforeProtectedHandler(t *testing.T) {
	response, calls := performProtectedRequest(t, RequireAdmin(middlewareSession()), "user-token")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusForbidden)
	}
	if calls != 0 {
		t.Fatal("admin handler ran for a non-admin user")
	}
}

func TestRequireAdminRejectsAnonymousWithoutRunningProtectedHandler(t *testing.T) {
	response, calls := performProtectedRequest(t, RequireAdmin(middlewareSession()), "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
	if calls != 0 {
		t.Fatal("admin handler ran for an anonymous request")
	}
}

func TestAuthMiddlewareAllowsAuthorizedHandlersOnce(t *testing.T) {
	for _, test := range []struct {
		name       string
		middleware gin.HandlerFunc
		token      string
	}{
		{name: "user", middleware: RequireUser(middlewareSession()), token: "user-token"},
		{name: "admin", middleware: RequireAdmin(middlewareSession()), token: "admin-token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, calls := performProtectedRequest(t, test.middleware, test.token)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusOK)
			}
			if calls != 1 {
				t.Fatalf("authorized handler calls=%d, want 1", calls)
			}
		})
	}
}
