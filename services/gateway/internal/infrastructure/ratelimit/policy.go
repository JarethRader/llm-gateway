package ratelimit

import "github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"

// Scope identifies which dimension a limit applies to
const (
	ScopeKey    model.Scope = "key"
	ScopeModel  model.Scope = "model"
	ScopeGlobal model.Scope = "global"
)

// Limit is a (rate, burst) pair. Rate is tokens/sec; burst is bucket capacity
type Limit struct {
	Rate  float64
	Burst float64
}

func (l Limit) Enabled() bool { return l.Rate > 0 && l.Burst > 0 }

// Policy holds the configured limits each scope. Per-key and per-model
// limits may be overridden by id in the infrastructure store.
// ---
// The check order is Key -> Model -> Global with reverse order refund.
// A tenant exceeding its own quota is rejected before consuming shared
// capacity, which avoids false rejections of well-behaved tenants.
type Policy struct {
	Global       Limit
	DefaultModel Limit
	DefaultKey   Limit
	PerModel     map[model.LargeLanguageModelID]Limit
	PerKey       map[model.KeyID]Limit
}

func (p *Policy) KeyLimit(key model.KeyID) Limit {
	if lim, ok := p.PerKey[key]; ok {
		return lim
	}
	return p.DefaultKey
}

func (p *Policy) ModelLimit(model model.LargeLanguageModelID) Limit {
	if lim, ok := p.PerModel[model]; ok {
		return lim
	}
	return p.DefaultModel
}
