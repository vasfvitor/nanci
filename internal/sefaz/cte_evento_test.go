package sefaz

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/uf"
)

const (
	testChaveCTe   = "35260912345678000195570010000001011123456784" // model 57, SP
	testChaveCTeOS = "35260912345678000195670010000001041456789014" // model 67
	testChaveGTVe  = "35260912345678000195640010000001051567890127" // model 64
)

func desacordoEvento() EventoDesacordo {
	return EventoDesacordo{
		ChaveAcesso: testChaveCTe,
		CNPJ:        testCNPJ,
		NSeqEvento:  1,
		DhEvento:    time.Date(2026, 11, 19, 10, 30, 58, 0, time.FixedZone("", -3*60*60)),
		XObs:        `Carga entregue "avariada" & fora do prazo`,
	}
}

func TestSignEventoDesacordo(t *testing.T) {
	tlsCert := loadMockCertificate(t)
	signer, err := NewSigner(tlsCert)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	signed, err := signer.SignEventoDesacordo(desacordoEvento(), TpAmbProducao)
	if err != nil {
		t.Fatalf("SignEventoDesacordo: %v", err)
	}

	// MOC CT-e 4.00, 5.1.1 and evPrestDesacordo_v4.00.xsd: no verEvento,
	// cOrgao from the chave, nSeqEvento with 3 digits in the Id.
	prefix := `<eventoCTe xmlns="http://www.portalfiscal.inf.br/cte" versao="4.00">` +
		`<infEvento Id="ID610110` + testChaveCTe + `001">` +
		`<cOrgao>35</cOrgao><tpAmb>1</tpAmb><CNPJ>70860312000150</CNPJ>` +
		`<chCTe>` + testChaveCTe + `</chCTe>` +
		`<dhEvento>2026-11-19T10:30:58-03:00</dhEvento><tpEvento>610110</tpEvento><nSeqEvento>1</nSeqEvento>` +
		`<detEvento versaoEvento="4.00"><evPrestDesacordo>` +
		`<descEvento>Prestacao do Servico em Desacordo</descEvento><indDesacordoOper>1</indDesacordoOper>` +
		`<xObs>Carga entregue "avariada" &amp; fora do prazo</xObs>` +
		`</evPrestDesacordo></detEvento></infEvento>` +
		`<Signature xmlns="http://www.w3.org/2000/09/xmldsig#"><SignedInfo>`
	if !strings.HasPrefix(string(signed), prefix) {
		t.Errorf("signed eventoCTe does not start with the expected infEvento:\n%s", signed)
	}
	if !strings.HasSuffix(string(signed), `</Signature></eventoCTe>`) {
		t.Errorf("Signature must be the last child of eventoCTe:\n%s", signed)
	}
	if id := "ID610110" + testChaveCTe + "001"; len(id) != 55 {
		t.Errorf("Id has %d characters, want 55", len(id))
	}
	verifySignedEventoIn(t, signed, tlsCert.Leaf, cteNamespace)

	os := desacordoEvento()
	os.ChaveAcesso = testChaveCTeOS
	if _, err := signer.SignEventoDesacordo(os, TpAmbHomologacao); err != nil {
		t.Errorf("CT-e OS: %v", err)
	}
}

func TestSignEventoDesacordo_Validation(t *testing.T) {
	signer := newMockSigner(t)
	tests := map[string]func(*EventoDesacordo){
		"GTV-e":           func(e *EventoDesacordo) { e.ChaveAcesso = testChaveGTVe },
		"NF-e key":        func(e *EventoDesacordo) { e.ChaveAcesso = testChave },
		"bad key":         func(e *EventoDesacordo) { e.ChaveAcesso = "123" },
		"bad CNPJ":        func(e *EventoDesacordo) { e.CNPJ = "11111111111111" },
		"nSeqEvento 0":    func(e *EventoDesacordo) { e.NSeqEvento = 0 },
		"nSeqEvento 21":   func(e *EventoDesacordo) { e.NSeqEvento = 21 },
		"no dhEvento":     func(e *EventoDesacordo) { e.DhEvento = time.Time{} },
		"short xObs":      func(e *EventoDesacordo) { e.XObs = "curta demais" },
		"no xObs":         func(e *EventoDesacordo) { e.XObs = "" },
		"curly quote obs": func(e *EventoDesacordo) { e.XObs = "Carga entregue “avariada” no destino" },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			e := desacordoEvento()
			change(&e)
			if _, err := signer.SignEventoDesacordo(e, TpAmbProducao); err == nil {
				t.Error("SignEventoDesacordo = nil error, want a validation error")
			}
		})
	}
	if _, err := signer.SignEventoDesacordo(desacordoEvento(), "3"); err == nil {
		t.Error("tpAmb 3: want an error")
	}
}

