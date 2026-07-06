package ratelimit

import (
	"time"
)

// TokenBucket implements the token-bucket rate-limiting algorithm.
// It is an internal detail of the shared Limiter. All access must be
// serialized by the caller holding the owning shard's mutex.
type TokenBucket struct {
	burst      int     // maximum burst for tokens
	tokens     float64 // current number of available tokens
	rate       float64 // tokens per second refill rate
	lastRefill time.Time
}

func NewBucket(burst int, tokensPerSecond float64, creationTime time.Time) *TokenBucket {
	return &TokenBucket{
		burst:      burst,
		tokens:     float64(burst),
		rate:       tokensPerSecond,
		lastRefill: creationTime,
	}
}

// tryTake refills the bucket to now, then attempts to remove n tokens.
// Returns whether the take succeeded, and — on failure — the duration until
// n tokens would be available. Must be called with the owning shard's mutex held.
func (b *TokenBucket) tryTake(n int, now time.Time) (bool, time.Duration) {
	b.refill(now)

	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		return true, 0
	}

	deficit := float64(n) - b.tokens
	wait := time.Duration(deficit / b.rate * float64(time.Second))
	return false, wait
}

// refill advances the bucket to now at the given rate, capped at burst.
// Must be called with the owning shard's held.
func (b *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	b.tokens += b.rate * elapsed
	if b.tokens > float64(b.burst) {
		b.tokens = float64(b.burst)
	}
	b.lastRefill = now
}

// Give returns N tokens (used to refund previously taken tokens).
func (b *TokenBucket) Give(n int) {
	b.tokens += float64(n)
	if b.tokens > float64(b.burst) {
		b.tokens = float64(b.burst)
	}
}
