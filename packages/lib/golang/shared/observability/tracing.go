package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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

type ProviderConfig struct {
	ServiceName string
	Version     string
	Environment string

	Enabled     bool
	Endpoint    string
	Insecure    bool
	SampleRatio float64
}

func InitTracer(ctx context.Context, cfg ProviderConfig) (*TelemetryProviders, error) {
	if !cfg.Enabled {
		return &TelemetryProviders{
			Tracer:   noop.NewTracerProvider(),
			Meter:    noopmetric.NewMeterProvider(),
			Logger:   nooplog.NewLoggerProvider(),
			Shutdown: func(context.Context) error { return nil },
		}, nil
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}
	tp, err := buildTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("telemetry: tracer provider: %w", err)
	}
	mp, err := buildMeterProvider(ctx, cfg, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("telemetry: meter provider: %w", err)
	}
	lp, err := buildLoggerProvider(ctx, cfg, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("telemetry: logger provider: %w", err)
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	global.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent / tracestate
		propagation.Baggage{},      // optional cross-cutting key/values
	))

	shutdown := func(ctx context.Context) error {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		var errs []error
		if err := tp.Shutdown(c); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
		if err := mp.Shutdown(c); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
		if err := lp.Shutdown(c); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
		}
		return errors.Join(errs...)
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

func buildResource(ctx context.Context, cfg ProviderConfig) (*resource.Resource, error) {
	// resource.WithFromEnv consumes OTEL_RESOURCE_ATTRIBUTES populated by
	// the Kubernetes Downward API (see Phase 4 pod spec).
	// Note: WithProcess() omitted as it requires CGO for username detection.
	opts := []resource.Option{
		resource.WithFromEnv(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
	}
	return resource.New(ctx, opts...)
}

func buildTracerProvider(ctx context.Context, cfg ProviderConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
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

func buildMeterProvider(ctx context.Context, cfg ProviderConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp,
			sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	), nil
}

func buildLoggerProvider(ctx context.Context, cfg ProviderConfig, res *resource.Resource) (*sdklog.LoggerProvider, error) {
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
