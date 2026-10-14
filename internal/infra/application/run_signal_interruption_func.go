package application

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
)

func RunSignalInterruptionFunc(fn func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := fn(ctx); err != nil {
		return fmt.Errorf("run fn: %w", err)
	}

	return nil
}
