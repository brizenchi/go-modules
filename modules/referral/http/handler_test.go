package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRespondAppErrorDoesNotLeakUnknownInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/referral/stats", nil)

	respondAppError(context, errors.New("database dsn secret-must-not-leak"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-must-not-leak") || !strings.Contains(response.Body.String(), "internal referral error") {
		t.Fatalf("unsafe response body: %s", response.Body.String())
	}
}

func TestBuildInviteLinkSetsEncodedRefWithoutRequiringAStringSuffix(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		baseLink string
		code     string
		want     string
	}{
		{name: "plain page", baseLink: "https://app.example.com/invite", code: "INV 123", want: "https://app.example.com/invite?ref=INV+123"},
		{name: "documented placeholder", baseLink: "https://app.example.com/invite?ref=", code: "INV123", want: "https://app.example.com/invite?ref=INV123"},
		{name: "existing query", baseLink: "https://app.example.com/invite?source=account&ref=old", code: "NEW", want: "https://app.example.com/invite?ref=NEW&source=account"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := buildInviteLink(testCase.baseLink, testCase.code)
			if err != nil {
				t.Fatalf("buildInviteLink() error = %v", err)
			}
			if got != testCase.want {
				t.Fatalf("buildInviteLink() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBuildInviteLinkRejectsInvalidBaseURL(t *testing.T) {
	if _, err := buildInviteLink("/invite?ref=", "INV123"); err == nil {
		t.Fatal("buildInviteLink() expected an error for a relative base URL")
	}
}
