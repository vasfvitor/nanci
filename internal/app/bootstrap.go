package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
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

// Close releases resources (such as the database connection).
func (a *App) Close() {
	if a.DB != nil {
		_ = a.DB.Close()
	}
}
