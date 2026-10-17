package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// mockPFXPassword opens internal/foundation/cert/testdata's mock PFX.
const mockPFXPassword = "mockdata"

type mockPasswords struct{}

func (mockPasswords) GetCertPassword(context.Context, app.CertPasswordRequest) ([]byte, error) {
	return []byte(mockPFXPassword), nil
}

// fakeDistribuicao is an httptest NFeDistribuicaoDFe that answers every
// request with response and counts the requests.
type fakeDistribuicao struct {
	response string
	requests int
}

func (f *fakeDistribuicao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	f.requests++
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	_, _ = io.WriteString(w, f.response)
}

// newNFeFetchTestRoot is newNFeTestRoot with the mock certificate, its
// password, and the NF-e distribution pointed at a fake SEFAZ answering
// response.
func newNFeFetchTestRoot(t *testing.T, response string) (*nfeTestRoot, *fakeDistribuicao) {
	t.Helper()
	ctx := context.Background()
	env := newNFeTestRoot(t)

	certPath, err := filepath.Abs(filepath.Join("..", "foundation", "cert", "testdata", "cert_a1_mock_70860312000150.pfx"))
	if err != nil {
		t.Fatal(err)
	}
	creds := credential.NewStore(env.db)
	cred, err := creds.CredentialByID(ctx, "cred-1")
	if err != nil {
		t.Fatal(err)
	}
	cred.CertPath = certPath
	if err := creds.UpdateCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}

	application, err := app.New(app.Dependencies{
		Log:                slog.New(slog.DiscardHandler),
		CompanyStore:       company.NewStore(env.db),
		CredentialStore:    creds,
		SyncRepo:           sync.NewStore(env.db),
		DocumentRepo:       store.NewDocumentRepository(env.db),
		NFeRepo:            env.repo,
		CTeRepo:            store.NewCTeRepository(env.db),
		XMLStore:           env.xml,
		DataDir:            t.TempDir(),
		CredentialProvider: mockPasswords{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeDistribuicao{response: response}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	application.SyncManager.SEFAZEndpoints = &sefaz.Endpoints{
		Distribuicao: server.URL + "/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
	}

	env.root, env.out = newTestRootForApp(application)
	env.root.SetErr(&bytes.Buffer{})
	return env, fake
}

// retDistDFeInt builds the SOAP answer of NFeDistribuicaoDFe with cStat and,
// when fixture is not empty, that NF-e fixture as a procNFe docZip.
func retDistDFeInt(t *testing.T, cStat int, xMotivo, fixture string) string {
	t.Helper()
	body := `<tpAmb>1</tpAmb><verAplic>1.0.0</verAplic>` +
		fmt.Sprintf(`<cStat>%d</cStat><xMotivo>%s</xMotivo>`, cStat, xMotivo) +
		`<dhResp>2026-09-23T10:00:00-03:00</dhResp><ultNSU>000000000000000</ultNSU><maxNSU>000000000000100</maxNSU>`
	if fixture != "" {
		data, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", fixture)) // #nosec G304 -- fixed testdata path.
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		body += `<loteDistDFeInt><docZip NSU="000000000000042" schema="procNFe_v4.00.xsd">` +
			base64.StdEncoding.EncodeToString(buf.Bytes()) + `</docZip></loteDistDFeInt>`
	}
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<nfeDistDFeInteresseResponse xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe"><nfeDistDFeInteresseResult>` +
		`<retDistDFeInt versao="1.01" xmlns="http://www.portalfiscal.inf.br/nfe">` + body + `</retDistDFeInt>` +
		`</nfeDistDFeInteresseResult></nfeDistDFeInteresseResponse></soap:Body></soap:Envelope>`
}

func TestNFeBuscar_Completa(t *testing.T) {
	env, fake := newNFeFetchTestRoot(t, retDistDFeInt(t, 138, "Documento localizado", "procnfe.xml"))

	if err := env.run("nfe", "buscar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc); err != nil {
		t.Fatalf("buscar: %v\n%s", err, env.out.String())
	}
	want := "Buscando NF-e 3526 0911 2223 3300 0181 5500 1000 0012 3411 2345 6787...\n" +
		"NF-e completa salva.\n" +
		"Consultas na última hora: 1/20\n"
	if got := env.out.String(); got != want {
		t.Errorf("output =\n%s\nwant\n%s", got, want)
	}
	if fake.requests != 1 {
		t.Errorf("SEFAZ requests = %d, want 1", fake.requests)
	}

	if err := env.run("nfe", "list", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(env.out.String(), nfeChaveProc) {
		t.Errorf("the fetched NF-e is not listed:\n%s", env.out.String())
	}
}

func TestNFeBuscar_NotFound(t *testing.T) {
	env, _ := newNFeFetchTestRoot(t, retDistDFeInt(t, 137, "Nenhum documento localizado", ""))

	if err := env.run("nfe", "buscar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc); err != nil {
		t.Fatalf("buscar: %v\n%s", err, env.out.String())
	}
	lines := strings.Split(strings.TrimSpace(env.out.String()), "\n")
	if len(lines) != 3 ||
		lines[1] != "NF-e não localizada. NF-e não localizada ou sem acesso para a empresa" ||
		lines[2] != "Consultas na última hora: 1/20" {
		t.Errorf("output =\n%s", env.out.String())
	}
}

func TestNFeBuscar_InvalidChaveSkipsAppFactory(t *testing.T) {
	calls, err := runWithoutApp(t, "nfe", "buscar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc[:43]+"0")
	if err == nil || !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Errorf("err = %v, want the invalid-chave error", err)
	}
	if calls != 0 {
		t.Errorf("AppFactory called %d times, want 0", calls)
	}
}
