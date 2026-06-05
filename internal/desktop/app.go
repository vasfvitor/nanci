package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction
type WailsCredentialProvider struct {
	ctx          context.Context
	passwordChan chan string
}

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided
func (p WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) (string, error) {
	// Notify the frontend to show the password dialog
	runtime.EventsEmit(p.ctx, "request-cert-password", req)

	// Block until the password is submitted by the frontend
	select {
	case pass := <-p.passwordChan:
		if pass == "" {
			return "", fmt.Errorf("operação cancelada pelo usuário")
		}
		return pass, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// App struct
type App struct {
	ctx          context.Context
	core         *app.App
	passwordChan chan string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		passwordChan: make(chan string),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	verbose := os.Getenv("NANCI_VERBOSE") == "1"
	coreApp, err := app.NewApp(verbose)
	if err != nil {
		fmt.Printf("failed to initialize core app: %v\n", err)
		return
	}

	// Inject the Wails-specific credential provider
	coreApp.CredentialProvider = WailsCredentialProvider{
		ctx:          ctx,
		passwordChan: a.passwordChan,
	}
	a.core = coreApp
}

func (a *App) shutdown(ctx context.Context) {
	if a.core != nil {
		a.core.Close()
	}
}

// --- Auth & Credentials ---

// SubmitCertPassword receives the password from the frontend dialog and unblocks GetCertPassword
func (a *App) SubmitCertPassword(password string) {
	// Non-blocking send in case it's called multiple times or nobody is listening
	select {
	case a.passwordChan <- password:
	default:
	}
}

// CancelCertPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (a *App) CancelCertPassword() {
	select {
	case a.passwordChan <- "":
	default:
	}
}

// SelectCertificate opens a file dialog to select a .pfx or .p12 file
func (a *App) SelectCertificate() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selecione o Certificado Digital",
		Filters: []runtime.FileFilter{
			{DisplayName: "Certificados (*.pfx; *.p12)", Pattern: "*.pfx;*.p12"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
}

// SelectExportDirectory opens a dialog to select an output directory for exports
func (a *App) SelectExportDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selecione a Pasta de Destino",
	})
}

// --- Core API Exposure ---

func (a *App) AddCompany(input app.AddCompanyInput) error {
	return a.core.AddCompany(a.ctx, input)
}

func (a *App) ListCompanies() ([]nfse.Company, error) {
	return a.core.ListCompanies(a.ctx)
}

func (a *App) Pull(input app.PullInput) (app.PullResult, error) {
	return a.core.Pull(a.ctx, input)
}

func (a *App) ListDocuments(input app.ListInput) ([]nfse.CompanyDocument, error) {
	return a.core.ListDocuments(a.ctx, input)
}

func (a *App) Status(cnpj string) (app.StatusResult, error) {
	return a.core.Status(a.ctx, cnpj)
}

func (a *App) ExportCSV(input app.ExportInput) error {
	return a.core.ExportCSV(a.ctx, input)
}

func (a *App) ExportXLSX(input app.ExportInput) error {
	return a.core.ExportXLSX(a.ctx, input)
}

func (a *App) ExportZIP(input app.ExportInput) error {
	return a.core.ExportZIP(a.ctx, input)
}
