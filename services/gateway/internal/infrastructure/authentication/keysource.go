package authentication

import (
	"fmt"
	"log/slog"
	"packages/lib/golang/shared/config"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

// Register selects a key source and feeds keys into the sink (Authenticator.Sync)
func Register(cfg config.Auth, lgr *slog.Logger, sink func([]model.Identity)) error {
	switch cfg.Mode {
	case "static":
		keys, err := LoadFile(cfg.KeyTablePath)
		if err != nil {
			return fmt.Errorf("keystore static: %w", err)
		}
		sink(keys)
		lgr.Info("keysource loaded static keys", slog.Int("count", len(keys)))
		return nil
	case "dynamic":
		// TODO
		return fmt.Errorf("not implemented")
	default:
		return fmt.Errorf("keysource: unknown mode: %q", cfg.Mode)
	}
}
