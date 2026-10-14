package middleware_test

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

type logEntry struct {
	level  string
	msg    string
	fields logger.Fields
}

type fakeLogger struct {
	fields  logger.Fields
	entries *[]logEntry
}

func newFakeLogger() *fakeLogger {
	return &fakeLogger{
		fields:  logger.Fields{},
		entries: &[]logEntry{},
	}
}

func (f *fakeLogger) Debugf(format string, args ...any) { f.log("debug", fmt.Sprintf(format, args...)) }
func (f *fakeLogger) Infof(format string, args ...any)  { f.log("info", fmt.Sprintf(format, args...)) }
func (f *fakeLogger) Warnf(format string, args ...any)  { f.log("warn", fmt.Sprintf(format, args...)) }
func (f *fakeLogger) Errorf(format string, args ...any) { f.log("error", fmt.Sprintf(format, args...)) }
func (f *fakeLogger) Debug(args ...any)                 { f.log("debug", args...) }
func (f *fakeLogger) Info(args ...any)                  { f.log("info", args...) }
func (f *fakeLogger) Warn(args ...any)                  { f.log("warn", args...) }
func (f *fakeLogger) Error(args ...any)                 { f.log("error", args...) }

func (f *fakeLogger) WithContext(_ context.Context) logger.Logger { return f }

func (f *fakeLogger) WithError(err error) logger.Logger {
	return f.WithField("error", err)
}

func (f *fakeLogger) WithField(key string, value any) logger.Logger {
	return f.WithFields(logger.Fields{key: value})
}

func (f *fakeLogger) WithFields(fields logger.Fields) logger.Logger {
	merged := maps.Clone(f.fields)
	maps.Copy(merged, fields)

	return &fakeLogger{
		fields:  merged,
		entries: f.entries,
	}
}

func (f *fakeLogger) log(level string, args ...any) {
	*f.entries = append(*f.entries, logEntry{
		level:  level,
		msg:    fmt.Sprint(args...),
		fields: maps.Clone(f.fields),
	})
}

func TestLogging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantLevel string
		wantCode  int
		wantBytes int
	}{
		{
			name: "ok implicit status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			wantLevel: "info",
			wantCode:  http.StatusOK,
			wantBytes: 5,
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantLevel: "warn",
			wantCode:  http.StatusNotFound,
		},
		{
			name: "internal error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantLevel: "error",
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			log := newFakeLogger()
			rec := httptest.NewRecorder()

			middleware.Logging(log)(tc.handler).ServeHTTP(
				rec,
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil),
			)

			if rec.Code != tc.wantCode {
				t.Fatalf("response code = %d, want %d", rec.Code, tc.wantCode)
			}

			if len(*log.entries) != 1 {
				t.Fatalf("got %d log entries, want 1", len(*log.entries))
			}

			entry := (*log.entries)[0]

			if entry.level != tc.wantLevel {
				t.Errorf("level = %q, want %q", entry.level, tc.wantLevel)
			}

			if entry.fields["status"] != tc.wantCode {
				t.Errorf("status field = %v, want %d", entry.fields["status"], tc.wantCode)
			}

			if entry.fields["bytes"] != tc.wantBytes {
				t.Errorf("bytes field = %v, want %d", entry.fields["bytes"], tc.wantBytes)
			}

			if entry.fields["method"] != http.MethodGet || entry.fields["path"] != "/test" {
				t.Errorf("unexpected request fields: %v", entry.fields)
			}
		})
	}
}

func TestLoggingContextLogger(t *testing.T) {
	t.Parallel()

	log := newFakeLogger()

	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info("inside handler")
	})

	middleware.Logging(log)(handler).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload", nil),
	)

	if len(*log.entries) != 2 {
		t.Fatalf("got %d log entries, want 2", len(*log.entries))
	}

	entry := (*log.entries)[0]

	if entry.msg != "inside handler" {
		t.Fatalf("first entry msg = %q, want %q", entry.msg, "inside handler")
	}

	if entry.fields["method"] != http.MethodPost || entry.fields["path"] != "/upload" {
		t.Errorf("handler logger missing request fields: %v", entry.fields)
	}
}
