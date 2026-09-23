package sync

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	gosync "sync"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/uf"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
)

var (
	loadPKCS12   = cert.LoadPKCS12
	newADNClient = adn.NewClient
	// newSEFAZClient builds the NF-e distribution client; tests swap it.
	newSEFAZClient = func(cfg sefaz.ClientConfig) (nfeFetcher, error) {
		return sefaz.NewClient(cfg)
	}
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
	// Purpose tells the user what the password is for, e.g. "Sincronização NFS-e".
	Purpose string
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

var newSyncRunner = func(repo *Store, src Source, log *slog.Logger) syncRunner {
	return NewSyncService(repo, src, log)
}

type Manager struct {
	Log                *slog.Logger
	CompanyProvider    companyProvider
	CredentialProvider credentialProvider
	DocProvider        documentProvider
	SyncRepo           *Store
	XMLStore           xmlStore
	// Certificates loads the certificate of each pull.
	Certificates *CertificateLoader
	// NFeRepo stores the NF-e distribution; pulls with Source nfe need it.
	NFeRepo *store.NFeRepository

	runningMu gosync.Mutex
	running   map[string]bool // "companyID:source" pulls in flight in this process
}

type PullInput struct {
	CNPJ string
	Mode string
	// Source is the distribution service to pull from. Empty means NFS-e.
	Source nfse.SyncSource
}

type PullResult struct {
	Source                   nfse.SyncSource
	CompanyName              string
	CNPJ                     string
	CredentialLabel          string
	CredentialCNPJ           string
	ConsultationBasis        string
	Status                   string
	StopReason               string
	LastProcessedNSU         int64 // the cursor; for NF-e, the last ultNSU
	LastFoundNSU             *int64
	MaxNSU                   int64 // highest NSU the source reported; 0 when unknown
	EmptyStreak              int
	DocumentsFound           int
	EventsFound              int
	DocumentsSaved           int
	EventsSaved              int
	CompletasSaved           int // NF-e procNFe stored
	ResumosSaved             int // NF-e resNFe stored
	DocumentsSkippedByPolicy int
	EventsSkippedByPolicy    int
	Errors                   int
	NextAllowedAt            *time.Time // set while the source must not be queried
	RequestsLastHour         int
	RequestBudget            int // requests allowed per hour; 0 means unlimited
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
	source, err := resolveSyncSource(input.Source)
	if err != nil {
		return PullResult{}, err
	}

	m.Log.InfoContext(ctx, "Iniciando sincronização de pull", slog.String("cnpj", cleanedCNPJ), slog.String("source", string(source)))

	company, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return PullResult{}, err
	}
	if err := m.checkSource(company, source); err != nil {
		return PullResult{}, err
	}
	sourceState, err := m.SyncRepo.SourceState(ctx, company.ID, source)
	if err != nil {
		return PullResult{}, fmt.Errorf("carregar estado da origem: %w", err)
	}
	if err := checkBlocked(source, sourceState, time.Now()); err != nil {
		return PullResult{}, err
	}
	release, err := m.ReserveSource(company.ID, source)
	if err != nil {
		return PullResult{}, err
	}
	defer release()

	loaded, err := m.Certificates.LoadForCompany(ctx, company, "Sincronização "+sourceLabel(source))
	if err != nil {
		return PullResult{}, err
	}
	credential := loaded.Credential

	src, err := m.newSource(company, source, loaded.TLS)
	if err != nil {
		return PullResult{}, err
	}
	svc := newSyncRunner(m.SyncRepo, src, m.Log)

	var result PullResult
	result.Source = source
	result.CompanyName = company.Name
	result.CNPJ = company.CNPJ
	result.CredentialLabel = credential.Label
	result.CredentialCNPJ = credential.OwnerCNPJ
	result.ConsultationBasis = string(loaded.Basis)

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
		if event.CompletasSaved > result.CompletasSaved {
			result.CompletasSaved = event.CompletasSaved
		}
		if event.ResumosSaved > result.ResumosSaved {
			result.ResumosSaved = event.ResumosSaved
		}
	}

	start := time.Now()
	if err := svc.Sync(ctx, company, credential, string(loaded.Basis), mode, progress); err != nil {
		return PullResult{}, fmt.Errorf("sincronização: %w", err)
	}
	result.Duration = time.Since(start)

	snapshot, err := m.SyncRepo.LatestSyncSnapshot(ctx, company.ID, source, company.Environment, company.CNPJ)
	if err != nil {
		return PullResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		result.LastProcessedNSU = snapshot.State.LastProcessedNSU
		result.LastFoundNSU = snapshot.State.LastFoundNSU
		result.EmptyStreak = snapshot.State.LastEmptyStreak
		if snapshot.State.MaxNSU != nil {
			result.MaxNSU = *snapshot.State.MaxNSU
		}
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
	limits, err := m.SourceLimits(ctx, company.ID, source)
	if err != nil {
		return PullResult{}, err
	}
	result.NextAllowedAt = limits.NextAllowedAt
	result.RequestsLastHour = limits.RequestsLastHour
	result.RequestBudget = limits.RequestBudget

	m.Log.InfoContext(
		ctx, "Sincronização concluída com sucesso",
		slog.Int("docs_found", result.DocumentsFound),
		slog.Int("errors", result.Errors),
		slog.Duration("duration", result.Duration),
	)

	return result, nil
}

