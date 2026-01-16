package pgxadapter

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MockTx struct {
	pgx.Tx

	exec     func(context.Context, string, ...interface{}) (Result, error)
	commit   func(context.Context) error
	rollback func(context.Context) error
}

func (t *MockTx) Exec(ctx context.Context, stmt string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (t *MockTx) Commit(ctx context.Context) error {
	return t.commit(ctx)
}

func (t *MockTx) Rollback(ctx context.Context) error {
	return t.rollback(ctx)
}
