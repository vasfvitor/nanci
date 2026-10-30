// App is the Wails service bound to the frontend. This file holds its fields,
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

	"github.com/wailsapp/wails/v3/pkg/application"

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
	Emit(name string, data ...any)
}

// wailsEvents emits through the Wails application main sets on App. Before
// that, and in tests without Wails, the event is dropped.
type wailsEvents struct{ app *App }

func (e wailsEvents) Emit(name string, data ...any) {
	if e.app.wails != nil {
		e.app.wails.Event.Emit(name, data...)
	}
}

// App struct
type App struct {
	ctx        context.Context
	wails      *application.App           // set by main before Run
	window     *application.WebviewWindow // the main window, parent of the dialogs
	svc        services
	cleanup    func()
	startupErr error // why startup failed; ready returns it
	cred       *WailsCredentialProvider
	logLevel   *slog.LevelVar
	logWriter  *rotatingFileWriter
	logPath    string
	guard      closeGuard
	events     eventEmitter
	quit       func() // asks Wails to quit; tests replace it
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{
		cred:     newWailsCredentialProvider(),
		logLevel: new(slog.LevelVar),
	}
	a.events = wailsEvents{app: a}
	a.cred.events = a.events
	a.quit = func() {
		if a.wails != nil {
			a.wails.Quit()
		}
	}
	return a
}

// errNotStarted is returned by every method that needs the core while
// startup has failed or has not run.
var errNotStarted = errors.New("o nanci não iniciou")

// ServiceStartup runs when Wails starts the App service. ctx lives until
// shutdown and bounds every core call. A failure is kept in startupErr and
// returned by every bound method that needs the core, so the frontend gets
// an error instead of a call that never settles. It never returns an error:
// that would stop Wails before the window opens.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx

	svc, cleanup, err := a.initRuntime(ctx)
	if err != nil {
		a.startupErr = err
		fmt.Printf("failed to start: %v\n", err)
		return nil
	}
	a.svc = svc
	a.cleanup = cleanup
	return nil
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

	log, writer, err := newDesktopLogger(a.emitLog, a.logLevel, a.logPath)
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

// ServiceShutdown runs when Wails stops the App service.
func (a *App) ServiceShutdown() error {
	if a.cleanup != nil {
		a.cleanup()
	}
	if a.logWriter != nil {
		_ = a.logWriter.Close()
	}
	return nil
}

// emitLog sends a desktop log record to the console in the frontend.
func (a *App) emitLog(event desktopLogEvent) {
	a.events.Emit("backend-log", event)
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
