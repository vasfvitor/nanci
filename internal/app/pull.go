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
	"github.com/vasfvitor/nanci/internal/store"
	syncrun "github.com/vasfvitor/nanci/internal/syncrun"
)

type syncRunner interface {
	Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error
}

// newSyncRunner wires the orchestrator with the wider concrete needed by
// syncrun.SyncRepository (8 methods). The app's own SyncSnapshotStore
// uses a 3-method consumer interface, but the orchestrator still needs the
// full 8-method surface; *store.SyncRepository satisfies both structurally.
var newSyncRunner = func(repo *store.SyncRepository, client *adn.Client, xmlStore files.XMLStore, log *slog.Logger) syncRunner {
	return syncrun.NewSyncService(repo, client, xmlStore, log)
}

var (
	loadPKCS12   = cert.LoadPKCS12
	newADNClient = adn.NewClient
)

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

// SyncService owns the sync lifecycle use cases: Pull (run a sync) and
// ResetSyncState (clear the cursor). It depends on the certificate, the
// ADN client constructor, and the orchestrator runner.
type SyncService struct {
	Log                *slog.Logger
	CompanyRepo        CompanyRepository
	CredentialRepo     CredentialRepository
	SyncRepo           *store.SyncRepository
	XMLStore           files.XMLStore
	CredentialProvider CredentialProvider
}

func newSyncService(d Dependencies) SyncService {
	return SyncService{
		Log:                d.Log,
		CompanyRepo:        d.CompanyRepo,
		CredentialRepo:     d.CredentialRepo,
		SyncRepo:           d.SyncRepo,
		XMLStore:           d.XMLStore,
		CredentialProvider: d.CredentialProvider,
	}
}

// Pull synchronises fiscal documents for the given company from the ADN API.
// It resolves the certificate password via the injected provider so that
// neither the CLI nor Wails need to wire cert loading themselves.
func (s SyncService) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return PullResult{}, err
	}
	mode, err := parsePullMode(input.Mode)
	if err != nil {
		return PullResult{}, err
	}

	s.Log.InfoContext(ctx, "Iniciando sincronização de pull", slog.String("cnpj", cleanedCNPJ))

	company, err := lookupCompanyByCNPJ(ctx, s.CompanyRepo, cleanedCNPJ)
	if err != nil {
		return PullResult{}, err
	}
	credential, err := lookupCredentialByID(ctx, s.CredentialRepo, company.CredentialID)
	if err != nil {
		return PullResult{}, fmt.Errorf("resolver credencial da empresa %s: %w", company.Name, err)
	}

	if err := validateCertificatePath(credential.CertPath); err != nil {
		return PullResult{}, err
	}
	pass, err := s.CredentialProvider.GetCertPassword(ctx, CertPasswordRequest{
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

	s.Log.DebugContext(ctx, "Carregando certificado TLS", slog.String("cert_path", credential.CertPath))
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
	if err := s.CredentialRepo.UpdateCredential(ctx, credential); err != nil {
		return PullResult{}, fmt.Errorf("persistir inspeção da credencial: %w", err)
	}

	consultationBasis, err := validateConsultationCompatibility(company, credential)
	if err != nil {
		return PullResult{}, err
	}

	apiClient, err := newADNClient(adn.ClientConfig{
		BaseURL:     resolveEnvironmentURL(company.Environment),
		Certificate: &tlsCert,
		Log:         s.Log,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("configurar cliente ADN: %w", err)
	}

	s.Log.DebugContext(ctx, "Construindo cliente ADN e SyncService")
	svc := newSyncRunner(s.SyncRepo, apiClient, s.XMLStore, s.Log)

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
		return PullResult{}, fmt.Errorf("sincronização: %w", err)
	}
	result.Duration = time.Since(start)

	snapshot, err := s.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
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

	s.Log.InfoContext(
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
