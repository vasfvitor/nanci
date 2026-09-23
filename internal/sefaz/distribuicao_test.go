package sefaz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const wantDistContentType = `application/soap+xml; charset=utf-8; action="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe/nfeDistDFeInteresse"`

// distRequest is the exact request body for one distDFeInt query.
func distRequest(tpAmb, query string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeDistDFeInteresse xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe">` +
		`<nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe">` +
		`<distDFeInt xmlns="http://www.portalfiscal.inf.br/nfe" versao="1.01">` +
		`<tpAmb>` + tpAmb + `</tpAmb><cUFAutor>35</cUFAutor><CNPJ>70860312000150</CNPJ>` + query +
		`</distDFeInt></nfeDadosMsg></nfeDistDFeInteresse></soap:Body></soap:Envelope>`
}

// retDistResponse is a SOAP answer shaped like the real one, with the
// retDistDFeInt element written by retDist.
func retDistResponse(retDist string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">` +
		`<soap:Body><nfeDistDFeInteresseResponse xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe"><nfeDistDFeInteresseResult>` +
		retDist +
		`</nfeDistDFeInteresseResult></nfeDistDFeInteresseResponse></soap:Body></soap:Envelope>`
}

func retDist(cStat int, xMotivo, ultNSU, maxNSU, lote string) string {
	return fmt.Sprintf(`<retDistDFeInt xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">`+
		`<tpAmb>1</tpAmb><verAplic>1.7.6</verAplic><cStat>%d</cStat><xMotivo>%s</xMotivo><dhResp>2026-09-23T10:00:00-03:00</dhResp>`+
		`<ultNSU>%s</ultNSU><maxNSU>%s</maxNSU>%s</retDistDFeInt>`, cStat, xMotivo, ultNSU, maxNSU, lote)
}

func TestDistribuicao_RequestBytes(t *testing.T) {
	tests := []struct {
		name string
		env  nfse.Environment
		call func(*Client) error
		want string
	}{
		{
			name: "distNSU",
			env:  nfse.EnvironmentProduction,
			call: func(c *Client) error {
				_, err := c.DistNSU(context.Background(), "70.860.312/0001-50", 35, 123)
				return err
			},
			want: distRequest("1", `<distNSU><ultNSU>000000000000123</ultNSU></distNSU>`),
		},
		{
			name: "consNSU in homologação",
			env:  nfse.EnvironmentRestricted,
			call: func(c *Client) error {
				_, err := c.ConsNSU(context.Background(), testCNPJ, 35, 999_999_999_999_999)
				return err
			},
			want: distRequest("2", `<consNSU><NSU>999999999999999</NSU></consNSU>`),
		},
		{
			name: "consChNFe",
			env:  nfse.EnvironmentProduction,
			call: func(c *Client) error {
				_, err := c.ConsChNFe(context.Background(), testCNPJ, 35, "35260911222333000181550010000012341123456787")
				return err
			},
			want: distRequest("1", `<consChNFe><chNFe>35260911222333000181550010000012341123456787</chNFe></consChNFe>`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := retDistResponse(retDist(CStatNenhumDocumento, "Nenhum documento localizado", "000000000000123", "000000000000123", ""))
			client, fake := newFakeClient(t, http.StatusOK, response, ClientConfig{Environment: tt.env})
			if err := tt.call(client); err != nil {
				t.Fatalf("call: %v", err)
			}

			requests := fake.captured()
			if len(requests) != 1 {
				t.Fatalf("got %d requests, want 1", len(requests))
			}
			got := requests[0]
			if got.Body != tt.want {
				t.Errorf("request body:\n got %s\nwant %s", got.Body, tt.want)
			}
			if ct := got.Header.Get("Content-Type"); ct != wantDistContentType {
				t.Errorf("Content-Type = %q, want %q", ct, wantDistContentType)
			}
			if _, ok := got.Header["Soapaction"]; ok {
				t.Error("SOAP 1.2 request must not send a SOAPAction header")
			}
		})
	}
}

