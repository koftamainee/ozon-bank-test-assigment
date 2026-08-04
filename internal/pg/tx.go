package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type PgxPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func WithTx(ctx context.Context, pool PgxPool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
