package fxmigrate

import (
	"context"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/structx/pgxadapter"
	pgxmigrate "github.com/structx/pgxadapter/migrate"

	"go.uber.org/fx"
)

// M
type M struct {
	Schema string
	Dir    string
	FS     embed.FS
}

// Params
type Params struct {
	fx.In

	Ctx context.Context `optional:"true"`

	Log migrate.Logger `optional:"true" name:"migrate_logger"`

	Ms []*M `group:"migrations"`

	DB pgxadapter.DB
}

// AsMigration uberfx helper function to quickly provide and annotate a migration
func AsMigration(schema, dir string, fs embed.FS) any {
	return fx.Annotate(
		func() *M {
			return &M{Schema: schema, Dir: dir, FS: fs}
		}, fx.ResultTags(`group:"migrations"`),
	)
}

// Module
var Module = fx.Module("pgx_migrations", fx.Invoke(invokeModule))

func invokeModule(p Params) error {
	baseCtx := p.Ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	for _, m := range p.Ms {
		src, err := iofs.New(m.FS, m.Dir)
		if err != nil {
			return fmt.Errorf("iofs.New: %w", err)
		}

		driver, err := pgxmigrate.WithInstance(baseCtx, p.DB, &pgxmigrate.Config{MigrationSchemaName: m.Schema})
		if err != nil {
			return fmt.Errorf("failed to create pgxadapter migrate instance: %w", err)
		}

		m, err := migrate.NewWithInstance("iofs", src, "pgxv5", driver)
		if err != nil {
			return fmt.Errorf("failed to create migration: %w", err)
		}

		if p.Log != nil {
			m.Log = p.Log
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
	}
	return nil
}
