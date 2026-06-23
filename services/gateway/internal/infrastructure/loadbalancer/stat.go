package loadbalancer

import "github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"

// BackendStat is a point-in-time snapshot of one backend's load/health, built
// by infrastructure from live atomics and passed to the pure selector.
type BackendStat struct {
	ID          model.BackendID
	Serves      bool    // serves the requested model
	Healthy     bool    // passive/active health
	BreakerOpen bool    // breaker phase == Open
	TtftEwmaMs  float64 // EWMA of time-to-first-token in ms (primary signal)
	InFlight    int     // current in-flght requests on this backend
	MaxInFlight int     // soft capacity for normalization (>0)
	CacheUsage  float64 // vLLM gpu_cache_usage_per in [0,1]; 0 if unknown
	Weight      float64 // static capacity hint (>0)
}

// Weights tune the score. Lower score is better.
type Weights struct {
	Latency   float64 // weight on normalized TTFT
	InFlight  float64 // weight on normalized in-flight
	Cache     float64 // weight on KV-cache utilization
	RefTTFTMs float64 // reference TTFT for normalization (e.g. 200ms)
}

// Eligible reports whether a backend may receive the request at all.
func (s *BackendStat) Eligible() bool {
	return s.Serves && s.Healthy && !s.BreakerOpen
}

// Score is the load cost of routing to this backend (lower = preferred).
func (s BackendStat) Score(w Weights) float64 {
	lat := s.TtftEwmaMs / w.RefTTFTMs
	inf := 0.0
	if s.MaxInFlight > 0 {
		inf = float64(s.InFlight) / float64(s.MaxInFlight)
	}
	return (w.Latency*lat + w.InFlight*inf + w.Cache*s.CacheUsage) / s.Weight
}
