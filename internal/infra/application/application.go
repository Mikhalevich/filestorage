package application

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/Mikhalevich/filestorage/internal/infra/logger"
	"github.com/Mikhalevich/filestorage/internal/infra/tracing"
)

const (
	tracerShutdownTimeout = time.Second * 10
)

type Configer interface {
	Level() string
	ServiceName() string
	TracingEndpoint() string
}

type RunFn func(ctx context.Context) error

// Run sets up infra, executes runFn and exits the process with a non-zero code on failure.
func Run(
	cfg Configer,
	runFn RunFn,
) {
	os.Exit(run(cfg, runFn))
}

// run returns the process exit code, so deferred cleanup (tracer flush) happens before os.Exit.
func run(
	cfg Configer,
	runFn RunFn,
) int {
	if err := LoadConfig(cfg); err != nil {
		logger.StdLogger().WithError(err).Error("failed to load config")

		return 1
	}

	log, err := SetupLogger(cfg.Level())
	if err != nil {
		logger.StdLogger().WithError(err).Error("failed to setup logger")

		return 1
	}

	serviceLogger := log.WithField("service_name", cfg.ServiceName())

	shutdownTracer, err := tracing.SetupTracer(cfg.TracingEndpoint(), cfg.ServiceName(), BuildVersion())
	if err != nil {
		serviceLogger.WithError(err).Error("failed to setup tracer")

		return 1
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), tracerShutdownTimeout)
		defer cancel()

		if err := shutdownTracer(ctx); err != nil {
			serviceLogger.WithError(err).Error("failed to shutdown tracer")
		}
	}()

	if err := RunSignalInterruptionFunc(func(ctx context.Context) error {
		serviceLogger.Info("service starting")
		defer serviceLogger.Info("service stopped")

		if err := runFn(logger.WithLogger(ctx, serviceLogger)); err != nil {
			return fmt.Errorf("run fn: %w", err)
		}

		return nil
	}); err != nil {
		serviceLogger.WithError(err).Error("service run error")

		return 1
	}

	return 0
}

// BuildVersion returns the VCS revision stamped by the go toolchain, or empty string if unavailable.
func BuildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	var (
		revision string
		modified bool
	)

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if revision != "" && modified {
		return revision + "-dirty"
	}

	return revision
}
