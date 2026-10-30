package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func Logging(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rspWriter http.ResponseWriter, req *http.Request) {
			var (
				start  = time.Now()
				ctx    = req.Context()
				res, _ = resource.FromContext(ctx)
			)

			reqLog := log.WithFields(logger.Fields{
				"method":      req.Method,
				"path":        req.URL.Path,
				"remote_addr": req.RemoteAddr,
				"user":        res.User,
			})

			rec := &statusRecorder{
				ResponseWriter: rspWriter,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, req.WithContext(logger.WithLogger(ctx, reqLog)))

			accessLog := reqLog.WithContext(req.Context()).WithFields(logger.Fields{
				"status":   rec.status,
				"bytes":    rec.bytes,
				"duration": time.Since(start).String(),
			})

			switch {
			case rec.status >= http.StatusInternalServerError:
				accessLog.Error("http request")
			case rec.status >= http.StatusBadRequest:
				accessLog.Warn("http request")
			default:
				accessLog.Info("http request")
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter

	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}

	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wroteHeader = true

	written, err := s.ResponseWriter.Write(b)
	s.bytes += written

	if err != nil {
		return written, fmt.Errorf("write response: %w", err)
	}

	return written, nil
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
