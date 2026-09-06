package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/gin-gonic/gin"
)

func TestAuthJSONHandlersRejectOversizedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(Deps{})
	router := gin.New()
	router.POST("/send", handler.SendCode)
	router.POST("/verify", handler.VerifyCode)
	router.POST("/exchange", handler.ExchangeToken)
	router.POST("/ticket", func(c *gin.Context) {
		SetIdentity(c, &domain.Identity{UserID: "user-1"})
		c.Next()
	}, handler.IssueWSTicket)

	largeValue := strings.Repeat("a", int(maxAuthJSONBodyBytes)+1)
	tests := []struct {
		path string
		body string
	}{
		{path: "/send", body: `{"email":"` + largeValue + `"}`},
		{path: "/verify", body: `{"email":"user@example.com","code":"` + largeValue + `"}`},
		{path: "/exchange", body: `{"code":"` + largeValue + `"}`},
		{path: "/ticket", body: `{"scope":{"source":"` + largeValue + `"}}`},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), "request body too large") {
				t.Fatalf("unexpected response body: %s", response.Body.String())
			}
		})
	}
}

func TestRespondAppErrorDoesNotLeakUnknownInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPost, "/auth/send-code", nil)

	respondAppError(context, errors.New("smtp password secret-must-not-leak"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-must-not-leak") || !strings.Contains(response.Body.String(), "internal authentication error") {
		t.Fatalf("unsafe response body: %s", response.Body.String())
	}
}
