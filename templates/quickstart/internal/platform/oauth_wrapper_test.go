package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authmodule "github.com/brizenchi/go-modules/modules/auth"
	"github.com/brizenchi/go-modules/modules/auth/adapter/memstore"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/gin-gonic/gin"
)

type wrapperOAuthUsers struct{ identity authdomain.Identity }

func (*wrapperOAuthUsers) FindByEmail(context.Context, string) (*authdomain.Identity, error) {
	return nil, authdomain.ErrUserNotFound
}
func (*wrapperOAuthUsers) FindOrCreateByEmail(context.Context, string) (*authdomain.Identity, error) {
	return nil, authdomain.ErrUserNotFound
}
func (*wrapperOAuthUsers) FindOrCreateFromOAuth(context.Context, authdomain.OAuthProfile) (*authdomain.Identity, error) {
	return nil, authdomain.ErrUserNotFound
}
func (u *wrapperOAuthUsers) FindByID(context.Context, string) (*authdomain.Identity, error) {
	copy := u.identity
	return &copy, nil
}
func (*wrapperOAuthUsers) MarkLogin(context.Context, string) error { return nil }

type wrapperOAuthSigner struct{}

func (wrapperOAuthSigner) Issue(id authdomain.Identity, ttl time.Duration) (*authdomain.Token, error) {
	return &authdomain.Token{Value: "token-" + id.UserID, ExpiresAt: time.Now().Add(ttl)}, nil
}
func (wrapperOAuthSigner) Parse(string) (*authdomain.Identity, error) {
	return nil, authdomain.ErrInvalidToken
}

func wrapperVerifier(seed byte) (string, string) {
	verifier := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 32))
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestQuickstartExchangeWrapperForwardsOAuthVerifierAndDoesNotConsumeOnMismatch(t *testing.T) {
	store := memstore.NewExchangeStore()
	verifier, challenge := wrapperVerifier(4)
	wrongVerifier, _ := wrapperVerifier(5)
	identity := authdomain.Identity{UserID: "user-1", Email: "user@example.com"}
	if err := store.Save(t.Context(), authdomain.ExchangeCode{
		Code: "exchange-code", UserID: identity.UserID, Provider: authdomain.ProviderGoogle,
		BindingHash: challenge, ExpiresAt: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	service := authapp.NewOAuthService(authapp.OAuthDeps{
		Users: &wrapperOAuthUsers{identity: identity}, Signer: wrapperOAuthSigner{}, ExchangeStore: store,
	})
	modules := &Modules{Auth: &authmodule.Module{OAuth: service}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/exchange", modules.exchangeToken())

	for _, body := range []string{
		`{"code":"exchange-code"}`,
		`{"code":"exchange-code","oauth_verifier":"` + wrongVerifier + `"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid verifier status=%d body=%s", response.Code, response.Body.String())
		}
	}

	validBody := `{"code":"exchange-code","oauth_verifier":"` + verifier + `","referral_code":"REF123"}`
	request := httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(validBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "token-user-1") {
		t.Fatalf("correct verifier status=%d body=%s", response.Code, response.Body.String())
	}

	// The correctly bound exchange code is still one-time.
	request = httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(validBody))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("replay status=%d body=%s", response.Code, response.Body.String())
	}
}
