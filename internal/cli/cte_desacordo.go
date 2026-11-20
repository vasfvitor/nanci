package cli

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// newCTeDesacordoCmd registers the prestação do serviço em desacordo (610110)
// of one CT-e. Without --confirmar it only prints what would be sent.
func newCTeDesacordoCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag      string
		observacaoFlag string
		confirmarFlag  bool
	)
	cmd := &cobra.Command{
		Use:   "desacordo",
		Short: "Registra a prestação do serviço em desacordo como tomador (simulação sem --confirmar)",
		Long: "Envia o evento 610110 à SEFAZ que autorizou o CT-e, declarando que o transporte não foi prestado " +
			"como consta no documento. Só o tomador pode enviá-lo, para o CT-e e o CT-e OS, em até 45 dias da " +
			"autorização: o nanci mostra o prazo e deixa a SEFAZ decidir. O evento enviado não pode ser desfeito pelo nanci.",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := dfe.ParseAccessKey(chaveFlag)
			if err != nil {
				return fmt.Errorf("chave de acesso inválida: %w", err)
			}
			if err := sefaz.CheckXObsDesacordo(observacaoFlag); err != nil {
				return fmt.Errorf("%w: %w", app.ErrInvalidObservacao, err)
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			input := app.CTeDesacordoInput{CNPJ: *cnpjFlag, ChaveAcesso: string(chave), Observacao: observacaoFlag}
			if !confirmarFlag {
				plan, err := application.CTe.PlanDesacordo(cmd.Context(), input)
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				if plan.BlockReason != "" {
					return errors.New("erro: " + plan.BlockReason)
				}
				if env.jsonOutput() {
					return printJSON(out, newCTeDesacordoPlanOutput(plan))
				}
				printCTeDesacordoPlan(out, plan)
				_, _ = fmt.Fprintln(out, "\nNada foi enviado. Use --confirmar para registrar a prestação em desacordo.")
				return nil
			}

			outcome, err := application.CTe.RegisterDesacordo(cmd.Context(), input)
			if outcome.ChaveAcesso != "" {
				if env.jsonOutput() {
					if jsonErr := printJSON(out, newCTeDesacordoOutcomeOutput(outcome)); jsonErr != nil {
						return jsonErr
					}
				} else {
					printCTeDesacordoOutcome(out, outcome)
				}
			}
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}
			if outcome.Status != cte.DesacordoStatusRegistrada && outcome.Status != cte.DesacordoStatusJaRegistrada {
				return fmt.Errorf("erro: a SEFAZ não registrou a prestação em desacordo (cStat %s)", outcome.CStat)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso do CT-e")
	cmd.Flags().StringVar(&observacaoFlag, "observacao", "", "O que não foi prestado como consta no CT-e, de 15 a 255 caracteres")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Envia o evento à SEFAZ; sem esta flag nada é enviado")
	_ = cmd.MarkFlagRequired("chave")
	_ = cmd.MarkFlagRequired("observacao")
	return cmd
}

// printCTeDesacordoPlan prints what `cte desacordo --confirmar` would send.
func printCTeDesacordoPlan(out io.Writer, plan app.CTeDesacordoPlan) {
	doc := plan.Document
	ambiente := "produção"
	if plan.TpAmb == sefaz.TpAmbHomologacao {
		ambiente = "homologação"
	}
	_, _ = fmt.Fprintf(out, "Evento: Prestação do Serviço em Desacordo (%s), ambiente de %s\n", sefaz.TpEventoPrestacaoDesacordo, ambiente)
	_, _ = fmt.Fprintf(out, "Chave de acesso: %s (modelo %s)\n", doc.ChaveAcesso, doc.Modelo)
	_, _ = fmt.Fprintf(out, "Número: %s | Série: %s | Emissão: %s | Valor (R$): %s\n",
		doc.Numero, doc.Serie, formatDate(doc.IssueDate), doc.TotalValue.FormatBRL())
	_, _ = fmt.Fprintf(out, "Emitente: %s %s\n", dashIfEmpty(cnpj.Format(doc.Emitente.CNPJ)), doc.Emitente.Name)
	_, _ = fmt.Fprintf(out, "Tomador: %s %s\n", cnpj.Format(plan.CNPJ), plan.CompanyName)
	if !plan.Due.IsZero() {
		daysLeft := fmt.Sprintf("faltam %d dia(s)", plan.DaysLeft)
		if plan.DaysLeft < 0 {
			daysLeft = fmt.Sprintf("vencido há %d dia(s)", -plan.DaysLeft)
		}
		_, _ = fmt.Fprintf(out, "Prazo: %s (%s)\n", formatDate(plan.Due), daysLeft)
		if plan.DaysLeft < 0 {
			_, _ = fmt.Fprintf(out, "Atenção: passaram os %d dias da autorização. A SEFAZ deve rejeitar o evento (cStat %d).\n",
				app.DesacordoPrazoDias, sefaz.CStatDesacordoForaDoPrazo)
		}
	}
	_, _ = fmt.Fprintf(out, "Observação: %s\n", plan.Observacao)
}

// printCTeDesacordoOutcome prints the answer to the event, in the table of
// the NF-e manifestação.
func printCTeDesacordoOutcome(w io.Writer, o app.CTeDesacordoOutcome) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "CHAVE DE ACESSO\tRESULTADO\tCSTAT\tMOTIVO\tPROTOCOLO")
	_, _ = fmt.Fprintln(tw, "---------------\t---------\t-----\t------\t---------")
	_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
		o.ChaveAcesso, nfeOutcomeLabel(o.Status), dashIfEmpty(o.CStat), dashIfEmpty(o.XMotivo), dashIfEmpty(o.Protocolo))
	_ = tw.Flush()
}
