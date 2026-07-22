package circuitbreaker

import (
	"context"
	"log/slog"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Manager struct {
	mu       sync.RWMutex
	circuits map[model.BackendID]*Breaker
	cfg      config.Circuit
	lgr      *slog.Logger
	meter    *observability.Metrics
}

func NewManager(cfg config.Circuit, lgr *slog.Logger, meter *observability.Metrics) ports.CircuitBreaker {
	manager := &Manager{
		circuits: make(map[model.BackendID]*Breaker),
		cfg:      cfg,
		lgr:      lgr,
		meter:    meter,
	}

	return manager
}

func (m *Manager) Allow(b model.BackendID, now time.Time) (admit, probe bool) {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		m.mu.Lock()
		breaker, exists = m.circuits[b]
		if !exists {
			breaker = NewBreaker(b, m.cfg, m.lgr, m.meter)
			m.circuits[b] = breaker
		}
		m.mu.Unlock()
	}
	return breaker.Allow(now)
}

func (m *Manager) IsFailure(statusCode int) bool {
	if slices.Contains(m.cfg.FailureStatuses, statusCode) {
		return true
	}
	return statusCode >= 500
}

func (m *Manager) RecordSuccess(b model.BackendID) {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		return
	}
	breaker.Record(time.Now(), true)
}

func (m *Manager) RecordFailure(b model.BackendID) {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		return
	}
	breaker.Record(time.Now(), false)
}

func (m *Manager) Release(b model.BackendID) {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		return
	}
	breaker.Release()
}

func (m *Manager) IsProbeReady(b model.BackendID) bool {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		return true
	}
	return breaker.IsProbeReady(time.Now())
}

func (m *Manager) Snapshot(b model.BackendID) model.CircuitBreakerState {
	m.mu.RLock()
	breaker, exists := m.circuits[b]
	m.mu.RUnlock()
	if !exists {
		return model.CircuitBreakerState{
			Phase: model.PhaseClosed,
		}
	}
	return breaker.Snapshot()
}

func (m *Manager) Sync(desired []model.Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()

	old := m.circuits
	next := make(map[model.BackendID]*Breaker, len(desired))
	for _, b := range desired {
		if c, ok := old[b.ID]; ok {
			next[b.ID] = c
			continue
		}
		next[b.ID] = NewBreaker(b.ID, m.cfg, m.lgr, m.meter)
	}
	m.circuits = next
}

type Breaker struct {
	mu            sync.Mutex // guards transitions only
	backendID     model.BackendID
	state         atomic.Int64 // circuit.Phase, hot-path readable
	openedAt      atomic.Int64 // unixnano
	trips         atomic.Int32
	halfSuccesses atomic.Int32
	halfInFlight  atomic.Int32
	window        *window
	lgr           *slog.Logger
	cfg           config.Circuit
	meter         *observability.Metrics
}

func NewBreaker(backend model.BackendID, cfg config.Circuit, lgr *slog.Logger, meter *observability.Metrics) *Breaker {
	span := cfg.Window
	if span == 0 {
		span = 10 * time.Second
	}
	numBuckets := cfg.Buckets
	if numBuckets == 0 {
		numBuckets = 10
	}
	if cfg.OpenBase == 0 {
		cfg.OpenBase = 5 * time.Second
	}
	if cfg.OpenMax == 0 {
		cfg.OpenMax = 30 * time.Second
	}
	if cfg.BackoffFactor == 0 {
		cfg.BackoffFactor = 2
	}
	if cfg.HalfOpenMax == 0 {
		cfg.HalfOpenMax = 3
	}
	if cfg.HalfOpenSuccess == 0 {
		cfg.HalfOpenSuccess = 3
	}
	if cfg.FailureRatio == 0 {
		cfg.FailureRatio = 0.5
	}
	if cfg.MinRequests == 0 {
		cfg.MinRequests = 10
	}
	window := newWindow(span, span/time.Duration(numBuckets), numBuckets)
	breaker := &Breaker{
		backendID: backend,
		window:    window,
		lgr:       lgr,
		cfg:       cfg,
		meter:     meter,
	}
	breaker.state.Store(int64(model.PhaseClosed))

	return breaker
}

