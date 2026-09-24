package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// fakeSEFAZ answers EnviarEventos with real signed eventos and a scripted
// cStat per chave (135 when not scripted).
type fakeSEFAZ struct {
	t           *testing.T
	clients     int // clients created through newSEFAZClient
	tlsErr      error
	tlsCalls    int
	cteTLSCalls int
	lotes       [][]sefaz.Evento
	cStats      map[string]int
	failLote    map[int]error // 0-based lote index -> transport error
}

func (f *fakeSEFAZ) CheckTLS(context.Context) error {
	f.tlsCalls++
	return f.tlsErr
}

func (f *fakeSEFAZ) CheckTLSCTe(context.Context) error {
	f.cteTLSCalls++
	return f.tlsErr
}

func (f *fakeSEFAZ) EnviarEventos(_ context.Context, signer *sefaz.Signer, idLote string, eventos []sefaz.Evento) (sefaz.LoteResult, error) {
	index := len(f.lotes)
	f.lotes = append(f.lotes, eventos)
	if err := f.failLote[index]; err != nil {
		return sefaz.LoteResult{}, err
	}

	result := sefaz.LoteResult{IDLote: idLote, CStat: sefaz.CStatLoteProcessado}
	for i, ev := range eventos {
		signed, err := signer.SignEvento(ev, sefaz.TpAmbProducao)
		if err != nil {
			f.t.Fatalf("sign evento: %v", err)
		}
		cStat, ok := f.cStats[ev.ChaveAcesso]
		if !ok {
			cStat = sefaz.CStatEventoVinculado
		}
		registeredAt := time.Date(2026, 9, 20, 10, 0, i, 0, time.UTC)
		protocolo := ""
		if sefaz.IsRegistered(cStat) {
			protocolo = fmt.Sprintf("8912600%08d", index*100+i)
		}
		ret := fmt.Sprintf(`<retEvento versao="1.00"><infEvento><cStat>%d</cStat><chNFe>%s</chNFe><tpEvento>%s</tpEvento><nProt>%s</nProt></infEvento></retEvento>`,
			cStat, ev.ChaveAcesso, ev.TpEvento, protocolo)
		result.Eventos = append(result.Eventos, sefaz.EventoResult{
			ChaveAcesso:  ev.ChaveAcesso,
			TpEvento:     ev.TpEvento,
			NSeqEvento:   ev.NSeqEvento,
			CStat:        cStat,
			XMotivo:      fmt.Sprintf("resposta %d", cStat),
			Protocolo:    protocolo,
			DhRegEvento:  &registeredAt,
			SignedEvento: signed,
			RetEvento:    []byte(ret),
		})
	}
	return result, nil
}

func (f *fakeSEFAZ) loteSizes() []int {
	sizes := make([]int, 0, len(f.lotes))
	for _, lote := range f.lotes {
		sizes = append(sizes, len(lote))
	}
	return sizes
}

// useFakeSEFAZ makes every SEFAZ client of the test the returned fake.
func useFakeSEFAZ(t *testing.T) *fakeSEFAZ {
	t.Helper()
	fake := &fakeSEFAZ{t: t}
	original := newSEFAZClient
	t.Cleanup(func() { newSEFAZClient = original })
	newSEFAZClient = func(cfg sefaz.ClientConfig) (sefazClient, error) {
		if cfg.Certificate == nil || cfg.Environment != nfse.EnvironmentProduction {
			t.Errorf("client config = %+v", cfg)
		}
		fake.clients++
		return fake, nil
	}
	return fake
}

// seedResumos stores count resumos addressed to the company and returns
// their chaves.
func (e *nfeTestEnv) seedResumos(count int) []string {
	e.t.Helper()
	chaves := make([]string, 0, count)
	for n := 100; n < 100+count; n++ {
		chaves = append(chaves, e.seedResumo(n, "2026-09-01T09:15:42-03:00", "11222333000181"))
	}
	return chaves
}

func (e *nfeTestEnv) manifestacaoStatuses() map[string]int {
	e.t.Helper()
	rows, err := e.db.QueryContext(context.Background(), `SELECT status, COUNT(*) FROM nfe_manifestacoes GROUP BY status`)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	statuses := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			e.t.Fatal(err)
		}
		statuses[status] = count
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return statuses
}

