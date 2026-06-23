package loadbalancer

import "github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"

// SelectP2C implements Power-of-Twi-Choised over eligible backends. rnd(n)
// must return a uniform int in [0,n); infrastructure supplies it (math/rand/v2).
// Returns the chosen BackendID and true, or false if no backend is eligible.
func SelectP2C(stats []BackendStat, w Weights, rnd func(n int) int) (model.BackendID, bool) {
	// collect eligible indices
	elig := stats[:0:0]
	for _, s := range stats {
		if s.Eligible() {
			elig = append(elig, s)
		}
	}
	switch len(elig) {
	case 0:
		return "", false
	case 1:
		return elig[0].ID, true
	}
	i := rnd(len(elig))
	j := rnd(len(elig) - 1)
	if j >= i {
		j++
	}
	a, b := elig[i], elig[j]
	if a.Score(w) <= b.Score(w) {
		return a.ID, true
	}
	return b.ID, true
}
