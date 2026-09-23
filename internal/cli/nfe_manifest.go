package cli

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"text/tabwriter"
	"time"

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

				_, _ = fmt.Fprintf(out, "NF-e elegíveis para a Ciência da Operação: %d (%d lote(s) de até %d)\n",
					len(plan.Eligible), plan.Lotes, sefaz.MaxEventosPorLote)
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
			_, _ = fmt.Fprintf(out, "\nSolicitadas: %d | Registradas: %d | Já registradas: %d | Rejeitadas: %d | Não enviadas: %d\n",
				summary.Requested, summary.Registered, summary.AlreadyRegistered, summary.Rejected, summary.NotSent)
			if summary.Interrupted != "" {
				_, _ = fmt.Fprintf(out, "Aviso: o envio foi interrompido (%s). As NF-e não enviadas podem ser enviadas de novo.\n", summary.Interrupted)
			}
			return nil
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
			tipo, err := parseTipoManifestacao(tipoFlag)
			if err != nil {
				return err
			}
			justificativa, err := nfe.ValidateJustificativa(tipo, justificativaFlag)
			if err != nil {
				return err
			}
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
			if !confirmarFlag {
				docs, err := application.NFe.ListDocuments(cmd.Context(), app.NFeListInput{
					CNPJ:         *cnpjFlag,
					ChavesAcesso: []string{string(chave)},
					Limit:        1,
				})
				if err != nil {
					return fmt.Errorf("erro: %w", err)
				}
				if len(docs) == 0 {
					return fmt.Errorf("erro: NF-e %s não encontrada para a empresa", chave)
				}
				doc := docs[0]
				if doc.CompanyRole != nfe.CompanyRoleDestinatario || doc.Situacao != nfe.SituacaoAutorizada {
					return fmt.Errorf("erro: a NF-e %s não pode ser manifestada pela empresa (papel %s, situação %s)",
						chave, doc.CompanyRole, doc.Situacao)
				}
				if reason := nfe.ConclusiveBlockReason(doc.Manifestacao); reason != "" {
					return errors.New("erro: " + reason)
				}

				_, _ = fmt.Fprintf(out, "Evento: %s (%s)\n", tipo.Label(), tipo.TpEvento())
				_, _ = fmt.Fprintf(out, "Chave de acesso: %s\n", doc.ChaveAcesso)
				_, _ = fmt.Fprintf(out, "Número: %s | Série: %s | Emissão: %s | Valor (R$): %s\n",
					doc.Numero, doc.Serie, formatNFeDate(doc.IssueDate), doc.TotalValue.FormatBRL())
				_, _ = fmt.Fprintf(out, "Emitente: %s %s\n", cnpj.Format(doc.EmitenteCNPJ), doc.EmitenteName)
				_, _ = fmt.Fprintf(out, "Manifestação atual: %s\n", doc.Manifestacao)
				now := time.Now()
				due := nfe.ManifestationDeadlines(doc.Document).ConclusiveDue
				if !due.IsZero() {
					_, _ = fmt.Fprintf(out, "Prazo da manifestação conclusiva: %s (%s)\n", formatNFeDate(due), describeDaysLeft(due, now))
				}
				if nfe.TacitlyConfirmed(doc, now) {
					_, _ = fmt.Fprintf(out, "Atenção: passados %d dias da autorização sem manifestação conclusiva, a operação já é considerada confirmada. A SEFAZ deve rejeitar o evento (cStat 596).\n",
						nfe.ConclusiveDeadlineDays)
				}
				if justificativa != "" {
					_, _ = fmt.Fprintf(out, "Justificativa: %s\n", justificativa)
				}
				_, _ = fmt.Fprintln(out, "\nNada foi enviado. Use --confirmar para registrar a manifestação.")
				return nil
			}

			outcome, err := application.NFe.RegisterManifestation(cmd.Context(), app.NFeManifestationInput{
				CNPJ:          *cnpjFlag,
				ChaveAcesso:   string(chave),
				Tipo:          string(tipo),
				Justificativa: justificativaFlag,
			})
			if outcome.ChaveAcesso != "" {
				printNFeOutcomes(out, []app.NFeEventOutcome{outcome})
			}
			if err != nil {
				return fmt.Errorf("erro: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chaveFlag, "chave", "", "Chave de acesso da NF-e")
	cmd.Flags().StringVar(&tipoFlag, "tipo", "", "Tipo: confirmacao, desconhecimento ou nao-realizada")
	cmd.Flags().StringVar(&justificativaFlag, "justificativa", "", "Justificativa (15 a 255 caracteres), obrigatória para nao-realizada")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Envia o evento à SEFAZ; sem esta flag nada é enviado")
	_ = cmd.MarkFlagRequired("chave")
	_ = cmd.MarkFlagRequired("tipo")
	return cmd
}

// parseTipoManifestacao reads the --tipo flag of `nfe manifestar`.
func parseTipoManifestacao(raw string) (nfe.ManifestationType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "confirmacao":
		return nfe.ManifestationConfirmacao, nil
	case "desconhecimento":
		return nfe.ManifestationDesconhecimento, nil
	case "nao-realizada", "nao_realizada":
		return nfe.ManifestationNaoRealizada, nil
	default:
		return "", fmt.Errorf("tipo de manifestação inválido %q: use confirmacao, desconhecimento ou nao-realizada", raw)
	}
}

// describeDaysLeft says how many whole days are left until due, or how long
// ago it passed.
func describeDaysLeft(due, now time.Time) string {
	days := int(math.Floor(due.Sub(now).Hours() / 24))
	if days < 0 {
		return fmt.Sprintf("vencido há %d dia(s)", -days)
	}
	return fmt.Sprintf("faltam %d dia(s)", days)
}