func (e *nfeTestEnv) manifestacao(chave string) nfe.Manifestacao {
	e.t.Helper()
	docs, err := e.app.NFe.ListDocuments(context.Background(), NFeListInput{CNPJ: nfeTestCNPJ, ChavesAcesso: []string{chave}})
	if err != nil || len(docs) != 1 {
		e.t.Fatalf("document %s: %d rows, %v", chave, len(docs), err)
	}
	return docs[0].Manifestacao
}

func TestNFePlanCienciaNeedsNoNetworkOrPassword(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	eligible := env.seedResumo(10, "2026-08-20T10:00:00-03:00", "11222333000181")
	emitida := env.seedResumo(11, "2026-08-21T10:00:00-03:00", nfeTestCNPJ)
	unknown := testChave(t, 12)
	fake := useFakeSEFAZ(t)

	plan, err := env.app.NFe.PlanCiencia(context.Background(), NFeCienciaInput{
		CNPJ:         nfeTestCNPJ,
		ChavesAcesso: []string{eligible, nfeChaveProc, nfeChaveCancelada, emitida, unknown, "123", eligible},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Eligible) != 1 || string(plan.Eligible[0].ChaveAcesso) != eligible {
		t.Fatalf("eligible = %+v, want only %s", plan.Eligible, eligible)
	}
	wantSkipped := []NFeSkipped{
		{ChaveAcesso: "123", Reason: "chave de acesso inválida"},
		{ChaveAcesso: eligible, Reason: "chave repetida"},
		{ChaveAcesso: nfeChaveProc, Reason: "já manifestada (ciencia)"},
		{ChaveAcesso: nfeChaveCancelada, Reason: "NF-e cancelada"},
		{ChaveAcesso: emitida, Reason: "a empresa não é a destinatária"},
		{ChaveAcesso: unknown, Reason: "não encontrada para a empresa"},
	}
	if !slices.Equal(plan.Skipped, wantSkipped) {
		t.Errorf("skipped =\n%+v\nwant\n%+v", plan.Skipped, wantSkipped)
	}

	all, err := env.app.NFe.PlanCiencia(context.Background(), NFeCienciaInput{CNPJ: nfeTestCNPJ, AllResumos: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Eligible) != 1 || string(all.Eligible[0].ChaveAcesso) != eligible {
		t.Errorf("all resumos = %+v, want only %s", all.Eligible, eligible)
	}
	if len(env.passwords.requests) != 0 || fake.clients != 0 {
		t.Errorf("password prompts = %d, SEFAZ clients = %d; want none", len(env.passwords.requests), fake.clients)
	}
}

func TestNFeRegisterCienciaSendsLotesOfTwenty(t *testing.T) {
	env := newNFeTestEnv(t)
	chaves := env.seedResumos(45)
	fake := useFakeSEFAZ(t)

	summary, err := env.app.NFe.RegisterCiencia(context.Background(), NFeCienciaInput{CNPJ: nfeTestCNPJ, AllResumos: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := fake.loteSizes(); !slices.Equal(got, []int{20, 20, 5}) {
		t.Errorf("lote sizes = %v, want [20 20 5]", got)
	}
	counts := outcomeCounts(summary.Outcomes)
	if counts[nfe.ManifestacaoStatusRegistrada] != 45 || summary.Interrupted != "" || len(summary.Outcomes) != 45 {
		t.Errorf("summary = counts %v, interrupted %q, outcomes %d", counts, summary.Interrupted, len(summary.Outcomes))
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Assinatura: Ciência da Operação (45 notas)" {
		t.Errorf("password requests = %+v, want one for 45 notas", env.passwords.requests)
	}
	for _, ev := range fake.lotes[0] {
		if ev.TpEvento != nfe.TpEventoCiencia || ev.CNPJ != nfeTestCNPJ || ev.NSeqEvento != 1 {
			t.Fatalf("evento = %+v", ev)
		}
	}
	for _, chave := range chaves {
		if got := env.manifestacao(chave); got != nfe.ManifestacaoCiencia {
			t.Fatalf("%s manifestacao = %s, want ciencia", chave, got)
		}
	}
	if got := env.manifestacaoStatuses(); got[nfe.ManifestacaoStatusRegistrada] != 45 || len(got) != 1 {
		t.Errorf("stored statuses = %v, want 45 registrada", got)
	}
	var tpAmbs, withoutTpAmb int
	err = env.db.QueryRowContext(context.Background(), `
		SELECT COUNT(DISTINCT tp_amb), COUNT(*) FILTER (WHERE tp_amb <> ?) FROM nfe_manifestacoes
	`, sefaz.TpAmbProducao).Scan(&tpAmbs, &withoutTpAmb)
	if err != nil {
		t.Fatal(err)
	}
	if tpAmbs != 1 || withoutTpAmb != 0 {
		t.Errorf("tp_amb: %d distinct values, %d rows not %s; want every row sent to produção", tpAmbs, withoutTpAmb, sefaz.TpAmbProducao)
	}
}

func TestNFeRegisterCienciaReportsMixedAnswers(t *testing.T) {
	env := newNFeTestEnv(t)
	chaves := env.seedResumos(4)
	fake := useFakeSEFAZ(t)
	fake.cStats = map[string]int{
		chaves[0]: sefaz.CStatEventoVinculado,
		chaves[1]: sefaz.CStatDuplicidadeEvento,
		chaves[2]: sefaz.CStatCienciaNFeCancelada,
		chaves[3]: sefaz.CStatCienciaAposManifestacao,
	}

	summary, err := env.app.NFe.RegisterCiencia(context.Background(), NFeCienciaInput{CNPJ: nfeTestCNPJ, ChavesAcesso: chaves})
	if err != nil {
		t.Fatal(err)
	}
	counts := outcomeCounts(summary.Outcomes)
	if counts[nfe.ManifestacaoStatusRegistrada] != 1 || counts[nfe.ManifestacaoStatusJaRegistrada] != 1 || counts[nfe.ManifestacaoStatusRejeitada] != 2 || counts[NFeOutcomeNaoEnviada] != 0 {
		t.Errorf("counts = %v, summary = %+v", counts, summary)
	}
	want := []struct{ status, cStat string }{
		{nfe.ManifestacaoStatusRegistrada, "135"},
		{nfe.ManifestacaoStatusJaRegistrada, "573"},
		{nfe.ManifestacaoStatusRejeitada, "650"},
		{nfe.ManifestacaoStatusRejeitada, "655"},
	}
	for i, o := range summary.Outcomes {
		if o.ChaveAcesso != chaves[i] || o.Status != want[i].status || o.CStat != want[i].cStat || o.TpEvento != nfe.TpEventoCiencia {
			t.Errorf("outcome %d = %+v, want %s/%s", i, o, want[i].status, want[i].cStat)
		}
	}
	if summary.Outcomes[0].Protocolo == "" || summary.Outcomes[0].RegisteredAt == nil {
		t.Errorf("registered outcome = %+v, want protocolo and time", summary.Outcomes[0])
	}

	if got := env.manifestacao(chaves[0]); got != nfe.ManifestacaoCiencia {
		t.Errorf("registered manifestacao = %s", got)
	}
	if got := env.manifestacao(chaves[1]); got != nfe.ManifestacaoCiencia {
		t.Errorf("already registered manifestacao = %s", got)
	}
	if got := env.manifestacao(chaves[2]); got != nfe.ManifestacaoNenhuma {
		t.Errorf("rejected manifestacao = %s", got)
	}
	// 655: SEFAZ did not register the ciência, so no event is stored.
	if got := env.manifestacao(chaves[3]); got != nfe.ManifestacaoNenhuma {
		t.Errorf("655 manifestacao = %s, want nenhuma", got)
	}
	if motivo := summary.Outcomes[3].XMotivo; !strings.HasPrefix(motivo, "NF-e já possui manifestação conclusiva") {
		t.Errorf("655 XMotivo = %q", motivo)
	}
	if events, err := env.app.NFe.ListEvents(context.Background(), nfeTestCNPJ, chaves[3]); err != nil || len(events) != 0 {
		t.Errorf("655 events = %+v, %v; want none", events, err)
	}
	wantStatuses := map[string]int{
		nfe.ManifestacaoStatusRegistrada:   1,
		nfe.ManifestacaoStatusJaRegistrada: 1,
		nfe.ManifestacaoStatusRejeitada:    2,
	}
	if got := env.manifestacaoStatuses(); fmt.Sprint(got) != fmt.Sprint(wantStatuses) {
		t.Errorf("stored statuses = %v, want %v", got, wantStatuses)
	}
	events, err := env.app.NFe.ListEvents(context.Background(), nfeTestCNPJ, chaves[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || !events[0].SentByNanci || events[0].RawHash == "" {
		t.Errorf("events = %+v, want the sent ciência with its procEventoNFe", events)
	}
	if _, err := env.xml.Get(events[0].RawHash); err != nil {
		t.Errorf("procEventoNFe blob: %v", err)
	}
}

func TestNFeRegisterCienciaTransportFailureInterruptsWithoutError(t *testing.T) {
	env := newNFeTestEnv(t)
	chaves := env.seedResumos(45)
	fake := useFakeSEFAZ(t)
	fake.failLote = map[int]error{1: errors.New("connection reset by peer")}

	summary, err := env.app.NFe.RegisterCiencia(context.Background(), NFeCienciaInput{CNPJ: nfeTestCNPJ, AllResumos: true})
	if err != nil {
		t.Fatalf("RegisterCiencia error = %v, want the failure in the summary", err)
	}
	if len(fake.lotes) != 2 {
		t.Errorf("lotes sent = %d, want 2 (the third is never sent)", len(fake.lotes))
	}
	counts := outcomeCounts(summary.Outcomes)
	if counts[nfe.ManifestacaoStatusRegistrada] != 20 || counts[NFeOutcomeNaoEnviada] != 25 || !strings.Contains(summary.Interrupted, "connection reset by peer") {
		t.Errorf("summary = counts %v, interrupted %q", counts, summary.Interrupted)
	}
	for _, o := range summary.Outcomes[20:] {
		if o.Status != NFeOutcomeNaoEnviada {
			t.Fatalf("outcome after the failure = %+v, want nao_enviada", o)
		}
	}

	registered := 0
	for _, chave := range chaves {
		if env.manifestacao(chave) == nfe.ManifestacaoCiencia {
			registered++
		}
	}
	if registered != 20 {
		t.Errorf("documents with ciência = %d, want 20", registered)
	}
	wantStatuses := map[string]int{nfe.ManifestacaoStatusRegistrada: 20, nfe.ManifestacaoStatusErro: 20}
	if got := env.manifestacaoStatuses(); fmt.Sprint(got) != fmt.Sprint(wantStatuses) {
		t.Errorf("stored statuses = %v, want %v", got, wantStatuses)
	}
}

func TestNFeRegisterManifestacaoValidatesBeforePassword(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	emitida := env.seedResumo(11, "2026-08-21T10:00:00-03:00", nfeTestCNPJ)
	fake := useFakeSEFAZ(t)
	ctx := context.Background()

	tests := []struct {
		name string
		in   NFeManifestacaoInput
		want string
	}{
		{"não realizada without justificativa", NFeManifestacaoInput{ChaveAcesso: nfeChaveProc, Tipo: "210240"}, "justificativa inválida"},
		{"não realizada with a short justificativa", NFeManifestacaoInput{ChaveAcesso: nfeChaveProc, Tipo: "nao_realizada", Justificativa: "curta"}, "justificativa inválida"},
		{"justificativa on confirmação", NFeManifestacaoInput{ChaveAcesso: nfeChaveProc, Tipo: "confirmacao", Justificativa: "mercadoria recebida conforme pedido"}, "justificativa só é aceita"},
		{"ciência", NFeManifestacaoInput{ChaveAcesso: nfeChaveProc, Tipo: "210210"}, "ciência em lote"},
		{"unknown tipo", NFeManifestacaoInput{ChaveAcesso: nfeChaveProc, Tipo: "aceite"}, "tipo de manifestação inválido"},
		{"emitente role", NFeManifestacaoInput{ChaveAcesso: emitida, Tipo: "confirmacao"}, "não é a destinatária"},
		{"cancelada", NFeManifestacaoInput{ChaveAcesso: nfeChaveCancelada, Tipo: "desconhecimento"}, "NF-e cancelada"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.CNPJ = nfeTestCNPJ
			_, err := env.app.NFe.RegisterManifestacao(ctx, tc.in)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want %q", err, tc.want)
			}
		})
	}
	if len(env.passwords.requests) != 0 || len(fake.lotes) != 0 {
		t.Fatalf("password prompts = %d, lotes = %d; want none before validation passes", len(env.passwords.requests), len(fake.lotes))
	}

	justificativa := "Mercadoria nunca foi entregue no endereço"
	outcome, err := env.app.NFe.RegisterManifestacao(ctx, NFeManifestacaoInput{
		CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, Tipo: "nao-realizada", Justificativa: justificativa,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Status != nfe.ManifestacaoStatusRegistrada || outcome.TpEvento != nfe.TpEventoNaoRealizada {
		t.Errorf("outcome = %+v", outcome)
	}
	if len(fake.lotes) != 1 || len(fake.lotes[0]) != 1 || fake.lotes[0][0].XJust != justificativa {
		t.Errorf("lotes = %+v", fake.lotes)
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Assinatura: Operação não realizada" {
		t.Errorf("password requests = %+v", env.passwords.requests)
	}
	if got := env.manifestacao(nfeChaveProc); got != nfe.ManifestacaoNaoRealizada {
		t.Errorf("manifestacao = %s, want nao_realizada", got)
	}

	// Once a conclusive manifestação is registered, every conclusive type
	// is refused before the password prompt, the same one included.
	for _, in := range []NFeManifestacaoInput{
		{Tipo: "nao_realizada", Justificativa: justificativa},
		{Tipo: "confirmacao"},
		{Tipo: "desconhecimento"},
	} {
		in.CNPJ, in.ChaveAcesso = nfeTestCNPJ, nfeChaveProc
		_, err = env.app.NFe.RegisterManifestacao(ctx, in)
		const want = "NF-e já possui manifestação conclusiva (Operação não realizada)"
		if err == nil || err.Error() != want {
			t.Errorf("%s after nao_realizada: %v, want %q", in.Tipo, err, want)
		}
	}
	if len(env.passwords.requests) != 1 || len(fake.lotes) != 1 {
		t.Errorf("password prompts = %d, lotes = %d; want still 1 and 1", len(env.passwords.requests), len(fake.lotes))
	}
}

func TestNFePlanManifestacaoNeedsNoNetworkOrPassword(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	emitida := env.seedResumo(11, "2026-08-21T10:00:00-03:00", nfeTestCNPJ)
	fake := useFakeSEFAZ(t)
	ctx := context.Background()
	env.app.NFe.now = func() time.Time { return mustTime(t, "2026-09-15T12:00:00-03:00") }

	plan, err := env.app.NFe.PlanManifestacao(ctx, NFeManifestacaoInput{
		CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, Tipo: "nao-realizada", Justificativa: "  Mercadoria nunca\tfoi entregue  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Tipo != nfe.TipoManifestacaoNaoRealizada || plan.Justificativa != "Mercadoria nunca foi entregue" ||
		string(plan.Document.ChaveAcesso) != nfeChaveProc || plan.BlockReason != "" {
		t.Errorf("plan = %+v", plan)
	}
	if plan.ConclusiveDue.IsZero() || plan.DaysLeft <= 0 || plan.TacitlyConfirmed {
		t.Errorf("deadline = %s, days left %d, tacitly confirmed %v", plan.ConclusiveDue, plan.DaysLeft, plan.TacitlyConfirmed)
	}

	env.app.NFe.now = func() time.Time { return mustTime(t, "2027-01-15T12:00:00-03:00") }
	late, err := env.app.NFe.PlanManifestacao(ctx, NFeManifestacaoInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, Tipo: "confirmacao"})
	if err != nil {
		t.Fatal(err)
	}
	if late.DaysLeft >= 0 || !late.TacitlyConfirmed || late.BlockReason != "" {
		t.Errorf("late plan: days left %d, tacitly confirmed %v, block %q", late.DaysLeft, late.TacitlyConfirmed, late.BlockReason)
	}

	blocked, err := env.app.NFe.PlanManifestacao(ctx, NFeManifestacaoInput{CNPJ: nfeTestCNPJ, ChaveAcesso: emitida, Tipo: "confirmacao"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "NF-e " + emitida + ": a empresa não é a destinatária"; blocked.BlockReason != want {
		t.Errorf("BlockReason = %q, want %q", blocked.BlockReason, want)
	}

	if len(env.passwords.requests) != 0 || fake.clients != 0 {
		t.Errorf("password prompts = %d, SEFAZ clients = %d; want none", len(env.passwords.requests), fake.clients)
	}
}

// outcomeCounts counts the outcomes by status.
func outcomeCounts(outcomes []NFeEventOutcome) map[string]int {
	counts := make(map[string]int)
	for _, o := range outcomes {
		counts[o.Status]++
	}
	return counts
}
