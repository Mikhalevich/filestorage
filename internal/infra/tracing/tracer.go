package tracing

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	//nolint:gochecknoglobals
	std Tracer = NewNoopTracer()
)

type Tracer interface {
	StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span)
}

type OtelTracer struct {
	Name string
}

func NewOtelTracer(
	endpoint string,
	name string,
	version string,
) (*OtelTracer, error) {
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithHeaders(map[string]string{
				"content-type": "application/json",
			}),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		),
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
		Name: name,
	}, nil
}

func (t *OtelTracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tr := otel.Tracer(t.Name)
	//nolint:spancheck
	return tr.Start(ctx, spanName)
}

func StartSpan(ctx context.Context) (context.Context, trace.Span) {
	return std.StartSpan(ctx, callingFuncName())
}

func StartSpanName(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return std.StartSpan(ctx, spanName)
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
) error {
	tr, err := NewOtelTracer(endpoint, name, version)
	if err != nil {
		return fmt.Errorf("creating otel tracer: %w", err)
	}

	std = tr

	return nil
}
