package authentication

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

// LoadFile reads a key table JSON file and returns []model.Identity.
//
//	[{"sha256":"<hex>","key_id":"k_123","tier":"free"}, ...]
func LoadFile(path string) ([]model.Identity, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load key table: %w", err)
	}

	var entries []struct {
		SHA256 string `json:"sha256"`
		KeyID  string `json:"key_id"`
		Tier   string `json:"tier"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse key table: %w", err)
	}

	keys := make([]model.Identity, 0, len(entries))
	for _, e := range entries {
		digest, err := hex.DecodeString(e.SHA256)
		if err != nil {
			return nil, fmt.Errorf("invalid sha256 for key_id %s: %w", e.KeyID, err)
		}
		if len(digest) != 32 {
			return nil, fmt.Errorf("invalid digest for key_id %s: expected 32 bytes but got %d", e.KeyID, len(digest))
		}
		var d [32]byte
		copy(d[:], digest)
		keys = append(keys, model.Identity{
			Digest: d,
			KeyID:  model.KeyID(e.KeyID),
			Tier:   model.Tier(e.Tier),
		})
	}
	return keys, nil
}
