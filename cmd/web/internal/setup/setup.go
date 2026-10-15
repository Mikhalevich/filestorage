package setup

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func Start(
	ctx context.Context,
	cfg config.Config,
) error {
	var (
		fileService = filesvc.New(cfg.Storage.RootDir)
	)
	if err := app.Start(
		ctx,
		cfg.HTTPServer,
		fileService,
	); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	return nil
}
