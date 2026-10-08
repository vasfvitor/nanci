package company

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/uf"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

var (
	ErrCredentialMismatch   = errors.New("a credencial informada não pertence à mesma raiz do CNPJ da empresa")
	ErrCredentialNoOwner    = errors.New("o certificado não expõe um CNPJ proprietário utilizável para consulta")
	ErrCompanyNoEnvironment = errors.New("a empresa não possui ambiente configurado")
)

type storeInterface interface {
	CreateCompany(ctx context.Context, c *Company) error
	ListCompanies(ctx context.Context) ([]Company, error)
	CompanyByCNPJ(ctx context.Context, cnpj string) (*Company, error)
	UpdateCompany(ctx context.Context, c *Company) error
	AssignCredential(ctx context.Context, companyID dfe.CompanyID, credentialID dfe.CredentialID) error
}

type credentialProvider interface {
	CredentialByID(ctx context.Context, id dfe.CredentialID) (*credential.Credential, error)
	CreateCredential(ctx context.Context, cred *credential.Credential) error
}

type syncProvider interface {
	LatestSyncSnapshot(ctx context.Context, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, cnpj string) (syncstate.SyncSnapshot, error)
	HasSyncState(ctx context.Context, params syncstate.HasSyncStateParams) (bool, error)
}

// AddCompanyInput carries the data required to register a new company.
type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     dfe.Environment
	UF              string // optional state sigla, e.g. "SP"
	SyncStartPolicy syncstate.SyncStartPolicy
	SyncStartDate   *time.Time
}

// UpdateCompanyInput carries data to update a company
type UpdateCompanyInput struct {
	CNPJ            string
	Name            string
	Environment     dfe.Environment
	UF              string // optional state sigla, e.g. "SP"
	SyncStartPolicy syncstate.SyncStartPolicy
	SyncStartDate   *time.Time
}

// AssignCredentialInput changes the active credential of a company.
type AssignCredentialInput struct {
	CompanyCNPJ  string
	CredentialID string
}

// Manager owns the company use cases.
type Manager struct {
	store       storeInterface
	credentials credentialProvider
	syncs       syncProvider
}

func NewManager(s storeInterface, creds credentialProvider, syncs syncProvider) *Manager {
	return &Manager{
		store:       s,
		credentials: creds,
		syncs:       syncs,
	}
}

// AddCompany registers a new company in the store.
func (m *Manager) AddCompany(ctx context.Context, input AddCompanyInput) error {
	cleanedCNPJ, err := cnpj.Normalize(input.CNPJ)
	if err != nil {
		return err
	}
	root, _ := cnpj.Root(cleanedCNPJ)
	sigla, err := normalizeUF(input.UF)
	if err != nil {
		return err
	}

	credential, err := m.resolveCredentialForCompany(ctx, input)
	if err != nil {
		return err
	}

	company := &Company{
		ID:                 dfe.CompanyID(dfe.GenerateID()),
		CNPJ:               cleanedCNPJ,
		CNPJRoot:           root,
		Name:               input.Name,
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        input.Environment,
		UF:                 sigla,
		SyncStartPolicy:    input.SyncStartPolicy,
		SyncStartDate:      input.SyncStartDate,
	}

	if err := m.store.CreateCompany(ctx, company); err != nil {
		return fmt.Errorf("salvar empresa: %w", err)
	}

	return nil
}

// ListCompanies returns all registered companies.
func (m *Manager) ListCompanies(ctx context.Context) ([]Company, error) {
	companies, err := m.store.ListCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	for i := range companies {
		snapshot, snapErr := m.syncs.LatestSyncSnapshot(ctx, companies[i].ID, syncstate.SyncSourceNFSe, companies[i].Environment, companies[i].CNPJ)
		if snapErr != nil {
			return nil, fmt.Errorf("carregar snapshot da empresa %s: %w", companies[i].Name, snapErr)
		}
		if snapshot.State != nil {
			companies[i].LastFoundNSU = snapshot.State.LastFoundNSU
			companies[i].LastSyncAt = snapshot.State.LastSuccessAt
		}
		if snapshot.Run != nil {
			companies[i].LastRunStatus = snapshot.Run.Status
			companies[i].LastRunStopReason = snapshot.Run.StopReason
			if snapshot.Run.FinishedAt != nil {
				companies[i].LastSyncAt = snapshot.Run.FinishedAt
			}
		}
	}
	return companies, nil
}

// AssignCredentialToCompany changes the active credential for an existing company.
func (m *Manager) AssignCredentialToCompany(ctx context.Context, input AssignCredentialInput) error {
	company, err := lookupCompanyByCNPJ(ctx, m.store, input.CompanyCNPJ)
	if err != nil {
		return err
	}

	credential, err := lookupCredentialByID(ctx, m.credentials, dfe.CredentialID(input.CredentialID))
	if err != nil {
		return err
	}

	if company.CNPJRoot != "" && credential.OwnerCNPJRoot != "" && company.CNPJRoot != credential.OwnerCNPJRoot {
		return ErrCredentialMismatch
	}

	if err := m.store.AssignCredential(ctx, company.ID, credential.ID); err != nil {
		return fmt.Errorf("atribuir credencial: %w", err)
	}
	return nil
}