func TestCheckXObsDesacordo(t *testing.T) {
	tests := []struct {
		xObs string
		ok   bool
	}{
		{"Mercadoria não entregue", true},
		{"Avaria: 10% da carga; ver ocorrência nº 12", true},
		{strings.Repeat("a", XObsDesacordoMin), true},
		{strings.Repeat("ç", XObsDesacordoMax), true},
		{strings.Repeat("a", XObsDesacordoMin-1), false},
		{strings.Repeat("a", XObsDesacordoMax+1), false},
		{" Mercadoria não entregue", false},
		{"Mercadoria não entregue ", false},
		{"Mercadoria não\nentregue", false},
		{"Mercadoria não entregue €", false},
		{"Mercadoria não entregue 🚚", false},
	}
	for _, tt := range tests {
		if err := CheckXObsDesacordo(tt.xObs); (err == nil) != tt.ok {
			t.Errorf("CheckXObsDesacordo(%q) = %v, want ok %t", tt.xObs, err, tt.ok)
		}
	}
}

// retEventoCTeResponse builds an answer of CTeRecepcaoEventoV4 by hand, from
// retEventoCTe_v4.00.xsd; no real answer was captured. The element around
// retEventoCTe is a guess, and the parser does not depend on it.
func retEventoCTeResponse(cStat int, xMotivo, chave, nProt string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteRecepcaoEventoResult xmlns="` + wsdlRecepcaoEventoCTe + `">` +
		`<retEventoCTe xmlns="http://www.portalfiscal.inf.br/cte" versao="4.00"><infEvento>` +
		`<tpAmb>1</tpAmb><verAplic>TESTE</verAplic><cOrgao>35</cOrgao>` +
		`<cStat>` + strconv.Itoa(cStat) + `</cStat><xMotivo>` + xMotivo + `</xMotivo>` +
		`<chCTe>` + chave + `</chCTe><tpEvento>610110</tpEvento>` +
		`<xEvento>Prestacao do Servico em Desacordo</xEvento><nSeqEvento>1</nSeqEvento>` +
		`<dhRegEvento>2026-11-19T10:31:02-03:00</dhRegEvento><nProt>` + nProt + `</nProt>` +
		`</infEvento></retEventoCTe></cteRecepcaoEventoResult></soap:Body></soap:Envelope>`
}

func ufCodeForTest(sigla string) (string, bool) {
	code, ok := uf.Code(sigla)
	return strconv.Itoa(code), ok
}

func TestEnviarEventoDesacordo(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusOK,
		retEventoCTeResponse(CStatEventoCTeVinculado, "Evento registrado e vinculado a CT-e", testChaveCTe, "135260000000123"), ClientConfig{})

	result, err := client.EnviarEventoDesacordo(context.Background(), newMockSigner(t), desacordoEvento())
	if err != nil {
		t.Fatalf("EnviarEventoDesacordo: %v", err)
	}
	if result.CStat != CStatEventoCTeVinculado || result.Protocolo != "135260000000123" ||
		result.XMotivo != "Evento registrado e vinculado a CT-e" || result.DhRegEvento == nil {
		t.Errorf("result = %+v", result)
	}
	if !IsRegisteredCTe(result.CStat) || IsAlreadyDoneCTe(result.CStat) {
		t.Errorf("cStat %d: registered %t, already done %t", result.CStat, IsRegisteredCTe(result.CStat), IsAlreadyDoneCTe(result.CStat))
	}
	if !strings.HasPrefix(string(result.SignedEvento), "<eventoCTe ") || !strings.HasPrefix(string(result.RetEvento), "<retEventoCTe ") {
		t.Errorf("SignedEvento or RetEvento not kept:\n%s\n%s", result.SignedEvento, result.RetEvento)
	}
	proc := string(ProcEventoCTe(result.SignedEvento, result.RetEvento))
	if !strings.HasPrefix(proc, `<procEventoCTe xmlns="http://www.portalfiscal.inf.br/cte" versao="4.00"><eventoCTe `) ||
		!strings.HasSuffix(proc, `</retEventoCTe></procEventoCTe>`) {
		t.Errorf("procEventoCTe = %s", proc)
	}

	requests := fake.captured()
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Path != "/CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx" {
		t.Errorf("path = %s", req.Path)
	}
	if ct := req.Header.Get("Content-Type"); !strings.Contains(ct, `action="`+actionRecepcaoEventoCTe+`"`) {
		t.Errorf("Content-Type = %s", ct)
	}
	if !strings.Contains(req.Body, `<soap:Body><cteDadosMsg xmlns="`+wsdlRecepcaoEventoCTe+`"><eventoCTe `) ||
		strings.Contains(req.Body, "envEvento") || strings.Contains(req.Body, "idLote") {
		t.Errorf("body is not one eventoCTe in cteDadosMsg:\n%s", req.Body)
	}
}

func TestEnviarEventoDesacordo_Answers(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantErr    bool
		wantCStat  int
		registered bool
		already    bool
	}{
		{"duplicate", http.StatusOK, retEventoCTeResponse(CStatDuplicidadeEventoCTe, "Rejeicao: Duplicidade de evento", testChaveCTe, ""), false, 631, false, true},
		{"not tomador", http.StatusOK, retEventoCTeResponse(CStatAutorNaoTomador, "Rejeicao: Autor do evento diverge do tomador", testChaveCTe, ""), false, 755, false, false},
		{"out of time", http.StatusOK, retEventoCTeResponse(CStatDesacordoForaDoPrazo, "Rejeicao: Prazo excedido", testChaveCTe, ""), false, 787, false, false},
		{"four-digit cStat", http.StatusOK, retEventoCTeResponse(1001, "Rejeicao nova", testChaveCTe, ""), false, 1001, false, false},
		{"other chave", http.StatusOK, retEventoCTeResponse(CStatEventoCTeVinculado, "ok", testChaveCTeOS, "135260000000123"), true, 0, false, false},
		{"no retEventoCTe", http.StatusOK, `<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body/></soap:Envelope>`, true, 0, false, false},
		{"SOAP fault", http.StatusInternalServerError, `<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body><soap:Fault><soap:Code><soap:Value>soap:Receiver</soap:Value></soap:Code><soap:Reason><soap:Text xml:lang="pt">falhou</soap:Text></soap:Reason></soap:Fault></soap:Body></soap:Envelope>`, true, 0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newFakeClient(t, tt.status, tt.body, ClientConfig{})
			result, err := client.EnviarEventoDesacordo(context.Background(), newMockSigner(t), desacordoEvento())
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %t", err, tt.wantErr)
			}
			if tt.name == "SOAP fault" {
				var fault *FaultError
				if !errors.As(err, &fault) {
					t.Errorf("err = %v, want a *FaultError", err)
				}
			}
			if err != nil {
				return
			}
			if result.CStat != tt.wantCStat || IsRegisteredCTe(result.CStat) != tt.registered || IsAlreadyDoneCTe(result.CStat) != tt.already {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestCTeRecepcaoEventoURL(t *testing.T) {
	for sigla := range cteAutorizadoraBySigla {
		code, _ := ufCodeForTest(sigla)
		for _, tpAmb := range []string{TpAmbProducao, TpAmbHomologacao} {
			url, err := CTeRecepcaoEventoURL(code, tpAmb)
			if err != nil || !strings.HasPrefix(url, "https://") || !strings.Contains(url, "CTeRecepcaoEventoV4") {
				t.Errorf("%s (%s) tpAmb %s = %q, %v", sigla, code, tpAmb, url, err)
			}
		}
	}
	if n := len(cteAutorizadoraBySigla); n != 27 {
		t.Errorf("table has %d UF, want 27", n)
	}

	tests := []struct{ cUF, tpAmb, want string }{
		{"35", TpAmbProducao, cteSVSP.producao},       // SP
		{"26", TpAmbProducao, cteSVSP.producao},       // PE uses SVSP
		{"43", TpAmbHomologacao, cteSVRS.homologacao}, // RS
		{"33", TpAmbProducao, cteSVRS.producao},       // RJ uses SVRS
		{"31", TpAmbProducao, cteMG.producao},
		{"50", TpAmbHomologacao, cteMS.homologacao},
		{"51", TpAmbProducao, cteMT.producao},
		{"41", TpAmbProducao, ctePR.producao},
	}
	for _, tt := range tests {
		if got, err := CTeRecepcaoEventoURL(tt.cUF, tt.tpAmb); err != nil || got != tt.want {
			t.Errorf("CTeRecepcaoEventoURL(%s, %s) = %q, %v; want %q", tt.cUF, tt.tpAmb, got, err, tt.want)
		}
	}
	for _, bad := range [][2]string{{"99", TpAmbProducao}, {"xx", TpAmbProducao}, {"35", "3"}} {
		if _, err := CTeRecepcaoEventoURL(bad[0], bad[1]); err == nil {
			t.Errorf("CTeRecepcaoEventoURL(%s, %s): want an error", bad[0], bad[1])
		}
	}
}

// TestEnviarEventoDesacordo_Autorizadora checks that without the test
// override the event goes to the autorizadora of the chave's UF.
func TestEnviarEventoDesacordo_Autorizadora(t *testing.T) {
	client, _ := newFakeClient(t, http.StatusOK, "", ClientConfig{})
	client.endpoints.RecepcaoEventoCTe = ""
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stop before the network: only the URL choice is under test
	_, err := client.EnviarEventoDesacordo(ctx, newMockSigner(t), desacordoEvento())
	if err == nil || !strings.Contains(err.Error(), "nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoEventoV4.asmx") {
		t.Errorf("err = %v, want a cancelled call to the SVSP host", err)
	}
}
