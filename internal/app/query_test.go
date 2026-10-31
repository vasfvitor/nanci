package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// failingPasswords records each request and refuses it, so a test reaches
// buildClient without loading the certificate or calling the ADN.
type failingPasswords struct {
	requests []CertPasswordRequest
}

func (p *failingPasswords) GetCertPassword(_ context.Context, req CertPasswordRequest) ([]byte, error) {
	p.requests = append(p.requests, req)
	return nil, errors.New("senha recusada")
}

// newFakeADNClient returns an ADN client whose base URL is a local server
// running handler. buildClient always uses the real ADN URL, so tests of
// queryGenericEndpoint build the client themselves.
func newFakeADNClient(t *testing.T, handler http.HandlerFunc) *adn.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := adn.NewClient(adn.ClientConfig{
		BaseURL: srv.URL + "/contribuintes",
		Retry:   adn.RetryConfig{MaxRetries: 1, Initial: time.Millisecond, MaxDelay: time.Millisecond},
		Log:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestQueryGenericEndpointIndentsJSON(t *testing.T) {
	t.Parallel()
	path := "NFSe/" + nfseChavePrestada + "/Eventos"
	client := newFakeADNClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/contribuintes/"+path {
			t.Errorf("request = %s %s, want GET /contribuintes/%s", r.Method, r.URL.Path, path)
		}
		_, _ = w.Write([]byte(`{"eventos":[{"tipo":"101101"}]}`))
	})

	got, err := queryGenericEndpoint(context.Background(), client, path)
	if err != nil {
		t.Fatalf("queryGenericEndpoint: %v", err)
	}
	want := "{\n  \"eventos\": [\n    {\n      \"tipo\": \"101101\"\n    }\n  ]\n}"
	if got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// TestQueryGenericEndpointErrors pins how ADN failures surface. A body that
// is not JSON fails in RawGet's decode, so it is an error too: the raw
// fallback after MarshalIndent never sees it.
func TestQueryGenericEndpointErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
		check   func(t *testing.T, err error)
	}{
		{
			name:    "ADN rejection",
			status:  http.StatusBadRequest,
			body:    `{"Erros":[{"Codigo":"E0001"}]}`,
			wantErr: "status 400",
			check: func(t *testing.T, err error) {
				var statusErr *httpclient.StatusError
				if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
					t.Errorf("err = %v, want a StatusError with status 400", err)
				}
			},
		},
		{
			name:    "empty result",
			status:  http.StatusNotFound,
			body:    `{"StatusProcessamento":"NENHUM_DOCUMENTO_LOCALIZADO","Erros":[{"Codigo":"E2220"}]}`,
			wantErr: "nenhum documento localizado",
			check: func(t *testing.T, err error) {
				if !errors.Is(err, adn.ErrNoDocumentsLocated) {
					t.Errorf("err = %v, want ErrNoDocumentsLocated", err)
				}
			},
		},
		{
			name:    "body is not JSON",
			status:  http.StatusOK,
			body:    "<html>manutenção</html>",
			wantErr: "failed to decode json response",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := newFakeADNClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			got, err := queryGenericEndpoint(context.Background(), client, "NFSe/"+nfseChavePrestada+"/Eventos")
			if err == nil || !strings.HasPrefix(err.Error(), "falha ao consultar API ADN: ") || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want a wrapped error with %q", err, tc.wantErr)
			}
			if got != "" {
				t.Errorf("output = %q, want empty on error", got)
			}
			if tc.check != nil {
				tc.check(t, err)
			}
		})
	}
}

func TestQueryNFSeEventsUnknownCompany(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)

	_, err := env.app.Query.QueryNFSeEvents(context.Background(), QueryNFSeInput{CNPJ: "11222333000181", ChaveAcesso: nfseChavePrestada})
	if err == nil || !strings.Contains(err.Error(), "empresa não encontrada") {
		t.Fatalf("err = %v, want empresa não encontrada", err)
	}
	if len(env.passwords.requests) != 0 {
		t.Errorf("password prompts = %d, want 0", len(env.passwords.requests))
	}
}

func TestQueryNFSeEventsReturnsCertificateError(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)
	passwords := &failingPasswords{}
	env.app.Query.Certificates.Passwords = passwords

	_, err := env.app.Query.QueryNFSeEvents(context.Background(), QueryNFSeInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfseChavePrestada})
	if err == nil || !strings.Contains(err.Error(), "obter senha do certificado: senha recusada") {
		t.Fatalf("err = %v, want the password error", err)
	}
	if len(passwords.requests) != 1 || passwords.requests[0].Purpose != "Consulta direta" {
		t.Errorf("password requests = %+v, want one for Consulta direta", passwords.requests)
	}
}

