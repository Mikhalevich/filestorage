package tracing

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	//nolint:gochecknoglobals
	std Tracer = NewNoopTracer()
)

type Tracer interface {
	StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

type OtelTracer struct {
	Name     string
	provider *sdktrace.TracerProvider
}

type ShutdownFn func(ctx context.Context) error

func NewOtelTracer(
	endpoint string,
	name string,
	version string,
) (*OtelTracer, error) {
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	attrs := []attribute.KeyValue{semconv.ServiceName(name)}
	if version != "" {
		attrs = append(attrs, semconv.ServiceVersion(version))
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("merge resource: %w", err)
	}

	tracerprovider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerprovider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &OtelTracer{
		Name:     name,
		provider: tracerprovider,
	}, nil
}

// Shutdown flushes pending spans and stops the exporter.
func (t *OtelTracer) Shutdown(ctx context.Context) error {
	if err := t.provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("tracer provider shutdown: %w", err)
	}

	return nil
}

func (t *OtelTracer) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	tr := otel.Tracer(t.Name)
	//nolint:spancheck
	return tr.Start(ctx, spanName, opts...)
}

func StartSpan(ctx context.Context) (context.Context, trace.Span) {
	return std.StartSpan(ctx, callingFuncName())
}

func StartSpanName(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return std.StartSpan(ctx, spanName, opts...)
}

func StdTracer() Tracer {
	return std
}

func callingFuncName() string {
	const skipCallers = 2

	pc, _, _, ok := runtime.Caller(skipCallers)
	if !ok {
		return ""
	}

	return runtime.FuncForPC(pc).Name()
}

func SetupTracer(
	endpoint string,
	name string,
	version string,
) (ShutdownFn, error) {
	otelTracer, err := NewOtelTracer(endpoint, name, version)
	if err != nil {
		return nil, fmt.Errorf("creating otel tracer: %w", err)
	}

	std = otelTracer

	return otelTracer.Shutdown, nil
}
