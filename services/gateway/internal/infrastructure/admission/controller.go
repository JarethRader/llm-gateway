package admission

import (
	"context"
	"packages/lib/golang/shared/config"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type Controller struct {
	sem      chan struct{}
	maxQueue int64
	queued   atomic.Int64
	inFlight atomic.Int64
	est      *Estimator
	maxRetry time.Duration
}

func NewController(cfg config.Admission) *Controller {
	maxRetry := cfg.MaxRetryAfter
	if maxRetry <= 0 {
		maxRetry = 30 * time.Second
	}
	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 64
	}
	maxQueue := cfg.MaxQueue
	if maxQueue <= 0 {
		maxQueue = 256
	}
	controller := &Controller{
		sem:      make(chan struct{}, maxConcurrent),
		maxQueue: int64(maxQueue),
		est:      NewEstimator(cfg.EwmaAlpha),
		maxRetry: maxRetry,
	}
	return controller
}

// Acquire blocks until a slot is free, the queue is full, or ctx is done.
func (c *Controller) Acquire(ctx context.Context) (ports.Permit, model.Decision) {
	if c.queued.Add(1) > c.maxQueue {
		c.queued.Add(-1)
		return nil, model.Decision{
			Allowed:    false,
			Scope:      model.Scope("admission"),
			RetryAfter: c.retryAfter(),
		}
	}

	select {
	case c.sem <- struct{}{}:
		c.queued.Add(-1)
		c.inFlight.Add(1)
		return &permit{c: c}, model.Decision{Allowed: true}
	case <-ctx.Done():
		c.queued.Add(-1)
		return nil, model.Decision{Allowed: false}
	}
}

func (c *Controller) retryAfter() time.Duration {
	mu := c.est.Rate()
	if mu <= 0 {
		return c.maxRetry
	}

	wait := time.Duration(float64(c.queued.Load()) / mu * float64(time.Second))
	if wait < 0 {
		wait = c.maxRetry
	}
	return min(max(wait, time.Second), c.maxRetry)
}

func (c *Controller) QueueDepth() int {
	return int(c.queued.Load())
}

func (c *Controller) InFlight() int {
	return int(c.inFlight.Load())
}

type permit struct {
	c    *Controller
	once sync.Once
}

func (p *permit) Release() {
	p.once.Do(func() {
		p.c.inFlight.Add(-1)
		<-p.c.sem
		p.c.est.MarkCompletion()
	})
}

var _ ports.Admitter = (*Controller)(nil)
