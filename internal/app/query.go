package app

import (
	"context"
	"encoding/json"
	"fmt"

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

func (a *App) QueryNFSe(ctx context.Context, input QueryNFSeInput) (string, error) {
	accessKey, err := validateQueryAccessKey(input.ChaveAcesso)
	if err != nil {
		return "", err
	}

	apiClient, err := a.buildClientForQuery(ctx, input.CNPJ)
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("NFSe/%s", accessKey)
	return a.queryGenericEndpoint(ctx, apiClient, path)
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
		return string(response), nil
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
		Environment: company.Environment,
		Certificate: &tlsCert,
		Log:         a.Log,
	})
	if err != nil {
		return nil, fmt.Errorf("configurar cliente ADN: %w", err)
	}
	return apiClient, nil
}
