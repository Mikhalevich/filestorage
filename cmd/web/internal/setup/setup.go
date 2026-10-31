package setup

import (
	"context"
	"database/sql"
	"errors"
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

	dbConn, dbDriver, cleanup, err := MakePGXConnection(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("make pgx connection: %w", err)
	}
	defer cleanup()

	var (
		sqlxDBConn          = sqlx.NewDb(dbConn, dbDriver.Name())
		transactionProvider = transaction.New(transaction.NewSqlxDB(sqlxDBConn))
		timeProvider        = timeprovider.New()

		fileService = filesvc.New(cfg.Storage.RootDir)
		userService = usersvc.New(
			pguser.New(dbDriver, transactionProvider),
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

// MakePGXConnection opens and pings the database. The returned cleanup closes the connection,
// it is a no-op when an error is returned.
func MakePGXConnection(
	ctx context.Context,
	cfg config.Postgres,
) (*sql.DB, *driver.Pgx, func(), error) {
	var (
		noop      = func() {}
		pgxDriver = driver.NewPgx()
	)

	dbConn, err := otelsql.Open(pgxDriver.Name(), cfg.Connection)
	if err != nil {
		return nil, nil, noop, fmt.Errorf("open database: %w", err)
	}

	if err := dbConn.PingContext(ctx); err != nil {
		return nil, nil, noop, fmt.Errorf("ping: %w", errors.Join(err, dbConn.Close()))
	}

	return dbConn, pgxDriver, func() {
		dbConn.Close()
	}, nil
}
