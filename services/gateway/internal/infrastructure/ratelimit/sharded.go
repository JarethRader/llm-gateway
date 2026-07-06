package ratelimit

import (
	"context"
	"math"
	"packages/lib/golang/shared/config"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

const shardCount = 256 // power of two; index = hash(id) & (shardCount-1)

type entry struct {
	bucket *TokenBucket
	limit  Limit
	seen   int64
}

type shard struct {
	mu      sync.Mutex
	buckets map[string]*entry // key = string(scope)+"\x00"+id
}

// getOrInit must be called with sh.mu already held. It does not acquire the lock.
func (sh *shard) getOrInit(key string, limit Limit, now time.Time) *entry {
	if sh.buckets == nil {
		sh.buckets = make(map[string]*entry)
	}

	if e, ok := sh.buckets[key]; ok {
		if e.limit != limit {
			e.limit = limit
			e.bucket.rate = limit.Rate
			e.bucket.burst = int(limit.Burst)
			e.bucket.tokens = math.Min(e.bucket.tokens, limit.Burst)
		}
		e.seen = now.UnixNano()
		return e
	}

	e := &entry{
		bucket: NewBucket(int(limit.Burst), limit.Rate, now),
		limit:  limit,
		seen:   now.UnixNano(),
	}
	sh.buckets[key] = e
	return e
}

type taken struct {
	sh *shard
	k  string
}

type Limiter struct {
	shards        [shardCount]shard
	policy        atomic.Pointer[Policy]
	maxRetry      time.Duration
	sweepInterval time.Duration
	idleTTL       time.Duration
}

func NewLimiter(cfg config.RateLimit) *Limiter {
	limiter := &Limiter{}

	if cfg.MaxRetryAfter == 0 {
		limiter.maxRetry = 30 * time.Second
	} else {
		limiter.maxRetry = cfg.MaxRetryAfter
	}

	if cfg.SweepInterval == 0 {
		limiter.sweepInterval = 30 * time.Second
	} else {
		limiter.sweepInterval = cfg.SweepInterval
	}

	if cfg.IdleTTL == 0 {
		limiter.idleTTL = 5 * time.Minute
	} else {
		limiter.idleTTL = cfg.IdleTTL
	}

	policy := PolicyFromConfig(cfg)
	limiter.policy.Store(&policy)
	return limiter
}

func (l *Limiter) SetPolicy(policy Policy) {
	l.policy.Store(&policy)
}

func PolicyFromConfig(c config.RateLimit) Policy {
	policy := Policy{
		Global:       convertPolicyToLimit(c.Global),
		DefaultModel: convertPolicyToLimit(c.DefaultModel),
		DefaultKey:   convertPolicyToLimit(c.DefaultKey),
		PerModel:     make(map[model.LargeLanguageModelID]Limit),
		PerKey:       make(map[model.KeyID]Limit),
	}
	for k, v := range c.PerModel {
		policy.PerModel[model.LargeLanguageModelID(k)] = convertPolicyToLimit(v)
	}
	for k, v := range c.PerKey {
		policy.PerKey[model.KeyID(k)] = convertPolicyToLimit(v)
	}

	return policy
}

func convertPolicyToLimit(p config.RatePolicy) Limit {
	return Limit{
		Rate:  float64(p.RatePerSec),
		Burst: float64(p.Burst),
	}
}

func (l *Limiter) Run(ctx context.Context) {
	interval := l.sweepInterval
	ttl := l.idleTTL

	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.sweep(time.Now().Add(-ttl))
		}
	}
}

func (l *Limiter) Allow(key model.KeyID, m model.LargeLanguageModelID, tokens int) model.Decision {
	policy := l.policy.Load()

	var got []taken
	order := []struct {
		scope model.Scope
		id    string
		limit Limit
	}{
		{ScopeKey, string(key), policy.KeyLimit(key)},
		{ScopeModel, string(m), policy.ModelLimit(m)},
		{ScopeGlobal, "-", policy.Global},
	}
	now := time.Now()
	for _, o := range order {
		if !o.limit.Enabled() {
			continue
		}
		sh, k := l.shardFor(o.scope, o.id)
		sh.mu.Lock()
		e := sh.getOrInit(k, o.limit, now)
		ok, wait := e.bucket.tryTake(tokens, now)
		sh.mu.Unlock()

		if !ok {
			for i := len(got) - 1; i >= 0; i-- {
				t := got[i]
				t.sh.mu.Lock()
				if pe, exists := t.sh.buckets[t.k]; exists {
					pe.bucket.Give(tokens)
				}
				t.sh.mu.Unlock()
			}
			return model.Decision{
				Allowed:    false,
				Scope:      o.scope,
				RetryAfter: l.clampRetry(wait),
			}
		}
		got = append(got, taken{sh: sh, k: k})
	}
	return model.Decision{Allowed: true}
}

func (l *Limiter) sweep(olderThan time.Time) {
	cutoff := olderThan.UnixNano()
	for i := range l.shards {
		sh := &l.shards[i]
		sh.mu.Lock()
		for k, e := range sh.buckets {
			if e.seen < cutoff {
				delete(sh.buckets, k)
			}
		}
		sh.mu.Unlock()
	}
}

func (l *Limiter) shardFor(scope model.Scope, id string) (*shard, string) {
	key := string(scope) + "\x00" + id
	h := fnv1a(key)

	// bitwise AND &(shardCount-1) is a branchless modulo to avoid costly `div` and `%` operations.
	// shardCount must be a power-of-two for this to work
	return &l.shards[h&(shardCount-1)], key
}

func fnv1a(s string) uint64 {
	h := uint64(0x811c9dc5) // FNV offset basis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i]) // XOR step first
		h *= 0x01000193   // Multiply step second by FNV prime
	}
	return h
}

func (l *Limiter) clampRetry(wait time.Duration) time.Duration {
	if wait < 0 {
		return 0
	}
	if wait > l.maxRetry {
		return l.maxRetry
	}
	return wait
}

var _ ports.Limiter = (*Limiter)(nil)
