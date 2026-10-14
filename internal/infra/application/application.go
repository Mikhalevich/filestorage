package application

import (
	"context"
	"fmt"
	"os"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

type Configer interface {
	Level() string
	ServiceName() string
	TracingEndpoint() string
}

type RunFn func(ctx context.Context) error

func Run(
	cfg Configer,
	runFn RunFn,
) {
	if err := LoadConfig(cfg); err != nil {
		logger.StdLogger().WithError(err).Error("failed to load config")
		os.Exit(1)
	}

	log, err := SetupLogger(cfg.Level())
	if err != nil {
		logger.StdLogger().WithError(err).Error("failed to setup logger")
		os.Exit(1)
	}

	serviceLogger := log.WithField("service_name", cfg.ServiceName())

	if err := tracing.SetupTracer(cfg.TracingEndpoint(), cfg.ServiceName(), ""); err != nil {
		serviceLogger.WithError(err).Error("failed to setup tracer")
		os.Exit(1)
	}

	if err := RunSignalInterruptionFunc(func(ctx context.Context) error {
		serviceLogger.Info("service starting")
		defer serviceLogger.Info("service stopped")

		if err := runFn(logger.WithLogger(ctx, serviceLogger)); err != nil {
			return fmt.Errorf("run fn: %w", err)
		}

		return nil
	}); err != nil {
		serviceLogger.WithError(err).Error("service run error")
		os.Exit(1)
	}
}
