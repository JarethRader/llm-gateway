package loadbalancer

import (
	"context"
	"log/slog"
	"math"
	"math/rand/v2"
	"packages/lib/golang/shared/config"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type backendLive struct {
	backend     model.Backend
	ttftBits    atomic.Uint64 // EWMA(TTFT ms) via math.Float64bits
	inFlight    atomic.Int64
	healthy     atomic.Bool
	cacheBits   atomic.Uint64 // float64 cache usage; updated by scraper on /metrics endpoint
	cacheStamp  atomic.Int64  // unixnano of last scrape
	maxInFlight int64
}

func (bl *backendLive) snapshot(m model.LargeLanguageModelID, refTTFTMs float64, ttl time.Duration) BackendStat {
	ttft := math.Float64frombits(bl.ttftBits.Load())
	if ttft == 0 {
		ttft = refTTFTMs
	}

	var cache uint64
	stamp := bl.cacheStamp.Load()
	if stamp != 0 && time.Since(time.Unix(0, stamp)) < ttl {
		cache = bl.cacheBits.Load()
	}

	return BackendStat{
		Serves:      bl.backend.Serves(m),
		Healthy:     bl.healthy.Load(),
		BreakerOpen: false, // TODO get circuit breaker status
		TtftEwmaMs:  ttft,
		InFlight:    int(bl.inFlight.Load()),
		MaxInFlight: int(bl.maxInFlight),
		CacheUsage:  math.Float64frombits(cache),
		Weight:      bl.backend.Weight,
	}
}

func (bl *backendLive) observe(sampleMS, alpha float64) {
	for {
		old := math.Float64frombits(bl.ttftBits.Load())
		next := old
		if old == 0 {
			next = sampleMS
		} else {
			next = old + alpha*(sampleMS-old)
		}
		if bl.ttftBits.CompareAndSwap(math.Float64bits(old), math.Float64bits(next)) {
			return
		}
	}
}

// TODO implement circuit breaker
type LoadBalancer struct {
	weights Weights
	mu      sync.RWMutex // guards the map shape only
	live    map[model.BackendID]*backendLive
	cfg     config.LoadBalancer
}

func New(cfg config.LoadBalancer, lgr *slog.Logger) ports.LoadBalancer {
	return &LoadBalancer{
		weights: Weights{
			Latency:   cfg.Weights.Latency,
			InFlight:  cfg.Weights.InFlight,
			Cache:     cfg.Weights.Cache,
			RefTTFTMs: float64(cfg.Weights.RefTTFTMS),
		},
		live: make(map[model.BackendID]*backendLive),
		cfg:  cfg,
	}
}

// Select implements [ports.LoadBalancer].
func (r *LoadBalancer) Select(ctx context.Context, m model.LargeLanguageModelID) (model.BackendID, bool) {
	r.mu.RLock()
	stats := make([]BackendStat, 0, len(r.live))
	for id, bl := range r.live {
		st := bl.snapshot(m, r.weights.RefTTFTMs, r.cfg.Scrape.CacheTTL)
		st.ID = id
		stats = append(stats, st)
	}
	r.mu.RUnlock()
	return SelectP2C(stats, r.weights, func(n int) int { return rand.IntN(n) })
}

// Observe implements [ports.LoadBalancer].
func (r *LoadBalancer) Observe(b model.BackendID, ttftMS float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if bl, ok := r.live[b]; ok {
		bl.observe(ttftMS, r.cfg.EwmaAlpha)
	}
}

// Sync implements [ports.LoadBalancer].
func (r *LoadBalancer) Sync(desired []model.Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()

	old := r.live
	next := make(map[model.BackendID]*backendLive, len(desired))
	for _, b := range desired {
		if c, ok := old[b.ID]; ok {
			next[b.ID] = c
			continue
		}
		next[b.ID] = &backendLive{
			backend:     b,
			maxInFlight: int64(r.cfg.MaxInFlightPerBackend),
		}
		next[b.ID].healthy.Store(true)
	}
	r.live = next
}

// Dec implements [ports.LoadBalancer].
func (r *LoadBalancer) Dec(b model.BackendID) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if bl, ok := r.live[b]; ok {
		bl.inFlight.Add(-1)
	}
}

// Inc implements [ports.LoadBalancer].
func (r *LoadBalancer) Inc(b model.BackendID) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if bl, ok := r.live[b]; ok {
		bl.inFlight.Add(1)
	}
}
