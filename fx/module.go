package fx

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/fx"

	"github.com/structx/pgxadapter"
)

// Config pg configuration
type Config struct {
	User        string `env:"PGX_USERNAME"`
	Password    string `env:"PGX_PASSWORD"`
	Host        string `env:"PGX_HOST"`
	Port        string `env:"PGX_PORT"`
	DB          string `env:"PGX_DATABASE"`
	ExtraParams string `env:"PGX_EXTRA_PARAMS"`

	MaxConns int32 `env:"PGX_MAX_CONNS"`
}

// Dial build url dial string from config
func (c *Config) Dial() string {
	var b strings.Builder

	b.WriteString("postgres://")
	b.WriteString(c.User + ":")
	b.WriteString(c.Password + "@")
	b.WriteString(c.Host + ":")
	b.WriteString(c.Port + "/")
	b.WriteString(c.DB + "?")
	b.WriteString(c.ExtraParams)

	return b.String()
}

// Params fx in
type Params struct {
	fx.In

	Lc fx.Lifecycle

	Ctx context.Context `optional:"true"`

	Cfg *Config
}

// Result fx out
type Result struct {
	fx.Out

	DB pgxadapter.DB
}

// Module pgxadapter fx module
var Module = fx.Module("pgxadapter", fx.Provide(newModule))

func newModule(p Params) (Result, error) {
	baseCtx := p.Ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	dsn := p.Cfg.Dial()
	opts := []pgxadapter.Option{
		pgxadapter.WithMaxConns(p.Cfg.MaxConns),
	}

	dbtx, err := pgxadapter.New(baseCtx, dsn, opts...)
	if err != nil {
		return Result{}, fmt.Errorf("failed to open pgx database: %w", err)
	}

	p.Lc.Append(fx.Hook{
		OnStart: dbtx.Ping,
		OnStop:  dbtx.Close,
	})

	return Result{DB: dbtx}, nil
}
