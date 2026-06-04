package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens the database connection and optionally runs migrations.
func NewSQLiteStore(dbPath string, runMigrations bool) (*SQLiteStore, error) {
	// Recommended parameters for SQLite
	// _pragma=journal_mode(wal) - Write-Ahead Logging for better concurrency
	// _pragma=foreign_keys(1) - Enables foreign key constraints
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha ao conectar no banco sqlite: %w", err)
	}

	if runMigrations {
		if err := RunMigrations(db); err != nil {
			return nil, fmt.Errorf("falha ao rodar migrations: %w", err)
		}
	}

	return &SQLiteStore{
		db: db,
	}, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
