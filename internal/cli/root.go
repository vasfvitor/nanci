package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/app"
)

var (
	verbose bool
)

// rootCmd is the base command when nanci is called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "nanci",
	Short: "CLI para sincronização de XMLs de NFS-e Nacional",
	Long: `nanci (nfse-sync) sincroniza documentos fiscais da API ADN (NFS-e Nacional)
usando certificado digital A1. Suporta extração de retenções e relatórios.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Habilita log detalhado (debug)")
}

// newApp is a helper for commands that need the App instance.
// It also injects the terminal-based CredentialProvider.
func newApp() (*app.App, error) {
	application, err := app.NewApp(verbose)
	if err != nil {
		return nil, err
	}
	application.CredentialProvider = TerminalCredentialProvider{
		In:  os.Stdin,
		Out: os.Stderr,
	}
	return application, nil
}
