package logger

import (
	"context"
)

type Fields map[string]any

//nolint:interfacebloat
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)

	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
	WithField(key string, value any) Logger
	WithFields(fields Fields) Logger
}
