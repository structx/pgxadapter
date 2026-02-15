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
var _ Conn = (*pgxConn)(nil)
var _ Pool = (*pgxPool)(nil)

type pgxPool struct {
	p *pgxpool.Pool
}

// Acquire implements [Pool].
func (p *pgxPool) Acquire(ctx context.Context) (Conn, error) {
	conn, err := p.p.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxConn{
		c:       conn,
		connErr: nil,
		close:   sync.Once{},
	}, nil
}

// Begin implements [Pool].
func (p *pgxPool) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{
		tx:    tx,
		txErr: nil,
		ro:    sync.Once{},
		co:    sync.Once{},
	}, nil
}

// Close implements [Pool].
func (p *pgxPool) Close(ctx context.Context) error {
	p.p.Close()
	return nil
}

// Exec implements [Pool].
func (p *pgxPool) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := p.p.Exec(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{
		c: result,
	}, nil
}

// Ping implements [Pool].
func (p *pgxPool) Ping(ctx context.Context) error {
	return p.p.Ping(ctx)
}

// Query implements [Pool].
func (p *pgxPool) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	rows, err := p.p.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{
		r:    rows,
		once: sync.Once{},
	}, nil
}

// QueryRow implements [Pool].
func (p *pgxPool) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	row := p.p.QueryRow(ctx, sql, args...)
	return &pgxRow{
		r: row,
	}
}

type pgxConn struct {
	c       *pgxpool.Conn
	connErr error
	close   sync.Once
}

// Begin implements [Conn].
func (p *pgxConn) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.c.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{
		tx:    tx,
		txErr: nil,
		ro:    sync.Once{},
		co:    sync.Once{},
	}, nil
}

// Close implements [Conn].
func (p *pgxConn) Close(ctx context.Context) error {
	p.close.Do(func() {
		p.connErr = p.c.Conn().Close(ctx)
	})
	return p.connErr
}

// Ping implements [Conn].
func (p *pgxConn) Ping(ctx context.Context) error {
	return p.c.Ping(ctx)
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

// Close implements [Result].
func (p *pgxResult) Close() error {
	panic("unimplemented")
}

// Exec implements [Result].
func (p *pgxResult) Exec() (pgconn.CommandTag, error) {
	panic("unimplemented")
}

// Query implements [Result].
func (p *pgxResult) Query() (pgx.Rows, error) {
	panic("unimplemented")
}

// QueryRow implements [Result].
func (p *pgxResult) QueryRow() pgx.Row {
	panic("unimplemented")
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

// CommandTag implements [Rows].
func (p *pgxRows) CommandTag() pgconn.CommandTag {
	panic("unimplemented")
}

// Conn implements [Rows].
func (p *pgxRows) Conn() *pgx.Conn {
	panic("unimplemented")
}

// FieldDescriptions implements [Rows].
func (p *pgxRows) FieldDescriptions() []pgconn.FieldDescription {
	panic("unimplemented")
}

// RawValues implements [Rows].
func (p *pgxRows) RawValues() [][]byte {
	panic("unimplemented")
}

// Values implements [Rows].
func (p *pgxRows) Values() ([]any, error) {
	panic("unimplemented")
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

// Begin implements [Tx].
func (p *pgxTx) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := p.tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{
		tx:    tx,
		txErr: nil,
		ro:    sync.Once{},
		co:    sync.Once{},
	}, nil
}

// Commit implements [Tx].
func (p *pgxTx) Commit(ctx context.Context) error {
	p.co.Do(func() {
		p.txErr = p.tx.Commit(ctx)
	})
	return p.txErr
}

// Conn implements [Tx].
func (p *pgxTx) Conn() *pgx.Conn {
	return p.tx.Conn()
}

// CopyFrom implements [Tx].
func (p *pgxTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return p.tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// Exec implements [Tx].
func (p *pgxTx) Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error) {
	return p.tx.Exec(ctx, sql, arguments...)
}

// LargeObjects implements [Tx].
func (p *pgxTx) LargeObjects() pgx.LargeObjects {
	return p.tx.LargeObjects()
}

// Prepare implements [Tx].
func (p *pgxTx) Prepare(ctx context.Context, name string, sql string) (*pgconn.StatementDescription, error) {
	return p.tx.Prepare(ctx, name, sql)
}

// Query implements [Tx].
func (p *pgxTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.tx.Query(ctx, sql, args...)
}

// QueryRow implements [Tx].
func (p *pgxTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.tx.QueryRow(ctx, sql, args)
}

// Rollback implements [Tx].
func (p *pgxTx) Rollback(ctx context.Context) error {
	panic("unimplemented")
}

// SendBatch implements [Tx].
func (p *pgxTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	panic("unimplemented")
}