func TestDistribuicao_InvalidInputSendsNothing(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusOK, "", ClientConfig{})
	ctx := context.Background()

	calls := map[string]func() error{
		"invalid CNPJ": func() error { _, err := client.DistNSU(ctx, "70860312000151", 35, 0); return err },
		"cUFAutor 91":  func() error { _, err := client.DistNSU(ctx, testCNPJ, 91, 0); return err },
		"negative NSU": func() error { _, err := client.DistNSU(ctx, testCNPJ, 35, -1); return err },
		"NSU too big":  func() error { _, err := client.ConsNSU(ctx, testCNPJ, 35, 1_000_000_000_000_000); return err },
		"invalid key": func() error {
			_, err := client.ConsChNFe(ctx, testCNPJ, 35, "35260911222333000181550010000012341123456780")
			return err
		},
	}
	for name, call := range calls {
		if err := call(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if n := len(fake.captured()); n != 0 {
		t.Errorf("sent %d requests for invalid input, want 0", n)
	}
}

func TestDistNSU_DocumentsFound(t *testing.T) {
	fixtures := []struct {
		path   string
		nsu    string
		schema string
		kind   nfe.SchemaKind
	}{
		{"../nfe/testdata/resnfe.xml", "000000000000101", "resNFe_v1.01.xsd", nfe.SchemaResNFe},
		{"../nfe/testdata/procnfe.xml", "000000000000102", "procNFe_v4.00.xsd", nfe.SchemaProcNFe},
		{"../nfe/testdata/resevento-cancelamento.xml", "000000000000103", "resEvento_v1.01.xsd", nfe.SchemaResEvento},
	}
	var lote strings.Builder
	lote.WriteString("<loteDistDFeInt>")
	for _, f := range fixtures {
		fmt.Fprintf(&lote, `<docZip NSU="%s" schema="%s">%s</docZip>`, f.nsu, f.schema, mustDocZip(t, f.path))
	}
	lote.WriteString("</loteDistDFeInt>")
	response := retDistResponse(retDist(CStatDocumentoLocalizado, "Documento(s) localizado(s)", "000000000000103", "000000000000250", lote.String()))

	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})
	result, err := client.DistNSU(context.Background(), testCNPJ, 35, 100)
	if err != nil {
		t.Fatalf("DistNSU: %v", err)
	}

	if result.CStat != CStatDocumentoLocalizado || result.XMotivo != "Documento(s) localizado(s)" {
		t.Errorf("cStat/xMotivo = %d %q", result.CStat, result.XMotivo)
	}
	if result.UltNSU != 103 || result.MaxNSU != 250 {
		t.Errorf("ultNSU/maxNSU = %d/%d, want 103/250", result.UltNSU, result.MaxNSU)
	}
	wantDhResp := time.Date(2026, 9, 23, 13, 0, 0, 0, time.UTC)
	if !result.DhResp.Equal(wantDhResp) {
		t.Errorf("DhResp = %v, want %v", result.DhResp, wantDhResp)
	}
	if len(result.Docs) != len(fixtures) {
		t.Fatalf("got %d docs, want %d", len(result.Docs), len(fixtures))
	}
	for i, f := range fixtures {
		doc := result.Docs[i]
		if doc.NSU != int64(101+i) || doc.Schema != f.schema || doc.Kind() != f.kind {
			t.Errorf("doc %d = NSU %d schema %q kind %v", i, doc.NSU, doc.Schema, doc.Kind())
		}
		decoded, err := gzipxml.Decode(doc.Content, gzipxml.Limits{CompressedBytes: 1 << 20, UncompressedBytes: 1 << 20})
		if err != nil {
			t.Fatalf("decode doc %d: %v", i, err)
		}
		want, _ := os.ReadFile(f.path) // #nosec G304 -- fixed testdata path.
		if string(decoded.XML) != string(want) {
			t.Errorf("doc %d content does not round-trip", i)
		}
	}
}

func TestDistNSU_NoDocumentsWithPrefixedResult(t *testing.T) {
	// The element is found by local name, whatever prefix the server uses.
	response := retDistResponse(`<ns2:retDistDFeInt xmlns:ns2="http://www.portalfiscal.inf.br/nfe" versao="1.01">` +
		`<ns2:tpAmb>1</ns2:tpAmb><ns2:cStat>137</ns2:cStat><ns2:xMotivo>Nenhum documento localizado</ns2:xMotivo>` +
		`<ns2:ultNSU>000000000000250</ns2:ultNSU><ns2:maxNSU>000000000000250</ns2:maxNSU></ns2:retDistDFeInt>`)
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})

	result, err := client.DistNSU(context.Background(), testCNPJ, 35, 250)
	if err != nil {
		t.Fatalf("DistNSU: %v", err)
	}
	if result.CStat != CStatNenhumDocumento || result.UltNSU != 250 || result.MaxNSU != 250 || len(result.Docs) != 0 {
		t.Errorf("result = %+v", result)
	}
	if !result.DhResp.IsZero() {
		t.Errorf("DhResp = %v, want zero without dhResp", result.DhResp)
	}
}

func TestDistNSU_ConsumoIndevidoIsAResult(t *testing.T) {
	xMotivo := "Rejeicao: Consumo Indevido (Deve ser utilizado o ultNSU nas solicitacoes subsequentes. Tente apos 1 hora)"
	response := retDistResponse(retDist(CStatConsumoIndevido, xMotivo, "000000000000180", "000000000000250", ""))
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})

	result, err := client.DistNSU(context.Background(), testCNPJ, 35, 100)
	if err != nil {
		t.Fatalf("DistNSU: %v", err)
	}
	if result.CStat != CStatConsumoIndevido || result.XMotivo != xMotivo || result.UltNSU != 180 {
		t.Errorf("result = %+v", result)
	}
}

func TestDistNSU_RejectionError(t *testing.T) {
	xMotivo := "Rejeicao: CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"
	response := retDistResponse(retDist(593, xMotivo, "000000000000000", "000000000000000", ""))
	client, _ := newFakeClient(t, http.StatusOK, response, ClientConfig{})

	_, err := client.DistNSU(context.Background(), testCNPJ, 35, 0)
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("err = %v, want *RejectionError", err)
	}
	if rejection.CStat != 593 || rejection.XMotivo != xMotivo {
		t.Errorf("rejection = %+v", rejection)
	}
}

func TestDistNSU_ResponseWithoutResult(t *testing.T) {
	client, _ := newFakeClient(t, http.StatusOK, `<html><body>maintenance</body></html>`, ClientConfig{})
	if _, err := client.DistNSU(context.Background(), testCNPJ, 35, 0); err == nil {
		t.Fatal("expected error for a response without retDistDFeInt")
	}
}
