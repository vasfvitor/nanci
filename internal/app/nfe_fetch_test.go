package app

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// nfeRetDistDFeInt builds the SOAP answer of NFeDistribuicaoDFe with cStat
// and the docZips, each a [NSU, schema, fixture] triple.
func nfeRetDistDFeInt(t *testing.T, cStat int, xMotivo string, docs ...[3]string) string {
	t.Helper()
	var lote strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&lote, `<docZip NSU="%s" schema="%s">%s</docZip>`, d[0], d[1], gzipBase64(t, readNFeFixture(t, d[2])))
	}
	body := `<tpAmb>1</tpAmb><verAplic>1.0.0</verAplic>` +
		fmt.Sprintf(`<cStat>%d</cStat><xMotivo>%s</xMotivo>`, cStat, xMotivo) +
		`<dhResp>2026-09-23T10:00:00-03:00</dhResp><ultNSU>000000000000000</ultNSU><maxNSU>000000000000100</maxNSU>`
	if lote.Len() > 0 {
		body += `<loteDistDFeInt>` + lote.String() + `</loteDistDFeInt>`
	}
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeDistDFeInteresseResponse xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe"><nfeDistDFeInteresseResult>` +
		`<retDistDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` + body + `</retDistDFeInt>` +
		`</nfeDistDFeInteresseResult></nfeDistDFeInteresseResponse></soap:Body></soap:Envelope>`
}

// useFakeNFeDistribuicao points the NF-e distribution at an httptest server
// that answers every request with response.
func (e *nfeTestEnv) useFakeNFeDistribuicao(response string) *fakeCTeDistribuicao {
	e.t.Helper()
	fake := &fakeCTeDistribuicao{response: response}
	server := httptest.NewServer(fake)
	e.t.Cleanup(server.Close)
	e.app.SyncManager.SEFAZEndpoints = &sefaz.Endpoints{
		Distribuicao: server.URL + "/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
	}
	return fake
}

func (e *nfeTestEnv) fetch(chave string) (FetchByChaveResult, error) {
	e.t.Helper()
	return e.app.NFe.FetchByChave(context.Background(), FetchByChaveInput{CNPJ: nfeTestCNPJ, ChaveAcesso: chave})
}

