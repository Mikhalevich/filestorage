package app

import (
	"context"
	"time"

	"github.com/Mikhalevich/tgbot"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

const (
	patternKey     = attribute.Key("telegram.handler.pattern")
	messageTypeKey = attribute.Key("telegram.message.type")
)

// tracer wraps every telegram update handling into a span and writes an access log entry for it.
// tgbot creates a new tracer per update, so it keeps the per update state.
type tracer struct {
	tracer tracing.Tracer

	start  time.Time
	span   trace.Span
	reqLog logger.Logger
	err    error
}

func newTracer(t tracing.Tracer) *tracer {
	return &tracer{
		tracer: t,
	}
}

func (t *tracer) Before(ctx context.Context, pattern string, msg tgbot.Message) context.Context {
	t.start = time.Now()

	msgType := messageTypeName(msg.Type)

	ctx, t.span = t.tracer.StartSpan(
		ctx,
		"telegram "+pattern,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			patternKey.String(pattern),
			messageTypeKey.String(msgType),
		),
	)

	t.reqLog = logger.FromContext(ctx).WithFields(logger.Fields{
		"pattern":      pattern,
		"chat_id":      msg.ChatID,
		"message_type": msgType,
		"username":     msg.User.Username,
	})

	return logger.WithLogger(ctx, t.reqLog)
}

func (t *tracer) OnSuccess(context.Context) {
}

func (t *tracer) OnError(_ context.Context, err error) {
	t.err = err

	t.span.RecordError(err)
	t.span.SetStatus(codes.Error, err.Error())
}

func (t *tracer) After(context.Context) {
	defer t.span.End()

	accessLog := t.reqLog.WithField("duration", time.Since(t.start).String())

	if t.err != nil {
		accessLog.WithError(t.err).Error("telegram update")

		return
	}

	accessLog.Info("telegram update")
}

func messageTypeName(msgType tgbot.Type) string {
	switch msgType {
	case tgbot.TypeText:
		return "text"
	case tgbot.TypeCallbackQuery:
		return "callback_query"
	case tgbot.TypePayment:
		return "payment"
	case tgbot.TypeDocument:
		return "document"
	case tgbot.TypePhoto:
		return "photo"
	case tgbot.TypeInvalid:
	}

	return "invalid"
}
