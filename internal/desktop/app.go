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
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction
type WailsCredentialProvider struct {
	ctx           context.Context
	passwordChans map[string]chan []byte
	mu            *sync.Mutex
}

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided
func (p WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) ([]byte, error) {
	ch := make(chan []byte, 1)

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
		if len(pass) == 0 {
			return nil, fmt.Errorf("operação cancelada pelo usuário")
		}
		return pass, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// App struct
type App struct {
	ctx           context.Context
	core          *app.App
	passwordChans map[string]chan []byte
	mu            sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		passwordChans: make(map[string]chan []byte),
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

	verbose := os.Getenv("NANCI_VERBOSE") == "1"
	trace := os.Getenv("NANCI_TRACE") == "1"
	
	level := slog.LevelInfo
	if trace {
		level = logpkg.LevelTrace
	} else if verbose {
		level = slog.LevelDebug
	}

	wWriter := wailsLogWriter{ctx: ctx}
	handler := slog.NewTextHandler(wWriter, &slog.HandlerOptions{
		Level: level,
	})
	log := slog.New(handler)

	dataDir, err := paths.DataDir()
	if err != nil {
		fmt.Printf("failed to resolve data dir: %v\n", err)
		return
	}

	if err := paths.EnsureDir(dataDir); err != nil {
		fmt.Printf("failed to create data dir: %v\n", err)
		return
	}

	dbPath := filepath.Join(dataDir, "nanci-v2.db")

	db, err := store.OpenDB(dbPath, true)
	if err != nil {
		fmt.Printf("failed to initialize db: %v\n", err)
		return
	}

	coreApp, err := app.New(app.Dependencies{
		Log:            log,
		DB:             db,
		CompanyRepo:    store.NewCompanyRepository(db),
		CredentialRepo: store.NewCredentialRepository(db),
		SyncRepo:       store.NewSyncRepository(db),
		DocumentReader: store.NewDocumentRepository(db),
		XMLStore:       files.NewBlobStore(dataDir),
		DataDir:        dataDir,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: WailsCredentialProvider{
				ctx:           ctx,
				passwordChans: a.passwordChans,
				mu:            &a.mu,
			},
			Log: log,
		},
	})
	if err != nil {
		_ = db.Close()
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
		passBytes := []byte(password)
		select {
		case ch <- passBytes:
		default:
			cert.ZeroBytes(passBytes)
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
	
	err = os.WriteFile(savePath, input, 0644)
	if err != nil {
		return "", err
	}

	return savePath, nil
}
