package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

var (
	ErrCredentialMismatch   = errors.New("a credencial informada não pertence à mesma raiz do CNPJ da empresa")
	ErrCredentialNoOwner    = errors.New("o certificado não expõe um CNPJ proprietário utilizável para consulta")
	ErrCompanyNoEnvironment = errors.New("a empresa não possui ambiente configurado")
)

// AddCompanyInput carries the data required to register a new company.
type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     nfse.Environment
	SyncStartPolicy nfse.SyncStartPolicy
	SyncStartDate   *time.Time
}

// CompanyService owns the company use cases.
type CompanyService struct {
	CompanyRepo    *store.CompanyRepository
	CredentialRepo *store.CredentialRepository
	SyncRepo       *store.SyncRepository
}

func NewCompanyService(d Dependencies) *CompanyService {
	return &CompanyService{

		CompanyRepo:    d.CompanyRepo,
		CredentialRepo: d.CredentialRepo,
		SyncRepo:       d.SyncRepo,
	}
}

// AddCompany registers a new company in the store.
func (s *CompanyService) AddCompany(ctx context.Context, input AddCompanyInput) error {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return err
	}
	root, _ := cnpj.Root(cleanedCNPJ)

	credential, err := s.resolveCredentialForCompany(ctx, input)
	if err != nil {
		return err
	}

	company := &nfse.Company{
		ID:                 nfse.CompanyID(nfse.GenerateID()),
		CNPJ:               cleanedCNPJ,
		CNPJRoot:           root,
		Name:               input.Name,
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        input.Environment,
		SyncStartPolicy:    input.SyncStartPolicy,
		SyncStartDate:      input.SyncStartDate,
	}

	if err := s.CompanyRepo.CreateCompany(ctx, company); err != nil {
		return fmt.Errorf("salvar empresa: %w", err)
	}

	return nil
}

// ListCompanies returns all registered companies.
func (s *CompanyService) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	companies, err := s.CompanyRepo.ListCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	for i := range companies {
		snapshot, snapErr := s.SyncRepo.LatestSyncSnapshot(ctx, companies[i].ID, companies[i].Environment, companies[i].CNPJ)
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
func (s *CompanyService) AssignCredentialToCompany(ctx context.Context, input AssignCredentialInput) error {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyRepo, input.CompanyCNPJ)
	if err != nil {
		return err
	}

	credential, err := lookupCredentialByID(ctx, s.CredentialRepo, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return err
	}

	if company.CNPJRoot != "" && credential.OwnerCNPJRoot != "" && company.CNPJRoot != credential.OwnerCNPJRoot {
		return ErrCredentialMismatch
	}

	if err := s.CompanyRepo.AssignCredential(ctx, company.ID, credential.ID); err != nil {
		return fmt.Errorf("atribuir credencial: %w", err)
	}
	return nil
}

// resolveCredentialForCompany resolves the credential that should be
// associated with a new company, either by ID or by creating a fresh
// one from the cert path.
func (s *CompanyService) resolveCredentialForCompany(ctx context.Context, input AddCompanyInput) (*nfse.Credential, error) {
	if input.CredentialID != "" {
		return lookupCredentialByID(ctx, s.CredentialRepo, nfse.CredentialID(input.CredentialID))
	}

	if err := validateCertificatePath(input.CertPath); err != nil {
		return nil, err
	}

	credential := &nfse.Credential{
		ID:       nfse.CredentialID(nfse.GenerateID()),
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

	if err := s.CredentialRepo.CreateCredential(ctx, credential); err != nil {
		return nil, fmt.Errorf("salvar credencial: %w", err)
	}
	return credential, nil
}

// UpdateCompanyInput carries data to update a company
type UpdateCompanyInput struct {
	CNPJ            string
	Name            string
	Environment     nfse.Environment
	SyncStartPolicy nfse.SyncStartPolicy
	SyncStartDate   *time.Time
}

// UpdateCompany updates the name and environment of an existing company.
func (s *CompanyService) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyRepo, input.CNPJ)
	if err != nil {
		return err
	}

	if input.SyncStartPolicy != company.SyncStartPolicy || !sameDate(input.SyncStartDate, company.SyncStartDate) {
		hasState, err := s.SyncRepo.HasSyncState(ctx, nfse.HasSyncStateParams{CompanyID: company.ID})
		if err != nil {
			return fmt.Errorf("verificar estado de sincronização: %w", err)
		}
		if hasState {
			return fmt.Errorf("não é possível alterar a política inicial depois que a sincronização já começou")
		}
	}

	company.Name = input.Name
	company.Environment = input.Environment
	company.SyncStartPolicy = input.SyncStartPolicy
	company.SyncStartDate = input.SyncStartDate

	if err := s.CompanyRepo.UpdateCompany(ctx, company); err != nil {
		return fmt.Errorf("atualizar empresa: %w", err)
	}

	return nil
}

func ParseSyncStartPolicyInput(rawPolicy, rawDate string) (nfse.SyncStartPolicy, *time.Time, error) {
	if rawPolicy == "" {
		rawPolicy = string(nfse.SyncStartPolicyFromNow)
	}
	policy, err := nfse.ParseSyncStartPolicy(rawPolicy)
	if err != nil {
		return "", nil, err
	}

	switch policy {
	case nfse.SyncStartPolicyAll:
		if rawDate != "" {
			return "", nil, fmt.Errorf("sync_start_date deve ficar vazio para política all")
		}
		return policy, nil, nil
	case nfse.SyncStartPolicySinceDate:
		if rawDate == "" {
			return "", nil, fmt.Errorf("sync_start_date é obrigatório para política since_date")
		}
		parsed, err := time.Parse("2006-01-02", rawDate)
		if err != nil {
			return "", nil, fmt.Errorf("sync_start_date inválido: use YYYY-MM-DD")
		}
		return policy, &parsed, nil
	case nfse.SyncStartPolicyFromNow:
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
		return "", nil, fmt.Errorf("invalid sync start policy %q: %w", policy, nfse.ErrInvalidEnum)
	}
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}
