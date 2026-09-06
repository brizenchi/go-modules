// Command preview runs a LOCAL, THROWAWAY fixture for browser acceptance testing.
// It never loads .env, deployment configuration, or a configured database. Email
// uses the log adapter, OAuth and billing are disabled, and uploads stay inside a
// new temporary directory. Do not deploy this command: login codes are visible.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/brizenchi/go-modules/modules/email"
	emaildomain "github.com/brizenchi/go-modules/modules/email/domain"
	emailport "github.com/brizenchi/go-modules/modules/email/port"
	refdomain "github.com/brizenchi/go-modules/modules/referral/domain"
	refevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/feature/credits"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/feature/operations"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	"github.com/brizenchi/quickstart-template/internal/http/middleware"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	previewAdmin                = "preview-admin@example.test"
	previewUser                 = "preview-user@example.test"
	previewInitialCredits int64 = 50
)

type previewApp struct {
	handler   http.Handler
	modules   *platform.Modules
	db        *sql.DB
	directory string
	closeOnce sync.Once
}

// The shared log adapter intentionally omits message bodies. This command-only
// decorator prints local fixture codes; it is never registered by the real app.
type previewLogSender struct{ emailport.Sender }

func (sender previewLogSender) Send(ctx context.Context, message *emaildomain.Message) (*emaildomain.Receipt, error) {
	receipt, err := sender.Sender.Send(ctx, message)
	if err == nil {
		log.Printf("LOCAL PREVIEW CODE · %s · %s", message.To[0].Email, message.TextBody)
	}
	return receipt, err
}

func (app *previewApp) Close() {
	app.closeOnce.Do(func() {
		if app.db != nil {
			_ = app.db.Close()
		}
		// This path is generated internally by MkdirTemp, never supplied by flags.
		if app.directory != "" {
			_ = os.RemoveAll(app.directory)
		}
	})
}

func localOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("frontend must be a loopback HTTP origin, such as http://localhost:3100")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return "", errors.New("frontend must use localhost or a loopback IP address")
	}
	if rawPort := u.Port(); rawPort != "" {
		port, err := strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return "", errors.New("invalid frontend port")
		}
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

func newPreview(frontend, adminPassword string) (_ *previewApp, err error) {
	frontend, err = localOrigin(frontend)
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "quickstart-local-preview-")
	if err != nil {
		return nil, err
	}
	app := &previewApp{directory: directory}
	defer func() {
		if err != nil {
			app.Close()
		}
	}()
	var secret [32]byte
	if _, err = rand.Read(secret[:]); err != nil {
		return nil, err
	}
	off, on := false, true
	// Every provider choice is explicit. Environment variables cannot enable an
	// external mailer, payment provider, OAuth provider, database, or object store.
	cfg := platform.Config{
		ServiceName: "quickstart-local-preview",
		Auth: platform.AuthConfig{Enabled: &on, UserJWTSecret: hex.EncodeToString(secret[:]), UserJWTExpireHours: 4, AdminEmails: []string{previewAdmin}, FrontendRedirect: frontend,
			Email:  platform.AuthEmailConfig{Enabled: &on, Debug: true, Code: platform.AuthEmailCodeConfig{TTLMinutes: 10, MinResendGapSeconds: 1, DailyCap: 1000, MaxAttempts: 5}},
			Google: platform.GoogleConfig{Enabled: &off}, GitHub: platform.GitHubConfig{Enabled: &off}},
		Email:    platform.EmailConfig{Provider: "log"},
		Billing:  platform.BillingConfig{Enabled: &off},
		Referral: platform.ReferralConfig{Enabled: &on, Prefix: "PREVIEW", BaseLink: frontend + "/invite", ActivationReward: 25, ActivationWindowDays: 30},
	}
	if adminPassword != "" {
		cfg.Auth.AdminEmail = previewAdmin
		cfg.Auth.AdminPassword = adminPassword
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(directory, "preview.sqlite")+"?_busy_timeout=10000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	app.db, err = db.DB()
	if err != nil {
		return nil, err
	}
	app.db.SetMaxOpenConns(4)
	if err = platform.Migrate(db, cfg); err != nil {
		return nil, err
	}
	models := append([]any{&note.Note{}}, credits.Models()...)
	models = append(models, operations.Models()...)
	if err = db.AutoMigrate(models...); err != nil {
		return nil, err
	}
	app.modules, err = platform.New(db, cfg)
	if err != nil {
		return nil, err
	}
	*app.modules.Email = *email.New(previewLogSender{Sender: app.modules.Email.Sender}, nil)
	for _, fixture := range []struct{ email, role string }{{previewAdmin, user.RoleAdmin}, {previewUser, user.RoleUser}} {
		account := &user.User{Email: fixture.email, Role: fixture.role, Credits: previewInitialCredits}
		if err = app.modules.Users.Create(context.Background(), account); err != nil {
			return nil, err
		}
		if err = db.Create(&note.Note{UserID: account.ID, Title: "Welcome to your local preview", Body: "This note belongs to a temporary local account. Export it to test credit consumption. All preview data is removed when the command exits."}).Error; err != nil {
			return nil, err
		}
	}
	installPreviewListeners(app.modules)
	deps := hostapi.Deps{DB: db, Modules: app.modules, Users: app.modules.Users, Config: hostcfg.Config{SignupCredits: previewInitialCredits, Uploads: hostcfg.UploadConfig{Enabled: true, Provider: "local", Directory: filepath.Join(directory, "private-images")}}}
	// Reuse production auth middleware, API routes, settings, and feature wiring.
	app.handler = middleware.BuildRouter(middleware.RouterConfig{ServiceName: cfg.ServiceName, AllowedOrigins: []string{frontend, "http://localhost:3000", "http://localhost:3100", "http://127.0.0.1:3000", "http://127.0.0.1:3100"}}, apphttp.NewRouter(app.modules, deps))
	return app, nil
}

