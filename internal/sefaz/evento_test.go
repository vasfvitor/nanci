package sefaz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
)

const (
	testChave2              = "35260911222333000181550010000012351234567894"
	wantEventoContentType   = `application/soap+xml; charset=utf-8; action="http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4/nfeRecepcaoEvento"`
	retEnvEventoEnvelopeFmt = `<?xml version="1.0" encoding="utf-8"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeResultMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4">` +
		`<retEnvEvento versao="1.00" xmlns="http://www.portalfiscal.inf.br/nfe"><idLote>260923103058123</idLote><tpAmb>1</tpAmb>` +
		`<verAplic>AN_1.0.0</verAplic><cOrgao>91</cOrgao><cStat>%d</cStat><xMotivo>%s</xMotivo>%s</retEnvEvento>` +
		`</nfeResultMsg></soap:Body></soap:Envelope>`
)

func retEventoXML(cStat int, xMotivo, chave, tpEvento, dhReg, nProt string) string {
	return fmt.Sprintf(`<retEvento versao="1.00"><infEvento Id="ID%s"><tpAmb>1</tpAmb><verAplic>AN_1.0.0</verAplic><cOrgao>91</cOrgao>`+
		`<cStat>%d</cStat><xMotivo>%s</xMotivo><chNFe>%s</chNFe><tpEvento>%s</tpEvento><xEvento>Ciencia da Operacao</xEvento>`+
		`<nSeqEvento>1</nSeqEvento><CNPJDest>70860312000150</CNPJDest><dhRegEvento>%s</dhRegEvento><nProt>%s</nProt></infEvento></retEvento>`,
		nProt, cStat, xMotivo, chave, tpEvento, dhReg, nProt)
}

func manifestacao(chave string, tipo nfe.ManifestationType) Evento {
	e := cienciaEvento()
	e.ChaveAcesso = chave
	e.TpEvento = tipo.TpEvento()
	e.DescEvento = tipo.DescEvento()
	return e
}

func TestEnviarEventos_MatchesEachRetEvento(t *testing.T) {
	eventos := []Evento{
		manifestacao(testChave, nfe.ManifestationCiencia),
		manifestacao(testChave, nfe.ManifestationConfirmacao),
		manifestacao(testChave2, nfe.ManifestationCiencia),
	}
	// Answers come in a different order than the eventos were sent.
	retEventos := retEventoXML(650, "Rejeicao: Evento de Ciencia da Operacao para NF-e Cancelada ou Denegada", testChave2, "210210", "2026-09-23T10:31:05-03:00", "") +
		retEventoXML(573, "Rejeicao: Duplicidade de evento", testChave, "210200", "2026-09-23T10:31:05-03:00", "") +
		retEventoXML(135, "Evento registrado e vinculado a NF-e", testChave, "210210", "2026-09-23T10:31:05-03:00", "891260000000001")
	response := fmt.Sprintf(retEnvEventoEnvelopeFmt, 128, "Lote de evento processado", retEventos)

	client, fake := newFakeClient(t, http.StatusOK, response, ClientConfig{})
	signer := newMockSigner(t)
	result, err := client.EnviarEventos(context.Background(), signer, "260923103058123", eventos)
	if err != nil {
		t.Fatalf("EnviarEventos: %v", err)
	}

	if result.IDLote != "260923103058123" || result.CStat != CStatLoteProcessado || result.XMotivo != "Lote de evento processado" {
		t.Errorf("lote = %s %d %q", result.IDLote, result.CStat, result.XMotivo)
	}
	wantCStat := []int{135, 573, 650}
	if len(result.Eventos) != len(wantCStat) {
		t.Fatalf("got %d results, want %d", len(result.Eventos), len(wantCStat))
	}
	for i, r := range result.Eventos {
		if r.ChaveAcesso != eventos[i].ChaveAcesso || r.TpEvento != eventos[i].TpEvento || r.NSeqEvento != 1 {
			t.Errorf("result %d is for %s/%s, want %s/%s", i, r.ChaveAcesso, r.TpEvento, eventos[i].ChaveAcesso, eventos[i].TpEvento)
		}
		if r.CStat != wantCStat[i] {
			t.Errorf("result %d cStat = %d, want %d", i, r.CStat, wantCStat[i])
		}
		if !strings.HasPrefix(string(r.RetEvento), "<retEvento") || !strings.Contains(string(r.RetEvento), eventos[i].TpEvento) {
			t.Errorf("result %d RetEvento = %s", i, r.RetEvento)
		}
	}

	registered := result.Eventos[0]
	if !IsRegistered(registered.CStat) || IsAlreadyDone(registered.CStat) {
		t.Errorf("135 must be registered")
	}
	if registered.Protocolo != "891260000000001" || registered.XMotivo != "Evento registrado e vinculado a NF-e" {
		t.Errorf("registered = %+v", registered)
	}
	wantReg := time.Date(2026, 9, 23, 13, 31, 5, 0, time.UTC)
	if registered.DhRegEvento == nil || !registered.DhRegEvento.Equal(wantReg) {
		t.Errorf("DhRegEvento = %v, want %v", registered.DhRegEvento, wantReg)
	}
	if !IsAlreadyDone(result.Eventos[1].CStat) || IsRegistered(result.Eventos[1].CStat) {
		t.Errorf("573 must be already done")
	}
	if IsRegistered(result.Eventos[2].CStat) || IsAlreadyDone(result.Eventos[2].CStat) {
		t.Errorf("650 must be a rejection")
	}
	if IsRegistered(CStatCienciaAposManifestacao) || IsAlreadyDone(CStatCienciaAposManifestacao) {
		t.Errorf("655 must be a rejection: SEFAZ did not register the ciência")
	}

	// The request carries the signed eventos verbatim, straight in nfeDadosMsg.
	requests := fake.captured()
	if len(requests) != 1 {
		t.Fatalf("sent %d requests, want 1", len(requests))
	}
	if ct := requests[0].Header.Get("Content-Type"); ct != wantEventoContentType {
		t.Errorf("Content-Type = %q, want %q", ct, wantEventoContentType)
	}
	var signed strings.Builder
	for _, r := range result.Eventos {
		signed.Write(r.SignedEvento)
	}
	wantBody := `<?xml version="1.0" encoding="UTF-8"?><soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4">` +
		`<envEvento xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00"><idLote>260923103058123</idLote>` +
		signed.String() +
		`</envEvento></nfeDadosMsg></soap:Body></soap:Envelope>`
	if requests[0].Body != wantBody {
		t.Errorf("request body:\n got %s\nwant %s", requests[0].Body, wantBody)
	}

	// The stored procEventoNFe parses as a registered ciência.
	proc := ProcEventoNFe(registered.SignedEvento, registered.RetEvento)
	event, err := nfe.ParseProcEventoNFe(proc)
	if err != nil {
		t.Fatalf("ParseProcEventoNFe: %v", err)
	}
	if !event.Registered || event.Type != nfe.EventTypeCiencia || event.ChaveAcesso.String() != testChave || event.Protocolo != "891260000000001" {
		t.Errorf("procEventoNFe parsed as %+v", event)
	}
}

