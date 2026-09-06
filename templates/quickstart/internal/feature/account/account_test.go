package account

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type profileEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data profileResponse `json:"data"`
}

func newAccountTestRouter(t *testing.T) (*gin.Engine, *user.Repository, *user.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := user.AutoMigrate(db); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	repository := user.NewRepository(db)
	account := &user.User{
		Email:         "owner@example.com",
		EmailVerified: true,
		Username:      "Before",
		AvatarURL:     "https://cdn.example.com/before.png",
		Credits:       75,
	}
	if err := repository.Create(context.Background(), account); err != nil {
		t.Fatalf("create user: %v", err)
	}

	router := gin.New()
	userGroup := router.Group("/api/v1")
	userGroup.Use(func(c *gin.Context) {
		authhttp.SetIdentity(c, &authdomain.Identity{
			UserID: account.ID,
			Email:  account.Email,
			Role:   authdomain.RoleAdmin,
		})
		c.Next()
	})
	New(repository).Register(hostapi.Groups{User: userGroup})
	return router, repository, account
}

func TestGetProfileReturnsAuthenticatedAccount(t *testing.T) {
	router, _, account := newAccountTestRouter(t)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/account/profile", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body profileEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != http.StatusOK || body.Data.ID != account.ID || body.Data.Email != account.Email {
		t.Fatalf("unexpected response: %#v", body)
	}
	if !body.Data.EmailVerified || body.Data.Role != "admin" || body.Data.Credits != 75 {
		t.Fatalf("profile fields missing: %#v", body.Data)
	}
}

func TestPatchProfileUpdatesOnlyEditableFields(t *testing.T) {
	router, repository, account := newAccountTestRouter(t)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/account/profile",
		bytes.NewBufferString(`{"username":"  New name  ","avatar_url":"https://cdn.example.com/new.png"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body profileEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Username != "New name" || body.Data.AvatarURL != "https://cdn.example.com/new.png" {
		t.Fatalf("unexpected profile: %#v", body.Data)
	}
	stored, err := repository.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("find updated user: %v", err)
	}
	if stored.Email != account.Email || stored.Credits != account.Credits {
		t.Fatalf("read-only fields changed: %#v", stored)
	}
}

func TestPatchProfileRejectsInvalidOrReadOnlyFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty patch", body: `{}`},
		{name: "email is read only", body: `{"email":"other@example.com"}`},
		{name: "username too long", body: `{"username":"` + strings.Repeat("a", 101) + `"}`},
		{name: "avatar must be absolute", body: `{"avatar_url":"/avatar.png"}`},
		{name: "avatar scheme", body: `{"avatar_url":"javascript:alert(1)"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, _ := newAccountTestRouter(t)
			request := httptest.NewRequest(http.MethodPatch, "/api/v1/account/profile", bytes.NewBufferString(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
