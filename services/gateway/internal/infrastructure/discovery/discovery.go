package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"packages/lib/golang/shared/config"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

func Register(ctx context.Context, cfg config.BackendDiscovery, lgr *slog.Logger, sink func([]model.Backend)) error {
	switch cfg.Mode {
	case "static":
		sink(LoadConfig(cfg))
		return nil
	case "dynamic":
		client := NewClient(
			lgr,
			cfg.ApiUrl,
			cfg.Interval,
			sink,
		)
		go client.Run(ctx)
		return nil
	case "endpointslice":
		// TODO
		return fmt.Errorf("not implemented")
	default:
		return fmt.Errorf("invalid backend discovery mode set: %s", cfg.Mode)
	}
}
