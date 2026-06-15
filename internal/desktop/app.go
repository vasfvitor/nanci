package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vasfvitor/nanci/internal/app"
	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction
type WailsCredentialProvider struct {
	ctx           context.Context
	passwordChans map[string]chan string
	mu            *sync.Mutex
}

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided
func (p WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) (string, error) {
	ch := make(chan string, 1)

	p.mu.Lock()
	p.passwordChans[req.RequestID] = ch
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.passwordChans, req.RequestID)
		p.mu.Unlock()
	}()

	// Notify the frontend to show the password dialog
	runtime.EventsEmit(p.ctx, "request-cert-password", req)

	// Block until the password is submitted by the frontend
	select {
	case pass := <-ch:
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
	ctx           context.Context
	core          *app.App
	passwordChans map[string]chan string
	mu            sync.Mutex
	logLevel      *slog.LevelVar
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		passwordChans: make(map[string]chan string),
		logLevel:      new(slog.LevelVar),
	}
}

type wailsLogWriter struct {
	ctx context.Context
}

func (w wailsLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	runtime.LogPrint(w.ctx, msg)
	if w.ctx != nil {
		runtime.EventsEmit(w.ctx, "backend-log", msg)
	}
	return len(p), nil
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := app.LoadRuntimeEnv(); err != nil {
		fmt.Printf("failed to load runtime env: %v\n", err)
		return
	}

	verbose := os.Getenv("NANCI_VERBOSE") == "1"
	trace := os.Getenv("NANCI_TRACE") == "1"

	if trace {
		a.logLevel.Set(logpkg.LevelTrace)
	} else if verbose {
		a.logLevel.Set(slog.LevelDebug)
	} else {
		a.logLevel.Set(slog.LevelInfo)
	}

	wWriter := wailsLogWriter{ctx: ctx}
	handler := slog.NewTextHandler(wWriter, &slog.HandlerOptions{
		Level: a.logLevel,
	})
	log := slog.New(handler)

	coreApp, err := app.NewRuntime(app.RuntimeOptions{
		Log: log,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: WailsCredentialProvider{
				ctx:           ctx,
				passwordChans: a.passwordChans,
				mu:            &a.mu,
			},
		},
		RunMigrations: true,
	})
	if err != nil {
		fmt.Printf("failed to configure app: %v\n", err)
		return
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
func (a *App) SubmitCertPassword(reqID string, password string) {
	a.mu.Lock()
	ch, ok := a.passwordChans[reqID]
	a.mu.Unlock()

	if ok {
		select {
		case ch <- password:
		default:
		}
	}
}

// CancelCertPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (a *App) CancelCertPassword(reqID string) {
	a.mu.Lock()
	ch, ok := a.passwordChans[reqID]
	a.mu.Unlock()

	if ok {
		select {
		case ch <- "":
		default:
		}
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

func (a *App) ToggleDebug(enable bool) {
	if enable {
		a.logLevel.Set(slog.LevelDebug)
	} else {
		a.logLevel.Set(slog.LevelInfo)
	}
}

func (a *App) AddCompany(input app.AddCompanyInput) error {
	return a.core.AddCompany(a.ctx, input)
}

func (a *App) AddCredential(input app.AddCredentialInput) error {
	return a.core.AddCredential(a.ctx, input)
}

func (a *App) ListCredentials() ([]nfse.Credential, error) {
	return a.core.ListCredentials(a.ctx)
}

func (a *App) UpdateCredentialPath(input app.UpdateCredentialPathInput) error {
	return a.core.UpdateCredentialPath(a.ctx, input)
}

func (a *App) UpdateCredentialData(input app.UpdateCredentialDataInput) error {
	return a.core.UpdateCredentialData(a.ctx, input)
}

func (a *App) UpdateCompany(input app.UpdateCompanyInput) error {
	return a.core.UpdateCompany(a.ctx, input)
}

func (a *App) AssignCredentialToCompany(input app.AssignCredentialInput) error {
	return a.core.AssignCredentialToCompany(a.ctx, input)
}

func (a *App) ListCompanies() ([]nfse.Company, error) {
	return a.core.ListCompanies(a.ctx)
}

func (a *App) Pull(input app.PullInput) (app.PullResult, error) {
	return a.core.Pull(a.ctx, input)
}

func (a *App) QueryNFSe(input app.QueryNFSeInput) (string, error) {
	return a.core.QueryNFSe(a.ctx, input)
}

func (a *App) QueryNFSeEvents(input app.QueryNFSeInput) (string, error) {
	return a.core.QueryNFSeEvents(a.ctx, input)
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

func (a *App) ExportLogs() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config dir: %w", err)
	}
	logFile := filepath.Join(configDir, "Nanci", "app.log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return "", fmt.Errorf("arquivo de log não encontrado")
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exportar Logs",
		DefaultFilename: "nanci_debug_logs.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "Arquivos de Texto (*.txt)", Pattern: "*.txt"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	input, err := os.ReadFile(logFile)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(savePath, input, 0o644)
	if err != nil {
		return "", err
	}

	return savePath, nil
}