func (e *nfeTestEnv) countSyncRequests() int {
	e.t.Helper()
	var count int
	if err := e.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sync_requests WHERE source = 'nfe'`).Scan(&count); err != nil {
		e.t.Fatal(err)
	}
	return count
}

func TestNFeFetchByChaveCompleta(t *testing.T) {
	env := newNFeTestEnv(t)
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 138, "Documento localizado",
		[3]string{"000000000000042", "procNFe_v4.00.xsd", "procnfe.xml"}))

	result, err := env.fetch(nfeChaveProc)
	if err != nil {
		t.Fatalf("FetchByChave: %v", err)
	}
	if result.ChaveAcesso != nfeChaveProc || result.Outcome != FetchOutcomeCompleta || result.Message != "" {
		t.Errorf("result = %+v, want completa", result)
	}
	if result.RequestsLastHour != 1 || result.RequestBudget != 20 {
		t.Errorf("limits = %d of %d, want 1 of 20", result.RequestsLastHour, result.RequestBudget)
	}
	if env.countSyncRequests() != 1 {
		t.Errorf("sync_requests = %d, want 1", env.countSyncRequests())
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Busca de NF-e pela chave" {
		t.Errorf("password requests = %+v, want one for the fetch", env.passwords.requests)
	}

	if len(fake.bodies) != 1 {
		t.Fatalf("SEFAZ requests = %d, want 1", len(fake.bodies))
	}
	for _, want := range []string{"<consChNFe><chNFe>" + nfeChaveProc + "</chNFe></consChNFe>", "<cUFAutor>35</cUFAutor>", "<CNPJ>" + nfeTestCNPJ + "</CNPJ>", "<tpAmb>1</tpAmb>"} {
		if !strings.Contains(fake.bodies[0], want) {
			t.Errorf("request body lacks %s:\n%s", want, fake.bodies[0])
		}
	}

	doc := env.nfeDocument(nfeChaveProc)
	if doc.Completeness != nfe.CompletenessCompleta || doc.CompanyRole != nfe.CompanyRoleDestinatario {
		t.Errorf("document = %s %s, want completa destinatario", doc.Completeness, doc.CompanyRole)
	}
	var firstNSU int64
	if err := env.db.QueryRowContext(context.Background(), `SELECT first_seen_nsu FROM company_nfe_documents WHERE company_id = ?`, string(env.company.ID)).Scan(&firstNSU); err != nil {
		t.Fatal(err)
	}
	if firstNSU != 42 {
		t.Errorf("first_seen_nsu = %d, want the docZip NSU 42", firstNSU)
	}

	// The fetch does not move the distribution cursor nor start a run.
	status, err := env.app.NFe.Status(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastNSU != 0 || status.NextAllowedAt != nil || status.RequestsLastHour != 1 {
		t.Errorf("status = %+v, want cursor 0, no block and 1 request", status)
	}
}

func TestNFeFetchByChaveResumo(t *testing.T) {
	env := newNFeTestEnv(t)
	env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 138, "Documento localizado",
		[3]string{"000000000000007", "resNFe_v1.01.xsd", "resnfe.xml"}))

	result, err := env.fetch(nfeChaveProc)
	if err != nil {
		t.Fatalf("FetchByChave: %v", err)
	}
	if result.Outcome != FetchOutcomeResumo || !strings.Contains(result.Message, "Ciência da Operação") {
		t.Errorf("result = %+v, want resumo with the ciência hint", result)
	}
	if doc := env.nfeDocument(nfeChaveProc); doc.Completeness != nfe.CompletenessResumo {
		t.Errorf("completeness = %s, want resumo", doc.Completeness)
	}
}

func TestNFeFetchByChaveNotFound(t *testing.T) {
	env := newNFeTestEnv(t)
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 137, "Nenhum documento localizado"))

	result, err := env.fetch(nfeChaveProc)
	if err != nil {
		t.Fatalf("FetchByChave: %v", err)
	}
	if result.Outcome != FetchOutcomeNotFound || result.Message != "NF-e não localizada ou sem acesso para a empresa" {
		t.Errorf("result = %+v, want not_found", result)
	}
	if len(fake.bodies) != 1 || env.countSyncRequests() != 1 || result.RequestsLastHour != 1 {
		t.Errorf("requests = %d sent, %d recorded, %d reported; want 1", len(fake.bodies), env.countSyncRequests(), result.RequestsLastHour)
	}
}

func TestNFeFetchByChaveConsumoIndevido(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 656, "Consumo Indevido"))

	_, err := env.fetch(nfeChaveProc)
	var blocked *sync.BlockedError
	if !errors.As(err, &blocked) || blocked.Reason != syncstate.SyncStopReasonConsumoIndevido || !blocked.Until.After(time.Now().Add(50*time.Minute)) {
		t.Fatalf("FetchByChave = %v, want a consumo_indevido block of about an hour", err)
	}

	limits, err := env.app.SyncManager.SourceLimits(ctx, env.company, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if limits.NextAllowedAt == nil || limits.BlockedReason != syncstate.SyncStopReasonConsumoIndevido {
		t.Errorf("limits = %+v, want the consumo_indevido block stored", limits)
	}

	if _, err := env.fetch(nfeChaveProc); !errors.Is(err, ErrSourceBlocked) {
		t.Errorf("second FetchByChave = %v, want ErrSourceBlocked", err)
	}
	if len(fake.bodies) != 1 || len(env.passwords.requests) != 1 {
		t.Errorf("after the blocked fetch: %d requests, %d prompts; want 1 and 1", len(fake.bodies), len(env.passwords.requests))
	}
}

func TestNFeFetchByChaveRejections(t *testing.T) {
	tests := []struct {
		cStat int
		want  string
	}{
		{640, "a empresa não é destinatária, transportadora nem autorizada nesta NF-e (cStat 640)"},
		{641, "a SEFAZ não entrega ao emitente a própria NF-e; importe o XML (cStat 641)"},
		{632, "chave fora da janela de 90 dias da distribuição (cStat 632)"},
		{653, "NF-e cancelada (cStat 653)"},
		{593, "cStat 593: Rejeição: CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.cStat), func(t *testing.T) {
			env := newNFeTestEnv(t)
			xMotivo := "Rejeição: CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"
			env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, tc.cStat, xMotivo))

			_, err := env.fetch(nfeChaveProc)
			if err == nil || err.Error() != tc.want {
				t.Errorf("FetchByChave = %v, want %q", err, tc.want)
			}
			if env.countSyncRequests() != 1 {
				t.Errorf("sync_requests = %d, want 1", env.countSyncRequests())
			}
		})
	}
}

// TestNFeFetchByChaveIgnoresCaughtUp: the hour of wait after the queue
// caught up applies to distNSU only.
func TestNFeFetchByChaveIgnoresCaughtUp(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 137, "Nenhum documento localizado"))
	if err := env.app.NFe.SyncRepo.SetBlockedUntil(ctx, env.company.ID, syncstate.SyncSourceNFe, env.company.Environment, time.Now().Add(time.Hour), syncstate.SyncStopReasonCaughtUp); err != nil {
		t.Fatal(err)
	}

	if _, err := env.fetch(nfeChaveProc); err != nil {
		t.Fatalf("FetchByChave with a caught_up wait: %v", err)
	}
	if len(fake.bodies) != 1 {
		t.Errorf("SEFAZ requests = %d, want 1", len(fake.bodies))
	}
}

func TestNFeFetchByChaveRateBudget(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 137, "Nenhum documento localizado"))
	now := time.Now().UTC()
	for i := range 20 {
		if err := env.app.NFe.SyncRepo.RecordRequest(ctx, env.company.ID, syncstate.SyncSourceNFe, env.company.Environment, now.Add(-time.Duration(30-i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	// The budget is spent after the password prompt, so the first refusal
	// comes after one prompt; it stores the rate_budget block.
	_, err := env.fetch(nfeChaveProc)
	var blocked *sync.BlockedError
	if !errors.As(err, &blocked) || blocked.Reason != syncstate.SyncStopReasonRateBudget {
		t.Fatalf("FetchByChave = %v, want a rate_budget block", err)
	}
	if len(fake.bodies) != 0 || len(env.passwords.requests) != 1 || env.countSyncRequests() != 20 {
		t.Errorf("%d requests, %d prompts, %d recorded; want 0, 1 and 20", len(fake.bodies), len(env.passwords.requests), env.countSyncRequests())
	}

	// The stored block refuses the next fetch before the prompt.
	if _, err := env.fetch(nfeChaveProc); !errors.Is(err, ErrSourceBlocked) {
		t.Errorf("second FetchByChave = %v, want ErrSourceBlocked", err)
	}
	if len(fake.bodies) != 0 || len(env.passwords.requests) != 1 || env.countSyncRequests() != 20 {
		t.Errorf("after the blocked fetch: %d requests, %d prompts, %d recorded; want 0, 1 and 20", len(fake.bodies), len(env.passwords.requests), env.countSyncRequests())
	}
}

// cancelledPassword answers every prompt as the desktop does when the user
// closes the password dialog.
type cancelledPassword struct{ prompts int }

func (p *cancelledPassword) GetCertPassword(context.Context, CertPasswordRequest) ([]byte, error) {
	p.prompts++
	return nil, ErrOperationCanceled
}

func TestNFeFetchByChaveCancelledPasswordSpendsNoRequest(t *testing.T) {
	env := newNFeTestEnv(t)
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 137, "Nenhum documento localizado"))
	cancelled := &cancelledPassword{}
	env.app.NFe.Certificates.Passwords = cancelled

	if _, err := env.fetch(nfeChaveProc); !errors.Is(err, ErrOperationCanceled) {
		t.Fatalf("FetchByChave = %v, want ErrOperationCanceled", err)
	}
	if cancelled.prompts != 1 || len(fake.bodies) != 0 || env.countSyncRequests() != 0 {
		t.Errorf("%d prompts, %d requests, %d recorded; want 1, 0 and 0", cancelled.prompts, len(fake.bodies), env.countSyncRequests())
	}
}

func TestNFeFetchByChaveRefusesBeforeSending(t *testing.T) {
	env := newNFeTestEnv(t)
	fake := env.useFakeNFeDistribuicao(nfeRetDistDFeInt(t, 137, "Nenhum documento localizado"))

	release, err := env.app.SyncManager.ReserveSource(env.company.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.fetch(nfeChaveProc); !errors.Is(err, ErrSyncRunning) {
		t.Errorf("FetchByChave during a pull = %v, want ErrSyncRunning", err)
	}
	release()

	if _, err := env.fetch(cteChaveProc); err == nil || err.Error() != "só NF-e (modelo 55) pode ser buscada pela chave" {
		t.Errorf("FetchByChave of a CT-e chave = %v, want the modelo error", err)
	}
	if _, err := env.fetch("123"); err == nil || !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Errorf("FetchByChave of an invalid chave = %v", err)
	}
	if len(fake.bodies) != 0 || len(env.passwords.requests) != 0 || env.countSyncRequests() != 0 {
		t.Errorf("%d requests, %d prompts, %d recorded; want none", len(fake.bodies), len(env.passwords.requests), env.countSyncRequests())
	}
}
