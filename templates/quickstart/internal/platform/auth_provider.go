package platform

import (
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth"
	"github.com/brizenchi/go-modules/modules/auth/adapter/emailcode"
	autheventbus "github.com/brizenchi/go-modules/modules/auth/adapter/eventbus"
	githuboauth "github.com/brizenchi/go-modules/modules/auth/adapter/github"
	"github.com/brizenchi/go-modules/modules/auth/adapter/google"
	authgormstore "github.com/brizenchi/go-modules/modules/auth/adapter/gormstore"
	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/brizenchi/go-modules/modules/email"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/gorm"
)

func buildAuth(db *gorm.DB, cfg Config, users *user.Repository, emailModule *email.Module) (*auth.Module, error) {
	if strings.TrimSpace(cfg.Auth.UserJWTSecret) == "" {
		return nil, fmt.Errorf("platform: auth.user_jwt_secret required when auth enabled")
	}
	signer, err := authjwt.NewSigner(authjwt.Config{
		Secret:  cfg.Auth.UserJWTSecret,
		Issuer:  cfg.ServiceName,
		UserTTL: time.Duration(cfg.Auth.UserJWTExpireHours) * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("platform: init jwt signer: %w", err)
	}
	ticketSigner, err := authjwt.NewTicketSigner(authjwt.Config{
		Secret:    cfg.Auth.UserJWTSecret,
		Issuer:    cfg.ServiceName + "-ws",
		TicketTTL: time.Duration(cfg.Auth.WSTicketTTLSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("platform: init websocket ticket signer: %w", err)
	}

	store := authgormstore.New(db)
	var issuer authport.EmailCodeIssuer
	var verifier authport.EmailCodeVerifier
	if cfg.EmailAuthEnabled() {
		if emailModule == nil {
			return nil, fmt.Errorf("platform: email provider required when auth.email.enabled=true")
		}
		issuer = emailcode.NewIssuer(emailcode.Config{
			TTL:          time.Duration(cfg.Auth.Email.Code.TTLMinutes) * time.Minute,
			MinResendGap: time.Duration(cfg.Auth.Email.Code.MinResendGapSeconds) * time.Second,
			DailyCap:     cfg.Auth.Email.Code.DailyCap,
			MaxAttempts:  cfg.Auth.Email.Code.MaxAttempts,
			TemplateRef:  "auth_email_code",
			Debug:        cfg.Auth.Email.Debug,
		}, store, verificationMailer{email: emailModule, serviceName: cfg.ServiceName})
		verifier = emailcode.NewVerifier(emailcode.Config{MaxAttempts: cfg.Auth.Email.Code.MaxAttempts}, store)
	}

	providers, err := buildIdentityProviders(cfg)
	if err != nil {
		return nil, err
	}
	return auth.New(auth.Deps{
		UserStore:         user.NewAuthStore(users),
		RoleResolver:      user.NewRoleResolver(cfg.Auth.AdminEmails),
		TokenSigner:       signer,
		WSTicketSigner:    ticketSigner,
		ExchangeCodeStore: store,
		EmailCodeIssuer:   issuer,
		EmailCodeVerifier: verifier,
		IdentityProviders: providers,
		Bus:               autheventbus.NewInProc(),
		FrontendURL:       cfg.Auth.FrontendRedirect,
	}), nil
}

func buildIdentityProviders(cfg Config) (map[string]authport.IdentityProvider, error) {
	providers := map[string]authport.IdentityProvider{}
	if cfg.GoogleEnabled() {
		provider, err := google.New(google.Config{
			ClientID:     cfg.Auth.Google.ClientID,
			ClientSecret: cfg.Auth.Google.ClientSecret,
			RedirectURL:  cfg.Auth.Google.RedirectURL,
			StateSecret:  cfg.Auth.Google.StateSecret,
			StateTTL:     time.Duration(cfg.Auth.Google.StateTTLMin) * time.Minute,
			Scope:        cfg.Auth.Google.Scope,
		})
		if err != nil {
			return nil, fmt.Errorf("platform: init google oauth: %w", err)
		}
		providers[string(authdomain.ProviderGoogle)] = provider
	}
	if cfg.GitHubEnabled() {
		provider, err := githuboauth.New(githuboauth.Config{
			ClientID:     cfg.Auth.GitHub.ClientID,
			ClientSecret: cfg.Auth.GitHub.ClientSecret,
			RedirectURL:  cfg.Auth.GitHub.RedirectURL,
			StateSecret:  cfg.Auth.GitHub.StateSecret,
			StateTTL:     time.Duration(cfg.Auth.GitHub.StateTTLMin) * time.Minute,
			Scope:        cfg.Auth.GitHub.Scope,
		})
		if err != nil {
			return nil, fmt.Errorf("platform: init github oauth: %w", err)
		}
		providers[string(authdomain.ProviderGitHub)] = provider
	}
	return providers, nil
}
