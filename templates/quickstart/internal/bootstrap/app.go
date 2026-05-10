package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/brizenchi/go-modules/foundation/pgx"
	fslog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/brizenchi/go-modules/stacks/saascore"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	billinghandler "github.com/brizenchi/quickstart-template/internal/http/handler/billing"
	httpmiddleware "github.com/brizenchi/quickstart-template/internal/http/middleware"
	billingentity "github.com/brizenchi/quickstart-template/internal/model/entity/billing"
	billingservice "github.com/brizenchi/quickstart-template/internal/service/billing"
)

type App struct {
	Config        AppConfig
	Server        *http.Server
	traceShutdown func(context.Context) error
}

func New() (app *App, err error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	fslog.Setup(fslog.Config{
		Level:    cfg.Log.Level,
		Format:   fslog.Format(cfg.Log.Format),
		Defaults: logDefaults(cfg),
	})

	traceShutdown, err := tracing.Setup(tracing.Config{
		ServiceName: cfg.Server.Name,
		Project:     cfg.Project,
		Environment: cfg.Env,
		Endpoint:    cfg.Tracing.Endpoint,
		Protocol:    cfg.Tracing.Protocol,
		Insecure:    cfg.Tracing.Insecure,
		SampleRate:  cfg.Tracing.SampleRate,
		Headers:     cfg.Tracing.ExporterHeaders(),
		URLPath:     cfg.Tracing.URLPath,
	})
	if err != nil {
		return nil, fmt.Errorf("tracing.Setup: %w", err)
	}
	defer func() {
		if err != nil {
			tracing.Shutdown(context.Background(), traceShutdown)
		}
	}()

	db, err := pgx.Open(cfg.DB.PGXConfig(cfg.Project, cfg.Env))
	if err != nil {
		return nil, fmt.Errorf("pgx.Open: %w", err)
	}
	if err := pgx.HealthCheck(context.Background(), db); err != nil {
		return nil, fmt.Errorf("pgx.HealthCheck: %w", err)
	}
	slog.Info("db ready", "dsn_safe", cfg.DB.SafeString())

	stack, err := saascore.New(
		db,
		cfg.SaaSCoreConfig(),
		saascore.HostHooks{
			OnReferralActivated: func(ctx context.Context, event saascore.ReferralActivatedEvent) error {
				return applyReferralReward(ctx, db, event)
			},
		},
		saascore.PolicyHooks{},
	)
	if err != nil {
		return nil, fmt.Errorf("saascore.New: %w", err)
	}
	if err := db.AutoMigrate(&billingentity.StripeTopUpEvent{}); err != nil {
		return nil, fmt.Errorf("migrate stripe top-up events: %w", err)
	}

	topUpService := billingservice.NewStripeTopUpService(
		cfg.StripeTopUpRuntimeConfig(),
		stack.DB,
		stack.Users,
		stack.Billing.Provider,
		stack.Billing.Customers,
		stack.Billing.UserResolver,
	)
	topUpHandler := billinghandler.NewStripeTopUpHandler(topUpService)
	router := apphttp.NewRouter(stack, topUpHandler)
	engine := httpmiddleware.BuildRouter(httpmiddleware.RouterConfig{ServiceName: cfg.Server.Name}, router)

	return &App{
		Config: cfg,
		Server: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: engine,
		},
		traceShutdown: traceShutdown,
	}, nil
}

func (a *App) ShutdownTracing(ctx context.Context) {
	if a == nil || a.traceShutdown == nil {
		return
	}
	tracing.Shutdown(ctx, a.traceShutdown)
}
