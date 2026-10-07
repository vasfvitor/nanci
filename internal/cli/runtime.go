package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/danfse/godanfsev2"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// prodAppFactory returns the production AppFactory: it opens a real SQLite
// database, wires the repositories, the blob store, the keyring-based
// credential provider (with a terminal fallback), and the DANFSe renderer.
//
// The terminal password prompt reads env.In and writes to env.Out (stderr),
// so stdout stays clean for piping.
func prodAppFactory(env CommandEnv) AppFactory {
	return func(ctx context.Context) (*app.App, func(), error) {
		if err := app.LoadRuntimeEnv(); err != nil {
			return nil, nil, fmt.Errorf("falha ao carregar runtime: %w", err)
		}

		log := logger.New(logLevels(env))

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
			CompanyStore:    company.NewStore(db),
			CredentialStore: credential.NewStore(db),
			SyncRepo:        sync.NewStore(db),
			DocumentRepo:    docRepo,
			NFeRepo:         store.NewNFeRepository(db),
			CTeRepo:         store.NewCTeRepository(db),

			XMLStore: files.NewBlobStore(dataDir),
			DataDir:  dataDir,
			CredentialProvider: app.KeyringCredentialProvider{
				Fallback: terminalPasswords(env),
				Log:      log,
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

// logLevels reads the --verbose and --trace flags when the factory runs,
// after Cobra has parsed them into env.Verbose and env.Trace. NANCI_TRACE=1
// also turns trace on; either source is enough and neither turns it off.
func logLevels(env CommandEnv) (verbose, trace bool) {
	return *env.Verbose, *env.Trace || os.Getenv("NANCI_TRACE") == "1"
}

// terminalPasswords builds the password prompt from the env's streams: it
// reads from In and writes the prompt to Out, which is stderr in production.
func terminalPasswords(env CommandEnv) TerminalCredentialProvider {
	return TerminalCredentialProvider{In: env.In, Out: env.Out}
}
