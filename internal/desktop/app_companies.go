// Company and credential methods of App: register and edit companies and
// certificates, assign a credential to a company and test the connection to
// the ADN and to the CT-e service.

package main

import (
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func (a *App) AddCompany(input desktopapi.AddCompanyInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	environment, err := dfe.ParseEnvironment(input.Environment)
	if err != nil {
		return err
	}
	policy, date, err := company.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
	if err != nil {
		return err
	}

	return a.svc.companies.AddCompany(a.ctx, company.AddCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		CredentialID:    input.CredentialID,
		CredentialLabel: input.CredentialLabel,
		CertPath:        input.CertPath,
		Environment:     environment,
		UF:              input.UF,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AddCredential(input desktopapi.AddCredentialInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.credentials.AddCredential(a.ctx, credential.AddCredentialInput{
		Label:    input.Label,
		CertPath: input.CertPath,
	})
}

func (a *App) ListCredentials() ([]desktopapi.CredentialSummary, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	credentials, err := a.svc.credentials.ListCredentials(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CredentialSummaries(credentials), nil
}

func (a *App) UpdateCredentialPath(input desktopapi.UpdateCredentialPathInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.credentials.UpdateCredentialPath(a.ctx, credential.UpdateCredentialPathInput{
		CredentialID: input.CredentialID,
		CertPath:     input.CertPath,
	})
}

func (a *App) UpdateCredentialData(input desktopapi.UpdateCredentialDataInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.credentials.UpdateCredentialData(a.ctx, credential.UpdateCredentialDataInput{
		CredentialID: input.CredentialID,
		Label:        input.Label,
	})
}

func (a *App) UpdateCompany(input desktopapi.UpdateCompanyInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	environment, err := dfe.ParseEnvironment(input.Environment)
	if err != nil {
		return err
	}
	// An empty policy keeps the stored one, as `nanci company update` does.
	var policy syncstate.SyncStartPolicy
	var date *time.Time
	if input.SyncStartPolicy == "" {
		current, err := a.svc.companies.CompanyByCNPJ(a.ctx, input.CNPJ)
		if err != nil {
			return err
		}
		policy, date = current.SyncStartPolicy, current.SyncStartDate
	} else {
		policy, date, err = company.ParseSyncStartPolicyInput(input.SyncStartPolicy, input.SyncStartDate)
		if err != nil {
			return err
		}
	}

	return a.svc.companies.UpdateCompany(a.ctx, company.UpdateCompanyInput{
		CNPJ:            input.CNPJ,
		Name:            input.Name,
		Environment:     environment,
		UF:              input.UF,
		SyncStartPolicy: policy,
		SyncStartDate:   date,
	})
}

func (a *App) AssignCredentialToCompany(input desktopapi.AssignCredentialInput) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.svc.companies.AssignCredentialToCompany(a.ctx, company.AssignCredentialInput{
		CompanyCNPJ:  input.CompanyCNPJ,
		CredentialID: input.CredentialID,
	})
}

func (a *App) ListCompanies() ([]desktopapi.CompanySummary, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	companies, err := a.svc.companies.ListCompanies(a.ctx)
	if err != nil {
		return nil, err
	}
	return desktopapi.CompanySummaries(companies), nil
}

func (a *App) TestConnection(companyCNPJ string) (desktopapi.ConnectionTestResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ConnectionTestResult{}, err
	}
	res, err := a.svc.query.TestConnection(a.ctx, companyCNPJ)
	if err != nil {
		return desktopapi.ConnectionTestResult{}, err
	}
	return desktopapi.ConnectionTestResult{
		CertLoaded:        res.CertLoaded,
		CertSubject:       res.CertSubject,
		CertExpiration:    res.CertExpiration,
		MTLSAccepted:      res.MTLSAccepted,
		EndpointReached:   res.EndpointReached,
		ResponseCode:      res.ResponseCode,
		ResponseDetail:    res.ResponseDetail,
		StatusExplanation: res.StatusExplanation,
	}, nil
}

// TestCTeConnection opens a TLS connection to the CT-e distribution host. It
// sends no query, so it does not use the hourly budget.
func (a *App) TestCTeConnection(cnpj string) (desktopapi.ConnectionTestResult, error) {
	if err := a.ready(); err != nil {
		return desktopapi.ConnectionTestResult{}, err
	}
	res, err := a.svc.cte.TestConnection(a.ctx, cnpj)
	if err != nil {
		return desktopapi.ConnectionTestResult{}, err
	}
	return desktopapi.ConnectionTestResult{
		CertLoaded:        res.CertLoaded,
		CertSubject:       res.CertSubject,
		CertExpiration:    res.CertExpiration,
		MTLSAccepted:      res.MTLSAccepted,
		EndpointReached:   res.EndpointReached,
		ResponseCode:      res.ResponseCode,
		ResponseDetail:    res.ResponseDetail,
		StatusExplanation: res.StatusExplanation,
	}, nil
}
