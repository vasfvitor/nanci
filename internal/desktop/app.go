package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction.
// It owns the passwordChans map and its mutex; the App delegates Submit/Cancel calls to it.
type WailsCredentialProvider struct {
	ctx           context.Context
	passwordChans map[string]chan string
	mu            sync.Mutex
}

func newWailsCredentialProvider() *WailsCredentialProvider {
	return &WailsCredentialProvider{
		passwordChans: make(map[string]chan string),
	}
}

func (p *WailsCredentialProvider) setCtx(ctx context.Context) { p.ctx = ctx }

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided
func (p *WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) (string, error) {
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
			return "", app.ErrOperationCanceled
		}
		return pass, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// SubmitPassword receives the password from the frontend dialog and unblocks GetCertPassword
func (p *WailsCredentialProvider) SubmitPassword(reqID, password string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if ok {
		select {
		case ch <- password:
		default:
		}
	}
}

// CancelPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (p *WailsCredentialProvider) CancelPassword(reqID string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if ok {
		select {
		case ch <- "":
		default:
		}
	}
}

// App struct
type App struct {
	ctx       context.Context
	core      *app.App
	cleanup   func()
	cred      *WailsCredentialProvider
	logLevel  *slog.LevelVar
	logWriter *rotatingFileWriter
	logPath   string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		cred:     newWailsCredentialProvider(),
		logLevel: new(slog.LevelVar),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cred.setCtx(ctx)

	if err := app.LoadRuntimeEnv(); err != nil {
		fmt.Printf("failed to load runtime env: %v\n", err)
		return
	}

	trace := os.Getenv("NANCI_TRACE") == "1"
	a.logLevel.Set(resolveDesktopBaseLevel(trace))

	logDir, err := desktopLogDir()
	if err != nil {
		fmt.Printf("failed to configure desktop log dir: %v\n", err)
		return
	}
	a.logPath = filepath.Join(logDir, desktopLogFileName)

	log, writer, err := newDesktopLogger(ctx, a.logLevel, a.logPath)
	if err != nil {
		fmt.Printf("failed to configure desktop logger: %v\n", err)
		return
	}
	a.logWriter = writer

	dataDir, err := app.ResolveRuntimeDataDir("")
	if err != nil {
		fmt.Printf("failed to resolve data dir: %v\n", err)
		return
	}
	if err := paths.EnsureDir(dataDir); err != nil {
		fmt.Printf("failed to create data dir: %v\n", err)
		return
	}

	db, err := store.OpenDB(ctx, app.RuntimeDBPath(dataDir), true)
	if err != nil {
		fmt.Printf("failed to initialize db: %v\n", err)
		return
	}

	a.cleanup = func() {
		_ = db.Close()
	}

	docRepo := store.NewDocumentRepository(db)

	coreApp, err := app.NewRuntime(app.Dependencies{
		Log:             log,
		CompanyRepo:     store.NewCompanyRepository(db),
		CredentialRepo:  store.NewCredentialRepository(db),
		SyncRepo:        store.NewSyncRepository(db),
		DocumentReader:  docRepo,
		DocumentTracker: docRepo,
		XMLStore:        files.NewBlobStore(dataDir),
		DataDir:         dataDir,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: a.cred,
		},
		DANFSeRenderer: godanfsev2.New(),
	})
	if err != nil {
		a.cleanup()
		fmt.Printf("failed to configure app: %v\n", err)
		return
	}

	a.core = coreApp
}

func (a *App) shutdown(ctx context.Context) {
	if a.cleanup != nil {
		a.cleanup()
	}
	if a.logWriter != nil {
		_ = a.logWriter.Close()
	}
}

// --- Auth & Credentials ---

// SubmitCertPassword receives the password from the frontend dialog and unblocks GetCertPassword
func (a *App) SubmitCertPassword(reqID string, password string) {
	a.cred.SubmitPassword(reqID, password)
}

// CancelCertPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (a *App) CancelCertPassword(reqID string) {
	a.cred.CancelPassword(reqID)
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

// SelectSaveFile opens a dialog to select an output file path for exports
func (a *App) SelectSaveFile(title, defaultFilename, pattern string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: title, Pattern: pattern},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
}

// --- Core API Exposure ---

func (a *App) SetLogLevel(level string) {
	a.logLevel.Set(parseDesktopLogLevel(level))
}

func (a *App) AddCompany(input desktopapi.AddCompanyInput) error {
	environment, err := nfse.ParseEnvironment(input.Environment)
	if err != nil {
		return err
	}
	policy, date, err := app.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
	if err != nil {
		return err
	}

	return a.core.AddCompany(a.ctx, app.AddCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		CredentialID:    input.CredentialID,
		CredentialLabel: input.CredentialLabel,
		CertPath:        input.CertPath,
		Environment:     environment,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AddCredential(input desktopapi.AddCredentialInput) error {
	return a.core.AddCredential(a.ctx, app.AddCredentialInput{
		Label:    input.Label,
		CertPath: input.CertPath,
	})
}

func (a *App) ListCredentials() ([]desktopapi.CredentialSummary, error) {
	credentials, err := a.core.ListCredentials(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CredentialSummaries(credentials), nil
}

func (a *App) UpdateCredentialPath(input desktopapi.UpdateCredentialPathInput) error {
	return a.core.UpdateCredentialPath(a.ctx, app.UpdateCredentialPathInput{
		CredentialID: input.CredentialID,
		CertPath:     input.CertPath,
	})
}

func (a *App) UpdateCredentialData(input desktopapi.UpdateCredentialDataInput) error {
	return a.core.UpdateCredentialData(a.ctx, app.UpdateCredentialDataInput{
		CredentialID: input.CredentialID,
		Label:        input.Label,
	})
}

func (a *App) UpdateCompany(input desktopapi.UpdateCompanyInput) error {
	environment, err := nfse.ParseEnvironment(input.Environment)
	if err != nil {
		return err
	}
	policy := nfse.SyncStartPolicyFromNow
	var date *time.Time
	if input.SyncStartPolicy != "" {
		policy, date, err = app.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
		if err != nil {
			return err
		}
	}

	return a.core.UpdateCompany(a.ctx, app.UpdateCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		Environment:     environment,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AssignCredentialToCompany(input desktopapi.AssignCredentialInput) error {
	return a.core.AssignCredentialToCompany(a.ctx, app.AssignCredentialInput{
		CompanyCNPJ:  input.CompanyCNPJ,
		CredentialID: input.CredentialID,
	})
}

func (a *App) ListCompanies() ([]desktopapi.CompanySummary, error) {
	companies, err := a.core.ListCompanies(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CompanySummaries(companies), nil
}

func (a *App) Pull(input desktopapi.PullInput) (desktopapi.PullResult, error) {
	res, err := a.core.Pull(a.ctx, app.PullInput{
		CNPJ: input.CNPJ,
		Mode: input.Mode,
	})
	if err != nil && errors.Is(err, app.ErrOperationCanceled) {
		return desktopapi.PullResult{}, fmt.Errorf("ERR_CANCELED: %w", err)
	}
	return desktopapi.PullResult{
		CompanyName:              res.CompanyName,
		CNPJ:                     res.CNPJ,
		CredentialLabel:          res.CredentialLabel,
		CredentialCNPJ:           res.CredentialCNPJ,
		ConsultationBasis:        res.ConsultationBasis,
		Status:                   res.Status,
		StopReason:               res.StopReason,
		LastProcessedNSU:         res.LastProcessedNSU,
		LastFoundNSU:             res.LastFoundNSU,
		EmptyStreak:              res.EmptyStreak,
		DocumentsFound:           res.DocumentsFound,
		EventsFound:              res.EventsFound,
		DocumentsSaved:           res.DocumentsSaved,
		EventsSaved:              res.EventsSaved,
		DocumentsSkippedByPolicy: res.DocumentsSkippedByPolicy,
		EventsSkippedByPolicy:    res.EventsSkippedByPolicy,
		Errors:                   res.Errors,
		Duration:                 res.Duration,
	}, err
}

func (a *App) ResetSyncState(input desktopapi.ResetSyncInput) error {
	return a.core.ResetSyncState(a.ctx, app.ResetSyncInput{
		CNPJ: input.CompanyCNPJ,
	})
}

func (a *App) QueryNFSeEvents(input desktopapi.QueryNFSeInput) (string, error) {
	return a.core.QueryNFSeEvents(a.ctx, app.QueryNFSeInput{
		CNPJ:        input.CompanyCNPJ,
		ChaveAcesso: input.ChaveAcesso,
	})
}

func (a *App) ListDocuments(input desktopapi.ListInput) ([]desktopapi.DocumentRow, error) {
	documents, err := a.core.ListDocuments(a.ctx, app.ListInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.DocumentRows(documents), nil
}

func (a *App) ListEventsForDocument(documentID string) ([]desktopapi.DocumentEvent, error) {
	events, err := a.core.ListEventsForDocument(a.ctx, documentID)
	if err != nil {
		return nil, err
	}
	return desktopapi.DocumentEvents(events), nil
}

func (a *App) Status(cnpj string) (desktopapi.StatusResult, error) {
	res, err := a.core.Status(a.ctx, cnpj)
	if err != nil {
		return desktopapi.StatusResult{}, err
	}
	return desktopapi.StatusResult{
		CompanyName:        res.CompanyName,
		CNPJ:               res.CNPJ,
		Environment:        res.Environment,
		ConsultationCNPJ:   res.ConsultationCNPJ,
		CredentialCNPJ:     res.CredentialCNPJ,
		CredentialNotAfter: res.CredentialNotAfter,
		LastProcessedNSU:   res.LastProcessedNSU,
		LastFoundNSU:       res.LastFoundNSU,
		LastSyncAt:         res.LastSyncAt,
		LastRunStatus:      res.LastRunStatus,
		LastRunStopReason:  res.LastRunStopReason,
		TotalEmitidas:      res.TotalEmitidas,
		TotalTomadas:       res.TotalTomadas,
	}, nil
}

func (a *App) ExportDANFSe(input desktopapi.ExportDANFSeInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err := a.core.ExportDANFSe(a.ctx, app.ExportDANFSeInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "danfse"}, nil
}

func (a *App) ExportXML(input desktopapi.ExportXMLInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err := a.core.ExportXML(a.ctx, app.ExportXMLInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "xml"}, nil
}

func (a *App) ExportDANFSeZIP(input desktopapi.ExportDocumentsInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	exportInput := app.ExportInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Direction:    input.Direction,
		OutPath:      input.OutPath,
		Incremental:  input.Incremental,
		ChavesAcesso: input.ChavesAcesso,
	}

	res, err := a.core.ExportDANFSeZIP(a.ctx, exportInput)
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{
		OutPath:       res.OutPath,
		Format:        res.Format,
		Incremental:   res.Incremental,
		ExportedCount: res.ExportedCount,
	}, nil
}

func (a *App) ExportDocuments(input desktopapi.ExportDocumentsInput) (desktopapi.ExportResult, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	exportInput := app.ExportInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Direction:    input.Direction,
		OutPath:      input.OutPath,
		Incremental:  input.Incremental,
		ChavesAcesso: input.ChavesAcesso,
	}

	var res app.ExportResult
	var err error
	switch format {
	case "csv":
		res, err = a.core.ExportCSV(a.ctx, exportInput)
	case "xlsx":
		res, err = a.core.ExportXLSX(a.ctx, exportInput)
	case "zip":
		res, err = a.core.ExportZIP(a.ctx, exportInput)
	default:
		return desktopapi.ExportResult{}, fmt.Errorf("formato de exportação desconhecido: %s", format)
	}

	if err != nil {
		return desktopapi.ExportResult{}, err
	}

	return desktopapi.ExportResult{
		OutPath:       res.OutPath,
		Format:        res.Format,
		Incremental:   res.Incremental,
		ExportedCount: res.ExportedCount,
	}, nil
}

func (a *App) CountPendingExports(input desktopapi.ExportDocumentsInput) (int, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "zip" {
		format = "xml"
	} else if format == "danfse-zip" {
		format = "danfse"
	}

	exportInput := app.ExportInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
	}
	return a.core.CountPendingExportDocuments(a.ctx, exportInput, format)
}

func (a *App) MarkDocumentsViewed(input desktopapi.ListInput) (int, error) {
	return a.core.MarkDocumentsViewed(a.ctx, app.ListInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	})
}

func formatExportError(err error) error {
	return err
}

func parseDesktopLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return logpkg.LevelTrace
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

func exportExtension(format string) (string, error) {
	switch format {
	case "csv":
		return ".csv", nil
	case "xlsx":
		return ".xlsx", nil
	case "zip":
		return ".zip", nil
	default:
		return "", fmt.Errorf("formato de exportação inválido: %s", format)
	}
}

func (a *App) ExportLogs() (string, error) {
	if a.logPath == "" {
		return "", fmt.Errorf("logger de desktop não configurado")
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exportar Logs",
		DefaultFilename: "nanci_desktop_logs.zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "Arquivos ZIP (*.zip)", Pattern: "*.zip"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	return savePath, exportRotatedLogs(savePath, a.logPath)
}

func exportRotatedLogs(savePath string, basePath string) error {
	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("criar arquivo de exportação: %w", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)

	added := 0
	for _, path := range collectRotatedLogPaths(basePath, logFileMaxBackups) {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat log %s: %w", path, err)
		}
		if info.Size() == 0 {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("ler log %s: %w", path, err)
		}

		entry, err := archive.Create(filepath.Base(path))
		if err != nil {
			return fmt.Errorf("criar entrada zip %s: %w", path, err)
		}
		if _, err := entry.Write(content); err != nil {
			return fmt.Errorf("escrever entrada zip %s: %w", path, err)
		}
		added++
	}

	if added == 0 {
		return fmt.Errorf("arquivo de log não encontrado")
	}

	if err := archive.Close(); err != nil {
		return fmt.Errorf("fechar zip: %w", err)
	}
	return nil
}

func (a *App) GetBuildInfo() desktopapi.BuildInfo {
	return desktopapi.BuildInfo{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		Date:    buildinfo.Date,
	}
}

func (a *App) GetDataDirectory() (string, error) {
	return paths.DataDir()
}

func openDir(dir string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

func (a *App) OpenDataDirectory() error {
	dir, err := paths.DataDir()
	if err != nil {
		return err
	}
	return openDir(dir)
}

func (a *App) OpenLogsDirectory() error {
	dir, err := desktopLogDir()
	if err != nil {
		return err
	}
	return openDir(dir)
}

func (a *App) TestConnection(companyCNPJ string) (desktopapi.ConnectionTestResult, error) {
	res, err := a.core.TestConnection(a.ctx, companyCNPJ)
	if err != nil {
		return desktopapi.ConnectionTestResult{}, err
	}
	return desktopapi.ConnectionTestResult{
		CertLoaded:        res.CertLoaded,
		CertSubject:       res.CertSubject,
		CertExpiration:    res.CertExpiration,
		MTLSAccepted:      res.MTLSAccepted,
		EndpointReached:   res.EndpointReached,
		ResponseCode:      res.ResponseCode,
		ResponseDetail:    res.ResponseDetail,
		StatusExplanation: res.StatusExplanation,
	}, nil
}
