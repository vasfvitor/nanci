package app

import (
	"context"
	"encoding/json"
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

	apiClient, err := s.buildClient(ctx, input.CNPJ, "Consulta direta")
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

func queryGenericEndpoint(ctx context.Context, apiClient *adn.Client, path string) (string, error) {
	var response json.RawMessage
	err := apiClient.RawGet(ctx, path, &response)
	if err != nil {
		return "", fmt.Errorf("falha ao consultar API ADN: %w", err)
	}

	pretty, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return string(response), nil //nolint:nilerr // intentional: return raw response if formatting fails
	}
	return string(pretty), nil
}

// buildClient loads the company certificate, asking for its password with
// purpose, and returns an ADN client that consults with it.
func (s *QueryService) buildClient(ctx context.Context, companyCNPJ, purpose string) (*adn.Client, error) {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, companyCNPJ)
	if err != nil {
		return nil, err
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, company, purpose)
	if err != nil {
		return nil, err
	}

	apiClient, err := adn.NewClient(adn.ClientConfig{
		BaseURL:     sync.ResolveEnvironmentURL(company.Environment),
		Certificate: &loaded.TLS,
		Log:         s.Log,
	})
	if err != nil {
		return nil, fmt.Errorf("configurar cliente ADN: %w", err)
	}
	return apiClient, nil
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

	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, companyCNPJ)
	if err != nil {
		return result, fmt.Errorf("empresa não encontrada: %w", err)
	}

	credential, err := lookupCredentialByID(ctx, s.CredentialStore, company.CredentialID)
	if err != nil {
		result.StatusExplanation = "Certificado digital não associado ou não encontrado para esta empresa."
		return result, nil //nolint:nilerr // intentional: return diagnostic error info in result
	}

	result.CertLoaded = true
	result.CertSubject = credential.SubjectName
	if credential.NotAfter != nil {
		result.CertExpiration = credential.NotAfter.Format("02/01/2006 15:04:05")
		if credential.NotAfter.Before(time.Now()) {
			result.StatusExplanation = "Certificado digital expirado."
			return result, nil
		}
	}

	apiClient, err := s.buildClient(ctx, companyCNPJ, "Teste de conexão")
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Erro ao carregar certificado/senha: %v", err)
		return result, nil
	}

	result.MTLSAccepted = true

	docResp, err := apiClient.FetchDocuments(ctx, adn.DistributionRequest{
		LastNSU:          0,
		ConsultationCNPJ: company.CNPJ,
	})

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
