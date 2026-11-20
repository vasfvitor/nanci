package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

// ErrDatabaseNewer means the database has migrations this binary does not
// know: a newer nanci (usually the desktop app after an automatic update)
// migrated it. goose would carry on without a word, and older code would then
// write to a schema it was not written for.
var ErrDatabaseNewer = errors.New("o banco de dados foi atualizado por uma versão mais nova do Nanci; use o nanci e o aplicativo da mesma versão")

//go:embed migrations_v2/*.sql
var embedMigrationsV2 embed.FS

// Migrations returns the embedded goose migrations.
func Migrations() (fs.FS, error) {
	return fs.Sub(embedMigrationsV2, "migrations_v2")
}

// OpenDB opens the SQLite database and optionally runs migrations.
func OpenDB(ctx context.Context, dbPath string, runMigrations bool) (*sql.DB, error) {
	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	// Syncs of different sources, the UI and the CLI write at the same time.
	// busy_timeout makes a writer wait for the lock instead of failing with
	// SQLITE_BUSY: 15 s, because a slow disk under load kept a writer waiting
	// past 5 s in CI, and every write here is short. _txlock=immediate takes
	// the write lock at BEGIN: a deferred transaction that reads first and
	// then writes fails at once when another connection wrote in between,
	// whatever the timeout.
	dsn := fmt.Sprintf("%s%s_pragma=journal_mode(wal)&_pragma=foreign_keys(1)&_pragma=busy_timeout(15000)&_txlock=immediate", dbPath, separator)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco sqlite: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("falha ao conectar no banco sqlite: %w", err)
	}

	if runMigrations {
		migrations, err := Migrations()
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao carregar migrations: %w", err)
		}

		provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao configurar migrations: %w", err)
		}

		if err := checkNotNewer(ctx, provider); err != nil {
			_ = db.Close()
			return nil, err
		}

		_, err = provider.Up(ctx)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao rodar migrations: %w", err)
		}
	}

	return db, nil
}

// checkNotNewer fails with ErrDatabaseNewer when the database was migrated
// past the last migration embedded in this binary.
func checkNotNewer(ctx context.Context, provider *goose.Provider) error {
	dbVersion, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("falha ao ler a versão do banco: %w", err)
	}
	var known int64
	for _, source := range provider.ListSources() {
		known = max(known, source.Version)
	}
	if dbVersion > known {
		return fmt.Errorf("%w (banco na migração %d, este programa conhece até a %d)", ErrDatabaseNewer, dbVersion, known)
	}
	return nil
}
