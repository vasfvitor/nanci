package cli

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// newNFeCienciaCmd registers Ciência da Operação. Without --confirmar it
// only prints what would be sent.
func newNFeCienciaCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlags       []string
		todosResumosFlag bool
		confirmarFlag    bool
	)
	cmd := &cobra.Command{
		Use:   "ciencia",
		Short: "Registra a Ciência da Operação (simulação sem --confirmar)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if todosResumosFlag == (len(chaveFlags) > 0) {
				return errors.New("informe --chave ou --todos-resumos, apenas um dos dois")
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			input := app.NFeCienciaInput{
				CNPJ:         *cnpjFlag,
				ChavesAcesso: chaveFlags,
				AllResumos:   todosResumosFlag,
			}
			out := cmd.OutOrStdout()

			if !confirmarFlag {
				plan, err := application.NFe.PlanCiencia(cmd.Context(), input)
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				if len(plan.Eligible) == 0 {
					_, _ = fmt.Fprintln(out, "Nenhuma NF-e elegível para a Ciência da Operação.")
					printNFeSkipped(out, plan.Skipped)
					return nil
				}

				lotes := (len(plan.Eligible) + sefaz.MaxEventosPorLote - 1) / sefaz.MaxEventosPorLote
				_, _ = fmt.Fprintf(out, "NF-e elegíveis para a Ciência da Operação: %d (%d lote(s) de até %d)\n",
					len(plan.Eligible), lotes, sefaz.MaxEventosPorLote)
				w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "CHAVE DE ACESSO\tEMITENTE\tNOME EMITENTE\tEMISSÃO\tVALOR (R$)\tCIÊNCIA ATÉ\tCONCLUSIVA ATÉ")
				_, _ = fmt.Fprintln(w, "---------------\t--------\t-------------\t-------\t----------\t-----------\t--------------")
				for _, c := range plan.Eligible {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						c.ChaveAcesso,
						cnpj.Format(c.EmitenteCNPJ),
						truncateText(c.EmitenteName, 30),
						formatNFeDate(c.IssueDate),
						c.TotalValue.FormatBRL(),
						formatNFeDate(c.CienciaDue),
						formatNFeDate(c.ConclusiveDue),
					)
				}
				_ = w.Flush()
				printNFeSkipped(out, plan.Skipped)
				_, _ = fmt.Fprintln(out, "\nNada foi enviado. Use --confirmar para registrar a Ciência da Operação.")
				return nil
			}

			summary, err := application.NFe.RegisterCiencia(cmd.Context(), input)
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}
			printNFeOutcomes(out, summary.Outcomes)
			printNFeSkipped(out, summary.Skipped)
			counts := make(map[string]int)
			for _, o := range summary.Outcomes {
				counts[o.Status]++
			}
			_, _ = fmt.Fprintf(out, "\nSolicitadas: %d | Registradas: %d | Já registradas: %d | Rejeitadas: %d | Não enviadas: %d\n",
				len(summary.Outcomes), counts[app.NFeOutcomeRegistrada], counts[app.NFeOutcomeJaRegistrada],
				counts[app.NFeOutcomeRejeitada], counts[app.NFeOutcomeNaoEnviada])
			if summary.Interrupted != "" {
				_, _ = fmt.Fprintf(out, "Aviso: o envio foi interrompido (%s). As NF-e não enviadas podem ser enviadas de novo.\n", summary.Interrupted)
			}
			return nfeOutcomesError(summary.Outcomes, summary.Interrupted)
		},
	}
	cmd.Flags().StringSliceVar(&chaveFlags, "chave", nil, "Chave de acesso da NF-e (pode repetir)")
	cmd.Flags().BoolVar(&todosResumosFlag, "todos-resumos", false, "Seleciona todos os resumos autorizados ainda sem manifestação")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Envia os eventos à SEFAZ; sem esta flag nada é enviado")
	return cmd
}