// ReserveSource marks the (company, source) pair as busy in this process, as
// a running pull does, and returns the func that clears the mark. Pull holds
// it for the whole run; callers that change the source's data hold it to keep
// pulls out. A second reservation of the same pair gets ErrSyncRunning
// instead of interrupting the first. This does not guard against another
// process; StartRun cleans up after a crashed one.
func (m *Manager) ReserveSource(companyID nfse.CompanyID, source nfse.SyncSource) (func(), error) {
	key := string(companyID) + ":" + string(source)

	m.runningMu.Lock()
	defer m.runningMu.Unlock()
	if m.running[key] {
		return nil, fmt.Errorf("%w (%s)", ErrSyncRunning, sourceLabel(source))
	}
	if m.running == nil {
		m.running = make(map[string]bool)
	}
	m.running[key] = true

	return func() {
		m.runningMu.Lock()
		defer m.runningMu.Unlock()
		delete(m.running, key)
	}, nil
}

// SourceLimits is when a source may be queried again and how much of its
// hourly request budget is spent.
type SourceLimits struct {
	NextAllowedAt    *time.Time          // set while the source must not be queried
	BlockedReason    nfse.SyncStopReason // why NextAllowedAt is set
	RequestsLastHour int
	RequestBudget    int // requests allowed per hour; 0 means unlimited
}

// SourceLimits reports the request limits of the company's source now.
func (m *Manager) SourceLimits(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource) (SourceLimits, error) {
	now := time.Now().UTC()
	state, err := m.SyncRepo.SourceState(ctx, companyID, source)
	if err != nil {
		return SourceLimits{}, fmt.Errorf("carregar estado da origem: %w", err)
	}
	var limits SourceLimits
	if state.BlockedUntil != nil && now.Before(*state.BlockedUntil) {
		limits.NextAllowedAt = state.BlockedUntil
		limits.BlockedReason = state.BlockedReason
	}
	count, _, err := m.SyncRepo.RequestsSince(ctx, companyID, source, now.Add(-time.Hour))
	if err != nil {
		return SourceLimits{}, fmt.Errorf("contar consultas da última hora: %w", err)
	}
	limits.RequestsLastHour = count
	if source == nfse.SyncSourceNFe {
		limits.RequestBudget = NFeRequestsPerHour
	}
	return limits, nil
}

// checkSource fails, before any password prompt, when the company cannot be
// pulled from source.
func (m *Manager) checkSource(company *nfse.Company, source nfse.SyncSource) error {
	switch source {
	case nfse.SyncSourceNFSe:
		return nil
	case nfse.SyncSourceNFe:
		if m.NFeRepo == nil {
			return errors.New("repositório de NF-e não configurado")
		}
		_, err := companyUFCode(company)
		return err
	default:
		return fmt.Errorf("origem de sincronização %q ainda não disponível", source)
	}
}

// newSource builds the Source of one pull once the certificate is loaded.
// checkSource must have accepted company and source.
func (m *Manager) newSource(company *nfse.Company, source nfse.SyncSource, tlsCert tls.Certificate) (Source, error) {
	switch source {
	case nfse.SyncSourceNFSe:
		apiClient, err := newADNClient(adn.ClientConfig{
			BaseURL:     ResolveEnvironmentURL(company.Environment),
			Certificate: &tlsCert,
			Log:         m.Log,
		})
		if err != nil {
			return nil, fmt.Errorf("configurar cliente ADN: %w", err)
		}
		return NewNFSeSource(apiClient, m.SyncRepo, m.XMLStore, m.Log), nil
	case nfse.SyncSourceNFe:
		cUFAutor, err := companyUFCode(company)
		if err != nil {
			return nil, err
		}
		client, err := newSEFAZClient(sefaz.ClientConfig{
			Environment: company.Environment,
			Certificate: &tlsCert,
			Log:         m.Log,
		})
		if err != nil {
			return nil, fmt.Errorf("configurar cliente SEFAZ: %w", err)
		}
		return NewNFeSource(client, m.NFeRepo, m.XMLStore, m.Log, cUFAutor), nil
	default:
		return nil, fmt.Errorf("origem de sincronização %q ainda não disponível", source)
	}
}

// companyUFCode returns the IBGE code of the company's UF, which the NF-e
// distribution requires as cUFAutor.
func companyUFCode(company *nfse.Company) (int, error) {
	if company.UF == "" {
		return 0, errors.New("empresa sem UF cadastrada; use company update --uf")
	}
	code, ok := uf.Code(company.UF)
	if !ok {
		return 0, fmt.Errorf("UF %q da empresa é inválida; use company update --uf", company.UF)
	}
	return code, nil
}

// resolveSyncSource defaults an empty source to NFS-e, the source every
// caller used before sources existed.
func resolveSyncSource(source nfse.SyncSource) (nfse.SyncSource, error) {
	if source == "" {
		return nfse.SyncSourceNFSe, nil
	}
	return nfse.ParseSyncSource(string(source))
}

func parsePullMode(raw string) (nfse.SyncMode, error) {
	if raw == "" {
		return nfse.SyncModeNormal, nil
	}
	return nfse.ParseSyncMode(raw)
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
	snapshot, err := m.SyncRepo.LatestSyncSnapshot(ctx, company.ID, nfse.SyncSourceNFSe, company.Environment, company.CNPJ)
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

type ResetSyncInput struct {
	CNPJ string
	// Source is the distribution service whose cursor is reset. Empty means NFS-e.
	Source nfse.SyncSource
}

func (m *Manager) ResetSyncState(ctx context.Context, input ResetSyncInput) error {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return err
	}
	source, err := resolveSyncSource(input.Source)
	if err != nil {
		return err
	}
	company, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return err
	}

	if err := m.SyncRepo.ResetSyncState(ctx, nfse.ResetSyncStateParams{
		CompanyID: company.ID,
		Source:    source,
	}); err != nil {
		return fmt.Errorf("resetar estado de sincronização: %w", err)
	}

	return nil
}
