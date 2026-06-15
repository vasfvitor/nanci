package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// AddCompanyInput carries the data required to register a new company.
type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     string // "producao" | "producao_restrita"
}

// AddCompany registers a new company in the store.
func (a *App) AddCompany(ctx context.Context, input AddCompanyInput) error {
	cleanedCNPJ, err := normalizeCNPJ(input.CNPJ)
	if err != nil {
		return err
	}
	root, _ := cnpj.Root(cleanedCNPJ)

	credential, err := a.resolveCredentialForCompany(ctx, input)
	if err != nil {
		return err
	}

	environment, err := parseEnvironment(input.Environment)
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
		Environment:        environment,
	}

	if err := a.CompanyRepo.CreateCompany(ctx, company); err != nil {
		return fmt.Errorf("salvar empresa: %w", err)
	}

	return nil
}

// ListCompanies returns all registered companies.
func (a *App) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	companies, err := a.CompanyRepo.ListCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	for i := range companies {
		snapshot, snapErr := a.SyncRepo.LatestSyncSnapshot(ctx, companies[i].ID, companies[i].Environment, companies[i].CNPJ)
		if snapErr != nil {
			return nil, fmt.Errorf("carregar snapshot da empresa %s: %w", companies[i].Name, snapErr)
		}
		if snapshot.State != nil {
			companies[i].LastNSU = snapshot.State.LastCheckedNSU
			companies[i].LastFoundNSU = snapshot.State.LastFoundNSU
			companies[i].LastFoundNSUValid = snapshot.State.LastFoundNSUValid
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
func (a *App) AssignCredentialToCompany(ctx context.Context, input AssignCredentialInput) error {
	company, err := a.companyByCNPJ(ctx, input.CompanyCNPJ)
	if err != nil {
		return err
	}

	credential, err := a.credentialByID(ctx, nfse.CredentialID(input.CredentialID))
	if err != nil {
		return err
	}

	if company.CNPJRoot != "" && credential.OwnerCNPJRoot != "" && company.CNPJRoot != credential.OwnerCNPJRoot {
		return fmt.Errorf("a credencial informada não pertence à mesma raiz do CNPJ da empresa")
	}

	if err := a.CompanyRepo.AssignCredential(ctx, company.ID, credential.ID); err != nil {
		return fmt.Errorf("atribuir credencial: %w", err)
	}
	return nil
}

func (a *App) resolveCredentialForCompany(ctx context.Context, input AddCompanyInput) (*nfse.Credential, error) {
	if input.CredentialID != "" {
		return a.credentialByID(ctx, nfse.CredentialID(input.CredentialID))
	}

	if err := validateCertificatePath(input.CertPath); err != nil {
		return nil, err
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

	if err := a.CredentialRepo.CreateCredential(ctx, credential); err != nil {
		return nil, fmt.Errorf("salvar credencial: %w", err)
	}
	return credential, nil
}

// UpdateCompanyInput carries data to update a company
type UpdateCompanyInput struct {
	CNPJ        string
	Name        string
	Environment string // "producao" | "producao_restrita"
}

// UpdateCompany updates the name and environment of an existing company.
func (a *App) UpdateCompany(ctx context.Context, input UpdateCompanyInput) error {
	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return err
	}

	environment, err := parseEnvironment(input.Environment)
	if err != nil {
		return err
	}

	if err := a.CompanyRepo.UpdateCompany(ctx, company.ID, input.Name, environment); err != nil {
		return fmt.Errorf("atualizar empresa: %w", err)
	}

	return nil
}