func TestTestConnectionExpiredCertificate(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)
	ctx := context.Background()
	cred, err := env.app.Query.CredentialStore.CredentialByID(ctx, env.company.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Date(2024, 3, 1, 12, 30, 0, 0, time.UTC)
	cred.SubjectName = "CN=EMPRESA MOCK:70860312000150"
	cred.NotAfter = &expired
	if err := env.app.Query.CredentialStore.UpdateCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}

	result, err := env.app.Query.TestConnection(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	want := ConnectionTestResult{
		CertLoaded:        true,
		CertSubject:       cred.SubjectName,
		CertExpiration:    "01/03/2024 12:30:00",
		StatusExplanation: "Certificado digital expirado.",
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}
	if len(env.passwords.requests) != 0 {
		t.Errorf("password prompts = %d, want 0 for an expired certificate", len(env.passwords.requests))
	}
}

// TestTestConnectionCertificateLoadFailure: a certificate that does not load
// is a diagnostic in the result, before any ADN call.
func TestTestConnectionCertificateLoadFailure(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)
	passwords := &failingPasswords{}
	env.app.Query.Certificates.Passwords = passwords

	result, err := env.app.Query.TestConnection(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if !result.CertLoaded || result.MTLSAccepted || result.EndpointReached {
		t.Errorf("result = %+v, want only CertLoaded", result)
	}
	if !strings.HasPrefix(result.StatusExplanation, "Erro ao carregar certificado/senha: ") || !strings.Contains(result.StatusExplanation, "senha recusada") {
		t.Errorf("StatusExplanation = %q, want the certificate load error", result.StatusExplanation)
	}
	if len(passwords.requests) != 1 || passwords.requests[0].Purpose != "Teste de conexão" {
		t.Errorf("password requests = %+v, want one for Teste de conexão", passwords.requests)
	}
}

func TestQueryNFSeEventsRejectsInvalidAccessKeyBeforeClientSetup(t *testing.T) {
	t.Parallel()

	application := &App{}

	_, err := application.Query.QueryNFSeEvents(context.Background(), QueryNFSeInput{
		CNPJ:        "11222333000181",
		ChaveAcesso: strings.Repeat("1", 49) + "?",
	})
	if err == nil {
		t.Fatal("expected invalid access key error")
	}
	if !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateQueryAccessKeyAcceptsAlphanumericCNPJ(t *testing.T) {
	t.Parallel()

	const chave = "355030812" + "12ABC34501DE35" + "0000000000123" + "2608" + "123456789" + "7"
	got, err := validateQueryAccessKey(" " + strings.ToLower(chave) + " ")
	if err != nil {
		t.Fatalf("validateQueryAccessKey: %v", err)
	}
	if string(got) != chave {
		t.Errorf("validateQueryAccessKey = %q, want %q", got, chave)
	}
}

func TestTestConnectionReportsUnknownCompanyOnce(t *testing.T) {
	t.Parallel()
	env := newNFeTestEnv(t)

	_, err := env.app.Query.TestConnection(context.Background(), "11222333000181")
	if err == nil {
		t.Fatal("expected error for an unregistered CNPJ")
	}
	if got := strings.Count(err.Error(), "empresa não encontrada"); got != 1 {
		t.Fatalf("error mentions %q %d times, want 1: %v", "empresa não encontrada", got, err)
	}
}

// TestTestConnectionCredentialLookup: a missing credential is a diagnostic
// in the result; a database error is returned.
func TestTestConnectionCredentialLookup(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		env := newNFeTestEnv(t)
		env.app.Query.CredentialStore = credential.NewStore(storetest.OpenTestDB(t)) // no credentials

		result, err := env.app.Query.TestConnection(context.Background(), nfeTestCNPJ)
		if err != nil {
			t.Fatalf("TestConnection: %v", err)
		}
		if result.CertLoaded || !strings.Contains(result.StatusExplanation, "não associado ou não encontrado") {
			t.Errorf("result = %+v, want the missing certificate explanation", result)
		}
	})

	t.Run("database error", func(t *testing.T) {
		t.Parallel()
		env := newNFeTestEnv(t)
		closed := storetest.OpenTestDB(t)
		if err := closed.Close(); err != nil {
			t.Fatal(err)
		}
		env.app.Query.CredentialStore = credential.NewStore(closed)

		result, err := env.app.Query.TestConnection(context.Background(), nfeTestCNPJ)
		if err == nil || !strings.HasPrefix(err.Error(), "buscar credencial: ") {
			t.Fatalf("TestConnection error = %v, want a wrapped credential lookup error", err)
		}
		if errors.Is(err, credential.ErrCredentialNotFound) || result.StatusExplanation != "" {
			t.Errorf("database error reported as a missing credential: %v, %+v", err, result)
		}
		if len(env.passwords.requests) != 0 {
			t.Errorf("password prompts = %d, want 0", len(env.passwords.requests))
		}
	})
}
