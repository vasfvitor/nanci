package app

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
)

// App encapsulates the global dependencies of the application.
type App struct {
	Log     *slog.Logger
	Store   store.Store
	DataDir string
}

// NewApp initializes the logger, resolves directories, and connects to the database.
func NewApp(verbose bool) (*App, error) {
	log := logger.New(verbose)

	dataDir, err := paths.DataDir()
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver diretório de dados: %w", err)
	}

	if err := paths.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório de dados: %w", err)
	}

	dbPath := filepath.Join(dataDir, "nanci.db")

	// Open the database and run migrations. For the CLI, running migrations
	// on startup is practical and ensures the schema is always up to date.
	sqliteStore, err := store.NewSQLiteStore(dbPath, true)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar banco de dados: %w", err)
	}

	return &App{
		Log:     log,
		Store:   sqliteStore,
		DataDir: dataDir,
	}, nil
}

// Close releases resources (such as the database connection).
func (a *App) Close() {
	if a.Store != nil {
		if s, ok := a.Store.(*store.SQLiteStore); ok {
			s.Close()
		}
	}
}
