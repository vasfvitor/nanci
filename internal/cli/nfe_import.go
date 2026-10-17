package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

func newNFeImportCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "import <arquivo>...",
		Short: "Importa arquivos XML de NF-e e de eventos sem consultar a SEFAZ",
		Long: "Importa arquivos XML de NF-e (procNFe ou resNFe) e de eventos que a empresa já tem, " +
			"como os emitidos pelo próprio sistema ou enviados pelo fornecedor. " +
			"Informe arquivos, não pastas. Um arquivo rejeitado não impede a importação dos outros.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkImportFiles(args); err != nil {
				return err
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			res, err := application.NFe.ImportXML(cmd.Context(), app.ImportXMLInput{CNPJ: *cnpjFlag, Files: args})
			if err != nil {
				return fmt.Errorf("erro ao importar XML: %w", err)
			}
			return printImportResult(cmd.OutOrStdout(), res)
		},
	}
}

// checkImportFiles refuses a path that does not exist or is a folder, so a
// typo never opens the database.
func checkImportFiles(paths []string) error {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("arquivo não encontrado: %s", path)
		}
		if info.IsDir() {
			return fmt.Errorf("%s é uma pasta; informe os arquivos XML", path)
		}
	}
	return nil
}

// importOutcomeLabel is the text shown for the outcome of an imported file.
func importOutcomeLabel(outcome string) string {
	switch outcome {
	case app.ImportOutcomeImported:
		return "importado"
	case app.ImportOutcomeUpdated:
		return "atualizado"
	default:
		return "rejeitado"
	}
}

// printImportResult writes one line per file and the counts, and returns an
// error when every file was rejected.
func printImportResult(w io.Writer, res app.ImportXMLResult) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, f := range res.Files {
		message := f.Message
		if f.Outcome == app.ImportOutcomeRejected {
			// The kind and chave may be unknown; the path tells which file it was.
			message = f.Path + ": " + message
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			importOutcomeLabel(f.Outcome),
			dashIfEmpty(f.Kind),
			dashIfEmpty(f.ChaveAcesso),
			message,
		)
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintf(w, "Importados: %d | Atualizados: %d | Rejeitados: %d\n", res.Imported, res.Updated, res.Rejected)

	if res.Imported+res.Updated == 0 {
		return errors.New("nenhum arquivo importado")
	}
	return nil
}
