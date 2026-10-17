// App is the struct bound to the Wails frontend. This file holds its fields,
// the startup that builds the core services, the readiness check every bound
// method runs, the shutdown, and the helpers shared by several areas.

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	dacterenderer "github.com/vasfvitor/nanci/internal/dacte/fiscalrenderer"
	danferenderer "github.com/vasfvitor/nanci/internal/danfe/fiscalrenderer"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// eventEmitter sends an event to the frontend. Tests replace wailsEvents so
// GetCertPassword runs without the Wails runtime.
type eventEmitter interface {
	Emit(ctx context.Context, name string, data ...any)
}

type wailsEvents struct{}

func (wailsEvents) Emit(ctx context.Context, name string, data ...any) {
	runtime.EventsEmit(ctx, name, data...)
}

// App struct
type App struct {
	ctx        context.Context
	svc        services
	cleanup    func()
	startupErr error // why startup failed; ready returns it
	cred       *WailsCredentialProvider
	logLevel   *slog.LevelVar
	logWriter  *rotatingFileWriter
	logPath    string
	guard      closeGuard
	events     eventEmitter
	quit       func(ctx context.Context) // runtime.Quit; tests replace it
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		cred:     newWailsCredentialProvider(),
		logLevel: new(slog.LevelVar),
		events:   wailsEvents{},
		quit:     runtime.Quit,
	}
}

// errNotStarted is returned by every method that needs the core while
// startup has failed or has not run.
var errNotStarted = errors.New("o nanci não iniciou")

// startup is called when the app starts. The context is saved
// so we can call the runtime methods. A failure is kept in startupErr and
// returned by every bound method that needs the core, so the frontend gets
// an error instead of a call that never settles.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cred.setCtx(ctx)

	svc, cleanup, err := a.initRuntime(ctx)
	if err != nil {
		a.startupErr = err
		fmt.Printf("failed to start: %v\n", err)
		return
	}
	a.svc = svc
	a.cleanup = cleanup
}

// initRuntime configures the logger, opens the database and builds the core
// services. The returned func closes the database.
func (a *App) initRuntime(ctx context.Context) (services, func(), error) {
	if err := app.LoadRuntimeEnv(); err != nil {
		return services{}, nil, fmt.Errorf("carregar configuração: %w", err)
	}

	trace := os.Getenv("NANCI_TRACE") == "1"
	a.logLevel.Set(resolveDesktopBaseLevel(trace))

	logDir, err := desktopLogDir()
	if err != nil {
		return services{}, nil, fmt.Errorf("configurar pasta de logs: %w", err)
	}
	a.logPath = filepath.Join(logDir, desktopLogFileName)

	log, writer, err := newDesktopLogger(ctx, a.logLevel, a.logPath)
	if err != nil {
		return services{}, nil, fmt.Errorf("configurar log: %w", err)
	}
	a.logWriter = writer

	dataDir, err := app.ResolveRuntimeDataDir("")
	if err != nil {
		return services{}, nil, fmt.Errorf("resolver pasta de dados: %w", err)
	}
	if err := paths.EnsureDir(dataDir); err != nil {
		return services{}, nil, fmt.Errorf("criar pasta de dados: %w", err)
	}

	db, err := store.OpenDB(ctx, app.RuntimeDBPath(dataDir), true)
	if err != nil {
		return services{}, nil, fmt.Errorf("abrir banco de dados: %w", err)
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
		DANFERenderer:  danferenderer.New(),
		DACTERenderer:  dacterenderer.New(),
	})
	if err != nil {
		_ = db.Close()
		return services{}, nil, fmt.Errorf("configurar aplicação: %w", err)
	}

	return servicesFrom(coreApp), func() { _ = db.Close() }, nil
}

// ready returns an error when the core services are not available: startup
// failed (the cause is wrapped) or never ran.
func (a *App) ready() error {
	if a.startupErr != nil {
		return fmt.Errorf("%w: %w", errNotStarted, a.startupErr)
	}
	if a.svc.companies == nil {
		return errNotStarted
	}
	return nil
}

func (a *App) shutdown(ctx context.Context) {
	if a.cleanup != nil {
		a.cleanup()
	}
	if a.logWriter != nil {
		_ = a.logWriter.Close()
	}
}

// validateImport refuses an import without a company or files before the
// core runs.
func validateImport(input desktopapi.ImportXMLInput) error {
	if input.CNPJ == "" {
		return errors.New("empresa não especificada")
	}
	if len(input.Files) == 0 {
		return errors.New("nenhum arquivo XML selecionado")
	}
	return nil
}
