package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/authentication"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/discovery"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/registry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	cfg       config.Config
	lgr       *slog.Logger
	router    *chi.Mux
	telemetry observability.TelemetryProviders
}

func NewServer(cfg config.Config, lgr *slog.Logger, telemetryProviders observability.TelemetryProviders) *Server {
	return &Server{
		cfg:       cfg,
		lgr:       lgr,
		router:    chi.NewMux(),
		telemetry: telemetryProviders,
	}
}

func (s *Server) Run(ctx context.Context) error {
	address := s.cfg.App.Addr + s.cfg.App.Port
	s.lgr.Info("server is starting...", slog.String("address", address))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wait, err := time.ParseDuration(fmt.Sprintf("%ds", s.cfg.ShutdownWaitSec))
	if err != nil {
		return fmt.Errorf("failed to create wait signal: %s", err)
	}

	registry, err := registry.Init(s.cfg, s.lgr)
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %s", err)
	}

	middleware := NewMiddleware(s.lgr.With("component", "middleware"))

	server := &http.Server{
		Addr:    address,
		Handler: s.router,
	}
	s.router.Use(otelhttp.NewMiddleware((s.cfg.App.Name)))
	s.router.Use(observability.HTTPMiddleware(nil))
	s.router.Use(middleware.Recovery)

	if err := authentication.Register(s.cfg.Authentication, s.lgr.With("component", "keysource"), registry.Authenticator.Sync); err != nil {
		return fmt.Errorf("failed to register key source: %w", err)
	}

	if s.cfg.RateLimit.Enabled {
		go registry.RateLimiter.Run(ctx)
	}

	if err := discovery.Register(ctx, s.cfg.BackendDiscovery, s.lgr.With("component", "discovery"), func(backends []model.Backend) {
		registry.ConnectionPool.Sync(backends)
		registry.LoadBalancer.Sync(backends)
	}); err != nil {
		return fmt.Errorf("failed to register backend discovery: %s", err)
	}

	if err := s.RegisterRouteHandlers(s.router, registry, middleware); err != nil {
		return fmt.Errorf("failed to register route handlers: %s", err)
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			s.lgr.Error("start server and listen", slog.Any("error", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	s.lgr.Info("server is stopping...")

	var shutdownWG sync.WaitGroup

	serverShutdownCtx, serverShutdownCancel := context.WithTimeout(ctx, wait)
	defer serverShutdownCancel()
	shutdownWG.Add(1)
	go func() {
		defer shutdownWG.Done()
		if err := server.Shutdown(serverShutdownCtx); err != nil {
			s.lgr.Error("shutdown http server", slog.Any("error", err))
		}
	}()

	serverTelemetryCtx, serverTelemetryCancel := context.WithTimeout(ctx, wait/4)
	defer serverTelemetryCancel()
	shutdownWG.Add(1)
	go func() {
		defer shutdownWG.Done()
		if err := s.telemetry.Shutdown(serverTelemetryCtx); err != nil {
			s.lgr.Error("shutdown telemetry providers", slog.Any("error", err))
		}
	}()

	shutdownWG.Wait()
	s.lgr.Info("server shutdown complete...")
	cancel()

	return nil
}
