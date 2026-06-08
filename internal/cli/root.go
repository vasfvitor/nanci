package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
)

var (
	verbose bool
	trace   bool
)

// rootCmd is the base command when nanci is called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "nanci",
	Short: "CLI para sincronização de XMLs de NFS-e Nacional",
	Long: `nanci (nfse-sync) sincroniza documentos fiscais da API ADN (NFS-e Nacional)
usando certificado digital A1. Suporta extração de retenções e relatórios.`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute(ctx context.Context) int {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Habilita log detalhado (debug)")
	rootCmd.PersistentFlags().BoolVar(&trace, "trace", false, "Habilita log de rastreamento extremo (trace)")
}

// newApp is a helper for commands that need the App instance.
// It also injects the terminal-based CredentialProvider.
func newApp() (*app.App, error) {
	// If env var is set, it overrides the flag
	if os.Getenv("NANCI_TRACE") == "1" {
		trace = true
	}
	log := logger.New(verbose, trace)

	dataDir, err := paths.DataDir()
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver diretório de dados: %w", err)
	}

	if err := paths.EnsureDir(dataDir); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório de dados: %w", err)
	}

	dbPath := filepath.Join(dataDir, "nanci-v2.db")

	db, err := store.OpenDB(dbPath, true)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar banco de dados v2: %w", err)
	}

	application, err := app.New(app.Dependencies{
		Log:                log,
		DB:                 db,
		CompanyRepo:        store.NewCompanyRepository(db),
		CredentialRepo:     store.NewCredentialRepository(db),
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     store.NewDocumentRepository(db),
		XMLStore:           files.NewBlobStore(dataDir),
		DataDir:            dataDir,
		CredentialProvider: app.KeyringCredentialProvider{
			Fallback: TerminalCredentialProvider{In: os.Stdin, Out: os.Stderr},
			Log:      log,
		},
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configurar aplicação: %w", err)
	}
	return application, nil
}
