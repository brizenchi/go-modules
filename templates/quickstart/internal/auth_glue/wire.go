// Package auth_glue wires the auth module + the project's User table.
//
// Replace `User` and the four UserStore methods with whatever your
// schema looks like.
package auth_glue

import (
	"context"
	"log/slog"
	"time"

	"github.com/brizenchi/go-modules/auth"
	"github.com/brizenchi/go-modules/auth/adapter/emailcode"
	"github.com/brizenchi/go-modules/auth/adapter/eventbus"
	"github.com/brizenchi/go-modules/auth/adapter/google"
	authjwt "github.com/brizenchi/go-modules/auth/adapter/jwt"
	"github.com/brizenchi/go-modules/auth/adapter/memstore"
	authdomain "github.com/brizenchi/go-modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/auth/http"
	"github.com/brizenchi/go-modules/auth/port"
	"github.com/brizenchi/go-modules/email"
	emaildomain "github.com/brizenchi/go-modules/email/domain"

	"github.com/brizenchi/go-modules/templates/quickstart/internal/email_glue"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// Init builds the auth module from viper config + your DB.
//
// You're expected to extend `userStore` (in user_store.go) so it
// reads/writes your project's User table. The signature below is
// the integration point that doesn't change between projects.
func Init(db *gorm.DB, mailer *email.Module) *auth.Module {
	jwtSecret := viper.GetString("auth.user_jwt_secret")
	if jwtSecret == "" {
		slog.Error("auth_glue: auth.user_jwt_secret missing — auth will be unusable")
		jwtSecret = "PLACEHOLDER-CHANGE-ME"
	}

	signer, _ := authjwt.NewSigner(authjwt.Config{
		Secret:  jwtSecret,
		Issuer:  viper.GetString("server.name"),
		UserTTL: 7 * 24 * time.Hour,
	})
	ticket, _ := authjwt.NewTicketSigner(authjwt.Config{
		Secret:    jwtSecret,
		Issuer:    viper.GetString("server.name") + "-ws",
		TicketTTL: 5 * time.Minute,
	})

	codeStore := memstore.NewCodeStore()
	exchangeStore := memstore.NewExchangeStore()

	issuer := emailcode.NewIssuer(emailcode.Config{
		TemplateRef: email_glue.VerificationTemplateRef,
		Debug:       viper.GetBool("auth.email.debug"),
	}, codeStore, mailerWrapper{mod: mailer})
	verifier := emailcode.NewVerifier(emailcode.Config{}, codeStore)

	providers := map[string]port.IdentityProvider{}
	if g := buildGoogle(); g != nil {
		providers[string(authdomain.ProviderGoogle)] = g
	}

	return auth.New(auth.Deps{
		UserStore:         &userStore{db: db},
		RoleResolver:      &roleResolver{},
		TokenSigner:       signer,
		WSTicketSigner:    ticket,
		ExchangeCodeStore: exchangeStore,
		EmailCodeIssuer:   issuer,
		EmailCodeVerifier: verifier,
		IdentityProviders: providers,
		Bus:               eventbus.NewInProc(),
		FrontendURL:       viper.GetString("auth.frontend_redirect"),
	})
}

// RequireUser returns the Bearer-auth middleware for the user route group.
func RequireUser(mod *auth.Module) gin.HandlerFunc {
	return authhttp.RequireUser(mod.Session)
}

func buildGoogle() port.IdentityProvider {
	if viper.GetString("auth.google.client_id") == "" {
		return nil
	}
	p, err := google.New(google.Config{
		ClientID:     viper.GetString("auth.google.client_id"),
		ClientSecret: viper.GetString("auth.google.client_secret"),
		RedirectURL:  viper.GetString("auth.google.redirect_url"),
		StateSecret:  viper.GetString("auth.google.state_secret"),
		Scope:        viper.GetString("auth.google.scope"),
	})
	if err != nil {
		slog.Error("auth_glue: google provider init failed", "error", err)
		return nil
	}
	return p
}

// mailerWrapper bridges pkg/auth/adapter/emailcode.Mailer → pkg/email.Module.
//
// auth doesn't import email — it declares its own tiny Mailer interface.
// This adapter is the connector both modules expect to live in the host.
type mailerWrapper struct{ mod *email.Module }

func (w mailerWrapper) SendProviderTemplate(ctx context.Context, ref string, to []emailcode.EmailAddress, vars map[string]any) error {
	addrs := make([]emaildomain.Address, len(to))
	for i, a := range to {
		addrs[i] = emaildomain.Address{Name: a.Name, Email: a.Email}
	}
	_, err := w.mod.SendProviderTemplate(ctx, ref, addrs, vars)
	return err
}
