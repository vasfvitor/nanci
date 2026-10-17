// Company and credential DTOs: the summaries the desktop lists, the inputs
// that add and edit companies and certificates, and their mappers.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
)

type CompanySummary struct {
	ID                 string
	CNPJ               string
	CNPJRoot           string
	Name               string
	CredentialID       string
	CredentialLabel    string
	CredentialCertPath string
	Environment        string
	UF                 string
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	SyncStartPolicy    string
	SyncStartDate      *time.Time
	InitialSyncDoneAt  *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CredentialSummary struct {
	ID                string
	Label             string
	CertPath          string
	OwnerCNPJ         string
	OwnerCNPJRoot     string
	FingerprintSHA256 string
	SubjectName       string
	NotBefore         *time.Time
	NotAfter          *time.Time
	InspectedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     string // "producao" | "producao_restrita"
	UF              string // state sigla such as "SP"; empty when unknown
	SyncStartPolicy string // "all" | "since_date" | "from_now"
	SyncStartDate   string // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type UpdateCompanyInput struct {
	CNPJ            string
	Name            string
	Environment     string // "producao" | "producao_restrita"
	UF              string // state sigla such as "SP"; empty when unknown
	SyncStartPolicy string // "all" | "since_date" | "from_now"
	SyncStartDate   string // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type AddCredentialInput struct {
	Label    string
	CertPath string
}

type UpdateCredentialPathInput struct {
	CredentialID string
	CertPath     string
}

type AssignCredentialInput struct {
	CompanyCNPJ  string
	CredentialID string
}

type UpdateCredentialDataInput struct {
	CredentialID string
	Label        string
}

func CompanySummaries(companies []company.Company) []CompanySummary {
	out := make([]CompanySummary, len(companies))
	for i, company := range companies {
		out[i] = CompanySummary{
			ID:                 string(company.ID),
			CNPJ:               company.CNPJ,
			CNPJRoot:           company.CNPJRoot,
			Name:               company.Name,
			CredentialID:       string(company.CredentialID),
			CredentialLabel:    company.CredentialLabel,
			CredentialCertPath: company.CredentialCertPath,
			Environment:        string(company.Environment),
			UF:                 company.UF,
			LastFoundNSU:       company.LastFoundNSU,
			LastSyncAt:         company.LastSyncAt,
			SyncStartPolicy:    string(company.SyncStartPolicy),
			SyncStartDate:      company.SyncStartDate,
			InitialSyncDoneAt:  company.InitialSyncDoneAt,
			LastRunStatus:      string(company.LastRunStatus),
			LastRunStopReason:  string(company.LastRunStopReason),
			CreatedAt:          company.CreatedAt,
			UpdatedAt:          company.UpdatedAt,
		}
	}
	return out
}

func CredentialSummaries(credentials []credential.Credential) []CredentialSummary {
	out := make([]CredentialSummary, len(credentials))
	for i, credential := range credentials {
		out[i] = CredentialSummary{
			ID:                string(credential.ID),
			Label:             credential.Label,
			CertPath:          credential.CertPath,
			OwnerCNPJ:         credential.OwnerCNPJ,
			OwnerCNPJRoot:     credential.OwnerCNPJRoot,
			FingerprintSHA256: credential.FingerprintSHA256,
			SubjectName:       credential.SubjectName,
			NotBefore:         credential.NotBefore,
			NotAfter:          credential.NotAfter,
			InspectedAt:       credential.InspectedAt,
			CreatedAt:         credential.CreatedAt,
			UpdatedAt:         credential.UpdatedAt,
		}
	}
	return out
}
