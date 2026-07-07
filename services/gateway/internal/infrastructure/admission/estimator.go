package admission

import (
	"math"
	"sync"
	"time"
)

// Estimator implements exponentially weighted moving average estimate for
// request completions/sec.
type Estimator struct {
	mu          sync.Mutex
	alpha       float64 // inverse (1/t) decay-time constant
	delta       float64 // EWMA completions/sec
	completions int64
	lastTime    time.Time
}

func NewEstimator(alpha float64) *Estimator {
	return &Estimator{
		alpha:       alpha,
		delta:       0,
		completions: 0,
		lastTime:    time.Now(),
	}
}

// MarkCompletion increments the completions in the current window and
// updates the moving average.
func (e *Estimator) MarkCompletion() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.completions++

	tau := time.Since(e.lastTime).Seconds()
	if tau < 0.01 {
		return
	}
	weight := math.Exp(-e.alpha * min(tau, 1/e.alpha))
	e.delta = (e.delta * weight) + (float64(e.completions)/min(tau, 1/e.alpha))*(1-weight)
	e.completions = 0
	e.lastTime = e.lastTime.Add(time.Duration(tau) * time.Second)
}

// delta returns the EWMA completions/sec.
func (e *Estimator) Rate() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	elapsed := time.Since(e.lastTime).Seconds()

	tau := min(elapsed, 1/e.alpha)
	if tau < 0.01 {
		return e.delta
	}

	if elapsed > 10/e.alpha {
		weight := math.Exp(-e.alpha * elapsed)
		return e.delta * weight
	}
	return e.delta * math.Exp(-e.alpha*tau)
}
