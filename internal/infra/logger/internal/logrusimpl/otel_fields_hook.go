package logrusimpl

import (
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type OtelFieldsHook struct {
}

var _ logrus.Hook = (*OtelFieldsHook)(nil)

func NewOtelFieldsHook() *OtelFieldsHook {
	return &OtelFieldsHook{}
}

func (hook *OtelFieldsHook) Fire(entry *logrus.Entry) error {
	ctx := entry.Context
	if ctx == nil {
		return nil
	}

	spanCtx := trace.SpanContextFromContext(ctx)

	if spanCtx.HasTraceID() {
		entry.Data["otel_trace_id"] = spanCtx.TraceID().String()
	}

	if spanCtx.HasSpanID() {
		entry.Data["otel_span_id"] = spanCtx.SpanID().String()
	}

	return nil
}

// Levels returns logrus levels on which this hook is fired.
func (hook *OtelFieldsHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}