func installPreviewListeners(modules *platform.Modules) {
	attribute := func(ctx context.Context, userID string) error {
		code := platform.ReferralCodeFromContext(ctx)
		if code == "" {
			return nil
		}
		_, err := modules.Referral.Attribute.AttributeReferral(ctx, userID, code)
		if errors.Is(err, refdomain.ErrAlreadyAttributed) {
			return nil
		}
		return err
	}
	modules.Auth.Subscribe(authevent.KindUserSignedUp, func(ctx context.Context, event authevent.Envelope) error {
		return errors.Join(attribute(ctx, event.UserID), modules.Users.GrantCredits(ctx, event.UserID, "signup", event.UserID, previewInitialCredits))
	})
	modules.Auth.Subscribe(authevent.KindUserLoggedIn, func(ctx context.Context, event authevent.Envelope) error {
		payload, ok := event.Payload.(authevent.UserLoggedIn)
		if !ok {
			return errors.New("unexpected preview login payload")
		}
		if payload.Identity.IsNew {
			return attribute(ctx, event.UserID)
		}
		return nil
	})
	modules.Referral.Subscribe(refevent.KindReferralActivated, func(ctx context.Context, event refevent.Envelope) error {
		payload, ok := event.Payload.(refevent.ReferralActivated)
		if !ok {
			return errors.New("unexpected preview referral payload")
		}
		return modules.Users.GrantCredits(ctx, payload.Referral.ReferrerID, "referral", strconv.FormatUint(payload.Referral.ID, 10), int64(payload.Referral.RewardCredits))
	})
}

func runPreview(ctx context.Context, port int, frontend string) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	// Only this explicitly supported credential is read from the environment.
	// Provider and database configuration remain isolated from deployment values.
	app, err := newPreview(frontend, os.Getenv("APP_AUTH_ADMIN_PASSWORD"))
	if err != nil {
		return err
	}
	defer app.Close()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: app.handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	stopped := make(chan struct{})
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		case <-stopped:
		}
	}()
	log.Printf("LOCAL FIXTURE PREVIEW ONLY: http://%s · temporary SQLite and files: %s", address, app.directory)
	log.Printf("Email codes appear in this terminal and the API debug response. No external email, OAuth or payments are enabled.")
	log.Printf("Sign in normally with %s (admin) or %s (user); each starts with %d credits. Frontend: %s", previewAdmin, previewUser, previewInitialCredits, frontend)
	if app.modules.Config.Auth.AdminEmail != "" {
		log.Printf("Administrator password login is enabled at %s/admin for %s; use the password supplied in APP_AUTH_ADMIN_PASSWORD.", frontend, previewAdmin)
	} else {
		log.Printf("To enable the /admin password form, restart with APP_AUTH_ADMIN_PASSWORD set to a 12–72 byte local test password.")
	}
	err = server.Serve(listener)
	close(stopped)
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Local browser acceptance fixture only: temporary SQLite, visible email codes, no external integrations. Never deploy this command.")
		flag.PrintDefaults()
	}
	port := flag.Int("port", 18081, "loopback API port (binds only 127.0.0.1)")
	frontend := flag.String("frontend", "http://localhost:3100", "loopback HTTP frontend origin used for CORS and invitation links")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := runPreview(ctx, *port, *frontend); err != nil {
		log.Printf("preview stopped: %v", err)
		os.Exit(1)
	}
}
