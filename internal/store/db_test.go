package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/vasfvitor/nanci/internal/store"
)

// TestOpenDBConcurrentWriters runs transactions that read and then write from
// several goroutines, as concurrent syncs do. Without busy_timeout and
// _txlock=immediate most of them fail with "database is locked".
func TestOpenDBConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "test.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.ExecContext(ctx, `CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	const writers, txPerWriter = 3, 50
	errs := make(chan error, writers*txPerWriter)
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range txPerWriter {
				errs <- readThenWrite(ctx, db)
			}
		})
	}
	wg.Wait()
	close(errs)

	failed := 0
	var first error
	for err := range errs {
		if err != nil {
			failed++
			if first == nil {
				first = err
			}
		}
	}
	if failed > 0 {
		t.Fatalf("%d of %d transactions failed, first: %v", failed, writers*txPerWriter, first)
	}
}

func readThenWrite(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM probe`).Scan(&n); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO probe (v) VALUES ('x')`); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// TestOpenDBRefusesNewerDatabase opens a database that a newer nanci migrated
// (a version above the last embedded migration). goose alone accepts it.
func TestOpenDBRefusesNewerDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "newer.db")
	db, err := store.OpenDB(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (9999, 1)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	_, err = store.OpenDB(ctx, path, true)
	if !errors.Is(err, store.ErrDatabaseNewer) {
		t.Fatalf("OpenDB on a newer database: got %v, want ErrDatabaseNewer", err)
	}
}

func TestOpenDBReopensAnUpToDateDatabase(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "current.db")
	for range 2 {
		db, err := store.OpenDB(ctx, path, true)
		if err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
	}
}
