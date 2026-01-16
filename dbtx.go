package pgxadapter

import (
	"context"
)

// DBTX
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (Result, error)
	QueryRow(context.Context, string, ...interface{}) Row
	Query(context.Context, string, ...interface{}) (Rows, error)
}

// Tx
type Tx interface {
	DBTX

	Commit(context.Context) error
	Rollback(context.Context) error
}

// Result
type Result interface {
	RowsAffected() int64
}

// Rows
type Rows interface {
	Next() bool
	Scan(...any) error
	Close()
	Err() error
}

// Row
type Row interface {
	Scan(...any) error
}

// DB
type DB interface {
	DBTX

	Begin(context.Context) (Tx, error)
	Ping(context.Context) error

	Close(context.Context) error
}
