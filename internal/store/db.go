package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var ErrNotFound = sql.ErrNoRows

//go:embed migrations_v2/*.sql
var embedMigrationsV2 embed.FS

// OpenDB opens the SQLite database and optionally runs migrations.
func OpenDB(dbPath string, runMigrations bool) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco sqlite: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("falha ao conectar no banco sqlite: %w", err)
	}

	if runMigrations {
		migrations, err := fs.Sub(embedMigrationsV2, "migrations_v2")
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao carregar migrations: %w", err)
		}

		provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao configurar migrations: %w", err)
		}

		_, err = provider.Up(context.Background())
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("falha ao rodar migrations: %w", err)
		}
	}

	return db, nil
}
