package transaction

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type transactionCtxKey struct{}

type DB interface {
	sqlx.ExtContext

	Begin(ctx context.Context) (DBTx, error)
}

type DBTx interface {
	sqlx.ExtContext

	Rollback() error
	Commit() error
}

type Transaction struct {
	db DB
}

func New(db DB) *Transaction {
	return &Transaction{
		db: db,
	}
}

func (t *Transaction) Transaction(
	ctx context.Context,
	trxFn func(ctx context.Context) error,
) error {
	if activeTrx := trxFromContext(ctx); activeTrx != nil {
		if err := trxFn(ctx); err != nil {
			return fmt.Errorf("trx fn with active transaction: %w", err)
		}

		return nil
	}

	trx, err := t.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	//nolint:errcheck
	defer trx.Rollback()

	if err := trxFn(context.WithValue(ctx, transactionCtxKey{}, trx)); err != nil {
		return fmt.Errorf("trx fn: %w", err)
	}

	if err := trx.Commit(); err != nil {
		return fmt.Errorf("trx commit: %w", err)
	}

	return nil
}

func (t *Transaction) ExtContext(ctx context.Context) sqlx.ExtContext {
	if activeTrx := trxFromContext(ctx); activeTrx != nil {
		return activeTrx
	}

	return t.db
}

func trxFromContext(ctx context.Context) DBTx {
	activeTrx, ok := ctx.Value(transactionCtxKey{}).(DBTx)

	if !ok {
		return nil
	}

	return activeTrx
}
