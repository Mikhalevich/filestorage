package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/router"
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
	mux := router.CreateMux(
		handler.New(fileSvc),
		middleware.Resource(),
		middleware.Tracing(tracing.StdTracer()),
		middleware.Logging(logger.FromContext(ctx)),
		middleware.Auth(tokenVerifier),
	)

	if err := startServer(ctx, cfg, mux); err != nil {
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
