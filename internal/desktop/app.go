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
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction.
// It owns the passwordChans map and its mutex; the App delegates Submit/Cancel calls to it.
type WailsCredentialProvider struct {
	ctx           context.Context
	passwordChans map[string]chan []byte
	mu            sync.Mutex
}

func newWailsCredentialProvider() *WailsCredentialProvider {
	return &WailsCredentialProvider{
		passwordChans: make(map[string]chan []byte),
	}
}

func (p *WailsCredentialProvider) setCtx(ctx context.Context) { p.ctx = ctx }

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided.
// The caller owns the returned slice and should zero it with cert.ZeroBytes once done.
func (p *WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) ([]byte, error) {
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
	runtime.EventsEmit(p.ctx, "request-cert-password", req) //nolint:contextcheck // Wails runtime calls need the app context from startup; ctx only bounds the wait.

	// Block until the password is submitted by the frontend
	select {
	case pass := <-ch:
		// CancelPassword sends nil; an empty submission means the same thing.
		if len(pass) == 0 {
			return nil, app.ErrOperationCanceled
		}
		return pass, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubmitPassword receives the password from the frontend dialog and unblocks GetCertPassword.
// Wails marshals the password from JS as a string, so the copy taken here is the
// first one that can be zeroed; the string itself cannot be.
func (p *WailsCredentialProvider) SubmitPassword(reqID, password string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if !ok {
		return
	}

	pass := []byte(password)
	select {
	case ch <- pass:
	default:
		// Nobody is waiting any more; do not leave the password in memory.
		cert.ZeroBytes(pass)
	}
}

// CancelPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (p *WailsCredentialProvider) CancelPassword(reqID string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if !ok {
		return
	}

	select {
	case ch <- nil:
	default:
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
		CompanyStore:    company.NewStore(db),
		CredentialStore: credential.NewStore(db),
		SyncRepo:        nsync.NewStore(db),
		DocumentRepo:    docRepo,
		NFeRepo:         store.NewNFeRepository(db),
		CTeRepo:         store.NewCTeRepository(db),
		XMLStore:        files.NewBlobStore(dataDir),
		DataDir:         dataDir,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: a.cred,
			Log:      log,
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
	policy, date, err := company.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
	if err != nil {
		return err
	}

	return a.core.Companies.AddCompany(a.ctx, company.AddCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		CredentialID:    input.CredentialID,
		CredentialLabel: input.CredentialLabel,
		CertPath:        input.CertPath,
		Environment:     environment,
		UF:              input.UF,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AddCredential(input desktopapi.AddCredentialInput) error {
	return a.core.Credentials.AddCredential(a.ctx, credential.AddCredentialInput{
		Label:    input.Label,
		CertPath: input.CertPath,
	})
}

func (a *App) ListCredentials() ([]desktopapi.CredentialSummary, error) {
	credentials, err := a.core.Credentials.ListCredentials(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CredentialSummaries(credentials), nil
}

func (a *App) UpdateCredentialPath(input desktopapi.UpdateCredentialPathInput) error {
	return a.core.Credentials.UpdateCredentialPath(a.ctx, credential.UpdateCredentialPathInput{
		CredentialID: input.CredentialID,
		CertPath:     input.CertPath,
	})
}

func (a *App) UpdateCredentialData(input desktopapi.UpdateCredentialDataInput) error {
	return a.core.Credentials.UpdateCredentialData(a.ctx, credential.UpdateCredentialDataInput{
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
		policy, date, err = company.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
		if err != nil {
			return err
		}
	}

	return a.core.Companies.UpdateCompany(a.ctx, company.UpdateCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		Environment:     environment,
		UF:              input.UF,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AssignCredentialToCompany(input desktopapi.AssignCredentialInput) error {
	return a.core.Companies.AssignCredentialToCompany(a.ctx, company.AssignCredentialInput{
		CompanyCNPJ:  input.CompanyCNPJ,
		CredentialID: input.CredentialID,
	})
}

func (a *App) ListCompanies() ([]desktopapi.CompanySummary, error) {
	companies, err := a.core.Companies.ListCompanies(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CompanySummaries(companies), nil
}

func (a *App) Pull(input desktopapi.PullInput) (desktopapi.PullResult, error) {
	res, err := a.core.SyncManager.Pull(a.ctx, nsync.PullInput{
		CNPJ:   input.CNPJ,
		Mode:   input.Mode,
		Source: nfse.SyncSourceNFSe,
	})
	if err != nil {
		return desktopapi.PullResult{}, err
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
	}, nil
}

func (a *App) ResetSyncState(input desktopapi.ResetSyncInput) error {
	return a.core.SyncManager.ResetSyncState(a.ctx, nsync.ResetSyncInput{
		CNPJ:   input.CompanyCNPJ,
		Source: nfse.SyncSourceNFSe,
	})
}

func (a *App) QueryNFSeEvents(input desktopapi.QueryNFSeInput) (string, error) {
	return a.core.Query.QueryNFSeEvents(a.ctx, app.QueryNFSeInput{
		CNPJ:        input.CompanyCNPJ,
		ChaveAcesso: input.ChaveAcesso,
	})
}

func (a *App) ListDocuments(input desktopapi.ListInput) ([]desktopapi.DocumentRow, error) {
	documents, err := a.core.Documents.ListDocuments(a.ctx, app.ListInput{
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
	events, err := a.core.Documents.ListEventsForDocument(a.ctx, documentID)
	if err != nil {
		return nil, err
	}
	return desktopapi.DocumentEvents(events), nil
}

func (a *App) Status(cnpj string) (desktopapi.StatusResult, error) {
	res, err := a.core.SyncManager.Status(a.ctx, cnpj)
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

	err := a.core.Exports.ExportDANFSe(a.ctx, app.ExportDANFSeInput{
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

	err := a.core.Exports.ExportXML(a.ctx, app.ExportXMLInput{
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

	res, err := a.core.Exports.ExportDANFSeZIP(a.ctx, exportInput)
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult(res), nil
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
		res, err = a.core.Exports.ExportCSV(a.ctx, exportInput)
	case "xlsx":
		res, err = a.core.Exports.ExportXLSX(a.ctx, exportInput)
	case "zip":
		res, err = a.core.Exports.ExportZIP(a.ctx, exportInput)
	default:
		return desktopapi.ExportResult{}, fmt.Errorf("formato de exportação desconhecido: %s", format)
	}

	if err != nil {
		return desktopapi.ExportResult{}, err
	}

	return desktopapi.ExportResult(res), nil
}

func (a *App) CountPendingExports(input desktopapi.ExportDocumentsInput) (int, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	switch format {
	case "zip":
		format = "xml"
	case "danfse-zip":
		format = "danfse"
	}

	exportInput := app.ExportInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
	}
	return a.core.Exports.CountPendingExportDocuments(a.ctx, exportInput, format)
}

func (a *App) MarkDocumentsViewed(input desktopapi.ListInput) (int, error) {
	return a.core.Documents.MarkDocumentsViewed(a.ctx, app.ListInput{
		CNPJ:       input.CNPJ,
		Competence: input.Competence,
		Direction:  input.Direction,
		OnlyUnread: input.OnlyUnread,
	})
}

func formatExportError(err error) error {
	if errors.Is(err, files.ErrBlobNotFound) {
		return fmt.Errorf("%w. Dica: o XML do documento não foi encontrado no disco. Isso pode ocorrer se o arquivo foi apagado manualmente ou se a sincronização não baixou o XML corretamente. Resetar NSU nas configurações da empresa pode forçar o download novamente", err)
	}
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
	file, err := os.Create(savePath) // #nosec G304 -- the user picks savePath in the save dialog.
	if err != nil {
		return fmt.Errorf("criar arquivo de exportação: %w", err)
	}
	defer func() { _ = file.Close() }()

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

		content, err := os.ReadFile(path) // #nosec G304 -- path is one of the app's own rotated log files.
		if err != nil {
			return fmt.Errorf("ler log %s: %w", path, err)
		}

		entry, err := archive.Create(filepath.Base(path))
		if err != nil {
			return fmt.Errorf("criar entrada zip %s: %w", path, err)
		}
		// Logs on disk keep CNPJs in clear; the exported copy is masked.
		if _, err := entry.Write(sanitizeLogContent(content)); err != nil {
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
	opener := "xdg-open" // linux, freebsd, etc.
	switch goruntime.GOOS {
	case "windows":
		opener = "explorer"
	case "darwin":
		opener = "open"
	}
	// #nosec G204 -- fixed opener per OS; dir is the app's data or log directory, passed without a shell.
	cmd := exec.Command(opener, dir) //nolint:noctx // fire-and-forget launch of the file browser; nothing to cancel.
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
	res, err := a.core.Query.TestConnection(a.ctx, companyCNPJ)
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

// --- NF-e ---

func (a *App) PullNFe(input desktopapi.PullNFeInput) (desktopapi.PullNFeResult, error) {
	res, err := a.core.NFe.Pull(a.ctx, input.CNPJ)
	if err != nil {
		return desktopapi.PullNFeResult{}, err
	}
	return desktopapi.PullNFeResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		Status:           res.Status,
		StopReason:       res.StopReason,
		LastNSU:          res.LastNSU,
		MaxNSU:           res.MaxNSU,
		CompletasSaved:   res.CompletasSaved,
		ResumosSaved:     res.ResumosSaved,
		EventsSaved:      res.EventsSaved,
		Errors:           res.Errors,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
		Duration:         res.Duration,
	}, nil
}

func (a *App) StatusNFe(cnpj string) (desktopapi.NFeStatusResult, error) {
	res, err := a.core.NFe.Status(a.ctx, cnpj)
	if err != nil {
		return desktopapi.NFeStatusResult{}, err
	}
	return desktopapi.NFeStatusResult{
		CompanyName:       res.CompanyName,
		CNPJ:              res.CNPJ,
		UF:                res.UF,
		TpAmb:             res.TpAmb,
		LastNSU:           res.LastNSU,
		MaxNSU:            res.MaxNSU,
		LastSyncAt:        res.LastSyncAt,
		LastRunStatus:     res.LastRunStatus,
		LastRunStopReason: res.LastRunStopReason,
		InitialSyncDoneAt: res.InitialSyncDoneAt,
		NextAllowedAt:     res.NextAllowedAt,
		BlockedReason:     res.BlockedReason,
		RequestsLastHour:  res.RequestsLastHour,
		RequestBudget:     res.RequestBudget,
		TotalDestinatario: res.TotalDestinatario,
		TotalEmitente:     res.TotalEmitente,
		TotalOutros:       res.TotalOutros,
		TotalResumos:      res.TotalResumos,
		TotalCompletas:    res.TotalCompletas,
		PendingCiencia:    res.PendingCiencia,
		PendingConclusiva: res.PendingConclusiva,
		CienciaOverdue:    res.CienciaOverdue,
	}, nil
}

func (a *App) ListNFe(input desktopapi.ListNFeInput) ([]desktopapi.NFeRow, error) {
	documents, err := a.core.NFe.ListDocuments(a.ctx, app.NFeListInput{
		CNPJ:         input.CNPJ,
		Competence:   input.Competence,
		Situacao:     input.Situacao,
		Completeness: input.Completeness,
		Role:         input.Role,
		Manifestacao: input.Manifestacao,
		EmitenteCNPJ: input.EmitenteCNPJ,
		ChavesAcesso: input.ChavesAcesso,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.NFeRows(documents), nil
}

func (a *App) ListNFeEvents(input desktopapi.NFeKeyInput) ([]desktopapi.NFeEvent, error) {
	events, err := a.core.NFe.ListEvents(a.ctx, input.CNPJ, input.ChaveAcesso)
	if err != nil {
		return nil, err
	}
	return desktopapi.NFeEvents(events), nil
}

func (a *App) ListNFePendingManifestacoes(input desktopapi.NFePendingInput) ([]desktopapi.NFePendingRow, error) {
	pending, err := a.core.NFe.ListPendingManifestacoes(a.ctx, app.NFePendingInput{
		CNPJ:          input.CNPJ,
		DueWithinDays: input.DueWithinDays,
	})
	if err != nil {
		return nil, err
	}
	return desktopapi.NFePendingRows(pending), nil
}

// PlanNFeCiencia lists which NF-e RegisterNFeCiencia would send and which it would
// skip. It sends nothing and asks for no password.
func (a *App) PlanNFeCiencia(input desktopapi.RegisterNFeCienciaInput) (desktopapi.NFeCienciaPlan, error) {
	plan, err := a.core.NFe.PlanCiencia(a.ctx, app.NFeCienciaInput{
		CNPJ:         input.CNPJ,
		ChavesAcesso: input.ChavesAcesso,
	})
	if err != nil {
		return desktopapi.NFeCienciaPlan{}, err
	}
	return desktopapi.NFeCienciaPlanFrom(plan), nil
}

// RegisterNFeCiencia sends Ciência da Operação for the eligible NF-e. Failures
// after sending started are reported per chave in the result, not as an error.
func (a *App) RegisterNFeCiencia(input desktopapi.RegisterNFeCienciaInput) (desktopapi.NFeEventBatchResult, error) {
	summary, err := a.core.NFe.RegisterCiencia(a.ctx, app.NFeCienciaInput{
		CNPJ:         input.CNPJ,
		ChavesAcesso: input.ChavesAcesso,
	})
	if err != nil {
		return desktopapi.NFeEventBatchResult{}, err
	}
	return desktopapi.NFeEventResults(summary), nil
}

func (a *App) RegisterNFeManifestacao(input desktopapi.RegisterNFeManifestacaoInput) (desktopapi.NFeEventResult, error) {
	outcome, err := a.core.NFe.RegisterManifestacao(a.ctx, app.NFeManifestacaoInput{
		CNPJ:          input.CNPJ,
		ChaveAcesso:   input.ChaveAcesso,
		Tipo:          input.Tipo,
		Justificativa: input.Justificativa,
	})
	if err != nil {
		return desktopapi.NFeEventResult{}, err
	}
	return desktopapi.NFeEventResultFrom(outcome), nil
}

func (a *App) ExportNFeXML(input desktopapi.ExportNFeXMLInput) (desktopapi.ExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.ExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	err := a.core.NFe.ExportXML(a.ctx, app.NFeExportXMLInput{
		CNPJ:        input.CNPJ,
		ChaveAcesso: input.ChaveAcesso,
		OutPath:     input.OutPath,
	})
	if err != nil {
		return desktopapi.ExportResult{}, err
	}
	return desktopapi.ExportResult{OutPath: input.OutPath, Format: "xml"}, nil
}

func (a *App) ExportNFeZIP(input desktopapi.ExportNFeZIPInput) (desktopapi.NFeExportResult, error) {
	if input.OutPath == "" {
		return desktopapi.NFeExportResult{}, fmt.Errorf("caminho de saída não especificado")
	}

	res, err := a.core.NFe.ExportXMLZip(a.ctx, app.NFeExportInput{
		CNPJ:           input.CNPJ,
		Competence:     input.Competence,
		Role:           input.Role,
		ChavesAcesso:   input.ChavesAcesso,
		IncludeResumos: input.IncludeResumos,
		Incremental:    input.Incremental,
		OutPath:        input.OutPath,
	})
	if err != nil {
		return desktopapi.NFeExportResult{}, err
	}
	return desktopapi.NFeExportResult{
		ExportResult:   desktopapi.ExportResult(res.ExportResult),
		SkippedResumos: res.SkippedResumos,
	}, nil
}

// ResetNFe removes the company's NF-e and resets its NF-e sync, which lets the
// company environment change again.
func (a *App) ResetNFe(cnpj string) (desktopapi.NFeResetResult, error) {
	res, err := a.core.NFe.Reset(a.ctx, cnpj)
	if err != nil {
		return desktopapi.NFeResetResult{}, err
	}
	return desktopapi.NFeResetResult{
		CompanyName:       res.CompanyName,
		CNPJ:              res.CNPJ,
		CompanyDocuments:  res.CompanyDocuments,
		Documents:         res.Documents,
		Events:            res.Events,
		ExportMarks:       res.ExportMarks,
		ManifestacoesKept: res.ManifestacoesKept,
	}, nil
}
