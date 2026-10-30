package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func Start(
	ctx context.Context,
	cfg config.Config,
) error {
	tokenVerifier, err := makeTokenVerifier(cfg.Auth)
	if err != nil {
		return fmt.Errorf("make token verifier: %w", err)
	}

	var (
		fileService = filesvc.New(cfg.Storage.RootDir)
	)
	if err := app.Start(
		ctx,
		cfg.HTTPServer,
		fileService,
		tokenVerifier,
	); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	return nil
}

// makeTokenVerifier returns nil when authorization is disabled.
//
//nolint:nilnil
func makeTokenVerifier(cfg config.Auth) (middleware.TokenVerifier, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	keyPEM, err := os.ReadFile(cfg.PublicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read public key file: %w", err)
	}

	publicKey, err := authsvc.ParsePublicKeyPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return authsvc.New(publicKey, authsvc.WithIssuer(cfg.Issuer)), nil
}
