package pgxadapter

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option pgxpool config option
type Option func(*pgxpool.Config)

// WithMaxConns
func WithMaxConns(conns int32) Option {
	return func(c *pgxpool.Config) {
		c.MaxConns = conns
	}
}

// WithTraceLogger
func WithTraceLogger(tracer pgx.QueryTracer) Option {
	return func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = tracer
	}
}

// New
func New(ctx context.Context, dsn string, opts ...Option) (Conn, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	for _, opt := range opts {
		opt(config)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create db pool: %w", err)
	}

	return &pgxPool{p: pool}, nil
}
