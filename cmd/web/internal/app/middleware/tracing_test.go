package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
)

func TestMain(m *testing.M) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	os.Exit(m.Run())
}

type testTracer struct {
	tracer trace.Tracer
}

func (t *testTracer) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	//nolint:spancheck
	return t.tracer.Start(ctx, spanName, opts...)
}

func newTestTracer() (*testTracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	return &testTracer{tracer: provider.Tracer("test")}, recorder
}

func serveTraced(t *testing.T, handler http.HandlerFunc, req *http.Request) sdktrace.ReadOnlySpan {
	t.Helper()

	tracer, recorder := newTestTracer()

	mux := http.NewServeMux()
	mux.HandleFunc("/test", handler)

	middleware.Tracing(tracer)(mux).ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d ended spans, want 1", len(spans))
	}

	return spans[0]
}

func spanAttr(span sdktrace.ReadOnlySpan, key attribute.Key) (attribute.Value, bool) {
	for _, attr := range span.Attributes() {
		if attr.Key == key {
			return attr.Value, true
		}
	}

	return attribute.Value{}, false
}

func TestTracingSpan(t *testing.T) {
	t.Parallel()

	span := serveTraced(
		t,
		func(w http.ResponseWriter, r *http.Request) {
			if !trace.SpanFromContext(r.Context()).SpanContext().IsValid() {
				t.Error("handler context has no span")
			}

			_, _ = w.Write([]byte("ok"))
		},
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil),
	)

	if span.Name() != "GET /test" {
		t.Errorf("span name = %q, want %q", span.Name(), "GET /test")
	}

	if span.SpanKind() != trace.SpanKindServer {
		t.Errorf("span kind = %v, want server", span.SpanKind())
	}

	if span.Status().Code != codes.Unset {
		t.Errorf("span status = %v, want unset", span.Status().Code)
	}

	if v, ok := spanAttr(span, semconv.HTTPResponseStatusCodeKey); !ok || v.AsInt64() != http.StatusOK {
		t.Errorf("status code attribute = %v, want %d", v.String(), http.StatusOK)
	}

	if v, ok := spanAttr(span, semconv.HTTPRouteKey); !ok || v.AsString() != "/test" {
		t.Errorf("route attribute = %q, want %q", v.String(), "/test")
	}
}

func TestTracingServerError(t *testing.T) {
	t.Parallel()

	span := serveTraced(
		t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil),
	)

	if span.Status().Code != codes.Error {
		t.Errorf("span status = %v, want error", span.Status().Code)
	}
}

func TestTracingPropagation(t *testing.T) {
	t.Parallel()

	const (
		traceID      = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpanID = "00f067aa0ba902b7"
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	req.Header.Set("Traceparent", "00-"+traceID+"-"+parentSpanID+"-01")

	span := serveTraced(t, func(http.ResponseWriter, *http.Request) {}, req)

	if got := span.SpanContext().TraceID().String(); got != traceID {
		t.Errorf("trace id = %s, want %s", got, traceID)
	}

	if got := span.Parent().SpanID().String(); got != parentSpanID {
		t.Errorf("parent span id = %s, want %s", got, parentSpanID)
	}
}
