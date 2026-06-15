package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/envfile"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

// CertPasswordRequest carries the context needed to ask for a certificate password.
type CertPasswordRequest struct {
	RequestID       string
	CompanyID       string
	CompanyName     string
	TargetCNPJ      string
	CredentialID    string
	CredentialLabel string
	CertPath        string
}

// CredentialProvider abstracts how certificate passwords are obtained.
// The CLI implements this via terminal prompts; Wails will implement it via
// a frontend dialog. internal/app must never import golang.org/x/term.
type CredentialProvider interface {
	GetCertPassword(ctx context.Context, req CertPasswordRequest) (string, error)
}

// App encapsulates the global dependencies of the application.
type App struct {
	Log                *slog.Logger
	DB                 *sql.DB
	CompanyRepo        nfse.CompanyRepository
	CredentialRepo     nfse.CredentialRepository
	SyncRepo           nfse.SyncRepository
	DocumentReader     nfse.DocumentReader
	XMLStore           files.XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
}

// Dependencies contains the infrastructure required by App.
type Dependencies struct {
	Log                *slog.Logger
	DB                 *sql.DB
	CompanyRepo        nfse.CompanyRepository
	CredentialRepo     nfse.CredentialRepository
	SyncRepo           nfse.SyncRepository
	DocumentReader     nfse.DocumentReader
	XMLStore           files.XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
}

// RuntimeOptions defines how a production app instance should be assembled.
type RuntimeOptions struct {
	Log                *slog.Logger
	CredentialProvider CredentialProvider
	DataDir            string
	RunMigrations      bool
}

// New constructs an App and rejects incomplete dependency graphs.
func New(deps Dependencies) (*App, error) {
	switch {
	case deps.Log == nil:
		return nil, errors.New("app: logger is required")
	case deps.DB == nil:
		return nil, errors.New("app: database is required")
	case deps.CompanyRepo == nil:
		return nil, errors.New("app: company repository is required")
	case deps.CredentialRepo == nil:
		return nil, errors.New("app: credential repository is required")
	case deps.SyncRepo == nil:
		return nil, errors.New("app: sync repository is required")
	case deps.DocumentReader == nil:
		return nil, errors.New("app: document reader is required")
	case deps.XMLStore == nil:
		return nil, errors.New("app: XML store is required")
	case deps.DataDir == "":
		return nil, errors.New("app: data directory is required")
	case deps.CredentialProvider == nil:
		return nil, errors.New("app: credential provider is required")
	}

	return &App{
		Log:                deps.Log,
		DB:                 deps.DB,
		CompanyRepo:        deps.CompanyRepo,
		CredentialRepo:     deps.CredentialRepo,
		SyncRepo:           deps.SyncRepo,
		DocumentReader:     deps.DocumentReader,
		XMLStore:           deps.XMLStore,
		DataDir:            deps.DataDir,
		CredentialProvider: deps.CredentialProvider,
	}, nil
}

// LoadRuntimeEnv loads supported .env.local files.
func LoadRuntimeEnv() error {
	if err := envfile.LoadLocal(); err != nil {
		return fmt.Errorf("carregar .env.local: %w", err)
	}
	return nil
}

// NewRuntime assembles the production app dependencies shared by CLI and desktop entrypoints.
// Callers are expected to load runtime environment variables before invoking it.
func NewRuntime(opts RuntimeOptions) (*App, error) {
	if opts.Log == nil {
		return nil, errors.New("app: logger is required")
	}
	if opts.CredentialProvider == nil {
		return nil, errors.New("app: credential provider is required")
	}

	dataDir, err := resolveRuntimeDataDir(opts.DataDir)
	if err != nil {
		return nil, err
	}
	if err := paths.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("criar diretório de dados: %w", err)
	}

	db, err := store.OpenDB(runtimeDBPath(dataDir), shouldRunMigrations(opts))
	if err != nil {
		return nil, fmt.Errorf("inicializar banco de dados: %w", err)
	}

	application, err := New(Dependencies{
		Log:                opts.Log,
		DB:                 db,
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     store.NewDocumentRepository(db),
		XMLStore:           files.NewBlobStore(dataDir),
		DataDir:            dataDir,
		CredentialProvider: opts.CredentialProvider,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configurar aplicação: %w", err)
	}
	return application, nil
}

func resolveRuntimeDataDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return "", fmt.Errorf("resolver diretório de dados: %w", err)
	}
	return dataDir, nil
}

func runtimeDBPath(dataDir string) string {
	return filepath.Join(dataDir, "nanci-v2.db")
}

func shouldRunMigrations(opts RuntimeOptions) bool {
	if !opts.RunMigrations {
		return false
	}
	return true
}

// Close releases resources (such as the database connection).
func (a *App) Close() {
	if a.DB != nil {
		_ = a.DB.Close()
	}
}
