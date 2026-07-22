package observability

import (
	"context"
	"errors"
	"fmt"
	"packages/lib/golang/shared/config"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	nooplog "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TelemetryProviders struct {
	Tracer   trace.TracerProvider
	Meter    metric.MeterProvider
	Logger   otellog.LoggerProvider
	Shutdown func(context.Context) error
}

func InitTracer(ctx context.Context, serviceName, version, environment string, cfg config.Telemetry, reg prometheus.Registerer) (*TelemetryProviders, error) {
	if !cfg.Enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		global.SetLoggerProvider(nooplog.NewLoggerProvider())
		return &TelemetryProviders{
			Tracer:   noop.NewTracerProvider(),
			Meter:    noopmetric.NewMeterProvider(),
			Logger:   nooplog.NewLoggerProvider(),
			Shutdown: func(c context.Context) error { return nil },
		}, nil
	}

	res, err := buildResource(ctx, serviceName, version, environment)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	mp, err := buildMeterProvider(reg, res)
	if err != nil {
		return nil, fmt.Errorf("telemetry: meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)

	tp, err := buildTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("telemetry: tracer provider: %w", err)
	}
	lp, err := buildLoggerProvider(ctx, cfg, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("telemetry: logger provider: %w", err)
	}

	otel.SetTracerProvider(tp)
	global.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent / tracestate
		propagation.Baggage{},      // optional cross-cutting key/values
	))

	shutdown := func(ctx context.Context) error {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return errors.Join(
			tp.Shutdown(c),
			mp.Shutdown(c),
			lp.Shutdown(c),
		)
	}
	return &TelemetryProviders{
		Tracer:   tp,
		Meter:    mp,
		Logger:   lp,
		Shutdown: shutdown,
	}, nil
}

// NewOtelSlogHandler returns an otelslog Handler that can be combined with
// the stdout JSON handler so the same logging call writes both a JSON line
// and an OTLP log record.
func NewOtelSlogHandler(name string) *otelslog.Handler {
	return otelslog.NewHandler(name, otelslog.WithLoggerProvider(global.GetLoggerProvider()))
}

func buildResource(ctx context.Context, serviceName, version, environment string) (*resource.Resource, error) {
	// resource.WithFromEnv consumes OTEL_RESOURCE_ATTRIBUTES populated by
	// the Kubernetes Downward API (see Phase 4 pod spec).
	// Note: WithProcess() omitted as it requires CGO for username detection.
	opts := []resource.Option{
		resource.WithFromEnv(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironmentName(environment),
		),
	}
	return resource.New(ctx, opts...)
}

func buildTracerProvider(ctx context.Context, cfg config.Telemetry, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithMaxQueueSize(2048),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithResource(res),
	), nil
}

func buildMeterProvider(reg prometheus.Registerer, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	reader, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
		sdkmetric.WithView(latencyHistogramViews()...),
	), nil
}

func buildLoggerProvider(ctx context.Context, cfg config.Telemetry, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	exp, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
		sdklog.WithResource(res),
	), nil
}

func latencyHistogramViews() []sdkmetric.View {
	ttft := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "gateway.ttft"},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: []float64{0.02, 0.05, 0.1, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 3, 5, 10},
		}},
	)
	dur := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "gateway.request.duration"},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}},
	)
	wait := sdkmetric.NewView(
		sdkmetric.Instrument{Name: "gateway.admission.wait"},
		sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
			Boundaries: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}},
	)
	return []sdkmetric.View{ttft, dur, wait}
}
