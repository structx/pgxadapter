package migrate

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/multistmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/structx/pgxadapter"
)

func init() {
	database.Register("pgxv5", &Pgxv5{})
}

const advisoryLockIDSalt uint = 1486364155

var multiStmtDelimiter = []byte(";")

const defaultMigrationTable = "schema_migrations"
const defaultMigrationSchema = "public"
const defaultMultiStatementMaxSize = 10 * 1 << 20 // 10 MB

var (
	// ErrMissingConfig
	ErrMissingConfig = errors.New("config is nil")
)

// Config
type Config struct {
	DatabaseName          string        `env:"PGM_DB_NAME"`
	MigrationTable        string        `env:"PGM_MIGRATION_TABLE"`
	MigrationSchemaName   string        `env:"PGM_MIGRATION_SCHEMA_NAME"`
	StatementTimeout      time.Duration `env:"PGM_STATEMENT_TIMEOUT"`
	MultiStatementEnabled bool          `env:"PGM_MULTI_STATEMENT_ENABLED"`
	MultiStatementMaxSize int           `env:"PGM_MULTI_STATEMENT_MAX_SIZE"`
}

// Pgxv5
type Pgxv5 struct {
	conn pgxadapter.DBTX
	db   pgxadapter.DB

	isLocked atomic.Bool

	cfg *Config
}

// interface compliance
var _ database.Driver = (*Pgxv5)(nil)

// WithInstance
func WithInstance(ctx context.Context, db pgxadapter.DB, config *Config) (database.Driver, error) {
	if config == nil {
		return nil, ErrMissingConfig
	}

	dbtx, err := db.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire db conn: %w", err)
	}

	p := &Pgxv5{
		conn:     dbtx,
		db:       db,
		isLocked: atomic.Bool{},
		cfg:      config,
	}

	if err := p.ensureVersionTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure version table exists: %w", err)
	}

	return p, nil
}

// Close implements [database.Driver].
func (p *Pgxv5) Close() error {
	return p.db.Close(context.Background())
}

// Drop implements [database.Driver].
func (p *Pgxv5) Drop() error {
	ctx := context.Background()

	stmt := `SELECT table_name FROM information_schema.tables WHERE table_schema=$1 AND table_type='BASE TABLE'`
	tables, err := p.conn.Query(ctx, stmt, p.cfg.MigrationSchemaName)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(stmt)}
	}
	defer tables.Close()

	tableNames := make([]string, 0)
	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			return err
		}
		if len(tableName) > 0 {
			tableNames = append(tableNames, tableName)
		}
	}

	if err := tables.Err(); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(stmt)}
	}

	if len(tableNames) > 0 {
		for _, t := range tableNames {
			stmt = fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", t)
			if _, err := p.conn.Exec(ctx, stmt); err != nil {
				return &database.Error{OrigErr: err, Query: []byte(stmt)}
			}
		}
	}

	return nil
}

// Lock implements [database.Driver].
func (p *Pgxv5) Lock() error {
	stmt := "SELECT pg_advisory_lock($1)"
	aid := generateAdvisoryLockID(p.cfg.DatabaseName, p.cfg.MigrationTable, p.cfg.MigrationSchemaName)
	if _, err := p.conn.Exec(context.TODO(), stmt, aid); err != nil {
		return fmt.Errorf("failed to execute advisory lock query: %w", err)
	}

	p.isLocked.Store(true)

	return nil
}

// Open implements [database.Driver].
func (p *Pgxv5) Open(dsn string) (database.Driver, error) {
	durl, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("dial string is not valid url: %w", err)
	}

	db, err := pgxadapter.New(context.Background(), migrate.FilterCustomQuery(durl).String())
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	migrationTable := durl.Query().Get("x-migrations-table")
	if len(migrationTable) == 0 {
		migrationTable = p.cfg.MigrationTable
	}

	if len(migrationTable) == 0 {
		migrationTable = defaultMigrationTable
	}

	migrationSchemaName := durl.Query().Get("x-schema-name")
	if len(migrationSchemaName) == 0 {
		migrationSchemaName = p.cfg.MigrationSchemaName
	}

	if len(migrationSchemaName) == 0 {
		migrationSchemaName = defaultMigrationSchema
	}

	statementTimeoutStr := durl.Query().Get("x-statement-timeout")
	statementTimeout := 0
	if len(statementTimeoutStr) == 0 {
		statementTimeout = int(p.cfg.StatementTimeout)
	} else {
		statementTimeout, err = strconv.Atoi(statementTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert statement timeout to int: %w", err)
		}
	}

	multiStatementMaxSize := defaultMultiStatementMaxSize
	maxSize := durl.Query().Get("x-multi-statement-max-size")
	if len(maxSize) != 0 {
		multiStatementMaxSize, err = strconv.Atoi(maxSize)
		if err != nil {
			return nil, fmt.Errorf("failed to convert multi statement max size to int: %w", err)
		}
	}

	multiStatementEnabled := false
	enabled := durl.Query().Get("x-multi-statement")
	if len(enabled) != 0 {
		multiStatementEnabled, err = strconv.ParseBool(enabled)
		if err != nil {
			return nil, fmt.Errorf("failed to convert string to bool: %w", err)
		}
	} else {
		multiStatementEnabled = p.cfg.MultiStatementEnabled
	}

	instance, err := WithInstance(context.Background(), db, &Config{
		DatabaseName:          durl.Path,
		MigrationTable:        migrationTable,
		MigrationSchemaName:   migrationSchemaName,
		StatementTimeout:      time.Duration(statementTimeout) * time.Second,
		MultiStatementEnabled: multiStatementEnabled,
		MultiStatementMaxSize: multiStatementMaxSize,
	})

	return instance, nil
}

