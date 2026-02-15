package pgxadapter

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type (
	// Conn
	Conn interface {
		Exec(context.Context, string, ...interface{}) (Result, error)
		QueryRow(context.Context, string, ...interface{}) Row
		Query(context.Context, string, ...interface{}) (Rows, error)
	}

	// Tx
	Tx pgx.Tx

	// Result
	Result pgx.BatchResults

	// Rows
	Rows pgx.Rows

	// Row
	Row pgx.Row

	// Pool
	Pool interface {
		Acquire(context.Context) (Conn, error)
		Begin(context.Context) (Tx, error)

		Ping(context.Context) error

		Exec(context.Context, string, ...interface{}) (Result, error)
		QueryRow(context.Context, string, ...interface{}) Row
		Query(context.Context, string, ...interface{}) (Rows, error)

		Close(context.Context) error
	}
)
