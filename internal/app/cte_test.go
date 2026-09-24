package app

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	gosync "sync"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
)

// Fixtures of internal/cte/testdata. The mock company (nfeTestCNPJ) is a
// party of each of them.
const (
	cteChaveProc  = "35260912345678000195570010000001011123456784" // procte.xml: tomador and destinatário
	cteChaveToma4 = "35260912345678000195570010000001021234567891" // procte-toma4.xml: autorizado
	cteChaveV200  = "35260912345678000195570010000001031345678907" // procte-v200-toma03.xml: tomador and remetente, tpAmb 2
	cteChaveOS    = "35260912345678000195670010000001041456789014" // procteos.xml: tomador
	cteChaveGTVe  = "35260912345678000195640010000001051567890127" // procgtve.xml: tomador and destinatário
	cteChaveSimp  = "35260912345678000195570020000001061678901234" // proctesimp.xml: tomador

	cteNFeChaveProc  = "35260911222333000181550010000012341123456787" // transported by procte.xml
	cteNFeChaveToma4 = "35260911222333000181550010000012361345678900" // transported by procte-toma4.xml
	cteEmitenteCNPJ  = "12345678000195"
	cteTomadorToma4  = "11223344000186" // the third-party tomador of procte-toma4.xml
	cteRemetenteCNPJ = "11222333000181" // remetente of procte.xml
)

func readCTeFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "cte", "testdata", name)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func cteRepo(e *nfeTestEnv) *store.CTeRepository {
	return e.app.CTe.CTeRepo
}

