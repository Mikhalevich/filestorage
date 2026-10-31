package middleware

import (
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func Logging(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rspWriter http.ResponseWriter, req *http.Request) {
			start := time.Now()

			reqLog := log.WithFields(logger.Fields{
				"method":      req.Method,
				"path":        req.URL.Path,
				"remote_addr": req.RemoteAddr,
			})

			rec := &responseRecorder{
				ResponseWriter: rspWriter,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, req.WithContext(logger.WithLogger(req.Context(), reqLog)))

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
