package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vasfvitor/nanci/internal/app"
)

var (
	verbose bool
	version = "dev"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nanci",
	Short: "CLI para sincronização de XMLs de NFS-e Nacional",
	Long: `nanci (nfse-sync) sincroniza documentos fiscais da API ADN (NFS-e Nacional)
usando certificado digital A1. Suporta extração de retenções e relatórios.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Habilita log detalhado (debug)")
}

// initApp is a utility for commands that need the App instance
func initApp() (*app.App, error) {
	return app.NewApp(verbose)
}