// Run implements [database.Driver].
func (p *Pgxv5) Run(migration io.Reader) error {
	if p.cfg.MultiStatementEnabled {
		var stmtErr error
		if err := multistmt.Parse(migration, multiStmtDelimiter, p.cfg.MultiStatementMaxSize, func(m []byte) bool {
			if stmtErr = p.runStatement(m); stmtErr != nil {
				return false
			}
			return true
		}); err != nil {
			return err
		}
		return stmtErr
	}

	m, err := io.ReadAll(migration)
	if err != nil {
		return fmt.Errorf("failed to read migration: %w", err)
	}

	return p.runStatement(m)
}

func (p *Pgxv5) runStatement(stmt []byte) error {
	ctx := context.Background()
	if p.cfg.StatementTimeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.cfg.StatementTimeout)
		defer cancel()
	}

	query := string(stmt)
	if strings.TrimSpace(query) == "" {
		return nil
	}

	if _, err := p.conn.Exec(ctx, query); err != nil {
		return &database.Error{OrigErr: err, Err: "migration failed", Query: stmt}
	}
	return nil
}

// SetVersion implements [database.Driver].
func (p *Pgxv5) SetVersion(version int, dirty bool) error {
	ctx := context.Background()

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}

	stmt1 := fmt.Sprintf("TRUNCATE %s.%s", p.cfg.MigrationSchemaName, p.cfg.MigrationTable)
	if _, err := tx.Exec(ctx, stmt1); err != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			err = errors.Join(err, errRollback)
		}
		return &database.Error{OrigErr: err, Query: []byte(stmt1)}
	}

	if version >= 0 || (version == database.NilVersion && dirty) {
		stmt2 := fmt.Sprintf("INSERT INTO %s.%s (version, is_dirty) VALUES ($1, $2)", p.cfg.MigrationSchemaName, p.cfg.MigrationTable)
		if _, err = tx.Exec(ctx, stmt2, version, dirty); err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = errors.Join(err, errRollback)
			}
			return &database.Error{OrigErr: err, Query: []byte(stmt2)}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}

	return nil
}

// Unlock implements [database.Driver].
func (p *Pgxv5) Unlock() error {
	stmt := "SELECT pg_advisory_unlock($1)"
	aid := generateAdvisoryLockID(p.cfg.DatabaseName, p.cfg.MigrationTable, p.cfg.MigrationSchemaName)
	if _, err := p.conn.Exec(context.TODO(), stmt, aid); err != nil {
		return fmt.Errorf("failed to execute advisory unlock query: %w", err)
	}
	p.isLocked.Store(false)
	return nil
}

// Version implements [database.Driver].
func (p *Pgxv5) Version() (version int, dirty bool, err error) {
	stmt := fmt.Sprintf("SELECT version, is_dirty FROM %s.%s LIMIT 1", p.cfg.MigrationSchemaName, p.cfg.MigrationTable)
	err = p.conn.QueryRow(context.TODO(), stmt).Scan(&version, &dirty)
	switch {
	case err == pgx.ErrNoRows:
		return database.NilVersion, false, nil
	case err != nil:
		if e, ok := err.(*pgconn.PgError); ok {
			if e.SQLState() == pgerrcode.UndefinedTable {
				return database.NilVersion, false, nil
			}
		}
		return 0, false, &database.Error{Query: []byte(stmt), OrigErr: err}
	default:
		return version, dirty, nil
	}
}

func (p *Pgxv5) ensureVersionTable() (err error) {
	if err = p.Lock(); err != nil {
		return fmt.Errorf("failed to lock: %w", err)
	}

	defer func() {
		if unlockErr := p.Unlock(); unlockErr != nil {
			err = errors.Join(err, unlockErr)
		}
	}()

	stmt := "SELECT COUNT(1) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2 LIMIT 1"
	r := p.conn.QueryRow(context.Background(), stmt, p.cfg.MigrationSchemaName, p.cfg.MigrationTable)

	var count int
	err = r.Scan(&count)
	if err != nil {
		return &database.Error{OrigErr: err, Query: []byte(stmt)}
	}

	if count == 1 {
		// migrations table already exists
		return nil
	}

	stmt2 := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (version BIGINT NOT NULL PRIMARY KEY, is_dirty BOOLEAN NOT NULL)",
		p.cfg.MigrationSchemaName, p.cfg.MigrationTable)
	if _, err := p.conn.Exec(context.Background(), stmt2); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(stmt2)}
	}

	return nil
}

func generateAdvisoryLockID(databaseName string, additionalNames ...string) string {
	if len(additionalNames) > 0 {
		databaseName = strings.Join(additionalNames, "\x00")
	}

	sum := crc32.ChecksumIEEE([]byte(databaseName))
	sum = sum * uint32(advisoryLockIDSalt)
	return fmt.Sprint(sum)
}
