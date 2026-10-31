package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

func Tracing(tracer tracing.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rspWriter http.ResponseWriter, req *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			ctx, span := tracer.StartSpan(
				ctx,
				req.Method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLPath(req.URL.Path),
					semconv.URLScheme(urlScheme(req)),
					semconv.ClientAddress(req.RemoteAddr),
					semconv.UserAgentOriginal(req.UserAgent()),
				),
			)
			defer span.End()

			rec := &responseRecorder{
				ResponseWriter: rspWriter,
				status:         http.StatusOK,
			}

			// ServeMux sets Pattern on the request it receives, so keep a reference to read it back.
			spanReq := req.WithContext(ctx)

			next.ServeHTTP(rec, spanReq)

			if spanReq.Pattern != "" {
				span.SetName(req.Method + " " + spanReq.Pattern)
				span.SetAttributes(semconv.HTTPRoute(spanReq.Pattern))
			}

			span.SetAttributes(semconv.HTTPResponseStatusCode(rec.status))

			if rec.status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(rec.status))
			}
		})
	}
}

func urlScheme(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}

	return "http"
}
