package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
)

// prodAppFactory returns the production AppFactory: it opens a real SQLite
// database, wires the repositories, the blob store, the keyring-based
// credential provider (with a terminal fallback), and the DANFSe renderer.
//
// verbose and trace route into the logger. stdin/stderr are *os.File
// because TerminalCredentialProvider passes them to golang.org/x/term,
// which requires Fd(); stdout is an io.Writer for general-purpose routing.
func prodAppFactory(verbose, trace bool, stdin, stderr *os.File, stdout io.Writer) AppFactory {
	return func(ctx context.Context) (*app.App, func(), error) {
		if err := app.LoadRuntimeEnv(); err != nil {
			return nil, nil, fmt.Errorf("falha ao carregar runtime: %w", err)
		}

		if os.Getenv("NANCI_TRACE") == "1" {
			trace = true
		}
		log := logger.New(verbose, trace)

		dataDir, err := app.ResolveRuntimeDataDir("")
		if err != nil {
			return nil, nil, err
		}
		if err := paths.EnsureDir(dataDir); err != nil {
			return nil, nil, fmt.Errorf("criar diretório de dados: %w", err)
		}

		db, err := store.OpenDB(ctx, app.RuntimeDBPath(dataDir), true)
		if err != nil {
			return nil, nil, fmt.Errorf("inicializar banco de dados: %w", err)
		}

		cleanup := func() {
			_ = db.Close()
		}

		docRepo := store.NewDocumentRepository(db)

		application, err := app.NewRuntime(app.Dependencies{
			Log:             log,
			CompanyRepo:     store.NewCompanyRepository(db),
			CredentialRepo:  store.NewCredentialRepository(db),
			SyncRepo:        store.NewSyncRepository(db),
			DocumentReader:  docRepo,
			DocumentTracker: docRepo,
			XMLStore:        files.NewBlobStore(dataDir),
			DataDir:         dataDir,
			CredentialProvider: app.KeyringCredentialProvider{
				Fallback: TerminalCredentialProvider{In: stdin, Out: stderr},
			},
			DANFSeRenderer: godanfsev2.New(),
		})
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("configurar aplicação: %w", err)
		}
		return application, cleanup, nil
	}
}