func TestEnviarEventos_MissingRetEvento(t *testing.T) {
	response := fmt.Sprintf(retEnvEventoEnvelopeFmt, 128, "Lote de evento processado", "")
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})

	result, err := client.EnviarEventos(context.Background(), newMockSigner(t), "1", []Evento{cienciaEvento()})
	if err != nil {
		t.Fatalf("EnviarEventos: %v", err)
	}
	r := result.Eventos[0]
	if r.CStat != 0 || r.RetEvento != nil || r.XMotivo == "" || len(r.SignedEvento) == 0 {
		t.Errorf("result without retEvento = %+v", r)
	}
}

func TestEnviarEventos_LoteRejected(t *testing.T) {
	response := fmt.Sprintf(retEnvEventoEnvelopeFmt, 215, "Rejeicao: Falha no schema XML", "")
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})

	_, err := client.EnviarEventos(context.Background(), newMockSigner(t), "1", []Evento{cienciaEvento()})
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("err = %v, want *RejectionError", err)
	}
	if rejection.CStat != 215 || rejection.XMotivo != "Rejeicao: Falha no schema XML" {
		t.Errorf("rejection = %+v", rejection)
	}
}

func TestEnviarEventos_InvalidLoteSendsNothing(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusOK, "", ClientConfig{})
	signer := newMockSigner(t)
	ctx := context.Background()

	tooMany := make([]Evento, MaxEventosPorLote+1)
	for i := range tooMany {
		tooMany[i] = cienciaEvento()
		tooMany[i].NSeqEvento = i%maxNSeqEvento + 1
	}
	invalid := cienciaEvento()
	invalid.CNPJ = ""

	tests := map[string]func() error{
		"21 eventos": func() error { _, err := client.EnviarEventos(ctx, signer, "1", tooMany); return err },
		"no eventos": func() error { _, err := client.EnviarEventos(ctx, signer, "1", nil); return err },
		"no signer":  func() error { _, err := client.EnviarEventos(ctx, nil, "1", []Evento{cienciaEvento()}); return err },
		"idLote": func() error {
			_, err := client.EnviarEventos(ctx, signer, "1234567890123456", []Evento{cienciaEvento()})
			return err
		},
		"duplicate": func() error {
			_, err := client.EnviarEventos(ctx, signer, "1", []Evento{cienciaEvento(), cienciaEvento()})
			return err
		},
		"invalid evento": func() error {
			_, err := client.EnviarEventos(ctx, signer, "1", []Evento{cienciaEvento(), invalid})
			return err
		},
	}
	for name, call := range tests {
		if err := call(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if n := len(fake.captured()); n != 0 {
		t.Errorf("sent %d requests, want 0", n)
	}
}

func TestNewIDLote(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 30, 58, 123_456_789, time.UTC)
	got := NewIDLote(now)
	if got != "260923103058123" {
		t.Errorf("NewIDLote = %s, want 260923103058123", got)
	}
	if len(got) != 15 || !isDigits(got) {
		t.Errorf("NewIDLote = %s, want 15 digits", got)
	}
}

func TestProcEventoNFe(t *testing.T) {
	got := string(ProcEventoNFe([]byte("<evento></evento>"), []byte("<retEvento></retEvento>")))
	want := `<procEventoNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.00"><evento></evento><retEvento></retEvento></procEventoNFe>`
	if got != want {
		t.Errorf("ProcEventoNFe = %s, want %s", got, want)
	}
}
