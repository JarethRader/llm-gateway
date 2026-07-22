package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Tracer() trace.Tracer { return otel.Tracer("llm-gateway") }