// seedCTe stores a CT-e fixture for the company (id, cnpj) as if the
// distribution had delivered it at nsu, and returns its blob hash. Event
// fixtures are stored for every company.
func (e *nfeTestEnv) seedCTe(companyID dfe.CompanyID, companyCNPJ, fixture string, nsu int64) string {
	e.t.Helper()
	data := []byte(readCTeFixture(e.t, fixture))
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := e.xml.Store(hash, data); err != nil {
		e.t.Fatal(err)
	}

	ctx := context.Background()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	if strings.HasPrefix(fixture, "proceventocte") {
		ev, err := cte.ParseProcEventoCTe(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		ev.RawHash = hash
		if _, err := cteRepo(e).ApplyEventTx(ctx, tx, store.ApplyCTeEventParams{Event: ev}); err != nil {
			e.t.Fatal(err)
		}
	} else {
		doc, err := cte.ParseProcCTe(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		doc.RawHash = hash
		params := store.ApplyCTeDocumentParams{Document: doc, CompanyID: companyID, CompanyCNPJ: companyCNPJ, NSU: nsu}
		if _, err := cteRepo(e).ApplyDocumentTx(ctx, tx, params); err != nil {
			e.t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatal(err)
	}
	return hash
}

// seedCTeFixtures stores for the mock company the five produção documents,
// the homologação one, the cancelamento of procte.xml and the comprovante de
// entrega of procte-toma4.xml.
func (e *nfeTestEnv) seedCTeFixtures() {
	e.t.Helper()
	for i, fixture := range []string{
		"procte.xml", "procte-toma4.xml", "procte-v200-toma03.xml", "procteos.xml", "procgtve.xml", "proctesimp.xml",
		"proceventocte-cancelamento.xml", "proceventocte-comprovante.xml",
	} {
		e.seedCTe(e.company.ID, e.company.CNPJ, fixture, int64(i+1))
	}
}

func cteChavesOf(docs []cte.CompanyDocument) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		out = append(out, string(d.ChaveAcesso))
	}
	return out
}

func gzipBase64(t *testing.T, data string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(data)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// fakeCTeDistribuicao is an httptest CTeDistribuicaoDFe that answers every
// request with one retDistDFeInt and records the requests.
type fakeCTeDistribuicao struct {
	response string

	mu       gosync.Mutex
	bodies   []string
	contents []string
}

func (f *fakeCTeDistribuicao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.bodies = append(f.bodies, string(body))
	f.contents = append(f.contents, r.Header.Get("Content-Type"))
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	_, _ = io.WriteString(w, f.response)
}

// cteRetDistDFeInt builds the SOAP answer of CTeDistribuicaoDFe with cStat
// 138 and the fixtures as docZips, NSU 1, 2, ...; ultNSU and maxNSU are the
// last NSU, so the pull catches up in one request.
func cteRetDistDFeInt(t *testing.T, fixtures ...[2]string) string {
	t.Helper()
	var docs strings.Builder
	for i, f := range fixtures {
		fmt.Fprintf(&docs, `<docZip NSU="%015d" schema="%s">%s</docZip>`, i+1, f[1], gzipBase64(t, readCTeFixture(t, f[0])))
	}
	last := fmt.Sprintf("%015d", len(fixtures))
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteDistDFeInteresseResponse xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe"><cteDistDFeInteresseResult>` +
		`<retDistDFeInt versao="1.00" xmlns="http://www.portalfiscal.inf.br/cte"><tpAmb>1</tpAmb><verAplic>1.0.0</verAplic>` +
		`<cStat>138</cStat><xMotivo>Documento(s) localizado(s)</xMotivo><dhResp>2026-09-23T10:00:00-03:00</dhResp>` +
		`<ultNSU>` + last + `</ultNSU><maxNSU>` + last + `</maxNSU><loteDistDFeInt>` + docs.String() + `</loteDistDFeInt>` +
		`</retDistDFeInt></cteDistDFeInteresseResult></cteDistDFeInteresseResponse></soap:Body></soap:Envelope>`
}

// TestCTePullEndToEnd pulls through the real SEFAZ client, with the mock PFX,
// from an httptest CTeDistribuicaoDFe.
func TestCTePullEndToEnd(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()
	fake := &fakeCTeDistribuicao{response: cteRetDistDFeInt(t,
		[2]string{"procte.xml", "procCTe_v4.00.xsd"},
		[2]string{"proceventocte-cancelamento.xml", "procEventoCTe_v4.00.xsd"},
	)}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	env.app.SyncManager.SEFAZEndpoints = &sefaz.Endpoints{
		DistribuicaoCTe: server.URL + "/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx",
	}

	result, err := env.app.CTe.Pull(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if result.Status != string(nfse.SyncStatusCompleted) || result.StopReason != string(nfse.SyncStopReasonCaughtUp) {
		t.Errorf("status/reason = %s/%s, want completed/caught_up", result.Status, result.StopReason)
	}
	if result.DocumentsSaved != 1 || result.EventsSaved != 1 || result.LastNSU != 2 || result.MaxNSU == nil || *result.MaxNSU != 2 {
		t.Errorf("result = %+v, want 1 document, 1 event and NSU 2", result)
	}
	if result.RequestsLastHour != 1 || result.RequestBudget != 20 || result.NextAllowedAt == nil {
		t.Errorf("limits = %d of %d, next %v; want 1 of 20 and a wait", result.RequestsLastHour, result.RequestBudget, result.NextAllowedAt)
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Sincronização CT-e" {
		t.Errorf("password requests = %+v, want one for Sincronização CT-e", env.passwords.requests)
	}

	if len(fake.bodies) != 1 {
		t.Fatalf("SEFAZ requests = %d, want 1", len(fake.bodies))
	}
	for _, want := range []string{"<cteDistDFeInteresse", "<cUFAutor>35</cUFAutor>", "<CNPJ>" + nfeTestCNPJ + "</CNPJ>", "<ultNSU>000000000000000</ultNSU>"} {
		if !strings.Contains(fake.bodies[0], want) {
			t.Errorf("request body lacks %s:\n%s", want, fake.bodies[0])
		}
	}
	if !strings.Contains(fake.contents[0], "cteDistDFeInteresse") {
		t.Errorf("Content-Type = %q, want the CT-e action", fake.contents[0])
	}

	docs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("documents = %d, want 1", len(docs))
	}
	doc := docs[0]
	if doc.ChaveAcesso != cteChaveProc || doc.CompanyRole != cte.CompanyRoleTomador || doc.Situacao != cte.SituacaoCancelada || doc.TpAmb != "1" || doc.EventCount != 1 {
		t.Errorf("document = %s %s %s tpAmb %s events %d; want procte.xml, tomador, cancelada, 1, 1",
			doc.ChaveAcesso, doc.CompanyRole, doc.Situacao, doc.TpAmb, doc.EventCount)
	}

	status, err := env.app.CTe.Status(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if status.LastNSU != 2 || status.TotalTomador != 1 || status.InitialSyncDoneAt == nil || status.BlockedReason != string(nfse.SyncStopReasonCaughtUp) {
		t.Errorf("status = %+v, want NSU 2, 1 tomador, initial sync done and the caught_up wait", status)
	}

	// Caught up: the next pull waits an hour and asks for no password.
	if _, err := env.app.CTe.Pull(ctx, nfeTestCNPJ); !errors.Is(err, ErrSourceBlocked) {
		t.Errorf("second Pull = %v, want ErrSourceBlocked", err)
	}
	if len(env.passwords.requests) != 1 || len(fake.bodies) != 1 {
		t.Errorf("after the blocked pull: %d prompts, %d requests; want 1 and 1", len(env.passwords.requests), len(fake.bodies))
	}

	// The NF-e source is untouched.
	nfeStatus, err := env.app.NFe.Status(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if nfeStatus.RequestsLastHour != 0 || nfeStatus.NextAllowedAt != nil || nfeStatus.LastNSU != 0 {
		t.Errorf("NF-e status = %+v, want untouched by the CT-e pull", nfeStatus)
	}
}

func TestCTeStatusCountsByRole(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()

	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	if err := env.app.CTe.SyncRepo.SetBlockedUntil(ctx, env.company.ID, nfse.SyncSourceCTe, until, nfse.SyncStopReasonConsumoIndevido); err != nil {
		t.Fatal(err)
	}

	status, err := env.app.CTe.Status(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	want := CTeStatusResult{
		CompanyName:   "Empresa Mock",
		CNPJ:          nfeTestCNPJ,
		UF:            "SP",
		TpAmb:         "1",
		NextAllowedAt: &until,
		BlockedReason: string(nfse.SyncStopReasonConsumoIndevido),
		RequestBudget: 20,
		TotalTomador:  4, // procte, OS, GTV-e and Simplificado; the homologação one is not counted
		TotalOutros:   1, // procte-toma4 as autorizado
	}
	if status.NextAllowedAt == nil || !status.NextAllowedAt.Equal(until) {
		t.Errorf("NextAllowedAt = %v, want %v", status.NextAllowedAt, until)
	}
	status.NextAllowedAt = want.NextAllowedAt
	if status != want {
		t.Errorf("Status =\n%+v\nwant\n%+v", status, want)
	}

	if _, err := env.app.CTe.Pull(ctx, nfeTestCNPJ); !errors.Is(err, ErrSourceBlocked) {
		t.Errorf("Pull while blocked = %v, want ErrSourceBlocked", err)
	}
	if len(env.passwords.requests) != 0 {
		t.Errorf("password prompts = %d, want 0", len(env.passwords.requests))
	}
}

func TestCTeListDocumentsFilters(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()

	all := []string{cteChaveSimp, cteChaveGTVe, cteChaveOS, cteChaveToma4, cteChaveProc}
	tests := []struct {
		name string
		in   ListCTeInput
		want []string
	}{
		{"all of the environment, newest first", ListCTeInput{}, all},
		{"competence", ListCTeInput{Competence: "2026-08"}, []string{}},
		{"situacao", ListCTeInput{Situacao: "cancelada"}, []string{cteChaveProc}},
		{"primary role", ListCTeInput{Role: "autorizado"}, []string{cteChaveToma4}},
		{"secondary role", ListCTeInput{Role: "destinatario"}, []string{cteChaveGTVe, cteChaveProc}},
		{"modelo 67", ListCTeInput{Modelo: "67"}, []string{cteChaveOS}},
		{"modelo 64", ListCTeInput{Modelo: "64"}, []string{cteChaveGTVe}},
		{"emitente", ListCTeInput{EmitenteCNPJ: "12.345.678/0001-95"}, all},
		{"tomador", ListCTeInput{TomadorCNPJ: cteTomadorToma4}, []string{cteChaveToma4}},
		{"nfe chave", ListCTeInput{NFeChave: cteNFeChaveProc}, []string{cteChaveProc}},
		{"nfe chave of toma4", ListCTeInput{NFeChave: cteNFeChaveToma4}, []string{cteChaveToma4}},
		{"chaves", ListCTeInput{ChavesAcesso: []string{cteChaveOS, cteChaveV200}}, []string{cteChaveOS}},
		{"limit", ListCTeInput{Limit: 1}, []string{cteChaveSimp}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.CNPJ = nfeTestCNPJ
			docs, err := env.app.CTe.ListDocuments(ctx, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got := cteChavesOf(docs); !slices.Equal(got, tc.want) {
				t.Errorf("chaves = %v, want %v", got, tc.want)
			}
		})
	}

	for name, in := range map[string]ListCTeInput{
		"situação": {CNPJ: nfeTestCNPJ, Situacao: "rascunho"},
		"papel":    {CNPJ: nfeTestCNPJ, Role: "transportador"},
		"modelo":   {CNPJ: nfeTestCNPJ, Modelo: "55"},
	} {
		if _, err := env.app.CTe.ListDocuments(ctx, in); !errors.Is(err, dfe.ErrInvalidEnum) {
			t.Errorf("invalid %s: err = %v, want dfe.ErrInvalidEnum", name, err)
		}
	}
	if _, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ, NFeChave: "123"}); !errors.Is(err, dfe.ErrInvalidAccessKey) {
		t.Errorf("invalid NF-e chave: err = %v, want dfe.ErrInvalidAccessKey", err)
	}

	events, err := env.app.CTe.ListEvents(ctx, nfeTestCNPJ, cteChaveProc)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].TpEvento != cte.TpEventoCancelamento || events[0].Type != cte.EventTypeCancelamento {
		t.Errorf("events = %+v, want the cancelamento", events)
	}
	if _, err := env.app.CTe.ListEvents(ctx, nfeTestCNPJ, cteNFeChaveProc); !errors.Is(err, cte.ErrDocumentNotFound) {
		t.Errorf("ListEvents of a chave the company does not see: err = %v, want cte.ErrDocumentNotFound", err)
	}
}

func TestCTeTestConnectionOnlyChecksCTeTLS(t *testing.T) {
	env := newNFeTestEnv(t)
	stub := useFakeSEFAZ(t)

	result, err := env.app.CTe.TestConnection(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if !result.CertLoaded || !result.EndpointReached || stub.cteTLSCalls != 1 || stub.tlsCalls != 0 {
		t.Errorf("result = %+v, CT-e TLS checks = %d, NF-e TLS checks = %d", result, stub.cteTLSCalls, stub.tlsCalls)
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Teste de conexão CT-e" {
		t.Errorf("password requests = %+v", env.passwords.requests)
	}
	status, err := env.app.CTe.Status(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if status.RequestsLastHour != 0 {
		t.Errorf("requests last hour = %d, want 0: the connection test spends no budget", status.RequestsLastHour)
	}

	stub.tlsErr = errors.New("handshake failed")
	result, err = env.app.CTe.TestConnection(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if result.EndpointReached || !strings.Contains(result.StatusExplanation, "handshake failed") {
		t.Errorf("failed check result = %+v", result)
	}
}

// TestCTeEnvironmentSwitchHidesAndShowsDocuments switches the company to
// homologação and back: each environment lists, counts, shows events of and
// exports only its own CT-e.
func TestCTeEnvironmentSwitchHidesAndShowsDocuments(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()

	switchTo := func(environment nfse.Environment) {
		t.Helper()
		err := env.app.Companies.UpdateCompany(ctx, company.UpdateCompanyInput{
			CNPJ:            env.company.CNPJ,
			Name:            env.company.Name,
			Environment:     environment,
			UF:              env.company.UF,
			SyncStartPolicy: env.company.SyncStartPolicy,
			SyncStartDate:   env.company.SyncStartDate,
		})
		if err != nil {
			t.Fatalf("switch to %s: %v", environment, err)
		}
	}
	listed := func() []string {
		t.Helper()
		docs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ})
		if err != nil {
			t.Fatal(err)
		}
		return cteChavesOf(docs)
	}

	switchTo(nfse.EnvironmentRestricted)
	if got := listed(); !slices.Equal(got, []string{cteChaveV200}) {
		t.Errorf("homologação lists %v, want only the homologação CT-e", got)
	}
	status, err := env.app.CTe.Status(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if status.TpAmb != "2" || status.TotalTomador != 1 || status.TotalOutros != 0 {
		t.Errorf("homologação status = %+v, want tpAmb 2 and one tomador", status)
	}
	if _, err := env.app.CTe.ListEvents(ctx, nfeTestCNPJ, cteChaveProc); !errors.Is(err, cte.ErrDocumentNotFound) {
		t.Errorf("ListEvents of a produção CT-e in homologação = %v, want ErrDocumentNotFound", err)
	}
	export, err := env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, OutPath: filepath.Join(t.TempDir(), "hom.zip")})
	if err != nil || export.ExportedCount != 1 {
		t.Errorf("homologação export = %d, %v; want the homologação CT-e only", export.ExportedCount, err)
	}

	switchTo(nfse.EnvironmentProduction)
	if got := listed(); len(got) != 5 || slices.Contains(got, cteChaveV200) {
		t.Errorf("back in produção lists %v, want the 5 produção CT-e", got)
	}
	if events, err := env.app.CTe.ListEvents(ctx, nfeTestCNPJ, cteChaveProc); err != nil || len(events) != 1 {
		t.Errorf("produção events = %d, %v; want the cancelamento", len(events), err)
	}
}