// newNFeManifestarCmd registers one conclusive manifestação. Without
// --confirmar it only prints what would be sent.
func newNFeManifestarCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	var (
		chaveFlag         string
		tipoFlag          string
		justificativaFlag string
		confirmarFlag     bool
	)
	cmd := &cobra.Command{
		Use:   "manifestar",
		Short: "Registra confirmação, desconhecimento ou operação não realizada (simulação sem --confirmar)",
		RunE: func(cmd *cobra.Command, args []string) error {
			chave, err := nfe.ParseAccessKey(chaveFlag)
			if err != nil {
				return fmt.Errorf("chave de acesso inválida: %w", err)
			}

			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			input := app.NFeManifestacaoInput{
				CNPJ:          *cnpjFlag,
				ChaveAcesso:   string(chave),
				Tipo:          tipoFlag,
				Justificativa: justificativaFlag,
			}
			if !confirmarFlag {
				plan, err := application.NFe.PlanManifestacao(cmd.Context(), input)
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				if plan.BlockReason != "" {
					return errors.New("erro: " + plan.BlockReason)
				}
				printNFeManifestacaoPlan(out, plan)
				_, _ = fmt.Fprintln(out, "\nNada foi enviado. Use --confirmar para registrar a manifestação.")
				return nil
			}

			outcome, err := application.NFe.RegisterManifestacao(cmd.Context(), input)
			if outcome.ChaveAcesso != "" {
				printNFeOutcomes(out, []app.NFeEventOutcome{outcome})
			}
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}
			return nfeOutcomesError([]app.NFeEventOutcome{outcome}, "")
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NF-e")
	cmd.Flags().StringVar(&tipoFlag, "tipo", "", "Tipo: confirmacao, desconhecimento ou nao_realizada")
	cmd.Flags().StringVar(&justificativaFlag, "justificativa", "", "Justificativa (15 a 255 caracteres), obrigatória para nao_realizada")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Envia o evento à SEFAZ; sem esta flag nada é enviado")
	_ = cmd.MarkFlagRequired("chave")
	_ = cmd.MarkFlagRequired("tipo")
	return cmd
}

// nfeOutcomesError makes ciencia and manifestar exit non-zero when an event
// was not registered (rejeitada or nao_enviada) or the sending was
// interrupted. The caller prints the outcome table first.
func nfeOutcomesError(outcomes []app.NFeEventOutcome, interrupted string) error {
	failed := 0
	for _, o := range outcomes {
		if o.Status != nfe.ManifestacaoStatusRegistrada && o.Status != nfe.ManifestacaoStatusJaRegistrada {
			failed++
		}
	}
	switch {
	case interrupted != "":
		return fmt.Errorf("erro: envio interrompido, %d de %d evento(s) sem registro", failed, len(outcomes))
	case failed > 0:
		return fmt.Errorf("erro: %d de %d evento(s) sem registro", failed, len(outcomes))
	default:
		return nil
	}
}

// printNFeManifestacaoPlan prints what `nfe manifestar --confirmar` would
// send.
func printNFeManifestacaoPlan(out io.Writer, plan app.NFeManifestacaoPlan) {
	doc := plan.Document
	_, _ = fmt.Fprintf(out, "Evento: %s (%s)\n", plan.Tipo.Label(), plan.Tipo.TpEvento())
	_, _ = fmt.Fprintf(out, "Chave de acesso: %s\n", doc.ChaveAcesso)
	_, _ = fmt.Fprintf(out, "Número: %s | Série: %s | Emissão: %s | Valor (R$): %s\n",
		doc.Numero, doc.Serie, formatNFeDate(doc.IssueDate), doc.TotalValue.FormatBRL())
	_, _ = fmt.Fprintf(out, "Emitente: %s %s\n", cnpj.Format(doc.EmitenteCNPJ), doc.EmitenteName)
	_, _ = fmt.Fprintf(out, "Manifestação atual: %s\n", doc.Manifestacao)
	if !plan.ConclusiveDue.IsZero() {
		daysLeft := fmt.Sprintf("faltam %d dia(s)", plan.DaysLeft)
		if plan.DaysLeft < 0 {
			daysLeft = fmt.Sprintf("vencido há %d dia(s)", -plan.DaysLeft)
		}
		_, _ = fmt.Fprintf(out, "Prazo da manifestação conclusiva: %s (%s)\n", formatNFeDate(plan.ConclusiveDue), daysLeft)
	}
	if plan.TacitlyConfirmed {
		_, _ = fmt.Fprintf(out, "Atenção: passados %d dias da autorização sem manifestação conclusiva, a operação já é considerada confirmada. A SEFAZ deve rejeitar o evento (cStat 596).\n",
			nfe.ConclusiveDeadlineDays)
	}
	if plan.Justificativa != "" {
		_, _ = fmt.Fprintf(out, "Justificativa: %s\n", plan.Justificativa)
	}
}
