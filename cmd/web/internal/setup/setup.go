package setup

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/config"
	"github.com/Mikhalevich/filestorage/internal/adapter/repository/postgres/driver"
	"github.com/Mikhalevich/filestorage/internal/adapter/repository/postgres/pguser"
	"github.com/Mikhalevich/filestorage/internal/adapter/repository/postgres/transaction"
	"github.com/Mikhalevich/filestorage/internal/adapter/timeprovider"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
	"github.com/Mikhalevich/filestorage/internal/domain/service/usersvc"
	"github.com/Mikhalevich/filestorage/internal/domain/usecase/login"
	"github.com/Mikhalevich/filestorage/internal/domain/usecase/register"
)

func Start(
	ctx context.Context,
	cfg config.Config,
) error {
	authService, err := authsvc.New(
		[]byte(cfg.Auth.PublicKey),
		[]byte(cfg.Auth.PrivateKey),
		authsvc.WithIssuer(cfg.Auth.Issuer),
		authsvc.WithTTL(cfg.Auth.TokenTTL),
	)

	if err != nil {
		return fmt.Errorf("make auth service: %w", err)
	}

	dbConn, driver, cleanup, err := MakePGXConnection(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("make pgx connection: %w", err)
	}
	defer cleanup()

	var (
		sqlxDBConn          = sqlx.NewDb(dbConn, driver.Name())
		transactionProvider = transaction.New(transaction.NewSqlxDB(sqlxDBConn))
		timeProvider        = timeprovider.New()

		fileService = filesvc.New(cfg.Storage.RootDir)
		userService = usersvc.New(
			pguser.New(driver, transactionProvider),
		)

		registerUsecase = register.New(userService, authService, timeProvider)
		loginUsecase    = login.New(userService, authService)
	)

	if err := app.Start(
		ctx,
		cfg.HTTPServer,
		fileService,
		registerUsecase,
		loginUsecase,
		authService,
	); err != nil {
		return fmt.Errorf("start app: %w", err)
	}

	return nil
}

func MakePGXConnection(ctx context.Context, cfg config.Postgres) (*sql.DB, *driver.Pgx, func(), error) {
	if cfg.Connection == "" {
		return nil, nil, func() {}, nil
	}

	driver := driver.NewPgx()

	dbConn, err := otelsql.Open(driver.Name(), cfg.Connection)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open database: %w", err)
	}

	if err := dbConn.PingContext(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("ping: %w", err)
	}

	return dbConn, driver, func() {
		dbConn.Close()
	}, nil
}
