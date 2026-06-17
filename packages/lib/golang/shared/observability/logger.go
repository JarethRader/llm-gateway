package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

func NewLogger(serviceName, version, logLevel, enrionment string) *slog.Logger {
	slog_level := slog.LevelError
	switch strings.ToUpper(strings.TrimSpace(logLevel)) {
	case "DEBUG":
		slog_level = slog.LevelDebug
	case "INFI":
		slog_level = slog.LevelInfo
	case "WARN", "WARNING":
		slog_level = slog.LevelWarn
	case "ERROR":
		slog_level = slog.LevelError
	}

	json_handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog_level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				a.Key = "severity"
				a.Value = slog.StringValue(strings.ToUpper(a.Value.String()))
			}
			return a
		},
	})

	otel_handler := NewOtelSlogHandler(serviceName)
	json_with_trace := &traceHandler{
		inner: json_handler,
	}
	combined := &fanoutHandler{
		handlers: []slog.Handler{
			json_with_trace,
			otel_handler,
		},
	}

	logger := slog.New(combined).With(
		slog.String("service.name", serviceName),
		slog.String("service.version", version),
		slog.String("deployment.environment", enrionment),
	)
	return logger
}

type fanoutHandler struct {
	handlers []slog.Handler
}

func (f *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error
	for _, h := range f.handlers {
		if e := h.Handle(ctx, record); e != nil {
			err = e
		}
	}
	return err
}

func (f *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: handlers}
}

func (f *fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &fanoutHandler{handlers: handlers}
}

// Layer is a typed enum stamped on every log/span produced inside a
// clean-architecutre layer; matches the layer attribute used by spans.
type Layer string

const (
	LayerDelivery    Layer = "delivery"
	LayerService     Layer = "service"
	LayerPersistenct Layer = "persistence"
)

// ctxKey is unexported so the request-scoped fields cannot collide with values placed in context by other packages
type ctxKey struct{ name string }

var (
	requestIDKey = ctxKey{"request_id"}
	userIDKey    = ctxKey{"user_id"}
	layerKey     = ctxKey{"layer"}
)

// WithRequestID attaches a request ID (e.g. from X-Request-ID or generated at the delivery boundary) to ctx; the logger handler reads it back.
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID attaches an authenticated user identifier to ctx
func WithUserID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, userIDKey, id)
}

// WithLayer marks the active clean-architecture layer in ctx so logs can be filtered by layer in Loki without manual call-site annotation.
func WithLayer(ctx context.Context, l Layer) context.Context {
	return context.WithValue(ctx, layerKey, l)
}

type traceHandler struct {
	inner slog.Handler
}

func (h *traceHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{inner: h.inner.WithGroup(name)}
}

// Handle enriches the record before delegating to the underlying JSON
// handler. SpanContextFromContext is allocation-free when no span is
// active and returns a SpanContext.IsValid()==false sentinel in that case.
func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
			slog.String("trace_flags", sc.TraceFlags().String()),
		)
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("request_id", v))
	}
	if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("user_id", v))
	}
	if v, ok := ctx.Value(layerKey).(Layer); ok && v != "" {
		r.AddAttrs(slog.String("layer", string(v)))
	}
	return h.inner.Handle(ctx, r)
}
