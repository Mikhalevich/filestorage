package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
)

const (
	shutdownTimeout = time.Second * 60
)

func Start(ctx context.Context, cfg config.HTTPServer) error {
	if err := startServer(ctx, cfg, createMux()); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

func createMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "it works!!!")
	})

	return mux
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
