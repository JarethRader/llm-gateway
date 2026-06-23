package discovery

import (
	"context"
	"log/slog"
	"packages/lib/golang/shared/config"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

func Register(ctx context.Context, cfg config.BackendDiscovery, lgr *slog.Logger, sink func([]model.Backend)) {
	switch cfg.Mode {
	case "static":
		sink(LoadConfig(cfg))
		return
	case "dynamic":
		client := NewClient(
			lgr,
			cfg.ApiUrl,
			cfg.Interval,
			sink,
		)
		go client.Run(ctx)
		return
	case "endpointslice":
		// TODO
	}
}
