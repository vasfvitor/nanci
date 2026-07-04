package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/envfile"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
)

var ErrOperationCanceled = errors.New("operação cancelada pelo usuário")

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
	Companies   *company.Manager
	Credentials *CredentialService
	Documents   *DocumentService
	Exports     *ExportService
	Status      *SyncStatusService
	Query       *QueryService
	Sync        *SyncService
}

// Dependencies contains the infrastructure required by App.
type Dependencies struct {
	Log                *slog.Logger
	CompanyStore       *company.Store
	CredentialRepo     *store.CredentialRepository
	SyncRepo           *store.SyncRepository
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
	case deps.CredentialRepo == nil:
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
		Companies:   company.NewManager(deps.CompanyStore, deps.CredentialRepo, deps.SyncRepo),
		Credentials: NewCredentialService(deps),
		Documents:   NewDocumentService(deps),
		Exports:     NewExportService(deps),
		Status:      NewSyncStatusService(deps),
		Query:       NewQueryService(deps),
		Sync:        NewSyncService(deps),
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
