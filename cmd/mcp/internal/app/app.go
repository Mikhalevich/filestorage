package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mikhalevich/filestorage/cmd/mcp/internal/app/tool"
	"github.com/Mikhalevich/filestorage/cmd/mcp/internal/config"
	"github.com/Mikhalevich/filestorage/internal/infra/application"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

const (
	serverName = "filestorage"
	mcpPath    = "/mcp"
)

func Start(
	ctx context.Context,
	cfg config.HTTPServer,
	limits config.Limits,
	fileSvc tool.FileService,
) error {
	server := NewServer(fileSvc, limits.MaxReadBytes, tracing.StdTracer())

	if err := startServer(
		ctx,
		cfg,
		NewHTTPHandler(server),
	); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

// NewServer creates mcp server with all file storage tools registered.
func NewServer(
	fileSvc tool.FileService,
	maxReadBytes int64,
	tracer tracing.Tracer,
) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: application.BuildVersion(),
	}, nil)

	server.AddReceivingMiddleware(Middleware(tracer))

	tool.New(fileSvc, maxReadBytes).Register(server)

	return server
}

// NewHTTPHandler serves the server over streamable http transport at /mcp.
func NewHTTPHandler(server *mcp.Server) http.Handler {
	mux := http.NewServeMux()

	// the server exposes only tools and never sends requests to the client, so sessions are not needed.
	mux.Handle(mcpPath, mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true},
	))

	return mux
}

func startServer(ctx context.Context, cfg config.HTTPServer, handler http.Handler) error {
	var (
		srv = &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadTimeout,
			// requests inherit the service logger, but are not canceled on shutdown signal.
			BaseContext: func(net.Listener) context.Context {
				return context.WithoutCancel(ctx)
			},
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
