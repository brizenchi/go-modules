// Package google implements port.IdentityProvider against Google's OAuth 2.0 endpoints.
//
// State handling: state values are stateless signed JWTs (HS256, 10-min TTL).
// This avoids needing a server-side state store and protects against CSRF.
package google

import (
	"context"
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
	StateTTL    time.Duration // default 10m
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
	if c.StateTTL == 0 {
		c.StateTTL = 10 * time.Minute
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

// IssueState mints an HS256 JWT carrying just an expiry — that's enough
// to prove the state came from this server within the TTL window.
func (p *Provider) IssueState() (string, error) {
	now := time.Now().UTC()
	claims := jwtv5.RegisteredClaims{
		IssuedAt:  jwtv5.NewNumericDate(now),
		ExpiresAt: jwtv5.NewNumericDate(now.Add(p.cfg.StateTTL)),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return tok.SignedString([]byte(p.cfg.StateSecret))
}

func (p *Provider) VerifyState(state string) error {
	if state == "" {
		return domain.ErrInvalidState
	}
	claims := &jwtv5.RegisteredClaims{}
	tok, err := jwtv5.ParseWithClaims(state, claims, func(*jwtv5.Token) (any, error) {
		return []byte(p.cfg.StateSecret), nil
	})
	if err != nil || !tok.Valid {
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
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (p *Provider) Exchange(ctx context.Context, q url.Values) (*domain.OAuthProfile, error) {
	if errParam := q.Get("error"); errParam != "" {
		return nil, fmt.Errorf("google: oauth error: %s", errParam)
	}
	code := q.Get("code")
	if code == "" {
		return nil, fmt.Errorf("google: missing code in callback")
	}
	if err := p.VerifyState(q.Get("state")); err != nil {
		return nil, err
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
	return &info, nil
}

var _ port.IdentityProvider = (*Provider)(nil)
