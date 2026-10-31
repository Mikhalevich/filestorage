package pguser

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Mikhalevich/filestorage/internal/adapter/repository/postgres/pguser/internal/model"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

const (
	// constraintUniqueName is the name Postgres gives to the UNIQUE constraint of users.name.
	constraintUniqueName = "users_name_key"
)

type Driver interface {
	IsConstraintError(err error, constraint string) bool
}

type Transactor interface {
	ExtContext(ctx context.Context) sqlx.ExtContext
}

type PgUser struct {
	driver     Driver
	transactor Transactor
}

func New(
	driver Driver,
	transactor Transactor,
) *PgUser {
	return &PgUser{
		driver:     driver,
		transactor: transactor,
	}
}

func (p *PgUser) CreateUser(
	ctx context.Context,
	usr user.User,
) error {
	var (
		query = `
			INSERT INTO users (
				name,
				password_hash,
				created_at
			) VALUES (
				:name,
				:password_hash,
				:created_at
			)
		`
	)
	_, err := sqlx.NamedExecContext(
		ctx,
		p.transactor.ExtContext(ctx),
		query,
		model.ToDBUser(usr),
	)
	if err != nil {
		if p.driver.IsConstraintError(err, constraintUniqueName) {
			return perror.AlreadyExists("user already exists")
		}

		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (p *PgUser) UserByName(
	ctx context.Context,
	name string,
) (user.User, error) {
	var (
		query = `
			SELECT
				id,
				name,
				password_hash,
				created_at
			FROM
				users
			WHERE
				name = $1
		`

		dbUser model.User
	)

	if err := sqlx.GetContext(
		ctx,
		p.transactor.ExtContext(ctx),
		&dbUser,
		query,
		name,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, perror.NotFound("user not found")
		}

		return user.User{}, fmt.Errorf("get context: %w", err)
	}

	return dbUser.ToDom(), nil
}
