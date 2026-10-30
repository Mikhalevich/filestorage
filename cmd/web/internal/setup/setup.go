package setup

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func Start(
	ctx context.Context,
	cfg config.Config,
) error {
	authService, err := authsvc.New(
		[]byte(cfg.Auth.PublicKey),
		[]byte(cfg.Auth.PrivateKey),
		authsvc.WithIssuer(cfg.Auth.Issuer),
	)

	if err != nil {
		return fmt.Errorf("make auth service: %w", err)
	}

	var (
		fileService = filesvc.New(cfg.Storage.RootDir)
	)
	if err := app.Start(
		ctx,
		cfg.HTTPServer,
		fileService,
		authService,
	); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	return nil
}
