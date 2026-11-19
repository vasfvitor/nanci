package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/sync"
)

// nsuActions are the NF-e or CT-e service methods recuperar-nsu calls.
type nsuActions struct {
	recover func(ctx context.Context, in app.RecoverNSUInput) (app.RecoverNSUResult, error)
	discard func(ctx context.Context, in app.DiscardNSUInput) (int64, error)
}

func newNFeRecuperarNSUCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return newRecuperarNSUCmd(env, cnpjFlag, "NF-e", "nfe", func(a *app.App) nsuActions {
		return nsuActions{recover: a.NFe.RecoverNSU, discard: a.NFe.DiscardNSU}
	})
}

func newCTeRecuperarNSUCmd(env CommandEnv, cnpjFlag *string) *cobra.Command {
	return newRecuperarNSUCmd(env, cnpjFlag, "CT-e", "cte", func(a *app.App) nsuActions {
		return nsuActions{recover: a.CTe.RecoverNSU, discard: a.CTe.DiscardNSU}
	})
}

func newRecuperarNSUCmd(env CommandEnv, cnpjFlag *string, label, group string, actionsFor func(*app.App) nsuActions) *cobra.Command {
	var limiteFlag int
	var descartarFlag, confirmarFlag bool
	cmd := &cobra.Command{
		Use:   "recuperar-nsu",
		Short: "Pede à SEFAZ, um a um, os NSUs de " + label + " que a distribuição pulou",
		Long: "Pede à distribuição da SEFAZ, com consNSU, cada NSU de " + label + " que a sincronização pulou sem receber documento " +
			"(por exemplo, os que outro programa consumiu com o mesmo CNPJ) e salva o que vier, sem mexer no NSU da sincronização. " +
			"Cada NSU gasta uma das 20 consultas por hora; sem --limite, usa o que sobra na hora menos 2, que ficam para a sincronização. " +
			"Uma execução interrompida continua de onde parou na próxima. " +
			"Com --descartar, encerra os NSUs faltantes sem pedi-los (simulação sem --confirmar); os documentos deles só entram importando o XML.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limiteFlag < 0 {
				return errors.New("--limite não pode ser negativo")
			}
			if descartarFlag && limiteFlag > 0 {
				return errors.New("--limite não vale com --descartar")
			}
			if confirmarFlag && !descartarFlag {
				return errors.New("--confirmar só vale com --descartar")
			}
			application, cleanup, err := env.AppFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("inicializar: %w", err)
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			actions := actionsFor(application)
			if descartarFlag {
				n, err := actions.discard(cmd.Context(), app.DiscardNSUInput{CNPJ: *cnpjFlag, Confirm: confirmarFlag})
				if err != nil {
					return env.pullError(out, err)
				}
				if env.jsonOutput() {
					return printJSON(out, discardNSUOutput{Discarded: n, Confirmed: confirmarFlag})
				}
				printDiscardNSUResult(out, n, confirmarFlag)
				return nil
			}
			res, err := actions.recover(cmd.Context(), app.RecoverNSUInput{CNPJ: *cnpjFlag, Limit: limiteFlag})
			if err != nil {
				return env.pullError(out, err)
			}
			if env.jsonOutput() {
				return printJSON(out, newRecoverNSUOutput(res))
			}
			printRecoverNSUResult(out, res, group, limiteFlag)
			return nil
		},
	}
	cmd.Flags().IntVar(&limiteFlag, "limite", 0, "Máximo de consultas desta execução (padrão: o que sobra na hora menos 2)")
	cmd.Flags().BoolVar(&descartarFlag, "descartar", false, "Encerra os NSUs faltantes sem pedi-los à SEFAZ")
	cmd.Flags().BoolVar(&confirmarFlag, "confirmar", false, "Com --descartar, descarta de fato; sem esta flag nada é alterado")
	return cmd
}

// printDiscardNSUResult prints what --descartar did, or would do without
// --confirmar.
func printDiscardNSUResult(out io.Writer, n int64, confirmed bool) {
	switch {
	case n == 0:
		_, _ = fmt.Fprintln(out, "Nenhum NSU faltando.")
	case confirmed:
		_, _ = fmt.Fprintf(out, "NSUs descartados: %d. A recuperação não vai pedi-los; os documentos deles só entram importando o XML.\n", n)
	default:
		_, _ = fmt.Fprintf(out, "NSUs que seriam descartados: %d. Nada foi alterado. Use --confirmar para descartar.\n", n)
	}
}

// printRecoverNSUResult prints the counts of a run; limit is the --limite the
// user gave, repeated in the hint to continue.
func printRecoverNSUResult(out io.Writer, res app.RecoverNSUResult, group string, limit int) {
	if res.Asked == 0 && res.Expired == 0 && res.Missing == 0 {
		_, _ = fmt.Fprintln(out, "Nenhum NSU faltando.")
		return
	}
	_, _ = fmt.Fprintf(out, "NSUs consultados: %d (%d com documento, %d sem documento", res.Asked, res.Recovered, res.Empty)
	if res.Unreadable > 0 {
		_, _ = fmt.Fprintf(out, ", %d ilegíveis; veja o log", res.Unreadable)
	}
	_, _ = fmt.Fprintln(out, ")")
	if res.Expired > 0 {
		_, _ = fmt.Fprintf(out, "NSUs fora da janela de 90 dias, encerrados sem consulta: %d\n", res.Expired)
	}
	_, _ = fmt.Fprintf(out, "NSUs ainda faltando: %d\n", res.Missing)
	switch res.StopReason {
	case sync.RecoverStopLimit:
		again := fmt.Sprintf("nanci %s recuperar-nsu --cnpj %s", group, res.CNPJ)
		if limit > 0 {
			again += fmt.Sprintf(" --limite %d", limit)
		}
		_, _ = fmt.Fprintf(out, "Parou no limite desta execução. Rode `%s` de novo para continuar.\n", again)
	case sync.RecoverStopRateBudget:
		_, _ = fmt.Fprintln(out, "Parou porque as consultas desta hora acabaram.")
	case sync.RecoverStopCanceled:
		_, _ = fmt.Fprintln(out, "Interrompida pelo usuário; o que falta continua na próxima vez.")
	case sync.RecoverStopConsumoIndevido:
		_, _ = fmt.Fprintln(out, "A SEFAZ bloqueou o CNPJ por consumo indevido (cStat 656).")
	}
	if res.NextAllowedAt != nil {
		_, _ = fmt.Fprintf(out, "Próxima consulta permitida após: %s\n", formatDateTime(*res.NextAllowedAt))
	}
	_, _ = fmt.Fprintf(out, "Consultas na última hora: %d/%d\n", res.RequestsLastHour, res.RequestBudget)
}

// printMissingNSUs tells the status reader how many NSUs a recovery can
// still ask for.
func printMissingNSUs(out io.Writer, status app.SefazSourceStatus, recoverCmd string) {
	if status.NSUFaltantes == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "  NSUs faltando: %d. Rode `%s` para pedi-los à SEFAZ um a um.\n", status.NSUFaltantes, recoverCmd)
}
