package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	readTimeout     = time.Second * 30
	writeTimeout    = time.Second * 30
	shutdownTimeout = time.Second * 60
)

func Start(ctx context.Context) error {
	if err := startServer(ctx, createMux()); err != nil {
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

func startServer(ctx context.Context, handler http.Handler) error {
	var (
		srv = &http.Server{
			Addr:         ":80",
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		}

		srvErrCh = make(chan error)
	)

	defer close(srvErrCh)

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
