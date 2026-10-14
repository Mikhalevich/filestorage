package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
)

const (
	shutdownTimeout = time.Second * 60
)

func Start(ctx context.Context, cfg config.HTTPServer) error {
	if err := startServer(
		ctx,
		cfg,
		createMux(handler.New()),
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

func createMux(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	registerHandler(mux, "/test", h.Test)
	registerHandler(mux, "/error_test", h.ErrorTest)

	return mux
}

func registerHandler(mux *http.ServeMux, path string, h HandlerFunc) {
	mux.HandleFunc(path, h.ToHTTPHandlerFunc())
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h HandlerFunc) ToHTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			if httpErr, ok := errors.AsType[httperror.HTTPError](err); ok {
				http.Error(w, httpErr.Msg, httpErr.Status)

				return
			}

			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}
