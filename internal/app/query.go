package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sync"
)

type QueryNFSeInput struct {
	CNPJ        string
	ChaveAcesso string
}

// QueryService owns the diagnostic and direct-query use cases.
type QueryService struct {
	Log             *slog.Logger
	CompanyStore    *company.Store
	CredentialStore *credential.Store
	Certificates    *sync.CertificateLoader

	// newADNClient builds the ADN client; nil means adn.NewClient. Tests set
	// it to point the client at a fake server.
	newADNClient func(adn.ClientConfig) (*adn.Client, error)
}

func NewQueryService(d Dependencies, certificates *sync.CertificateLoader) *QueryService {
	return &QueryService{
		Log:             d.Log,
		CompanyStore:    d.CompanyStore,
		CredentialStore: d.CredentialStore,
		Certificates:    certificates,
	}
}

func (s *QueryService) QueryNFSeEvents(ctx context.Context, input QueryNFSeInput) (string, error) {
	accessKey, err := validateQueryAccessKey(input.ChaveAcesso)
	if err != nil {
		return "", err
	}

	apiClient, _, err := s.buildClient(ctx, input.CNPJ, "Consulta direta")
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("NFSe/%s/Eventos", accessKey)
	return queryGenericEndpoint(ctx, apiClient, path)
}

func validateQueryAccessKey(raw string) (nfse.AccessKey, error) {
	accessKey, err := nfse.ParseAccessKey(raw)
	if err != nil {
		return "", fmt.Errorf("chave de acesso inválida: %w", err)
	}
	return accessKey, nil
}

// queryGenericEndpoint returns the ADN answer for path as indented JSON.
// RawGet already rejects a body that is not JSON, so indenting cannot fail on
// what it returns.
func queryGenericEndpoint(ctx context.Context, apiClient *adn.Client, path string) (string, error) {
	var response json.RawMessage
	err := apiClient.RawGet(ctx, path, &response)
	if err != nil {
		return "", fmt.Errorf("falha ao consultar API ADN: %w", err)
	}

	pretty, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("formatar resposta da API ADN: %w", err)
	}
	return string(pretty), nil
}

// buildClient loads the company certificate, asking for its password with
// purpose, and returns an ADN client that consults with it and the
// credential as the certificate itself describes it.
func (s *QueryService) buildClient(ctx context.Context, companyCNPJ, purpose string) (*adn.Client, *credential.Credential, error) {
	company, err := company.LookupByCNPJ(ctx, s.CompanyStore, companyCNPJ)
	if err != nil {
		return nil, nil, err
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, company, purpose)
	if err != nil {
		return nil, nil, err
	}

	newClient := s.newADNClient
	if newClient == nil {
		newClient = adn.NewClient
	}
	apiClient, err := newClient(adn.ClientConfig{
		BaseURL:     sync.ResolveEnvironmentURL(company.Environment),
		Certificate: &loaded.TLS,
		Log:         s.Log,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("configurar cliente ADN: %w", err)
	}
	return apiClient, loaded.Credential, nil
}

// ConnectionTestResult contains the diagnostics returned by TestConnection.
type ConnectionTestResult struct {
	CertLoaded        bool
	CertSubject       string
	CertExpiration    string
	MTLSAccepted      bool
	EndpointReached   bool
	ResponseCode      string
	ResponseDetail    string
	StatusExplanation string
}

// TestConnection verifies that the certificate can be loaded, mTLS works, and the ADN endpoint can be queried.
func (s *QueryService) TestConnection(ctx context.Context, companyCNPJ string) (ConnectionTestResult, error) {
	result := ConnectionTestResult{}

	company, err := company.LookupByCNPJ(ctx, s.CompanyStore, companyCNPJ)
	if err != nil {
		return result, err
	}

	cred, err := s.CredentialStore.CredentialByID(ctx, company.CredentialID)
	if errors.Is(err, credential.ErrCredentialNotFound) {
		result.StatusExplanation = "Certificado digital não associado ou não encontrado para esta empresa."
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("buscar credencial: %w", err)
	}

	// A certificate inspected before and already expired stops here, without
	// asking for its password. One never loaded has no inspection yet: the
	// check runs again on what loading it reads.
	result.CertLoaded = true
	if expired := describeCertificate(&result, cred); expired {
		return result, nil
	}

	apiClient, loaded, err := s.buildClient(ctx, companyCNPJ, "Teste de conexão")
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Erro ao carregar certificado/senha: %v", err)
		return result, nil
	}
	if expired := describeCertificate(&result, loaded); expired {
		return result, nil
	}

	docResp, err := apiClient.FetchDocuments(ctx, adn.DistributionRequest{
		LastNSU:          0,
		ConsultationCNPJ: company.CNPJ,
	})
	result.MTLSAccepted = true
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Conexão falhou: %v", err)
		return result, nil
	}

	result.EndpointReached = true
	result.ResponseCode = "Sucesso"
	if len(docResp.Docs) > 0 {
		result.ResponseDetail = fmt.Sprintf("Documentos retornados: %d. Último NSU: %d", len(docResp.Docs), docResp.UltNSU)
		result.StatusExplanation = "Conexão OK, documentos localizados com sucesso."
	} else {
		result.ResponseDetail = "Nenhum documento localizado para o NSU 0 (Erro E2220 / fila vazia)."
		result.StatusExplanation = "Conexão OK, sem novos documentos disponíveis para este NSU."
	}

	return result, nil
}

// describeCertificate fills the subject and expiry of result from cred and
// reports whether the certificate has expired, setting the explanation then.
func describeCertificate(result *ConnectionTestResult, cred *credential.Credential) bool {
	result.CertSubject = cred.SubjectName
	if cred.NotAfter == nil {
		return false
	}
	result.CertExpiration = formatCertExpiration(*cred.NotAfter)
	if !cred.NotAfter.Before(time.Now()) {
		return false
	}
	result.StatusExplanation = "Certificado digital expirado."
	return true
}

// formatCertExpiration shows a certificate's expiry in the user's local time.
// The credential store keeps NotAfter in UTC, so formatting it as is would
// put the expiry three hours ahead for a user in Brazil.
func formatCertExpiration(notAfter time.Time) string {
	return notAfter.Local().Format("02/01/2006 15:04:05")
}
