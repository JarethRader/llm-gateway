package authentication

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"sync/atomic"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

// ConfigAuthenticator holds the hot-swappable key table.
// Authenticate reads the current snapshot; Sync replaces it atomically.
type ConfigAuthenticator struct {
	table atomic.Pointer[[]model.Identity]
}

func New() *ConfigAuthenticator {
	a := &ConfigAuthenticator{}
	var empty []model.Identity
	a.table.Store(&empty) // fail-closed until first Sync
	return a
}

// Sync replaces the whole table in one atomic store.
// In-flight Authenticate calls keep reading the previous snapshot;
// the next call sees the new one.
func (a *ConfigAuthenticator) Sync(keys []model.Identity) {
	cp := make([]model.Identity, len(keys))
	copy(cp, keys)
	a.table.Store(&cp)
}

func (a *ConfigAuthenticator) Authenticate(_ context.Context, bearer string) (model.Identity, error) {
	if bearer == "" {
		return model.Identity{}, model.ErrEmptyBearer
	}
	sum := sha256.Sum256([]byte(bearer))
	keys := *a.table.Load()

	var match model.Identity
	found := false
	for i := range keys {
		if subtle.ConstantTimeCompare(keys[i].Digest[:], sum[:]) == 1 {
			match = keys[i]
			found = true
		}
	}
	if !found {
		return model.Identity{}, model.ErrUnknownKey
	}
	return match, nil
}
