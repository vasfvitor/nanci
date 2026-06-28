package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/foundation/envfile"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
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
	Log                *slog.Logger
	CompanyRepo        CompanyRepository
	CredentialRepo     CredentialRepository
	SyncRepo           SyncRepository
	DocumentReader     DocumentReader
	DocumentTracker    DocumentTracker
	XMLStore           XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
	DANFSeRenderer     danfse.Renderer
}

// Dependencies contains the infrastructure required by App.
type Dependencies struct {
	Log                *slog.Logger
	CompanyRepo        CompanyRepository
	CredentialRepo     CredentialRepository
	SyncRepo           SyncRepository
	DocumentReader     DocumentReader
	DocumentTracker    DocumentTracker
	XMLStore           XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
	DANFSeRenderer     danfse.Renderer
}

// New constructs an App and rejects incomplete dependency graphs.
func New(deps Dependencies) (*App, error) {
	switch {
	case deps.Log == nil:
		return nil, errors.New("app: logger is required")
	case deps.CompanyRepo == nil:
		return nil, errors.New("app: company repository is required")
	case deps.CredentialRepo == nil:
		return nil, errors.New("app: credential repository is required")
	case deps.SyncRepo == nil:
		return nil, errors.New("app: sync repository is required")
	case deps.DocumentReader == nil:
		return nil, errors.New("app: document reader is required")
	case deps.DocumentTracker == nil:
		return nil, errors.New("app: document tracker is required")
	case deps.XMLStore == nil:
		return nil, errors.New("app: XML store is required")
	case deps.DataDir == "":
		return nil, errors.New("app: data directory is required")
	case deps.CredentialProvider == nil:
		return nil, errors.New("app: credential provider is required")
	}

	return &App{
		Log:                deps.Log,
		CompanyRepo:        deps.CompanyRepo,
		CredentialRepo:     deps.CredentialRepo,
		SyncRepo:           deps.SyncRepo,
		DocumentReader:     deps.DocumentReader,
		DocumentTracker:    deps.DocumentTracker,
		XMLStore:           deps.XMLStore,
		DataDir:            deps.DataDir,
		CredentialProvider: deps.CredentialProvider,
		DANFSeRenderer:     deps.DANFSeRenderer,
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
