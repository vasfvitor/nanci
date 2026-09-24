package sefaz

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	wantCTeDistContentType = `application/soap+xml; charset=utf-8; action="http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe/cteDistDFeInteresse"`
	cteDistPath            = "/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx"
	// testCTeKey is the fictitious access key inside the docZips of
	// testdata/retdist-cte-138.xml.
	testCTeKey = "35260970860312000150570010000001231123456788"
)

// cteDistRequest is the exact request body for one CT-e distDFeInt query.
func cteDistRequest(tpAmb, query string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteDistDFeInteresse xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe">` +
		`<cteDadosMsg xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe">` +
		`<distDFeInt xmlns="http://www.portalfiscal.inf.br/cte" versao="1.00">` +
		`<tpAmb>` + tpAmb + `</tpAmb><cUFAutor>35</cUFAutor><CNPJ>70860312000150</CNPJ>` + query +
		`</distDFeInt></cteDadosMsg></cteDistDFeInteresse></soap:Body></soap:Envelope>`
}

// cteRetDistResponse is a CT-e SOAP answer with a retDistDFeInt in the CT-e
// namespace, for the cStat values without a fixture file.
func cteRetDistResponse(cStat int, xMotivo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteDistDFeInteresseResponse xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe"><cteDistDFeInteresseResult>` +
		fmt.Sprintf(`<retDistDFeInt versao="1.00" xmlns="http://www.portalfiscal.inf.br/cte">`+
			`<tpAmb>1</tpAmb><verAplic>1.0.0</verAplic><cStat>%d</cStat><xMotivo>%s</xMotivo>`+
			`<dhResp>2026-09-23T10:00:00-03:00</dhResp><ultNSU>000000000000000</ultNSU><maxNSU>000000000000000</maxNSU></retDistDFeInt>`, cStat, xMotivo) +
		`</cteDistDFeInteresseResult></cteDistDFeInteresseResponse></soap:Body></soap:Envelope>`
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}

func TestCTeDistribuicao_RequestBytes(t *testing.T) {
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
				_, err := c.DistCTeNSU(context.Background(), "70.860.312/0001-50", 35, 123)
				return err
			},
			want: cteDistRequest("1", `<distNSU><ultNSU>000000000000123</ultNSU></distNSU>`),
		},
		{
			name: "consNSU in homologação",
			env:  nfse.EnvironmentRestricted,
			call: func(c *Client) error {
				_, err := c.ConsCTeNSU(context.Background(), testCNPJ, 35, 42)
				return err
			},
			want: cteDistRequest("2", `<consNSU><NSU>000000000000042</NSU></consNSU>`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fake := newFakeClient(t, http.StatusOK, readFixture(t, "retdist-cte-137.xml"), ClientConfig{Environment: tt.env})
			if err := tt.call(client); err != nil {
				t.Fatalf("call: %v", err)
			}

			requests := fake.captured()
			if len(requests) != 1 {
				t.Fatalf("got %d requests, want 1", len(requests))
			}
			got := requests[0]
			if got.Path != cteDistPath {
				t.Errorf("path = %q, want %q", got.Path, cteDistPath)
			}
			if got.Body != tt.want {
				t.Errorf("request body:\n got %s\nwant %s", got.Body, tt.want)
			}
			if ct := got.Header.Get("Content-Type"); ct != wantCTeDistContentType {
				t.Errorf("Content-Type = %q, want %q", ct, wantCTeDistContentType)
			}
			if _, ok := got.Header["Soapaction"]; ok {
				t.Error("SOAP 1.2 request must not send a SOAPAction header")
			}
		})
	}
}

