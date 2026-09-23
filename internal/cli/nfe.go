package cli

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

// newNFeCommand builds the `nfe` subcommand tree. Every leaf acts on the
// company given by the persistent --cnpj flag.
func newNFeCommand(env CommandEnv) *cobra.Command {
	var cnpjFlag string
	nfeCmd := &cobra.Command{
		Use:   "nfe",
		Short: "Sincroniza, lista, manifesta, exporta e redefine NF-e (modelo 55) da SEFAZ",
	}
	nfeCmd.PersistentFlags().StringVarP(&cnpjFlag, "cnpj", "c", "", "CNPJ da empresa")
	_ = nfeCmd.MarkPersistentFlagRequired("cnpj")

	nfeCmd.AddCommand(newNFePullCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeStatusCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeListCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeCienciaCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeManifestarCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFePendentesCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeExportCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeTestarConexaoCmd(env, &cnpjFlag))
	nfeCmd.AddCommand(newNFeResetCmd(env, &cnpjFlag))
	return nfeCmd
}

// formatNFeDate prints the calendar date of t, or "-" when t is zero.
func formatNFeDate(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}

// formatNFeDateTime prints t in local time, to the minute.
func formatNFeDateTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}

// formatNSU prints an NSU with the 15 digits SEFAZ uses.
func formatNSU(nsu int64) string {
	return fmt.Sprintf("%015d", nsu)
}

// truncateText cuts s to at most n characters.
func truncateText(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// nfeOutcomeLabel is the text shown for a manifestação outcome.
func nfeOutcomeLabel(status string) string {
	switch status {
	case app.NFeOutcomeRegistrada:
		return "registrada"
	case app.NFeOutcomeJaRegistrada:
		return "já registrada"
	case app.NFeOutcomeRejeitada:
		return "rejeitada"
	default:
		return "não enviada"
	}
}

// printNFeOutcomes writes one line per manifestação event.
func printNFeOutcomes(w io.Writer, outcomes []app.NFeEventOutcome) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "CHAVE DE ACESSO\tRESULTADO\tCSTAT\tMOTIVO\tPROTOCOLO")
	_, _ = fmt.Fprintln(tw, "---------------\t---------\t-----\t------\t---------")
	for _, o := range outcomes {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			o.ChaveAcesso,
			nfeOutcomeLabel(o.Status),
			dashIfEmpty(o.CStat),
			dashIfEmpty(o.XMotivo),
			dashIfEmpty(o.Protocolo),
		)
	}
	_ = tw.Flush()
}

// printNFeSkipped lists the chaves left out of a manifestação, and why.
func printNFeSkipped(w io.Writer, skipped []app.NFeSkipped) {
	if len(skipped) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "Ignoradas: %d\n", len(skipped))
	for _, s := range skipped {
		_, _ = fmt.Fprintf(w, "  %s: %s\n", s.ChaveAcesso, s.Reason)
	}
}
