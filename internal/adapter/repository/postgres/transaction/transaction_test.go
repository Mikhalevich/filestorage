package transaction_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Mikhalevich/filestorage/internal/adapter/repository/postgres/transaction"
)

func TestTransaction(t *testing.T) {
	t.Parallel()

	t.Run("begin error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			trx    = transaction.New(mockDB)
		)

		mockDB.EXPECT().Begin(t.Context()).Return(nil, errors.New("some begin tx error"))

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return nil
		})

		require.EqualError(t, err, "begin: some begin tx error")
	})

	t.Run("fn error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return fmt.Errorf("some fn error")
		})

		require.EqualError(t, err, "trx fn: some fn error")
	})

	t.Run("commit error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Commit().Return(errors.New("some commit error")),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return nil
		})

		require.EqualError(t, err, "trx commit: some commit error")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Commit().Return(nil),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return nil
		})

		require.NoError(t, err)
	})
}

func TestTransactionNested(t *testing.T) {
	t.Parallel()

	t.Run("fn inner error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return trx.Transaction(ctx, func(ctx context.Context) error {
				return errors.New("some inner trx error")
			})
		})

		require.EqualError(t, err, "trx fn: trx fn with active transaction: some inner trx error")
	})

	t.Run("fn outer error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			if err := trx.Transaction(ctx, func(ctx context.Context) error {
				return nil
			}); err != nil {
				return fmt.Errorf("unexpected error: %w", err)
			}

			return errors.New("some outer fn error")
		})

		require.EqualError(t, err, "trx fn: some outer fn error")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl   = gomock.NewController(t)
			mockDB = transaction.NewMockDB(ctrl)
			mockTx = transaction.NewMockDBTx(ctrl)
			trx    = transaction.New(mockDB)
		)

		gomock.InOrder(
			mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
			mockTx.EXPECT().Commit().Return(nil),
			mockTx.EXPECT().Rollback().Return(nil),
		)

		err := trx.Transaction(t.Context(), func(ctx context.Context) error {
			return trx.Transaction(ctx, func(ctx context.Context) error {
				return nil
			})
		})

		require.NoError(t, err)
	})
}

func TestExtContext(t *testing.T) {
	t.Parallel()

	var (
		ctrl   = gomock.NewController(t)
		mockDB = transaction.NewMockDB(ctrl)
		mockTx = transaction.NewMockDBTx(ctrl)
		trx    = transaction.New(mockDB)
	)

	gomock.InOrder(
		mockDB.EXPECT().Begin(t.Context()).Return(mockTx, nil),
		mockTx.EXPECT().Commit().Return(nil),
		mockTx.EXPECT().Rollback().Return(nil),
	)

	err := trx.Transaction(t.Context(), func(ctx context.Context) error {
		ext := trx.ExtContext(ctx)
		require.NotNil(t, ext)

		return nil
	})

	require.NoError(t, err)
}
