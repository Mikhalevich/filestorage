package router

import (
	"errors"
	"net/http"
	"slices"

	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func registerHandler(
	mux *http.ServeMux,
	pattern string,
	h HandlerFunc,
	middlewares []Middleware,
) {
	var hdl http.Handler = h.ToHTTPHandlerFunc()

	for _, mw := range slices.Backward(middlewares) {
		hdl = mw(hdl)
	}

	mux.Handle(pattern, hdl)
}

type Middleware func(http.Handler) http.Handler
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h HandlerFunc) ToHTTPHandlerFunc() http.HandlerFunc {
	return func(rspWriter http.ResponseWriter, r *http.Request) {
		if err := h(rspWriter, r); err != nil {
			logger.FromContext(r.Context()).WithError(err).Error("handler error")
			trace.SpanFromContext(r.Context()).RecordError(err)

			if httpErr, ok := errors.AsType[*httperror.HTTPError](err); ok {
				http.Error(rspWriter, httpErr.Msg, errorStatus(httpErr.Status))

				return
			}

			http.Error(rspWriter, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func errorStatus(status int) int {
	if status < http.StatusBadRequest || status > 599 {
		return http.StatusInternalServerError
	}

	return status
}
