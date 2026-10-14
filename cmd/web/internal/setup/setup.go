package setup

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
)

func Start(
	ctx context.Context,
	cfg config.Config,
) error {
	if err := app.Start(ctx); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	return nil
}
