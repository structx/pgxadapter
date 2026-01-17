package pgxadapter

import (
	"context"
)

type (

	// DBTX connection and transaction wrapper
	DBTX interface {
		Exec(context.Context, string, ...interface{}) (Result, error)
		QueryRow(context.Context, string, ...interface{}) Row
		Query(context.Context, string, ...interface{}) (Rows, error)
	}

	// Tx
	Tx interface {
		DBTX

		Commit(context.Context) error
		Rollback(context.Context) error
	}

	// Result
	Result interface {
		RowsAffected() int64
	}

	// Rows
	Rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}

	// Row
	Row interface {
		Scan(...any) error
	}

	// DB
	DB interface {
		DBTX

		Acquire(context.Context) (DBTX, error)

		Begin(context.Context) (Tx, error)
		Ping(context.Context) error

		Close(context.Context) error
	}
)
