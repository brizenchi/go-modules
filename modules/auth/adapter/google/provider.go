// Package google implements port.IdentityProvider against Google's OAuth 2.0 endpoints.
//
// State integrity uses signed JWTs (HS256, 20-min default TTL). Browser
// binding and one-time consumption are enforced by auth/app with the shared
// OAuthFlowStore; a signed state alone is not treated as sufficient CSRF
// protection.
package google

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Config holds Google OAuth + endpoint settings.
//
// Sensible defaults for AuthorizeURL/TokenURL/UserInfoURL are filled in
// when blank; ClientID, ClientSecret, RedirectURL, StateSecret are required.
type Config struct {
	ProviderName domain.Provider // defaults to ProviderGoogle; set to ProviderAnthropic for the legacy alias
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scope        string // space-separated; defaults to "openid email profile"

	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string

	StateSecret string        // HS256 secret for state JWT
	StateTTL    time.Duration // default 20m
	HTTPTimeout time.Duration // default 15s
}

const (
	defaultAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenURL     = "https://oauth2.googleapis.com/token"
	defaultUserInfoURL  = "https://www.googleapis.com/oauth2/v3/userinfo"
	defaultScope        = "openid email profile"
)

func (c Config) withDefaults() Config {
	if c.ProviderName == "" {
		c.ProviderName = domain.ProviderGoogle
	}
	if c.AuthorizeURL == "" {
		c.AuthorizeURL = defaultAuthorizeURL
	}
	if c.TokenURL == "" {
		c.TokenURL = defaultTokenURL
	}
	if c.UserInfoURL == "" {
		c.UserInfoURL = defaultUserInfoURL
	}
	if c.Scope == "" {
		c.Scope = defaultScope
	}
	if c.StateTTL <= 0 {
		c.StateTTL = 20 * time.Minute
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	return c
}

// Provider implements port.IdentityProvider.
type Provider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) (*Provider, error) {
	cfg = cfg.withDefaults()
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("google: client_id and client_secret required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("google: redirect_url required")
	}
	if cfg.StateSecret == "" {
		return nil, fmt.Errorf("google: state_secret required")
	}
	return &Provider{cfg: cfg, client: &http.Client{Timeout: cfg.HTTPTimeout}}, nil
}

func (p *Provider) Name() domain.Provider { return p.cfg.ProviderName }

func (p *Provider) OAuthStateTTL() time.Duration { return p.cfg.StateTTL }

// IssueState mints an expiring HS256 JWT with a cryptographically random ID.
// The random jti prevents two flows started in the same second from sharing
// an identical state value.
func (p *Provider) IssueState() (string, error) {
	jti, err := randomStateID()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	claims := jwtv5.RegisteredClaims{
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(p.cfg.StateTTL)),
		ID:        jti,
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return tok.SignedString([]byte(p.cfg.StateSecret))
}

func (p *Provider) VerifyState(state string) error {
	if state == "" {
		return domain.ErrInvalidState
	}
	claims := &jwtv5.RegisteredClaims{}
	parser := jwtv5.NewParser(
		jwtv5.WithLeeway(30*time.Second),
		jwtv5.WithExpirationRequired(),
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
	)
	tok, err := parser.ParseWithClaims(state, claims, func(*jwtv5.Token) (any, error) {
		return []byte(p.cfg.StateSecret), nil
	})
	if err != nil || !tok.Valid || claims.ID == "" || claims.IssuedAt == nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidState, err)
	}
	return nil
}

func (p *Provider) AuthorizeURL(state string, extra url.Values) (string, error) {
	q := url.Values{
		"client_id":     {p.cfg.ClientID},
		"redirect_uri":  {p.cfg.RedirectURL},
		"response_type": {"code"},
		"scope":         {p.cfg.Scope},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	return p.cfg.AuthorizeURL + "?" + q.Encode(), nil
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

type userInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (p *Provider) Exchange(ctx context.Context, q url.Values) (*domain.OAuthProfile, error) {
	if err := p.VerifyState(q.Get("state")); err != nil {
		return nil, err
	}
	if errParam := strings.ToLower(strings.TrimSpace(q.Get("error"))); errParam != "" {
		if errParam == "access_denied" {
			return nil, domain.ErrOAuthDenied
		}
		return nil, domain.ErrOAuthCallback
	}
	code := strings.TrimSpace(q.Get("code"))
	if code == "" {
		return nil, domain.ErrOAuthCallback
	}

	form := url.Values{
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	}
	tok, err := p.postForm(ctx, p.cfg.TokenURL, form)
	if err != nil {
		return nil, err
	}
	var tr tokenResp
	if err := json.Unmarshal(tok, &tr); err != nil {
		return nil, fmt.Errorf("google: parse token resp: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("google: empty access_token")
	}

	info, err := p.fetchUserInfo(ctx, tr.AccessToken)
	if err != nil {
		return nil, err
	}
	return &domain.OAuthProfile{
		Provider:  p.cfg.ProviderName,
		Subject:   strings.TrimSpace(info.Sub),
		Email:     strings.ToLower(strings.TrimSpace(info.Email)),
		Username:  strings.TrimSpace(info.Name),
		AvatarURL: strings.TrimSpace(info.Picture),
	}, nil
}

func randomStateID() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("google: generate oauth state id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (p *Provider) postForm(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google: token endpoint %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (p *Provider) fetchUserInfo(ctx context.Context, accessToken string) (*userInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: userinfo request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google: userinfo %d: %s", resp.StatusCode, string(body))
	}
	var info userInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("google: parse userinfo: %w", err)
	}
	if info.Email == "" {
		return nil, fmt.Errorf("google: userinfo missing email")
	}
	if !info.EmailVerified {
		return nil, fmt.Errorf("google: userinfo email is not verified")
	}
	return &info, nil
}

var _ port.IdentityProvider = (*Provider)(nil)
