package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand builds a fresh *cobra.Command tree from the given env.
// A fresh tree is built per Execute call so that repeated invocations
// (and tests) don't share cobra state.
func NewRootCommand(env CommandEnv) *cobra.Command {
	root := &cobra.Command{
		Use:   "nanci",
		Short: "CLI para sincronização de XMLs de NFS-e Nacional, NF-e e CT-e",
		Long: `nanci (nfse-sync) sincroniza documentos fiscais da API ADN (NFS-e Nacional)
usando certificado digital A1. Suporta extração de retenções e relatórios.

Com "nanci nfe", baixa NF-e (modelo 55) da distribuição DF-e da SEFAZ e
registra a manifestação do destinatário.

Com "nanci cte", baixa CT-e, CT-e OS e GTV-e (modelos 57, 67 e 64) da
distribuição DF-e da SEFAZ.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		DisableSuggestions:    true,
	}
	root.SetOut(env.Stdout)
	root.SetErr(env.Out)
	root.PersistentFlags().BoolVarP(env.Verbose, "verbose", "v", false, "Habilita log detalhado (debug)")
	root.PersistentFlags().BoolVar(env.Trace, "trace", false, "Habilita log de rastreamento extremo (trace)")

	root.AddCommand(newInitCommand(env))
	root.AddCommand(newCompanyCommand(env))
	root.AddCommand(newCredentialCommand(env))
	root.AddCommand(newCTeCommand(env))
	root.AddCommand(newExportCommand(env))
	root.AddCommand(newListCommand(env))
	root.AddCommand(newNFeCommand(env))
	root.AddCommand(newPullCommand(env))
	root.AddCommand(newStatusCommand(env))
	return root
}
