package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"packages/lib/golang/shared/observability"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/registry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type (
	Server struct {
		cfg       Config
		lgr       *slog.Logger
		router    *chi.Mux
		telemetry observability.TelemetryProviders
	}

	Config struct {
		Name                        string
		Addr                        string
		Port                        string
		ShutdownWaitSec             int
		GracefulShutdownWaitTimeSec int
	}
)

func NewServer(cfg Config, lgr *slog.Logger, telemetryProviders observability.TelemetryProviders) *Server {
	return &Server{
		cfg:       cfg,
		lgr:       lgr,
		router:    chi.NewMux(),
		telemetry: telemetryProviders,
	}
}

func (s *Server) Run(ctx context.Context) error {
	address := s.cfg.Addr + s.cfg.Port
	s.lgr.Info("server is starting...", slog.String("address", address))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wait, err := time.ParseDuration(fmt.Sprintf("%ds", s.cfg.ShutdownWaitSec))
	if err != nil {
		return fmt.Errorf("failed to create wait signal: %s", err)
	}

	registry, err := registry.Init(s.lgr)
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %s", err)
	}

	server := &http.Server{
		Addr:    address,
		Handler: s.router,
	}
	s.router.Use(otelhttp.NewMiddleware((s.cfg.Name)))
	s.router.Use(observability.HTTPMiddleware(nil))

	if err := s.RegisterRouteHandlers(s.router, registry); err != nil {
		return fmt.Errorf("failed to register route handlersL %s", err)
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

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, wait)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		s.lgr.Error("shutdown http server", slog.Any("error", err))
	}
	if err := s.telemetry.Shutdown(shutdownCtx); err != nil {
		s.lgr.Error("shutdown telemetry providers", slog.Any("error", err))
	}
	s.lgr.Info("server shutdown complete...")

	return nil
}
