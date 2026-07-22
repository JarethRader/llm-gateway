package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	// Request (transport, end-to-end)
	RequestsTotal   metric.Int64Counter     // gateway_request_total {model,outcome,stream}
	RequestDuration metric.Float64Histogram // gateway_request_duration [gateway_request_duration_seconds] {model,outcome}
	TokensTotal     metric.Int64Counter     // gateway_tokens_total {model,backend,kind}
	RequestAttempts metric.Int64Histogram   // gateway_request_attempts {model}

	// Routing
	RouteSelected   metric.Int64Counter       // gateway_route_selected_total {backend,model}
	RouteNoBackend  metric.Int64Counter       // gateway_route_no_backend_total {model}
	BackendInflight metric.Int64UpDownCounter // gateway_backend_inflight {backend}

	// Rate limiting
	RatelimitDecisions metric.Int64Counter // gateway_ratelimit_decisions_total {scope,decision}

	// Admission
	AdmissionDecisions metric.Int64Counter       // gateway_admission_decisions_total {decision}
	QueueDepth         metric.Int64Gauge         // gateway_queue_depth
	Inflight           metric.Int64UpDownCounter // gateway_inflight
	AdmissionWait      metric.Float64Histogram   // gateway_admission_wait [gateway_admission_wait_seconds]

	// Streaming relay
	TTFT             metric.Float64Histogram // gateway_ttft [gateway_ttft_seconds] {model,backend}
	StreamBytes      metric.Int64Counter     // gateway_stream_bytes_total {backend,dir}
	StreamErrors     metric.Int64Counter     // gateway_stream_errors_total {backend,phase}
	ClientDisconnect metric.Int64Counter     // gateway_client_disconnects_total {phase}

	// Circuit breaker
	CircuitState        metric.Int64Gauge   // gateway_circuit_state {backend}
	CircuitTransitions  metric.Int64Counter // gateway_circuit_transitions_total {backend,from,to}
	CircuitShortCircuit metric.Int64Counter // gateway_circuit_short_circuits_total {backend}

	// Retry
	RetriesTotal metric.Int64Counter // gateway_retries_total {reason}
}

func NewMetrics() (*Metrics, error) {
	m := otel.Meter("llm-gateway")
	var err error
	mm := &Metrics{}

	if mm.RequestsTotal, err = m.Int64Counter("gateway.requests", metric.WithDescription("requests by terminal outcome")); err != nil {
		return nil, err
	}
	if mm.RequestDuration, err = m.Float64Histogram("gateway.request.duration", metric.WithUnit("s"), metric.WithDescription("accept->last-byte wall time")); err != nil {
		return nil, err
	}
	if mm.TokensTotal, err = m.Int64Counter("gateway.tokens", metric.WithDescription("tokens by kind {prompt,completions}")); err != nil {
		return nil, err
	}
	if mm.RequestAttempts, err = m.Int64Histogram("gateway.request.attempts", metric.WithDescription("requested attempts {model}")); err != nil {
		return nil, err
	}

	if mm.RouteSelected, err = m.Int64Counter("gateway.route.selected", metric.WithDescription("gateway selected backend {backend,model}")); err != nil {
		return nil, err
	}
	if mm.RouteNoBackend, err = m.Int64Counter("gateway.route.no.backend", metric.WithDescription("no supporting backend {model}")); err != nil {
		return nil, err
	}
	if mm.BackendInflight, err = m.Int64UpDownCounter("gateway.backend.inflight", metric.WithDescription("inflight requests {backend}")); err != nil {
		return nil, err
	}

	if mm.RatelimitDecisions, err = m.Int64Counter("gateway.ratelimit.decisions", metric.WithDescription("rate limit decisions made {scope,decision}")); err != nil {
		return nil, err
	}

	if mm.AdmissionDecisions, err = m.Int64Counter("gateway.admission.decisions", metric.WithDescription("admission decisions made {decision}")); err != nil {
		return nil, err
	}
	if mm.QueueDepth, err = m.Int64Gauge("gateway.queue.depth", metric.WithDescription("requests in queue")); err != nil {
		return nil, err
	}
	if mm.Inflight, err = m.Int64UpDownCounter("gateway.inflight", metric.WithDescription("requests in flight")); err != nil {
		return nil, err
	}
	if mm.AdmissionWait, err = m.Float64Histogram("gateway.admission.wait", metric.WithUnit("s")); err != nil {
		return nil, err
	}

	if mm.TTFT, err = m.Float64Histogram("gateway.ttft", metric.WithUnit("s"), metric.WithDescription("time to first token")); err != nil {
		return nil, err
	}
	if mm.StreamBytes, err = m.Int64Counter("gateway.stream.bytes", metric.WithDescription("bytes streamed through gateway")); err != nil {
		return nil, err
	}
	if mm.StreamErrors, err = m.Int64Counter("gateway.stream.errors", metric.WithDescription("stream errors thrown")); err != nil {
		return nil, err
	}
	if mm.ClientDisconnect, err = m.Int64Counter("gateway.client.disconnects", metric.WithDescription("clients disconnected")); err != nil {
		return nil, err
	}

	if mm.CircuitState, err = m.Int64Gauge("gateway.circuit.state", metric.WithDescription("circuit breaker current state {backend}")); err != nil {
		return nil, err
	}
	if mm.CircuitTransitions, err = m.Int64Counter("gateway.circuit.transitions", metric.WithDescription("circuit breaker state transitions")); err != nil {
		return nil, err
	}
	if mm.CircuitShortCircuit, err = m.Int64Counter("gateway.circuit.short.circuits", metric.WithDescription("circuit breaker short circuits")); err != nil {
		return nil, err
	}

	if mm.RetriesTotal, err = m.Int64Counter("gateway.retries", metric.WithDescription("reasons for request retries")); err != nil {
		return nil, err
	}

	return mm, nil
}
