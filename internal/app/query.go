package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/nfse"
)

var loadPKCS12 = cert.LoadPKCS12
var newADNClient = adn.NewClient

type QueryNFSeInput struct {
	CNPJ        string
	ChaveAcesso string
}

func (a *App) QueryNFSeEvents(ctx context.Context, input QueryNFSeInput) (string, error) {
	accessKey, err := validateQueryAccessKey(input.ChaveAcesso)
	if err != nil {
		return "", err
	}

	apiClient, err := a.buildClientForQuery(ctx, input.CNPJ)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("NFSe/%s/Eventos", accessKey)
	return a.queryGenericEndpoint(ctx, apiClient, path)
}

func validateQueryAccessKey(raw string) (nfse.AccessKey, error) {
	accessKey, err := nfse.ParseAccessKey(raw)
	if err != nil {
		return "", fmt.Errorf("chave de acesso inválida: %w", err)
	}
	return accessKey, nil
}

func (a *App) queryGenericEndpoint(ctx context.Context, apiClient *adn.Client, path string) (string, error) {
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

func (a *App) buildClientForQuery(ctx context.Context, companyCNPJ string) (*adn.Client, error) {
	company, err := a.companyByCNPJ(ctx, companyCNPJ)
	if err != nil {
		return nil, err
	}

	credential, err := a.credentialByID(ctx, company.CredentialID)
	if err != nil {
		return nil, err
	}

	if err := validateCertificatePath(credential.CertPath); err != nil {
		return nil, err
	}

	pass, err := a.CredentialProvider.GetCertPassword(ctx, CertPasswordRequest{
		RequestID:       nfse.GenerateID(),
		CompanyID:       string(company.ID),
		CompanyName:     company.Name,
		TargetCNPJ:      company.CNPJ,
		CredentialID:    string(credential.ID),
		CredentialLabel: credential.Label,
		CertPath:        credential.CertPath,
	})
	if err != nil {
		return nil, fmt.Errorf("senha do certificado: %w", err)
	}

	loadedCert, err := loadPKCS12(credential.CertPath, pass)
	if err != nil {
		return nil, fmt.Errorf("carregar certificado: %w", err)
	}

	tlsCert := loadedCert.TLS
	apiClient, err := newADNClient(adn.ClientConfig{
		BaseURL:     resolveEnvironmentURL(company.Environment),
		Certificate: &tlsCert,
		Log:         a.Log,
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
func (a *App) TestConnection(ctx context.Context, companyCNPJ string) (ConnectionTestResult, error) {
	result := ConnectionTestResult{}

	company, err := a.companyByCNPJ(ctx, companyCNPJ)
	if err != nil {
		return result, fmt.Errorf("empresa não encontrada: %w", err)
	}

	credential, err := a.credentialByID(ctx, company.CredentialID)
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

	// Try to build client (which loads certificate and asks for password if needed)
	apiClient, err := a.buildClientForQuery(ctx, companyCNPJ)
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Erro ao carregar certificado/senha: %v", err)
		return result, nil
	}

	result.MTLSAccepted = true

	// Call FetchDocuments for lastNSU 0 to test connectivity
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
