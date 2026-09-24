package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/cte"
)

// newCTeCommand builds the `cte` subcommand tree. Every leaf acts on the
// company given by the persistent --cnpj flag.
func newCTeCommand(env CommandEnv) *cobra.Command {
	var cnpjFlag string
	cteCmd := &cobra.Command{
		Use:   "cte",
		Short: "Sincroniza, lista, exporta e redefine CT-e (modelos 57, 64 e 67) da SEFAZ",
	}
	cteCmd.PersistentFlags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	_ = cteCmd.MarkPersistentFlagRequired("cnpj")

	cteCmd.AddCommand(newCTePullCmd(env, &cnpjFlag))
	cteCmd.AddCommand(newCTeStatusCmd(env, &cnpjFlag))
	cteCmd.AddCommand(newCTeListCmd(env, &cnpjFlag))
	cteCmd.AddCommand(newCTeExportCmd(env, &cnpjFlag))
	cteCmd.AddCommand(newCTeTestarConexaoCmd(env, &cnpjFlag))
	cteCmd.AddCommand(newCTeResetCmd(env, &cnpjFlag))
	return cteCmd
}

// ctePapelLabel is the company's primary role, or all its roles joined by
// commas when it plays more than one.
func ctePapelLabel(doc cte.CompanyDocument) string {
	if len(doc.Papeis) <= 1 {
		return string(doc.CompanyRole)
	}
	papeis := make([]string, len(doc.Papeis))
	for i, p := range doc.Papeis {
		papeis[i] = string(p)
	}
	return strings.Join(papeis, ",")
}
