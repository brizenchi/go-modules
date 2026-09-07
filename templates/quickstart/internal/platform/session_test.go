package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func sessionRequest(router http.Handler, method, path, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/v1"+path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func sessionLogin(t *testing.T, router http.Handler) string {
	t.Helper()
	response := adminPasswordRequest(router, `{"email":"owner@example.test","password":"`+testAdminPassword+`"}`, "192.0.2.50")
	if response.Code != http.StatusOK {
		t.Fatalf("login: %d %s", response.Code, response.Body)
	}
	var result struct{ Data struct{ Token string } }
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.Data.Token
}

func TestConfiguredSessionLifetimes(t *testing.T) {
	config := adminPasswordConfig()
	config.Auth.UserJWTExpireHours = 1
	config.Auth.WSTicketTTLSeconds = 30
	modules, router := newAdminPasswordTestServer(t, config)
	token := sessionLogin(t, router)
	identity, err := modules.Auth.Session.VerifyToken(token)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := modules.Auth.Session.Refresh(t.Context(), identity.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if remaining := time.Until(refreshed.Token.ExpiresAt); remaining < 59*time.Minute || remaining > time.Hour {
		t.Errorf("refresh TTL = %s, want 1h", remaining)
	}
	ticket, err := modules.Auth.Session.IssueWSTicket(t.Context(), identity.UserID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if remaining := time.Until(ticket.ExpiresAt); remaining < 29*time.Second || remaining > 30*time.Second {
		t.Errorf("ticket TTL = %s, want 30s", remaining)
	}
}

func TestLogoutRevokesOnlySubmittedToken(t *testing.T) {
	_, router := newAdminPasswordTestServer(t, adminPasswordConfig())
	token := sessionLogin(t, router)
	otherToken := sessionLogin(t, router)
	response := sessionRequest(router, http.MethodPost, "/auth/logout", token)
	if response.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", response.Code, response.Body)
	}
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/admin/protected"},
		{http.MethodPost, "/auth/refresh"},
		{http.MethodPost, "/auth/logout"},
	} {
		response = sessionRequest(router, route.method, route.path, token)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("revoked token accepted at %s: %d %s", route.path, response.Code, response.Body)
		}
	}
	response = sessionRequest(router, http.MethodGet, "/admin/protected", otherToken)
	if response.Code != http.StatusOK {
		t.Fatalf("independent session revoked: %d", response.Code)
	}
}

func TestSessionStoreFailureFailsClosed(t *testing.T) {
	modules, router := newAdminPasswordTestServer(t, adminPasswordConfig())
	token := sessionLogin(t, router)
	database, err := modules.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/admin/protected"},
		{http.MethodPost, "/auth/logout"},
	} {
		response := sessionRequest(router, route.method, route.path, token)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("store failure at %s: %d %s", route.path, response.Code, response.Body)
		}
	}
}
