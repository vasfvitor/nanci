package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
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
	LastNSU            int64
	LastFoundNSU       int64
	LastFoundNSUValid  bool
	LastSyncAt         *time.Time
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

type DocumentRow struct {
	ID                 string
	ChaveAcesso        string
	IssueDate          time.Time
	Competence         string
	PrestadorCNPJ      string
	PrestadorName      string
	TomadorCNPJ        string
	TomadorName        string
	IntermediarioCNPJ  string
	IntermediarioName  string
	ServiceValue       int64
	ISSValue           int64
	IRRFValue          int64
	INSSValue          int64
	PISValue           int64
	COFINSValue        int64
	CSLLValue          int64
	TotalRetentions    int64
	Status             string
	LayoutVersion      string
	XMLPath            string
	RawHash            string
	ParseWarnings      []string
	NFSeNumber         string
	ServiceDescription string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	RelationID         string
	CompanyID          string
	DocumentID         string
	CompanyRole        string
	VisibilityReason   string
	FirstSeenNSU       int64
	LastSeenNSU        int64
	FirstSeenNSUValid  bool
	LastSeenNSUValid   bool
	FirstSyncedAt      time.Time
	LastSyncedAt       time.Time
}

type DocumentEvent struct {
	ID                     string
	Type                   string
	EventAt                string
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
}

type ExportDocumentsInput struct {
	CNPJ       string
	Competence string
	Direction  string
	Format     string
	OutDir     string
	BaseName   string
}

type ExportDANFSeInput struct {
	CNPJ        string
	ChaveAcesso string
	OutDir      string
	BaseName    string
}

type ExportResult struct {
	OutPath string
	Format  string
}

func CompanySummaries(companies []nfse.Company) []CompanySummary {
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
			LastNSU:            company.LastNSU,
			LastFoundNSU:       company.LastFoundNSU,
			LastFoundNSUValid:  company.LastFoundNSUValid,
			LastSyncAt:         company.LastSyncAt,
			LastRunStatus:      string(company.LastRunStatus),
			LastRunStopReason:  string(company.LastRunStopReason),
			CreatedAt:          company.CreatedAt,
			UpdatedAt:          company.UpdatedAt,
		}
	}
	return out
}

func CredentialSummaries(credentials []nfse.Credential) []CredentialSummary {
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

func DocumentRows(documents []nfse.CompanyDocument) []DocumentRow {
	out := make([]DocumentRow, len(documents))
	for i, document := range documents {
		out[i] = DocumentRow{
			ID:                 string(document.ID),
			ChaveAcesso:        string(document.ChaveAcesso),
			IssueDate:          document.IssueDate,
			Competence:         document.Competence,
			PrestadorCNPJ:      document.PrestadorCNPJ,
			PrestadorName:      document.PrestadorName,
			TomadorCNPJ:        document.TomadorCNPJ,
			TomadorName:        document.TomadorName,
			IntermediarioCNPJ:  document.IntermediarioCNPJ,
			IntermediarioName:  document.IntermediarioName,
			ServiceValue:       document.ServiceValue.Cents(),
			ISSValue:           document.ISSValue.Cents(),
			IRRFValue:          document.IRRFValue.Cents(),
			INSSValue:          document.INSSValue.Cents(),
			PISValue:           document.PISValue.Cents(),
			COFINSValue:        document.COFINSValue.Cents(),
			CSLLValue:          document.CSLLValue.Cents(),
			TotalRetentions:    document.TotalRetentions.Cents(),
			Status:             string(document.Status),
			LayoutVersion:      document.LayoutVersion,
			XMLPath:            document.XMLPath,
			RawHash:            document.RawHash,
			ParseWarnings:      document.ParseWarnings,
			NFSeNumber:         document.NFSeNumber,
			ServiceDescription: document.ServiceDescription,
			CreatedAt:          document.CreatedAt,
			UpdatedAt:          document.UpdatedAt,
			RelationID:         document.RelationID,
			CompanyID:          string(document.CompanyID),
			DocumentID:         string(document.DocumentID),
			CompanyRole:        string(document.CompanyRole),
			VisibilityReason:   string(document.VisibilityReason),
			FirstSeenNSU:       document.FirstSeenNSU,
			LastSeenNSU:        document.LastSeenNSU,
			FirstSeenNSUValid:  document.FirstSeenNSUValid,
			LastSeenNSUValid:   document.LastSeenNSUValid,
			FirstSyncedAt:      document.FirstSyncedAt,
			LastSyncedAt:       document.LastSyncedAt,
		}
	}
	return out
}

func DocumentEvents(events []app.EventView) []DocumentEvent {
	out := make([]DocumentEvent, len(events))
	for i, event := range events {
		out[i] = DocumentEvent(event)
	}
	return out
}