func TestCTeDistribuicao_InvalidInputSendsNothing(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusOK, "", ClientConfig{})
	ctx := context.Background()

	calls := map[string]func() error{
		"invalid CNPJ": func() error { _, err := client.DistCTeNSU(ctx, "70860312000151", 35, 0); return err },
		"cUFAutor 91":  func() error { _, err := client.DistCTeNSU(ctx, testCNPJ, 91, 0); return err },
		"negative NSU": func() error { _, err := client.DistCTeNSU(ctx, testCNPJ, 35, -1); return err },
		"NSU too big":  func() error { _, err := client.ConsCTeNSU(ctx, testCNPJ, 35, 1_000_000_000_000_000); return err },
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

// Endpoints set by hand without the CT-e URL must not send the CT-e request
// anywhere else.
func TestDistCTeNSU_WithoutEndpoint(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Environment: nfse.EnvironmentProduction,
		Certificate: &tls.Certificate{},
		Endpoints:   &Endpoints{Distribuicao: "https://127.0.0.1:1/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx"},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.DistCTeNSU(context.Background(), testCNPJ, 35, 0)
	if err == nil || !strings.Contains(err.Error(), "CT-e distribution URL is not configured") {
		t.Errorf("err = %v, want the missing CT-e URL", err)
	}
}

func TestDistCTeNSU_DocumentsFound(t *testing.T) {
	client, _ := newFakeClient(t, http.StatusOK, readFixture(t, "retdist-cte-138.xml"), ClientConfig{})
	result, err := client.DistCTeNSU(context.Background(), testCNPJ, 35, 200)
	if err != nil {
		t.Fatalf("DistCTeNSU: %v", err)
	}

	if result.CStat != CStatDocumentoLocalizado || result.XMotivo != "Documento(s) localizado(s)" {
		t.Errorf("cStat/xMotivo = %d %q", result.CStat, result.XMotivo)
	}
	if result.UltNSU != 202 || result.MaxNSU != 310 {
		t.Errorf("ultNSU/maxNSU = %d/%d, want 202/310", result.UltNSU, result.MaxNSU)
	}
	wantDhResp := time.Date(2026, 9, 23, 13, 0, 0, 0, time.UTC)
	if !result.DhResp.Equal(wantDhResp) {
		t.Errorf("DhResp = %v, want %v", result.DhResp, wantDhResp)
	}

	want := []struct {
		nsu    int64
		schema string
		root   string
	}{
		{201, "procCTe_v4.00.xsd", "<cteProc "},
		{202, "procEventoCTe_v4.00.xsd", "<procEventoCTe "},
	}
	if len(result.Docs) != len(want) {
		t.Fatalf("got %d docs, want %d", len(result.Docs), len(want))
	}
	for i, w := range want {
		doc := result.Docs[i]
		if doc.NSU != w.nsu || doc.Schema != w.schema {
			t.Errorf("doc %d = NSU %d schema %q, want %d %q", i, doc.NSU, doc.Schema, w.nsu, w.schema)
		}
		decoded, err := gzipxml.Decode(doc.Content, gzipxml.Limits{CompressedBytes: 1 << 20, UncompressedBytes: 1 << 20})
		if err != nil {
			t.Fatalf("decode doc %d: %v", i, err)
		}
		xml := string(decoded.XML)
		if !strings.HasPrefix(xml, w.root) || !strings.Contains(xml, "<chCTe>"+testCTeKey+"</chCTe>") {
			t.Errorf("doc %d content = %s", i, xml)
		}
	}
}

func TestDistCTeNSU_NoDocuments(t *testing.T) {
	client, _ := newFakeClient(t, http.StatusOK, readFixture(t, "retdist-cte-137.xml"), ClientConfig{})
	result, err := client.DistCTeNSU(context.Background(), testCNPJ, 35, 310)
	if err != nil {
		t.Fatalf("DistCTeNSU: %v", err)
	}
	if result.CStat != CStatNenhumDocumento || result.UltNSU != 310 || result.MaxNSU != 310 || len(result.Docs) != 0 {
		t.Errorf("result = %+v", result)
	}
}

func TestDistCTeNSU_ConsumoIndevidoIsAResult(t *testing.T) {
	client, _ := newFakeClient(t, http.StatusOK, readFixture(t, "retdist-cte-656.xml"), ClientConfig{})
	result, err := client.DistCTeNSU(context.Background(), testCNPJ, 35, 100)
	if err != nil {
		t.Fatalf("DistCTeNSU: %v", err)
	}
	if result.CStat != CStatConsumoIndevido || result.UltNSU != 280 || result.MaxNSU != 310 ||
		!strings.HasPrefix(result.XMotivo, "Rejeicao: Consumo Indevido") {
		t.Errorf("result = %+v", result)
	}
}

func TestDistCTeNSU_RejectionError(t *testing.T) {
	tests := []struct {
		cStat   int
		xMotivo string
	}{
		{593, "Rejeicao: CNPJ-Base consultado difere do CNPJ-Base do Certificado Digital"},
		{589, "Rejeicao: Numero do NSU informado superior ao maior NSU da base de dados do Ambiente Nacional"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.cStat), func(t *testing.T) {
			client, _ := newFakeClient(t, http.StatusOK, cteRetDistResponse(tt.cStat, tt.xMotivo), ClientConfig{})

			_, err := client.ConsCTeNSU(context.Background(), testCNPJ, 35, 999)
			var rejection *RejectionError
			if !errors.As(err, &rejection) {
				t.Fatalf("err = %v, want *RejectionError", err)
			}
			if rejection.CStat != tt.cStat || rejection.XMotivo != tt.xMotivo {
				t.Errorf("rejection = %+v", rejection)
			}
		})
	}
}

func TestDistCTeNSU_SOAPFault(t *testing.T) {
	client, fake := newFakeClient(t, http.StatusInternalServerError, soapFault12, ClientConfig{})

	_, err := client.DistCTeNSU(context.Background(), testCNPJ, 35, 0)
	var fault *FaultError
	if !errors.As(err, &fault) {
		t.Fatalf("err = %v, want *FaultError", err)
	}
	if fault.Code != "soap:Receiver" || fault.Reason != "Server was unable to process request." {
		t.Errorf("fault = %+v", fault)
	}
	if n := len(fake.captured()); n != 1 {
		t.Errorf("sent %d requests, want 1", n)
	}
}
