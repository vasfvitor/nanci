package app

import (
	"context"
	"database/sql"
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

// Close releases resources (such as the database connection).
func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
