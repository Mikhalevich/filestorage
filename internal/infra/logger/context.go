package logger

import (
	"context"
)

type contextKey string

const (
	contextLogger = contextKey("contextLogger")
)

func FromContext(ctx context.Context) Logger {
	l, ok := ctx.Value(contextLogger).(Logger)
	if !ok {
		return StdLogger().WithContext(ctx)
	}

	return l.WithContext(ctx)
}

func WithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextLogger, l)
}
