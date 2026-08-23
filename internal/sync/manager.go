package sync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	companypkg "github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

var (
	loadPKCS12   = cert.LoadPKCS12
	newADNClient = adn.NewClient
)

// CertPasswordRequest carries the context needed to ask for a certificate password.
type CertPasswordRequest struct {
	RequestID       string
	CompanyID       string
	CompanyName     string
	TargetCNPJ      string
	CredentialID    string
	CredentialLabel string
	CertPath        string
}

// CredentialProvider obtains the password of a certificate.
//
// The password is returned as a []byte so it can be zeroed after use; the
// caller owns the returned slice and should defer cert.ZeroBytes on it.
type CredentialProvider interface {
	GetCertPassword(ctx context.Context, req CertPasswordRequest) ([]byte, error)
}

type companyProvider interface {
	CompanyByCNPJ(ctx context.Context, cnpj string) (*nfse.Company, error)
}

type credentialProvider interface {
	CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error)
	UpdateCredential(ctx context.Context, c *nfse.Credential) error
}

type documentProvider interface {
	CountDocumentsByRole(ctx context.Context, companyID nfse.CompanyID) (map[string]int64, error)
}

// xmlStore reuses files.XMLStore
type xmlStore interface {
	files.XMLStore
}

type syncRunner interface {
	Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error
}

var newSyncRunner = func(repo *Store, client *adn.Client, xStore files.XMLStore, log *slog.Logger) syncRunner {
	return NewSyncService(repo, client, xStore, log)
}

type Manager struct {
	Log                *slog.Logger
	CompanyProvider    companyProvider
	CredentialProvider credentialProvider
	DocProvider        documentProvider
	SyncRepo           *Store
	XMLStore           xmlStore
	PassProvider       CredentialProvider
}

type PullInput struct {
	CNPJ string
	Mode string
}

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

func (m *Manager) Pull(ctx context.Context, input PullInput) (PullResult, error) {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return PullResult{}, err
	}
	mode, err := parsePullMode(input.Mode)
	if err != nil {
		return PullResult{}, err
	}

	m.Log.InfoContext(ctx, "Iniciando sincronização de pull", slog.String("cnpj", cleanedCNPJ))

	company, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return PullResult{}, err
	}
	credential, err := m.CredentialProvider.CredentialByID(ctx, company.CredentialID)
	if err != nil {
		return PullResult{}, fmt.Errorf("resolver credencial da empresa %s: %w", company.Name, err)
	}

	if err := validateCertificatePath(credential.CertPath); err != nil {
		return PullResult{}, err
	}
	pass, err := m.PassProvider.GetCertPassword(ctx, CertPasswordRequest{
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
	defer cert.ZeroBytes(pass)

	m.Log.DebugContext(ctx, "Carregando certificado TLS", slog.String("cert_path", credential.CertPath))
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
	if err := m.CredentialProvider.UpdateCredential(ctx, credential); err != nil {
		return PullResult{}, fmt.Errorf("persistir inspeção da credencial: %w", err)
	}

	consultationBasis, err := validateConsultationCompatibility(company, credential)
	if err != nil {
		return PullResult{}, err
	}

	apiClient, err := newADNClient(adn.ClientConfig{
		BaseURL:     ResolveEnvironmentURL(company.Environment),
		Certificate: &tlsCert,
		Log:         m.Log,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("configurar cliente ADN: %w", err)
	}

	m.Log.DebugContext(ctx, "Construindo cliente ADN e SyncService")
	svc := newSyncRunner(m.SyncRepo, apiClient, m.XMLStore, m.Log)

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

	snapshot, err := m.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
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

	m.Log.InfoContext(
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
		return "", companypkg.ErrCredentialNoOwner
	}
	if company.Environment == "" {
		return "", companypkg.ErrCompanyNoEnvironment
	}
	if company.CNPJRoot != credential.OwnerCNPJRoot {
		return "", fmt.Errorf("%w: credencial (raiz %s) vs empresa (%s)", companypkg.ErrCredentialMismatch, credential.OwnerCNPJRoot, cnpj.Format(company.CNPJ))
	}
	if company.CNPJ == credential.OwnerCNPJ {
		return nfse.ConsultationBasisExactCertificateCNPJ, nil
	}
	return nfse.ConsultationBasisSameRootCertificate, nil
}

func ResolveEnvironmentURL(env nfse.Environment) string {
	switch env {
	case nfse.EnvironmentProduction:
		return adn.BaseURLProduction
	case nfse.EnvironmentRestricted:
		return adn.BaseURLRestrictedProduction
	default:
		return ""
	}
}

type StatusResult struct {
	CompanyName        string
	CNPJ               string
	Environment        string
	ConsultationCNPJ   string
	CredentialCNPJ     string
	CredentialNotAfter *time.Time
	LastProcessedNSU   int64
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	TotalEmitidas      int64
	TotalTomadas       int64
}

func (m *Manager) Status(ctx context.Context, rawCNPJ string) (StatusResult, error) {
	cleanedCNPJ, err := normalizeCNPJ(rawCNPJ)
	if err != nil {
		return StatusResult{}, err
	}
	company, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return StatusResult{}, err
	}
	credential, err := m.CredentialProvider.CredentialByID(ctx, company.CredentialID)
	if err != nil {
		return StatusResult{}, fmt.Errorf("resolver credencial da empresa %s: %w", company.Name, err)
	}
	snapshot, err := m.SyncRepo.LatestSyncSnapshot(ctx, company.ID, company.Environment, company.CNPJ)
	if err != nil {
		return StatusResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}

	counts, err := m.DocProvider.CountDocumentsByRole(ctx, company.ID)
	if err != nil {
		return StatusResult{}, fmt.Errorf("contar documentos: %w", err)
	}

	result := StatusResult{
		CompanyName:        company.Name,
		CNPJ:               company.CNPJ,
		Environment:        string(company.Environment),
		ConsultationCNPJ:   company.CNPJ,
		CredentialCNPJ:     credential.OwnerCNPJ,
		CredentialNotAfter: credential.NotAfter,
		TotalEmitidas:      counts["prestada"],
		TotalTomadas:       counts["tomada"],
	}
	if snapshot.State != nil {
		result.LastProcessedNSU = snapshot.State.LastProcessedNSU
		result.LastFoundNSU = snapshot.State.LastFoundNSU
		if snapshot.State.LastSuccessAt != nil {
			result.LastSyncAt = snapshot.State.LastSuccessAt
		}
	}
	if snapshot.Run != nil {
		result.LastRunStatus = string(snapshot.Run.Status)
		result.LastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			result.LastSyncAt = snapshot.Run.FinishedAt
		}
	}
	return result, nil
}

func normalizeCNPJ(raw string) (string, error) {
	if err := cnpj.Validate(raw); err != nil {
		return "", fmt.Errorf("CNPJ inválido: %w", err)
	}
	return cnpj.Clean(raw), nil
}

func validateCertificatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo de certificado não encontrado: %s", path)
		}
		return fmt.Errorf("verificar certificado: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("caminho do certificado aponta para um diretório: %s", path)
	}
	return nil
}

type ResetSyncInput struct {
	CNPJ string
}

func (m *Manager) ResetSyncState(ctx context.Context, input ResetSyncInput) error {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return err
	}
	company, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return err
	}

	if err := m.SyncRepo.ResetSyncState(ctx, nfse.ResetSyncStateParams{
		CompanyID: company.ID,
	}); err != nil {
		return fmt.Errorf("resetar estado de sincronização: %w", err)
	}

	return nil
}