func (b *Breaker) Allow(now time.Time) (bool, bool) {
	phase := model.BreakerPhase(b.state.Load())
	switch phase {
	case model.PhaseClosed:
		return true, false
	case model.PhaseOpen:
		b.mu.Lock()
		if now.UnixNano()-b.openedAt.Load() < int64(OpenTimeout(b.cfg, b.trips.Load())) {
			b.mu.Unlock()
			return false, false
		}
		b.doTransition(now, EventClock)
		phase = model.BreakerPhase(b.state.Load())
		b.mu.Unlock()
		if phase == model.PhaseHalfOpen {
			if b.halfInFlight.Add(1) <= int32(b.cfg.HalfOpenMax) {
				return true, true
			}
			b.halfInFlight.Add(-1)
			return false, false
		}
		return false, false
	case model.PhaseHalfOpen:
		if b.halfInFlight.Add(1) <= int32(b.cfg.HalfOpenMax) {
			return true, true
		}
		b.halfInFlight.Add(-1)
		return false, false
	}
	return false, false
}

func (b *Breaker) Snapshot() model.CircuitBreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()

	var canProbe bool
	if model.BreakerPhase(b.state.Load()) != model.PhaseOpen {
		canProbe = true
	} else {
		canProbe = time.Now().UnixNano()-b.openedAt.Load() >= int64(OpenTimeout(b.cfg, b.trips.Load()))
	}
	phase := model.BreakerPhase(b.state.Load())
	halfOpenInFlight := b.halfInFlight.Load()

	return model.CircuitBreakerState{
		Phase:            phase,
		OpenedAt:         time.Unix(0, b.openedAt.Load()),
		Trips:            b.trips.Load(),
		HalfOpenSuccess:  b.halfSuccesses.Load(),
		HalfOpenInFlight: halfOpenInFlight,
		IsOpen:           (phase == model.PhaseOpen && !canProbe) || (phase == model.PhaseHalfOpen && halfOpenInFlight >= b.cfg.HalfOpenMax),
	}
}

func (b *Breaker) Record(now time.Time, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.window.record(now, ok)
	event := EventSuccess
	if !ok {
		event = EventFailed
	}
	current := model.BreakerPhase(b.state.Load())
	b.doTransition(now, event)
	next := model.BreakerPhase(b.state.Load())
	if current == model.PhaseHalfOpen && next != current {
		b.halfInFlight.Store(0)
	} else if current == model.PhaseHalfOpen {
		val := b.halfInFlight.Load()
		if val > 0 {
			b.halfInFlight.Add(-1)
		}
	}
}

func (b *Breaker) IsProbeReady(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if model.BreakerPhase(b.state.Load()) != model.PhaseOpen {
		return true
	}
	return now.UnixNano()-b.openedAt.Load() >= int64(OpenTimeout(b.cfg, b.trips.Load()))
}

func (b *Breaker) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if model.BreakerPhase(b.state.Load()) == model.PhaseHalfOpen {
		val := b.halfInFlight.Load()
		if val > 0 {
			b.halfInFlight.Add(-1)
		}
	}
}

func (b *Breaker) doTransition(now time.Time, ev Event) {
	successes, failures := b.window.totals(now)
	total := successes + failures
	in := Inputs{
		Total:    total,
		Failures: failures,
	}
	if total == 0 {
		in.FailureRatio = 0
	} else {
		in.FailureRatio = float64(failures) / float64(total)
	}

	currentState := model.CircuitBreakerState{
		Phase:            model.BreakerPhase(b.state.Load()),
		OpenedAt:         time.Unix(0, b.openedAt.Load()),
		Trips:            b.trips.Load(),
		HalfOpenSuccess:  b.halfSuccesses.Load(),
		HalfOpenInFlight: b.halfInFlight.Load(),
	}
	nextState := Reduce(currentState, ev, in, b.cfg, now)

	if currentState.Phase != nextState.Phase {
		b.meter.CircuitState.Record(context.Background(), nextState.Phase.Value(), metric.WithAttributes(attribute.String("backend", string(b.backendID))))
		b.meter.CircuitTransitions.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("backend", string(b.backendID)),
				attribute.String("from", currentState.Phase.String()),
				attribute.String("to", nextState.Phase.String()),
			),
		)

		b.state.Store(nextState.Phase.Value())

		switch nextState.Phase {
		case model.PhaseOpen:
			b.openedAt.Store(nextState.OpenedAt.UnixNano())
			b.trips.Store(nextState.Trips)
			b.halfSuccesses.Store(0)
			b.halfInFlight.Store(0)
			b.lgr.Info("circuit opened",
				slog.String("backend", string(b.backendID)),
				slog.Int("trips", int(nextState.Trips)))
		case model.PhaseHalfOpen:
			b.halfSuccesses.Store(0)
		case model.PhaseClosed:
			b.trips.Store(0)
			b.halfSuccesses.Store(0)
			b.halfInFlight.Store(0)
			b.lgr.Info("circuit closed",
				slog.String("backend", string(b.backendID)))
		}
	} else {
		b.trips.Store(nextState.Trips)
		b.halfSuccesses.Store(nextState.HalfOpenSuccess)
	}
}