// resolveCredentialForCompany resolves the credential that should be
// associated with a new company, either by ID or by creating a fresh
// one from the cert path.
func (m *Manager) resolveCredentialForCompany(ctx context.Context, input AddCompanyInput) (*credential.Credential, error) {
	if input.CredentialID != "" {
		return lookupCredentialByID(ctx, m.credentials, dfe.CredentialID(input.CredentialID))
	}

	if err := credential.ValidateCertificatePath(input.CertPath); err != nil {
		return nil, err
	}

	credential := &credential.Credential{
		ID:       dfe.CredentialID(dfe.GenerateID()),
		Label:    input.CredentialLabel,
		CertPath: input.CertPath,
	}
	if credential.Label == "" {
		if input.Name != "" {
			credential.Label = input.Name
		} else {
			credential.Label = input.CertPath
		}
	}

	if err := m.credentials.CreateCredential(ctx, credential); err != nil {
		return nil, fmt.Errorf("salvar credencial: %w", err)
	}
	return credential, nil
}

// CompanyByCNPJ returns one registered company.
func (m *Manager) CompanyByCNPJ(ctx context.Context, rawCNPJ string) (*Company, error) {
	return lookupCompanyByCNPJ(ctx, m.store, rawCNPJ)
}

// UpdateCompany replaces the editable fields of an existing company: name,
// UF, the environment and, before the first NFS-e sync, the initial sync
// policy. The environment stays editable because sync state is kept per
// environment and every NF-e records its tpAmb.
func (m *Manager) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	company, err := lookupCompanyByCNPJ(ctx, m.store, input.CNPJ)
	if err != nil {
		return err
	}
	sigla, err := normalizeUF(input.UF)
	if err != nil {
		return err
	}

	// The start policy only applies to NFS-e; an NF-e cursor does not lock it.
	if input.SyncStartPolicy != company.SyncStartPolicy || !sameDate(input.SyncStartDate, company.SyncStartDate) {
		hasState, err := m.syncs.HasSyncState(ctx, syncstate.HasSyncStateParams{CompanyID: company.ID, Source: syncstate.SyncSourceNFSe})
		if err != nil {
			return fmt.Errorf("verificar estado de sincronização: %w", err)
		}
		if hasState {
			return fmt.Errorf("não é possível alterar a política inicial depois que a sincronização já começou")
		}
	}

	company.Name = input.Name
	company.Environment = input.Environment
	company.UF = sigla
	company.SyncStartPolicy = input.SyncStartPolicy
	company.SyncStartDate = input.SyncStartDate

	if err := m.store.UpdateCompany(ctx, company); err != nil {
		return fmt.Errorf("atualizar empresa: %w", err)
	}

	return nil
}

func ParseSyncStartPolicyInput(rawPolicy, rawDate string) (syncstate.SyncStartPolicy, *time.Time, error) {
	if rawPolicy == "" {
		rawPolicy = string(syncstate.SyncStartPolicyFromNow)
	}
	policy, err := syncstate.ParseSyncStartPolicy(rawPolicy)
	if err != nil {
		return "", nil, err
	}

	switch policy {
	case syncstate.SyncStartPolicyAll:
		if rawDate != "" {
			return "", nil, fmt.Errorf("sync_start_date deve ficar vazio para política all")
		}
		return policy, nil, nil
	case syncstate.SyncStartPolicySinceDate:
		if rawDate == "" {
			return "", nil, fmt.Errorf("sync_start_date é obrigatório para política since_date")
		}
		parsed, err := time.Parse("2006-01-02", rawDate)
		if err != nil {
			return "", nil, fmt.Errorf("sync_start_date inválido: use YYYY-MM-DD")
		}
		return policy, &parsed, nil
	case syncstate.SyncStartPolicyFromNow:
		if rawDate == "" {
			now := time.Now()
			parsed, err := time.Parse("2006-01-02", now.Format("2006-01-02"))
			if err != nil {
				return "", nil, err
			}
			return policy, &parsed, nil
		}
		parsed, err := time.Parse("2006-01-02", rawDate)
		if err != nil {
			return "", nil, fmt.Errorf("sync_start_date inválido: use YYYY-MM-DD")
		}
		return policy, &parsed, nil
	default:
		return "", nil, fmt.Errorf("invalid sync start policy %q: %w", policy, dfe.ErrInvalidEnum)
	}
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

// normalizeUF trims and upper-cases a state sigla. Empty is allowed and
// means the UF is unknown.
func normalizeUF(raw string) (string, error) {
	sigla := strings.ToUpper(strings.TrimSpace(raw))
	if sigla != "" && !uf.Valid(sigla) {
		return "", fmt.Errorf("UF inválida %q: use uma sigla como SP, RJ ou MG", raw)
	}
	return sigla, nil
}

func lookupCompanyByCNPJ(ctx context.Context, repo storeInterface, raw string) (*Company, error) {
	cleanedCNPJ, err := cnpj.Normalize(raw)
	if err != nil {
		return nil, err
	}

	company, err := repo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		if errors.Is(err, ErrCompanyNotFound) {
			return nil, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
		}
		return nil, fmt.Errorf("buscar empresa: %w", err)
	}
	return company, nil
}

func lookupCredentialByID(ctx context.Context, repo credentialProvider, id dfe.CredentialID) (*credential.Credential, error) {
	cred, err := repo.CredentialByID(ctx, id)
	if err != nil {
		if errors.Is(err, credential.ErrCredentialNotFound) {
			return nil, fmt.Errorf("credencial não encontrada")
		}
		return nil, fmt.Errorf("buscar credencial: %w", err)
	}
	return cred, nil
}
