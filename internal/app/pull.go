package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	syncservice "github.com/vasfvitor/nanci/internal/service/sync"
)

type syncRunner interface {
	Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error
}

var newSyncRunner = func(repo SyncRepository, client *adn.Client, xmlStore files.XMLStore, log *slog.Logger) syncRunner {
	return syncservice.NewSyncService(repo, client, xmlStore, log)
}

// PullInput is the input for the Pull use case.
type PullInput struct {
	CNPJ string
	Mode string
}

// PullResult summarises a completed sync run.
type PullResult struct {
	CompanyName              string
	CNPJ                     string
	CredentialLabel          string
	CredentialCNPJ           string
	ConsultationBasis        string
	Status                   string
	StopReason               string
	LastProcessedNSU         int64
	LastFoundNSU             *int64
	EmptyStreak              int
	DocumentsFound           int
	EventsFound              int
	DocumentsSaved           int
	EventsSaved              int
	DocumentsSkippedByPolicy int
	EventsSkippedByPolicy    int
	Errors                   int
	Duration                 time.Duration
}

// Pull synchronises fiscal documents for the given company from the ADN API.
// It resolves the certificate password via App.CredentialProvider so that
// neither the CLI nor Wails need to wire cert loading themselves.
func (a *App) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return PullResult{}, err
	}
	mode, err := parsePullMode(input.Mode)
	if err != nil {
		return PullResult{}, err
	}

	a.Log.InfoContext(ctx, "Iniciando sincronização de pull", slog.String("cnpj", cleanedCNPJ))

	// 1. Resolve company
	company, err := a.companyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return PullResult{}, err
	}
	credential, err := a.credentialByID(ctx, company.CredentialID)
	if err != nil {
		return PullResult{}, fmt.Errorf("resolver credencial da empresa %s: %w", company.Name, err)
	}

	// 2. Obtain certificate password via the injected provider
	if err := validateCertificatePath(credential.CertPath); err != nil {
		return PullResult{}, err
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
	loadedCert, err := loadPKCS12(credential.CertPath, pass)
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
	apiClient, err := newADNClient(adn.ClientConfig{
		BaseURL:     resolveEnvironmentURL(company.Environment),
		Certificate: &tlsCert,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("configurar cliente ADN: %w", err)
	}

	// 5. Build sync service
	a.Log.DebugContext(ctx, "Construindo cliente ADN e SyncService")
	svc := newSyncRunner(a.SyncRepo, apiClient, a.XMLStore, a.Log)

	// 6. Run sync, collecting progress into result counters
	var result PullResult
	result.CompanyName = company.Name
	result.CNPJ = company.CNPJ
	result.CredentialLabel = credential.Label
	result.CredentialCNPJ = credential.OwnerCNPJ
	result.ConsultationBasis = string(consultationBasis)

	progress := func(event nfse.ProgressEvent) {
		if event.Errors > result.Errors {
			result.Errors = event.Errors
		}
		if event.DocsFound > result.DocumentsFound {
			result.DocumentsFound = event.DocsFound
		}
		if event.DocumentsSaved > result.DocumentsSaved {
			result.DocumentsSaved = event.DocumentsSaved
		}
		if event.EventsSaved > result.EventsSaved {
			result.EventsSaved = event.EventsSaved
		}
		if event.DocumentsSkippedByPolicy > result.DocumentsSkippedByPolicy {
			result.DocumentsSkippedByPolicy = event.DocumentsSkippedByPolicy
		}
		if event.EventsSkippedByPolicy > result.EventsSkippedByPolicy {
			result.EventsSkippedByPolicy = event.EventsSkippedByPolicy
		}
	}

	start := time.Now()
	if err := svc.Sync(ctx, company, credential, string(consultationBasis), mode, progress); err != nil {
		a.Log.ErrorContext(ctx, "Sincronização finalizada com erro", slog.String("error", err.Error()))
		return PullResult{}, fmt.Errorf("sincronização: %w", err)
	}
	result.Duration = time.Since(start)

	snapshot, err := a.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
	if err != nil {
		return PullResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		result.LastProcessedNSU = snapshot.State.LastProcessedNSU
		result.LastFoundNSU = snapshot.State.LastFoundNSU
		result.EmptyStreak = snapshot.State.LastEmptyStreak
	}
	if snapshot.Run != nil {
		result.Status = string(snapshot.Run.Status)
		result.StopReason = string(snapshot.Run.StopReason)
		result.Errors = snapshot.Run.ErrorsCount
		result.DocumentsFound = snapshot.Run.DocumentsFound
	}
	if result.DocumentsSaved == 0 {
		result.DocumentsSaved = result.DocumentsFound
	}

	a.Log.InfoContext(
		ctx, "Sincronização concluída com sucesso",
		slog.Int("docs_found", result.DocumentsFound),
		slog.Int("errors", result.Errors),
		slog.Duration("duration", result.Duration),
	)

	return result, nil
}

func parsePullMode(raw string) (nfse.SyncMode, error) {
	if raw == "" {
		return nfse.SyncModeNormal, nil
	}
	return nfse.ParseSyncMode(raw)
}

func validateConsultationCompatibility(company *nfse.Company, credential *nfse.Credential) (nfse.ConsultationBasis, error) {
	if credential.OwnerCNPJ == "" || credential.OwnerCNPJRoot == "" {
		return "", ErrCredentialNoOwner
	}
	if company.Environment == "" {
		return "", ErrCompanyNoEnvironment
	}
	if company.CNPJRoot != credential.OwnerCNPJRoot {
		return "", fmt.Errorf("%w: credencial (raiz %s) vs empresa (%s)", ErrCredentialMismatch, credential.OwnerCNPJRoot, cnpj.Format(company.CNPJ))
	}
	if company.CNPJ == credential.OwnerCNPJ {
		return nfse.ConsultationBasisExactCertificateCNPJ, nil
	}
	return nfse.ConsultationBasisSameRootCertificate, nil
}

func resolveEnvironmentURL(env nfse.Environment) string {
	switch env {
	case nfse.EnvironmentProduction:
		return adn.BaseURLProduction
	case nfse.EnvironmentRestricted:
		return adn.BaseURLRestrictedProduction
	default:
		return ""
	}
}
