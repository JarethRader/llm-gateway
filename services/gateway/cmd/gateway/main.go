package main

import (
	"context"
	"log"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/server"
)

var (
	CommitHash string
	Tag        string
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx, CommitHash, Tag)
	if err != nil || cfg == nil {
		log.Fatalf("failed to load application configuration: %s", err)
	}

	telemetry, err := observability.InitTracer(ctx, observability.ProviderConfig{
		ServiceName: cfg.App.Name,
		Version:     cfg.App.Version,
		Environment: cfg.App.Environment,
		Enabled:     cfg.Telemetry.Enabled,
		Endpoint:    cfg.Telemetry.Endpoint,
		Insecure:    cfg.Telemetry.Insecure,
		SampleRatio: cfg.Telemetry.SampleRatio,
	})
	if err != nil {
		log.Fatalf("failed to initialize telemetry providers: %s", err)
	}

	lgr := observability.NewLogger(cfg.App.Name, cfg.App.Version, cfg.App.LogLevel, cfg.App.Environment)
	lgr.Info("starting llm-gateway service...")

	s := server.NewServer(
		*cfg,
		lgr,
		*telemetry,
	)

	if err := s.Run(ctx); err != nil {
		log.Fatalf("failed to run server: %s", err)
	}
}
