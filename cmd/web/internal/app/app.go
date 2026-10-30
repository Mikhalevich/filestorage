package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

func Start(
	ctx context.Context,
	cfg config.HTTPServer,
	fileSvc handler.FileService,
	tokenVerifier middleware.TokenVerifier,
) error {
	mux := createMux(handler.New(fileSvc))

	// nil verifier means authorization is disabled.
	if tokenVerifier != nil {
		mux = middleware.Auth(tokenVerifier)(mux)
	}

	if err := startServer(
		ctx,
		cfg,
		middleware.Tracing(tracing.StdTracer())(
			middleware.Logging(logger.FromContext(ctx))(
				mux,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	//nolint:contextcheck
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// connections still open after the timeout are closed forcibly.
		if closeErr := srv.Close(); closeErr != nil {
			return fmt.Errorf("shutdown: %w", errors.Join(err, closeErr))
		}

		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}

func createMux(hdl *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	registerHandler(mux, "GET /files/{"+handler.PathValueDirPath+"...}", hdl.FileList)
	registerHandler(mux, "POST /files/{"+handler.PathValueDirPath+"...}", hdl.FileUpload)
	registerHandler(mux, "DELETE /files/{"+handler.PathValueFilePath+"...}", hdl.FileDelete)

	registerHandler(mux, "POST /dirs/{"+handler.PathValueDirPath+"...}", hdl.DirectoryCreate)

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
