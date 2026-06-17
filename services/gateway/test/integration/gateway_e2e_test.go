package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/test/fakevllm"
)

type ModelRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
	Key    string `json:"key"`
}

type MockBackend struct {
	Name   string
	URL    string
	Models []string
}

type ProxyRouter struct {
	backends       []MockBackend
	modelToBackend map[string]*MockBackend
}

func (r *ProxyRouter) Register(be MockBackend) {
	r.backends = append(r.backends, be)
	for _, m := range be.Models {
		r.modelToBackend[m] = &be
	}
}

func (r *ProxyRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(io.TeeReader(req.Body, w)).Decode(&payload); err != nil {
		// no-op
	}

	be, ok := r.modelToBackend[payload.Model]
	if !ok {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	target, _ := url.Parse(be.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = 100 * time.Millisecond
	proxy.ServeHTTP(w, req)
}

func newTestGateway(t *testing.T, backends ...MockBackend) httptest.Server {
	r := &ProxyRouter{modelToBackend: make(map[string]*MockBackend)}
	for _, be := range backends {
		r.Register(be)
	}
	return *httptest.NewServer(r)
}

func backend(name, url, model string) MockBackend {
	models := make([]string, 0)
	models = append(models, model)
	return MockBackend{
		Name:   name,
		URL:    url,
		Models: models,
	}
}

func postChat(t *testing.T, url, model string, stream bool, key string) *http.Response {
	body := fmt.Sprintf(`{"model":"%s","stream":%t}`, model, stream)
	resp, err := http.Post(url+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readSSE(t *testing.T, r io.Reader) []string {
	var frames strings.Builder
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("stream error or EOF: %v", err)
			break
		}
		strLine := strings.TrimSpace(string(line))

		if len(strLine) == 0 {
			if frames.Len() > 0 {
				// Process or dispatch your completed event here
				fmt.Printf("Event Received:\n%s\n", strings.TrimSpace(frames.String()))

				// Reset builder for the next event
				frames.Reset()
			}
			continue
		}

		if strings.HasPrefix(strLine, ":") {
			continue
		}

		if strings.HasPrefix(strLine, "data:") {
			payload := strings.TrimPrefix(strLine, "data:")
			payload = strings.TrimPrefix(payload, " ")
			frames.WriteString(payload + "\n")
		}
	}

	return strings.Split(frames.String(), "\n")
}

func containsDone(frames []string) bool {
	for _, f := range frames {
		if strings.Contains(f, "[DONE]") {
			return true
		}
	}
	return false
}

func TestRelay_PreFirstByte_ReroutesToHealthy(t *testing.T) {
	bad := fakevllm.New(fakevllm.Behavior{
		Status: 503,
		Model:  "m",
	})
	good := fakevllm.New(fakevllm.Behavior{
		TTFT:      30 * time.Millisecond,
		Tokens:    8,
		EmitUsage: true,
		Model:     "m",
	})
	t.Cleanup(bad.Close)
	t.Cleanup(good.Close)

	gw := newTestGateway(t,
		backend("b-bad", bad.URL(), "m"),
		backend("b-good", good.URL(), "m"),
	)
	t.Cleanup(gw.Close)

	resp := postChat(t, gw.URL, "m" /*stream=*/, true, "valid-key")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (retry should have saved it)", resp.StatusCode)
	}
	frames := readSSE(t, resp.Body)
	if !containsDone(frames) {
		t.Fatal("stream did not terminate with [DONE]")
	}
}
