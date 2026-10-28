package main

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/cmd/bot/internal/config"
	"github.com/Mikhalevich/filestorage/cmd/bot/internal/setup"
	"github.com/Mikhalevich/filestorage/internal/infra/application"
)

func main() {
	var cfg config.Config
	application.Run(&cfg, func(ctx context.Context) error {
		if err := setup.Start(ctx, cfg); err != nil {
			return fmt.Errorf("start bot: %w", err)
		}

		return nil
	})
}
