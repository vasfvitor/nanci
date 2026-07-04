package app

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/envfile"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

var ErrOperationCanceled = errors.New("operação cancelada pelo usuário")

// CertPasswordRequest carries the context needed to ask for a certificate password.
type CertPasswordRequest = sync.CertPasswordRequest

// CredentialProvider abstracts how certificate passwords are obtained.
type CredentialProvider = sync.CredentialProvider

// App encapsulates the global dependencies of the application.
type App struct {
	Companies   *company.Manager
	Credentials *credential.Manager
	Documents   *DocumentService
	Exports     *ExportService
	Query       *QueryService
	SyncManager *sync.Manager
}

// Dependencies contains the infrastructure required by App.
type Dependencies struct {
	Log                *slog.Logger
	CompanyStore       *company.Store
	CredentialStore    *credential.Store
	SyncRepo           *sync.Store
	DocumentRepo       *store.DocumentRepository
	XMLStore           files.XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
	DANFSeRenderer     danfse.Renderer
}

// New constructs an App and rejects incomplete dependency graphs.
func New(deps Dependencies) (*App, error) {
	switch {
	case deps.Log == nil:
		return nil, errors.New("app: logger is required")
	case deps.CompanyStore == nil:
		return nil, errors.New("app: company repository is required")
	case deps.CredentialStore == nil:
		return nil, errors.New("app: credential repository is required")
	case deps.SyncRepo == nil:
		return nil, errors.New("app: sync repository is required")
	case deps.DocumentRepo == nil:
		return nil, errors.New("app: document repository is required")
	case deps.XMLStore == nil:
		return nil, errors.New("app: XML store is required")
	case deps.DataDir == "":
		return nil, errors.New("app: data directory is required")
	case deps.CredentialProvider == nil:
		return nil, errors.New("app: credential provider is required")
	}

	return &App{
		Companies:   company.NewManager(deps.CompanyStore, deps.CredentialStore, deps.SyncRepo),
		Credentials: credential.NewManager(deps.CredentialStore),
		Documents:   NewDocumentService(deps),
		Exports:     NewExportService(deps),
		Query:       NewQueryService(deps),
		SyncManager: &sync.Manager{
			Log:                deps.Log,
			CompanyProvider:    deps.CompanyStore,
			CredentialProvider: deps.CredentialStore,
			DocProvider:        deps.DocumentRepo,
			SyncRepo:           deps.SyncRepo,
			XMLStore:           deps.XMLStore,
			PassProvider:       deps.CredentialProvider,
		},
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
func NewRuntime(deps Dependencies) (*App, error) {
	return New(deps)
}

func ResolveRuntimeDataDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return "", fmt.Errorf("resolver diretório de dados: %w", err)
	}
	return dataDir, nil
}

func RuntimeDBPath(dataDir string) string {
	return filepath.Join(dataDir, "nanci-v1.db")
}
