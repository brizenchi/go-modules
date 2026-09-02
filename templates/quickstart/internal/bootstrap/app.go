// TEMPLATE-OWNED — avoid editing; changes here conflict on upgrade.
// Use host_hooks.go / host_routes.go / host_jobs.go / host_migrate.go
// and internal/hostcfg to plug in your own code.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/foundation/pgx"
	fslog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	httpmiddleware "github.com/brizenchi/quickstart-template/internal/http/middleware"
	"github.com/brizenchi/quickstart-template/internal/platform"
)

// shutdownTimeout bounds graceful shutdown: in-flight requests first,
// then background runners.
const shutdownTimeout = 30 * time.Second

type App struct {
	Config  AppConfig
	Server  *http.Server
	Modules *platform.Modules

	runners       []Runner
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

	moduleCfg := cfg.ModuleConfig()
	if err := platform.Migrate(db, moduleCfg); err != nil {
		return nil, fmt.Errorf("platform.Migrate: %w", err)
	}
	modules, err := platform.New(db, moduleCfg)
	if err != nil {
		return nil, fmt.Errorf("platform.New: %w", err)
	}

	// Host feature tables are migrated after the host user and module tables, so they may
	// reference users(id).
	if models := hostModels(); len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			return nil, fmt.Errorf("migrate host models: %w", err)
		}
		slog.Info("host models migrated", "count", len(models))
	}

	deps := hostapi.Deps{DB: db, Modules: modules, Users: modules.Users, Config: cfg.Host}
	subscribeModuleEvents(deps, cfg)

	router := apphttp.NewRouter(modules, deps)
	engine := httpmiddleware.BuildRouter(httpmiddleware.RouterConfig{
		ServiceName:    cfg.Server.Name,
		AllowedOrigins: cfg.HTTP.AllowedOriginList(),
	}, router)

	return &App{
		Config:  cfg,
		Modules: modules,
		Server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:           engine,
			ReadHeaderTimeout: time.Duration(cfg.HTTP.ReadHeaderTimeoutSecond) * time.Second,
			ReadTimeout:       time.Duration(cfg.HTTP.ReadTimeoutSeconds) * time.Second,
			WriteTimeout:      time.Duration(cfg.HTTP.WriteTimeoutSeconds) * time.Second,
			IdleTimeout:       time.Duration(cfg.HTTP.IdleTimeoutSeconds) * time.Second,
		},
		runners:       buildHostJobs(deps),
		traceShutdown: traceShutdown,
	}, nil
}

// Run starts the HTTP server and every registered runner, then blocks
// until ctx is cancelled or the server fails. On return everything has
// been shut down.
func (a *App) Run(ctx context.Context) error {
	runnerCtx, stopRunners := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for _, r := range a.runners {
		wg.Add(1)
		go func(r Runner) {
			defer wg.Done()
			slog.Info("runner started", "runner", r.Name())
			if err := r.Start(runnerCtx); err != nil {
				slog.Error("runner stopped with error", "runner", r.Name(), "error", err)
			}
		}(r)
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", a.Config.Server.Port)
		if err := a.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	var runErr error
	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-serverErr:
		runErr = err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Drain in-flight requests first, then let runners finish.
	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown", "error", err)
	}

	stopRunners()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
		slog.Warn("runners did not stop before timeout")
	}

	a.ShutdownTracing(shutdownCtx)
	return runErr
}

func (a *App) ShutdownTracing(ctx context.Context) {
	if a == nil || a.traceShutdown == nil {
		return
	}
	tracing.Shutdown(ctx, a.traceShutdown)
}
