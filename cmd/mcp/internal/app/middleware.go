package app

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

const (
	methodNameKey = attribute.Key("mcp.method.name")
	toolNameKey   = attribute.Key("gen_ai.tool.name")
)

// Middleware wraps every incoming mcp request into a span and writes an access log entry for it.
func Middleware(tracer tracing.Tracer) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()

			var (
				spanName = method
				attrs    = []attribute.KeyValue{methodNameKey.String(method)}
				fields   = logger.Fields{"method": method}
			)

			if params, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
				spanName += " " + params.Name
				attrs = append(attrs, toolNameKey.String(params.Name))
				fields["tool"] = params.Name
			}

			ctx, span := tracer.StartSpan(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(attrs...),
			)
			defer span.End()

			reqLog := logger.FromContext(ctx).WithFields(fields)

			res, err := next(logger.WithLogger(ctx, reqLog), method, req)

			accessLog := reqLog.WithField("duration", time.Since(start).String())

			switch {
			case err != nil:
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				accessLog.WithError(err).Error("mcp request")
			case isToolError(res):
				span.SetStatus(codes.Error, "tool error")
				accessLog.Warn("mcp request")
			default:
				accessLog.Info("mcp request")
			}

			return res, err
		}
	}
}

func isToolError(res mcp.Result) bool {
	toolRes, ok := res.(*mcp.CallToolResult)

	return ok && toolRes.IsError
}
