package driver

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Pgx struct {
}

func NewPgx() *Pgx {
	return &Pgx{}
}

func (p *Pgx) Name() string {
	return "pgx"
}

func (p *Pgx) IsConstraintError(err error, constraint string) bool {
	var pgxError *pgconn.PgError
	if !errors.As(err, &pgxError) {
		return false
	}

	if pgxError.ConstraintName == constraint {
		return true
	}

	return false
}
