package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

const (
	shutdownTimeout = time.Second * 60
)

func Start(
	ctx context.Context,
	cfg config.HTTPServer,
	fileSvc handler.FileService,
) error {
	if err := startServer(
		ctx,
		cfg,
		middleware.Tracing(tracing.StdTracer())(
			middleware.Logging(logger.FromContext(ctx))(
				createMux(handler.New(fileSvc)),
			),
		),
	); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

func startServer(ctx context.Context, cfg config.HTTPServer, handler http.Handler) error {
	var (
		srv = &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		}

		srvErrCh = make(chan error, 1)
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
	}()

	select {
	case err := <-srvErrCh:
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	//nolint:contextcheck
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}

func createMux(hdl *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	registerHandler(mux, "/test", hdl.Test)
	registerHandler(mux, "/error_test", hdl.ErrorTest)
	registerHandler(mux, "GET /files/{"+handler.PathValueDirPath+"...}", hdl.FileList)
	registerHandler(mux, "GET /download/{"+handler.PathValueFilePath+"...}", hdl.FileDownload)

	return mux
}

func registerHandler(mux *http.ServeMux, path string, h HandlerFunc) {
	mux.HandleFunc(path, h.ToHTTPHandlerFunc())
}

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
