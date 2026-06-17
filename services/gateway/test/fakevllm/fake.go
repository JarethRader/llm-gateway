package fakevllm

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

// Behavior controls one response. Fields are read per-request, so a test can
// flip a backend from healthy to faulty mid-run to drive the breaker.
type Behavior struct {
	Addr       *string
	TTFT       time.Duration // delay before the first SSE frame
	InterToken time.Duration // delay between subsequent tokens
	Tokens     int           // number of completion tokens to emit
	Status     int           // non-200 => error response (no stream)
	HangFirst  bool          // accept, send headers, then never write (idle test)
	DropAfter  int           // close the connection after N frames (mid-stream fault)
	EmitUsage  bool          // send a terminal usage frame (token accounting)
	Model      string        // echoed in chucks
}

// Fake is a controllable vLLM stand-in. Concurrency-safe: Behavior and the
// /metric gauges can be mutated while requests are in flight.
type Fake struct {
	srv      *httptest.Server
	mu       sync.RWMutex
	behavior Behavior

	running  atomic.Int64  // exported as vllm:num_requests_running
	waiting  atomic.Int64  // exported as vllm:num_requests_waiting
	cachePct atomic.Uint64 // *1000, exported as vllm:gpu_cache_usage_perc
}

func New(b Behavior) *Fake {
	f := &Fake{behavior: b}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", f.handleChat)
	mux.HandleFunc("/v1/models", f.handleModels)
	mux.HandleFunc("/metrics", f.handleMetrics)
	f.srv = httptest.NewUnstartedServer(mux)

	if b.Addr != nil {
		l, err := net.Listen("tcp", "127.0.0.1"+*b.Addr)
		if err != nil {
			log.Fatalf("Failed to listen on port %s: %v", *b.Addr, err)
		}
		f.srv.Listener = l
	}
	f.srv.Start()

	return f
}

func (f *Fake) URL() string { return f.srv.URL }
func (f *Fake) Close()      { f.srv.Close() }

func (f *Fake) SetBehavior(b Behavior) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.behavior = b
}
func (f *Fake) SetCacheUsage(pct float64) {
	f.cachePct.Store(uint64(pct * 1000))
}

func (f *Fake) handleChat(w http.ResponseWriter, r *http.Request) {
	f.running.Add(1)
	defer f.running.Add(-1)

	f.mu.RLock()
	b := f.behavior
	f.mu.RUnlock()

	if b.Status != 0 && b.Status != http.StatusOK {
		http.Error(w, `{"error":{"message":"injected"}}`, b.Status)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no flush", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)

	if b.HangFirst {
		<-r.Context().Done()
		return
	}

	select {
	case <-time.After(b.TTFT):
	case <-r.Context().Done():
		return
	}

	for i := 0; i < b.Tokens; i++ {
		if b.DropAfter > 0 && i >= b.DropAfter {
			if hj, ok := w.(http.Hijacker); ok {
				if conn, _, err := hj.Hijack(); err == nil {
					_ = conn.Close()
				}
			}
			return
		}
		chunk := map[string]any{
			"id":     "chatcmpl-fake",
			"object": "chat.completion.chunk",
			"model":  b.Model,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]string{"content": fmt.Sprintf("t%d", i)},
			}},
		}
		writeSSE(w, flusher, chunk)
		select {
		case <-time.After(b.InterToken):
		case <-r.Context().Done():
			return
		}
	}

	if b.EmitUsage {
		writeSSE(w, flusher, map[string]any{
			"id":      "chatcmpl-fake",
			"object":  "chat.completion.chunk",
			"model":   b.Model,
			"choices": []any{},
			"usage": map[string]int{
				"prompt_tokens":     16,
				"completion_tokens": b.Tokens,
				"total_tokens":      16 + b.Tokens,
			},
		})
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeSSE(w http.ResponseWriter, fl http.Flusher, v any) {
	buf, _ := json.Marshal(v)
	fmt.Fprintf(w, "data: %s\n\n", buf)
	fl.Flush()
}

func (f *Fake) handleModels(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   []map[string]string{{"id": f.behavior.Model, "object": "model"}},
	})
}

func (f *Fake) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "vllm:num_requests_running %d\n", f.running.Load())
	fmt.Fprintf(w, "vllm:num_requests_waiting %d\n", f.waiting.Load())
	fmt.Fprintf(w, "vllm:gpu_cache_usage_perc %.2f\n", float64(f.cachePct.Load()/1000))
}
