package cli

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store"
)

const (
	cteChaveGTVe        = "35260912345678000195640010000001051567890127" // procgtve.xml
	desacordoObservacao = "Carga entregue avariada no destino"
)

func (e *cteTestRoot) desacordoCount() int {
	e.t.Helper()
	var n int
	if err := e.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM cte_desacordos`).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestCTeDesacordo_DryRun(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte.xml", 1)

	if err := env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", desacordoObservacao); err != nil {
		t.Fatalf("desacordo: %v", err)
	}
	out := env.out.String()
	for _, want := range []string{
		"Prestação do Serviço em Desacordo (610110), ambiente de produção",
		"Chave de acesso: " + cteChaveProc + " (modelo 57)",
		"Prazo: 2026-10-17",
		"Observação: " + desacordoObservacao,
		"Nada foi enviado. Use --confirmar",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if env.passwords.requests != 0 || env.desacordoCount() != 0 {
		t.Errorf("password requests %d, desacordos recorded %d; want 0 and 0", env.passwords.requests, env.desacordoCount())
	}
}

func TestCTeDesacordo_JSONDryRun(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte.xml", 1)

	if err := env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", desacordoObservacao, "--json"); err != nil {
		t.Fatalf("desacordo --json: %v", err)
	}
	var plan map[string]any
	decodeJSONOutput(t, env.out, &plan)
	if plan["TpEvento"] != "610110" || plan["ChaveAcesso"] != cteChaveProc || plan["Modelo"] != "57" ||
		plan["Observacao"] != desacordoObservacao || plan["NSeqEvento"] != float64(1) || plan["Due"] == nil {
		t.Errorf("plan = %v", plan)
	}
	if _, ok := plan["DaysLeft"].(float64); !ok {
		t.Errorf("DaysLeft = %v, want a number", plan["DaysLeft"])
	}
	assertNoJSONKeys(t, plan, internalJSONKeys...)
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestCTeDesacordo_RejectsBeforeTheDatabase(t *testing.T) {
	calls, err := runWithoutApp(t, "cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", "curta")
	if !errors.Is(err, app.ErrInvalidObservacao) || calls != 0 {
		t.Errorf("short observação: err = %v, AppFactory calls %d", err, calls)
	}
	calls, err = runWithoutApp(t, "cte", "desacordo", "-c", nfeTestCNPJ, "--chave", "123", "--observacao", desacordoObservacao)
	if !errors.Is(err, dfe.ErrInvalidAccessKey) || calls != 0 {
		t.Errorf("bad chave: err = %v, AppFactory calls %d", err, calls)
	}
}

func TestCTeDesacordo_Blocked(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procgtve.xml", 1)
	err := env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveGTVe, "--observacao", desacordoObservacao, "--confirmar")
	if err == nil || !strings.Contains(err.Error(), "modelo 64") {
		t.Errorf("err = %v, want the GTV-e refused", err)
	}
	if env.passwords.requests != 0 || env.desacordoCount() != 0 {
		t.Errorf("password requests %d, desacordos recorded %d; want 0 and 0", env.passwords.requests, env.desacordoCount())
	}
}

// retEventoCTeSOAP is a CTeRecepcaoEventoV4 answer built by hand from
// retEventoCTe_v4.00.xsd; no real answer was captured.
func retEventoCTeSOAP(cStat int, chave, nProt string) string {
	return `<?xml version="1.0" encoding="utf-8"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteRecepcaoEventoResult xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4">` +
		`<retEventoCTe xmlns="http://www.portalfiscal.inf.br/cte" versao="4.00"><infEvento>` +
		`<tpAmb>1</tpAmb><verAplic>TESTE</verAplic><cOrgao>35</cOrgao><cStat>` + strconv.Itoa(cStat) + `</cStat>` +
		`<xMotivo>resposta ` + strconv.Itoa(cStat) + `</xMotivo><chCTe>` + chave + `</chCTe><tpEvento>610110</tpEvento>` +
		`<xEvento>Prestacao do Servico em Desacordo</xEvento><nSeqEvento>1</nSeqEvento>` +
		`<dhRegEvento>2026-11-19T10:31:02-03:00</dhRegEvento><nProt>` + nProt + `</nProt>` +
		`</infEvento></retEventoCTe></cteRecepcaoEventoResult></soap:Body></soap:Envelope>`
}

func newDesacordoSEFAZRoot(t *testing.T, response string) (*cteTestRoot, *fakeDistribuicao) {
	t.Helper()
	nfeEnv, fake := newNFeFetchTestRoot(t, response)
	env := &cteTestRoot{nfeTestRoot: nfeEnv, cteRepo: store.NewCTeRepository(nfeEnv.db)}
	env.seedCTe("procte.xml", 1)
	return env, fake
}

func TestCTeDesacordo_ConfirmedJSON(t *testing.T) {
	env, fake := newDesacordoSEFAZRoot(t, retEventoCTeSOAP(135, cteChaveProc, "135260000000123"))

	err := env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", desacordoObservacao, "--confirmar", "--json")
	if err != nil {
		t.Fatalf("desacordo --confirmar --json: %v\n%s", err, env.out.String())
	}
	var outcome map[string]any
	decodeJSONOutput(t, env.out, &outcome)
	if outcome["Status"] != "registrada" || outcome["CStat"] != "135" || outcome["Protocolo"] != "135260000000123" || outcome["TpEvento"] != "610110" {
		t.Errorf("outcome = %v", outcome)
	}
	if fake.requests != 1 || env.desacordoCount() != 1 {
		t.Errorf("SEFAZ requests %d, desacordos recorded %d; want 1 and 1", fake.requests, env.desacordoCount())
	}
	events, err := env.cteRepo.ListEventsByChave(context.Background(), cteChaveProc)
	if err != nil || len(events) != 1 || events[0].Observacao != desacordoObservacao {
		t.Errorf("events = %+v, %v; want the registered desacordo", events, err)
	}

	// A second run is refused before the password and the SEFAZ.
	err = env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", desacordoObservacao, "--confirmar")
	if err == nil || !strings.Contains(err.Error(), "já tem prestação em desacordo registrada") || fake.requests != 1 {
		t.Errorf("second send: err = %v, SEFAZ requests %d", err, fake.requests)
	}
}

func TestCTeDesacordo_RejectedPrintsBoth(t *testing.T) {
	env, _ := newDesacordoSEFAZRoot(t, retEventoCTeSOAP(787, cteChaveProc, ""))

	err := env.run("cte", "desacordo", "-c", nfeTestCNPJ, "--chave", cteChaveProc, "--observacao", desacordoObservacao, "--confirmar", "--json")
	if err == nil || !strings.Contains(err.Error(), "cStat 787") {
		t.Fatalf("err = %v, want the rejection", err)
	}
	var outcome map[string]any
	decodeJSONOutput(t, env.out, &outcome)
	if outcome["Status"] != "rejeitada" || outcome["CStat"] != "787" {
		t.Errorf("outcome = %v", outcome)
	}
}
