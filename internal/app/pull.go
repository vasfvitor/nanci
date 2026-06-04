package app

import (
	"context"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	syncservice "github.com/vasfvitor/nanci/internal/service/sync"
)

// PullInput is the input for the Pull use case.
type PullInput struct {
	CNPJ string
}

// PullResult summarises a completed sync run.
type PullResult struct {
	CompanyName    string
	CNPJ           string
	DocumentsFound int
	EventsFound    int
	Errors         int
	Duration       time.Duration
}

// Pull synchronises fiscal documents for the given company from the ADN API.
// It resolves the certificate password via App.CredentialProvider so that
// neither the CLI nor Wails need to wire cert loading themselves.
func (a *App) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	if err := cnpj.Validate(input.CNPJ); err != nil {
		return PullResult{}, fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)

	// 1. Resolve company
	company, err := a.Store.GetCompany(ctx, cleanedCNPJ)
	if err != nil {
		return PullResult{}, fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return PullResult{}, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	// 2. Obtain certificate password via the injected provider
	if a.CredentialProvider == nil {
		return PullResult{}, fmt.Errorf("CredentialProvider não configurado")
	}
	pass, err := a.CredentialProvider.GetCertPassword(ctx, CertPasswordRequest{
		CompanyID: company.ID,
		CNPJ:      company.CNPJ,
		CertPath:  company.CertPath,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("obter senha do certificado: %w", err)
	}

	// 3. Load TLS certificate
	tlsCert, err := cert.LoadPKCS12(company.CertPath, pass)
	if err != nil {
		return PullResult{}, fmt.Errorf("carregar certificado: %w", err)
	}

	// 4. Build ADN client
	httpClient := adn.NewHTTPClient(tlsCert)
	apiClient, err := adn.NewClient(httpClient, company.Environment)
	if err != nil {
		return PullResult{}, fmt.Errorf("configurar cliente ADN: %w", err)
	}

	// 5. Build file writer
	fileWriter := files.NewWriter(a.DataDir)

	// 6. Build sync service
	svc := syncservice.NewSyncService(a.Store, apiClient, fileWriter)

	// 7. Run sync, collecting progress into result counters
	var result PullResult
	result.CompanyName = company.Name
	result.CNPJ = company.CNPJ

	progress := func(event nfse.ProgressEvent) {
		if event.Errors > result.Errors {
			result.Errors = event.Errors
		}
		if event.DocsFound > result.DocumentsFound {
			result.DocumentsFound = event.DocsFound
		}
	}

	start := time.Now()
	if err := svc.Sync(ctx, company, progress); err != nil {
		return PullResult{}, fmt.Errorf("sincronização: %w", err)
	}
	result.Duration = time.Since(start)

	return result, nil
}
