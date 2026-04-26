// Quickstart server: wires every go-modules package into a runnable Gin app.
//
// Boot order (mirrored in initialization here):
//
//	logger → config → DB → Redis → email → auth → billing → referral → routes → listen
//
// Each step is small enough to read top-to-bottom. To make this your
// project, copy this directory, rename the module path, and fill out
// the four `internal/*_glue` packages.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authevent "github.com/brizenchi/go-modules/auth/event"
	billingevent "github.com/brizenchi/go-modules/billing/event"
	"github.com/brizenchi/go-modules/foundation/config"
	"github.com/brizenchi/go-modules/foundation/ginx"
	"github.com/brizenchi/go-modules/foundation/pgx"
	fslog "github.com/brizenchi/go-modules/foundation/slog"

	"github.com/brizenchi/go-modules/templates/quickstart/internal/auth_glue"
	"github.com/brizenchi/go-modules/templates/quickstart/internal/billing_glue"
	"github.com/brizenchi/go-modules/templates/quickstart/internal/email_glue"
	"github.com/brizenchi/go-modules/templates/quickstart/internal/referral_glue"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// 1. Logger — must be first so everything else lands in slog.
	cfg := loadConfig()
	fslog.Setup(fslog.Config{
		Level:    cfg.Log.Level,
		Format:   fslog.Format(cfg.Log.Format),
		Defaults: map[string]any{"service": cfg.Server.Name},
	})

	// 2. DB — every module that persists takes *gorm.DB.
	db, err := pgx.Open(pgx.Config{DSN: cfg.DB.DSN})
	if err != nil {
		log.Fatalf("pgx.Open: %v", err)
	}
	if err := pgx.HealthCheck(context.Background(), db); err != nil {
		log.Fatalf("pgx.HealthCheck: %v", err)
	}
	slog.Info("db ready", "dsn_safe", cfg.DB.SafeString())

	// 3. Email — used by auth's email-code flow.
	emailMod := email_glue.Init(email_glue.Config{
		BrevoAPIKey:        viper.GetString("auth.email.brevo_api_key"),
		BrevoSenderEmail:   viper.GetString("auth.email.sender_email"),
		BrevoSenderName:    viper.GetString("auth.email.sender_name"),
		VerificationTplRef: "3",
	})

	// 4. Auth — wires email-code + Google OAuth + JWT sessions.
	authMod := auth_glue.Init(db, emailMod)

	// 5. Billing — Stripe checkout + webhooks. Listeners do project work.
	billingMod := billing_glue.Init(db)

	// 6. Referral — owns its own schema; auto-migrates.
	referralMod := referral_glue.Init(db)

	// 7. Cross-module bridges — host project decides what fires what.
	authMod.Subscribe(authevent.KindUserSignedUp, billing_glue.OnUserSignedUp)
	billingMod.Subscribe(billingevent.KindSubscriptionActivated, referral_glue.OnSubscriptionActivated)

	// 8. HTTP.
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginx.Recover(), ginx.RequestID(), ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: []string{"/health"}}))
	r.Use(ginx.CORS(ginx.CORSConfig{AllowedOrigins: []string{"*"}}))
	r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{}))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	publicGroup := r.Group("/api/v1")
	publicGroup.POST("/auth/send-code", authMod.Handler.SendCode)
	publicGroup.POST("/auth/verify-code", authMod.Handler.VerifyCode)
	publicGroup.GET("/auth/:provider/authorize", authMod.Handler.StartOAuth)
	publicGroup.GET("/auth/:provider/callback", authMod.Handler.OAuthCallback)
	publicGroup.POST("/auth/exchange-token", authMod.Handler.ExchangeToken)
	publicGroup.POST("/stripe/webhook", billingMod.Handler.HandleWebhook)

	userGroup := r.Group("/api/v1")
	userGroup.Use(auth_glue.RequireUser(authMod)) // Bearer auth
	{
		userGroup.POST("/auth/refresh", authMod.Handler.Refresh)
		userGroup.POST("/auth/logout", authMod.Handler.Logout)
		userGroup.POST("/websocket/ticket", authMod.Handler.IssueWSTicket)

		userGroup.POST("/stripe/checkout/session", billingMod.Handler.CreateCheckoutSession)
		userGroup.POST("/stripe/subscription/cancel", billingMod.Handler.CancelSubscription)
		userGroup.POST("/stripe/subscription/reactivate", billingMod.Handler.ReactivateSubscription)
		userGroup.GET("/stripe/subscription", billingMod.Handler.GetSubscription)
		userGroup.GET("/stripe/invoices", billingMod.Handler.ListInvoices)

		userGroup.GET("/referral/code", referralMod.Handler.GetMyCode)
		userGroup.GET("/referral/list", referralMod.Handler.ListMyReferrals)
		userGroup.GET("/referral/stats", referralMod.Handler.GetMyStats)
	}

	// 9. Listen + graceful shutdown.
	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: r}
	go func() {
		slog.Info("listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "error", err)
	}
}

// AppConfig is the project-specific config shape.
type AppConfig struct {
	Server struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Log struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"log"`
	DB DBConfig `mapstructure:"db"`
}

type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

// SafeString returns a logging-friendly representation of the DSN with
// password masked. Adjust if your DSN format differs.
func (c DBConfig) SafeString() string {
	return "<dsn-masked>"
}

func loadConfig() AppConfig {
	path := os.Getenv("CONFIG")
	if path == "" {
		path = "deploy/config.yaml"
	}
	var cfg AppConfig
	if err := config.LoadGlobal(path, "APP", &cfg); err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Server.Name == "" {
		cfg.Server.Name = "quickstart"
	}
	return cfg
}
