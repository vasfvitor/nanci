package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

type QueryNFSeInput struct {
	CNPJ        string
	ChaveAcesso string
}

func (a *App) QueryNFSe(ctx context.Context, input QueryNFSeInput) (string, error) {
	apiClient, err := a.buildClientForQuery(ctx, input.CNPJ)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("NFSe/%s", input.ChaveAcesso)
	return a.queryGenericEndpoint(ctx, apiClient, path)
}

func (a *App) QueryNFSeEvents(ctx context.Context, input QueryNFSeInput) (string, error) {
	apiClient, err := a.buildClientForQuery(ctx, input.CNPJ)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("NFSe/%s/Eventos", input.ChaveAcesso)
	return a.queryGenericEndpoint(ctx, apiClient, path)
}

func (a *App) queryGenericEndpoint(ctx context.Context, apiClient *adn.Client, path string) (string, error) {
	var response json.RawMessage
	err := apiClient.RawGet(ctx, path, &response)
	if err != nil {
		return "", fmt.Errorf("falha ao consultar API ADN: %w", err)
	}

	pretty, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return string(response), nil
	}
	return string(pretty), nil
}

func (a *App) buildClientForQuery(ctx context.Context, companyCNPJ string) (*adn.Client, error) {
	cleanedCNPJ := cnpj.Clean(companyCNPJ)
	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil || company == nil {
		return nil, fmt.Errorf("empresa não encontrada: %v", err)
	}

	credential, err := a.CredentialRepo.CredentialByID(ctx, company.CredentialID)
	if err != nil || credential == nil {
		return nil, fmt.Errorf("credencial não encontrada")
	}

	if a.CredentialProvider == nil {
		return nil, fmt.Errorf("CredentialProvider não configurado")
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

	loadedCert, err := cert.LoadPKCS12(credential.CertPath, pass)
	if err != nil {
		return nil, fmt.Errorf("carregar certificado: %w", err)
	}

	tlsCert := loadedCert.TLS
	apiClient, err := adn.NewClient(adn.ClientConfig{
		Environment: company.Environment,
		Certificate: &tlsCert,
		Log:         a.Log,
	})
	if err != nil {
		return nil, fmt.Errorf("configurar cliente ADN: %w", err)
	}
	return apiClient, nil
}
