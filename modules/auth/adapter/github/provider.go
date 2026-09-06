// Package github implements port.IdentityProvider with GitHub OAuth Apps.
package github

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const (
	defaultAuthorizeURL = "https://github.com/login/oauth/authorize"
	defaultTokenURL     = "https://github.com/login/oauth/access_token"
	defaultUserURL      = "https://api.github.com/user"
	defaultEmailsURL    = "https://api.github.com/user/emails"
	defaultScope        = "read:user user:email"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scope        string

	AuthorizeURL string
	TokenURL     string
	UserURL      string
	EmailsURL    string

	StateSecret string
	StateTTL    time.Duration
	HTTPTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.AuthorizeURL) == "" {
		c.AuthorizeURL = defaultAuthorizeURL
	}
	if strings.TrimSpace(c.TokenURL) == "" {
		c.TokenURL = defaultTokenURL
	}
	if strings.TrimSpace(c.UserURL) == "" {
		c.UserURL = defaultUserURL
	}
	if strings.TrimSpace(c.EmailsURL) == "" {
		c.EmailsURL = defaultEmailsURL
	}
	if strings.TrimSpace(c.Scope) == "" {
		c.Scope = defaultScope
	}
	if c.StateTTL <= 0 {
		c.StateTTL = 20 * time.Minute
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 15 * time.Second
	}
	return c
}

type Provider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) (*Provider, error) {
	cfg = cfg.withDefaults()
	if strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, fmt.Errorf("github: client_id and client_secret required")
	}
	if strings.TrimSpace(cfg.RedirectURL) == "" {
		return nil, fmt.Errorf("github: redirect_url required")
	}
	if strings.TrimSpace(cfg.StateSecret) == "" {
		return nil, fmt.Errorf("github: state_secret required")
	}
	return &Provider{cfg: cfg, client: &http.Client{Timeout: cfg.HTTPTimeout}}, nil
}

func (p *Provider) Name() domain.Provider { return domain.ProviderGitHub }

func (p *Provider) OAuthStateTTL() time.Duration { return p.cfg.StateTTL }

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
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(p.cfg.StateSecret))
}

func (p *Provider) VerifyState(state string) error {
	if strings.TrimSpace(state) == "" {
		return domain.ErrInvalidState
	}
	claims := &jwtv5.RegisteredClaims{}
	parser := jwtv5.NewParser(
		jwtv5.WithLeeway(30*time.Second),
		jwtv5.WithExpirationRequired(),
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
	)
	token, err := parser.ParseWithClaims(state, claims, func(*jwtv5.Token) (any, error) {
		return []byte(p.cfg.StateSecret), nil
	})
	if err != nil || !token.Valid || claims.ID == "" || claims.IssuedAt == nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidState, err)
	}
	return nil
}

func (p *Provider) AuthorizeURL(state string, extra url.Values) (string, error) {
	query := url.Values{
		"client_id":    {p.cfg.ClientID},
		"redirect_uri": {p.cfg.RedirectURL},
		"scope":        {p.cfg.Scope},
		"state":        {state},
	}
	for key, values := range extra {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	return p.cfg.AuthorizeURL + "?" + query.Encode(), nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type userResponse struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type emailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (p *Provider) Exchange(ctx context.Context, query url.Values) (*domain.OAuthProfile, error) {
	if err := p.VerifyState(query.Get("state")); err != nil {
		return nil, err
	}
	if oauthErr := strings.ToLower(strings.TrimSpace(query.Get("error"))); oauthErr != "" {
		if oauthErr == "access_denied" {
			return nil, domain.ErrOAuthDenied
		}
		return nil, domain.ErrOAuthCallback
	}
	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		return nil, domain.ErrOAuthCallback
	}

	accessToken, err := p.exchangeToken(ctx, code)
	if err != nil {
		return nil, err
	}
	user, err := p.fetchUser(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	email := normalizeEmail(user.Email)
	if email == "" {
		email, err = p.fetchPrimaryEmail(ctx, accessToken)
		if err != nil {
			return nil, err
		}
	}
	if email == "" {
		return nil, fmt.Errorf("github: verified email required")
	}
	username := strings.TrimSpace(user.Name)
	if username == "" {
		username = strings.TrimSpace(user.Login)
	}
	return &domain.OAuthProfile{
		Provider:  domain.ProviderGitHub,
		Subject:   strconv.FormatInt(user.ID, 10),
		Email:     email,
		Username:  username,
		AvatarURL: strings.TrimSpace(user.AvatarURL),
	}, nil
}

func randomStateID() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("github: generate oauth state id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (p *Provider) exchangeToken(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURL},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var response tokenResponse
	if err := p.doJSON(req, &response); err != nil {
		return "", fmt.Errorf("github: token request: %w", err)
	}
	if response.Error != "" {
		return "", fmt.Errorf("github: token error %s: %s", response.Error, response.Description)
	}
	if strings.TrimSpace(response.AccessToken) == "" {
		return "", fmt.Errorf("github: empty access_token")
	}
	return response.AccessToken, nil
}

func (p *Provider) fetchUser(ctx context.Context, accessToken string) (*userResponse, error) {
	var user userResponse
	if err := p.getJSON(ctx, p.cfg.UserURL, accessToken, &user); err != nil {
		return nil, fmt.Errorf("github: user request: %w", err)
	}
	if user.ID == 0 {
		return nil, fmt.Errorf("github: user response missing id")
	}
	return &user, nil
}

func (p *Provider) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	var emails []emailResponse
	if err := p.getJSON(ctx, p.cfg.EmailsURL, accessToken, &emails); err != nil {
		return "", fmt.Errorf("github: emails request: %w", err)
	}
	for _, candidate := range emails {
		if candidate.Primary && candidate.Verified {
			return normalizeEmail(candidate.Email), nil
		}
	}
	for _, candidate := range emails {
		if candidate.Verified {
			return normalizeEmail(candidate.Email), nil
		}
	}
	return "", nil
}

func (p *Provider) getJSON(ctx context.Context, endpoint, accessToken string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return p.doJSON(req, out)
}

func (p *Provider) doJSON(req *http.Request, out any) error {
	response, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("endpoint returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var _ port.IdentityProvider = (*Provider)(nil)
