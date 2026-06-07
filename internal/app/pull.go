package app

import (
	"context"
	"fmt"
	"log/slog"
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
	CompanyName       string
	CNPJ              string
	CredentialLabel   string
	CredentialCNPJ    string
	ConsultationBasis string
	DocumentsFound    int
	EventsFound       int
	Errors            int
	Duration          time.Duration
}

// Pull synchronises fiscal documents for the given company from the ADN API.
// It resolves the certificate password via App.CredentialProvider so that
// neither the CLI nor Wails need to wire cert loading themselves.
func (a *App) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	if err := cnpj.Validate(input.CNPJ); err != nil {
		return PullResult{}, fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)

	a.Log.InfoContext(ctx, "Iniciando sincronização de pull", slog.String("cnpj", cleanedCNPJ))

	// 1. Resolve company
	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return PullResult{}, fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return PullResult{}, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}
	credential, err := a.CredentialRepo.CredentialByID(ctx, company.CredentialID)
	if err != nil {
		return PullResult{}, fmt.Errorf("buscar credencial: %w", err)
	}
	if credential == nil {
		return PullResult{}, fmt.Errorf("credencial não encontrada para a empresa %s", company.Name)
	}

	// 2. Obtain certificate password via the injected provider
	if a.CredentialProvider == nil {
		return PullResult{}, fmt.Errorf("CredentialProvider não configurado")
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
		return PullResult{}, fmt.Errorf("obter senha do certificado: %w", err)
	}

	// 3. Load TLS certificate
	a.Log.DebugContext(ctx, "Carregando certificado TLS", slog.String("cert_path", credential.CertPath))
	loadedCert, err := cert.LoadPKCS12(credential.CertPath, pass)
	if err != nil {
		return PullResult{}, fmt.Errorf("carregar certificado: %w", err)
	}
	tlsCert := loadedCert.TLS
	inspection := loadedCert.Inspection
	credential.OwnerCNPJ = inspection.OwnerCNPJ
	credential.OwnerCNPJRoot = inspection.OwnerCNPJRoot
	credential.FingerprintSHA256 = inspection.FingerprintSHA256
	credential.SubjectName = inspection.SubjectName
	credential.NotBefore = &inspection.NotBefore
	credential.NotAfter = &inspection.NotAfter
	now := time.Now().UTC()
	credential.InspectedAt = &now
	if err := a.CredentialRepo.UpdateCredential(ctx, credential); err != nil {
		return PullResult{}, fmt.Errorf("persistir inspeção da credencial: %w", err)
	}

	consultationBasis, err := validateConsultationCompatibility(company, credential)
	if err != nil {
		return PullResult{}, err
	}

	// 4. Build ADN client
	apiClient, err := adn.NewClient(adn.ClientConfig{
		Environment: credential.Environment,
		Certificate: &tlsCert,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("configurar cliente ADN: %w", err)
	}

	// 5. Build file writer
	fileWriter := files.NewBlobStore(a.DataDir)

	// 6. Build sync service
	a.Log.DebugContext(ctx, "Construindo cliente ADN e SyncService")
	svc := syncservice.NewSyncService(a.SyncRepo, apiClient, fileWriter, a.Log)

	// 7. Run sync, collecting progress into result counters
	var result PullResult
	result.CompanyName = company.Name
	result.CNPJ = company.CNPJ
	result.CredentialLabel = credential.Label
	result.CredentialCNPJ = credential.OwnerCNPJ
	result.ConsultationBasis = consultationBasis

	progress := func(event nfse.ProgressEvent) {
		if event.Errors > result.Errors {
			result.Errors = event.Errors
		}
		if event.DocsFound > result.DocumentsFound {
			result.DocumentsFound = event.DocsFound
		}
	}

	start := time.Now()
	if err := svc.Sync(ctx, company, credential, consultationBasis, progress); err != nil {
		a.Log.ErrorContext(ctx, "Sincronização finalizada com erro", slog.String("error", err.Error()))
		return PullResult{}, fmt.Errorf("sincronização: %w", err)
	}
	result.Duration = time.Since(start)

	a.Log.InfoContext(ctx, "Sincronização concluída com sucesso",
		slog.Int("docs_found", result.DocumentsFound),
		slog.Int("errors", result.Errors),
		slog.Duration("duration", result.Duration),
	)

	return result, nil
}

func validateConsultationCompatibility(company *nfse.Company, credential *nfse.Credential) (string, error) {
	if credential.OwnerCNPJ == "" || credential.OwnerCNPJRoot == "" {
		return "", fmt.Errorf("o certificado não expõe um CNPJ proprietário utilizável para consulta")
	}
	if credential.Environment == "" {
		return "", fmt.Errorf("a credencial não possui ambiente configurado")
	}
	if company.CNPJRoot != credential.OwnerCNPJRoot {
		return "", fmt.Errorf("a credencial pertence à raiz %s e não pode consultar a empresa %s", credential.OwnerCNPJRoot, cnpj.Format(company.CNPJ))
	}
	if company.CNPJ == credential.OwnerCNPJ {
		return "exact_certificate_cnpj", nil
	}
	return "same_root_certificate", nil
}
