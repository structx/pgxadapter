package pgxadapter

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// interface compliance
var _ Result = (*pgxResult)(nil)
var _ Row = (*pgxRow)(nil)
var _ Rows = (*pgxRows)(nil)
var _ Tx = (*pgxTx)(nil)
var _ DB = (*dbtx)(nil)
var _ DBTX = (*pgxConn)(nil)

type pgxConn struct {
	c *pgxpool.Conn
}

// Exec implements [DBTX].
func (p *pgxConn) Exec(ctx context.Context, stmt string, args ...interface{}) (Result, error) {
	result, err := p.c.Exec(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{result}, nil
}

// Query implements [DBTX].
func (p *pgxConn) Query(ctx context.Context, stmt string, args ...interface{}) (Rows, error) {
	rows, err := p.c.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{
		r:    rows,
		once: sync.Once{},
	}, nil
}

// QueryRow implements [DBTX].
func (p *pgxConn) QueryRow(ctx context.Context, stmt string, args ...interface{}) Row {
	row := p.c.QueryRow(ctx, stmt, args...)
	return &pgxRow{row}
}

type pgxResult struct {
	c pgconn.CommandTag
}

// RowsAffected implements [Result].
func (p *pgxResult) RowsAffected() int64 {
	return p.c.RowsAffected()
}

type pgxRow struct {
	r pgx.Row
}

// Scan implements [Row].
func (p *pgxRow) Scan(dest ...any) error {
	return p.r.Scan(dest...)
}

type pgxRows struct {
	r    pgx.Rows
	once sync.Once
}

// Close implements [Rows].
func (p *pgxRows) Close() {
	p.once.Do(func() {
		p.r.Close()
	})
}

// Err implements [Rows].
func (p *pgxRows) Err() error {
	return p.r.Err()
}

// Next implements [Rows].
func (p *pgxRows) Next() bool {
	return p.r.Next()
}

// Scan implements [Rows].
func (p *pgxRows) Scan(dest ...any) error {
	return p.r.Scan(dest...)
}

type pgxTx struct {
	tx    pgx.Tx
	ro    sync.Once
	co    sync.Once
	txErr error
}

// Exec implements [DBTX].
func (p *pgxTx) Exec(ctx context.Context, stmt string, args ...interface{}) (Result, error) {
	result, err := p.tx.Exec(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{
		c: result,
	}, nil
}

// Query implements [DBTX].
func (p *pgxTx) Query(ctx context.Context, stmt string, args ...interface{}) (Rows, error) {
	rows, err := p.tx.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{r: rows}, nil
}

// QueryRow implements [DBTX].
func (p *pgxTx) QueryRow(ctx context.Context, stmt string, args ...interface{}) Row {
	row := p.tx.QueryRow(ctx, stmt, args...)
	return &pgxRow{r: row}
}

// Commit implements [DBTX].
func (p *pgxTx) Commit(ctx context.Context) error {
	p.co.Do(func() {
		p.txErr = p.tx.Commit(ctx)
	})
	return p.txErr
}

// Rollback implements [DBTX].
func (p *pgxTx) Rollback(ctx context.Context) error {
	p.ro.Do(func() {
		p.txErr = p.tx.Rollback(ctx)
	})
	return p.txErr
}

type dbtx struct {
	p    *pgxpool.Pool
	once sync.Once
}

// Acquire implements [DB].
func (d *dbtx) Acquire(ctx context.Context) (DBTX, error) {
	conn, err := d.p.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxConn{conn}, nil
}

// Begin implements [DB].
func (d *dbtx) Begin(ctx context.Context) (Tx, error) {
	tx, err := d.p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{
		tx:    tx,
		ro:    sync.Once{},
		co:    sync.Once{},
		txErr: nil,
	}, nil
}

// Close implements [DB].
func (d *dbtx) Close(_ context.Context) error {
	d.once.Do(func() {
		d.p.Close()
	})
	return nil
}

// Exec implements [DB].
func (d *dbtx) Exec(ctx context.Context, stmt string, args ...interface{}) (Result, error) {
	result, err := d.p.Exec(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{result}, nil
}

// Ping implements [DB].
func (d *dbtx) Ping(ctx context.Context) error {
	return d.p.Ping(ctx)
}

// Query implements [DB].
func (d *dbtx) Query(ctx context.Context, stmt string, args ...interface{}) (Rows, error) {
	rows, err := d.p.Query(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{r: rows}, nil
}

// QueryRow implements [DB].
func (d *dbtx) QueryRow(ctx context.Context, stmt string, args ...interface{}) Row {
	row := d.p.QueryRow(ctx, stmt, args...)
	return &pgxRow{row}
}
