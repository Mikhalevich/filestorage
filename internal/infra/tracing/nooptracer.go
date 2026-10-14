package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type noopTracer struct {
}

func NewNoopTracer() *noopTracer {
	return &noopTracer{}
}

func (t *noopTracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return ctx, noop.Span{}
}
