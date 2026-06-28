package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
)

type StatusResult struct {
	CompanyName        string `json:"CompanyName"`
	CNPJ               string `json:"CNPJ"`
	Environment        string `json:"Environment"`
	ConsultationCNPJ   string `json:"ConsultationCNPJ"`
	CredentialCNPJ     string `json:"CredentialCNPJ"`
	CredentialNotAfter *time.Time `json:"CredentialNotAfter"`
	LastProcessedNSU   int64 `json:"LastProcessedNSU"`
	LastFoundNSU       *int64 `json:"LastFoundNSU"`
	LastSyncAt         *time.Time `json:"LastSyncAt"`
	LastRunStatus      string `json:"LastRunStatus"`
	LastRunStopReason  string `json:"LastRunStopReason"`
	TotalEmitidas      int64 `json:"TotalEmitidas"`
	TotalTomadas       int64 `json:"TotalTomadas"`
}

type CompanySummary struct {
	ID                 string `json:"ID"`
	CNPJ               string `json:"CNPJ"`
	CNPJRoot           string `json:"CNPJRoot"`
	Name               string `json:"Name"`
	CredentialID       string `json:"CredentialID"`
	CredentialLabel    string `json:"CredentialLabel"`
	CredentialCertPath string `json:"CredentialCertPath"`
	Environment        string `json:"Environment"`
	LastFoundNSU       *int64 `json:"LastFoundNSU"`
	LastSyncAt         *time.Time `json:"LastSyncAt"`
	SyncStartPolicy    string `json:"SyncStartPolicy"`
	SyncStartDate      *time.Time `json:"SyncStartDate"`
	InitialSyncDoneAt  *time.Time `json:"InitialSyncDoneAt"`
	LastRunStatus      string `json:"LastRunStatus"`
	LastRunStopReason  string `json:"LastRunStopReason"`
	CreatedAt          time.Time `json:"CreatedAt"`
	UpdatedAt          time.Time `json:"UpdatedAt"`
}

type CredentialSummary struct {
	ID                string `json:"ID"`
	Label             string `json:"Label"`
	CertPath          string `json:"CertPath"`
	OwnerCNPJ         string `json:"OwnerCNPJ"`
	OwnerCNPJRoot     string `json:"OwnerCNPJRoot"`
	FingerprintSHA256 string `json:"FingerprintSHA256"`
	SubjectName       string `json:"SubjectName"`
	NotBefore         *time.Time `json:"NotBefore"`
	NotAfter          *time.Time `json:"NotAfter"`
	InspectedAt       *time.Time `json:"InspectedAt"`
	CreatedAt         time.Time `json:"CreatedAt"`
	UpdatedAt         time.Time `json:"UpdatedAt"`
}

type DocumentRow struct {
	ID                 string `json:"ID"`
	ChaveAcesso        string `json:"ChaveAcesso"`
	IssueDate          time.Time `json:"IssueDate"`
	Competence         string `json:"Competence"`
	PrestadorCNPJ      string `json:"PrestadorCNPJ"`
	PrestadorName      string `json:"PrestadorName"`
	TomadorCNPJ        string `json:"TomadorCNPJ"`
	TomadorName        string `json:"TomadorName"`
	IntermediarioCNPJ  string `json:"IntermediarioCNPJ"`
	IntermediarioName  string `json:"IntermediarioName"`
	ServiceValue       int64 `json:"ServiceValue"`
	ISSValue           int64 `json:"ISSValue"`
	IRRFValue          int64 `json:"IRRFValue"`
	INSSValue          int64 `json:"INSSValue"`
	PISValue           int64 `json:"PISValue"`
	COFINSValue        int64 `json:"COFINSValue"`
	CSLLValue          int64 `json:"CSLLValue"`
	TotalRetentions    int64 `json:"TotalRetentions"`
	Status             string `json:"Status"`
	LayoutVersion      string `json:"LayoutVersion"`
	XMLPath            string `json:"XMLPath"`
	RawHash            string `json:"RawHash"`
	ParseWarnings      []string `json:"ParseWarnings"`
	NFSeNumber         string `json:"NFSeNumber"`
	ServiceDescription string `json:"ServiceDescription"`
	CreatedAt          time.Time `json:"CreatedAt"`
	UpdatedAt          time.Time `json:"UpdatedAt"`
	RelationID         string `json:"RelationID"`
	CompanyID          string `json:"CompanyID"`
	DocumentID         string `json:"DocumentID"`
	CompanyRole        string `json:"CompanyRole"`
	VisibilityReason   string `json:"VisibilityReason"`
	FirstSeenNSU       *int64 `json:"FirstSeenNSU"`
	LastSeenNSU        *int64 `json:"LastSeenNSU"`
	FirstSyncedAt      time.Time `json:"FirstSyncedAt"`
	LastSyncedAt       time.Time `json:"LastSyncedAt"`
	ViewedAt           *time.Time `json:"ViewedAt"`
}

type DocumentEvent struct {
	ID                     string `json:"ID"`
	Type                   string `json:"Type"`
	EventAt                *time.Time `json:"EventAt"`
	ReplacementChaveAcesso string `json:"ReplacementChaveAcesso"`
	Description            string `json:"Description"`
	RawXMLPath             string `json:"RawXMLPath"`
}

type AddCompanyInput struct {
	CNPJ            string `json:"CNPJ"`
	Name            string `json:"Name"`
	CredentialID    string `json:"CredentialID"`
	CredentialLabel string `json:"CredentialLabel"`
	CertPath        string `json:"CertPath"`
	Environment     string `json:"Environment"` // "producao" | "producao_restrita"
	SyncStartPolicy string `json:"SyncStartPolicy"` // "all" | "since_date" | "from_now"
	SyncStartDate   string `json:"SyncStartDate"` // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type UpdateCompanyInput struct {
	CNPJ            string `json:"CNPJ"`
	Name            string `json:"Name"`
	Environment     string `json:"Environment"` // "producao" | "producao_restrita"
	SyncStartPolicy string `json:"SyncStartPolicy"` // "all" | "since_date" | "from_now"
	SyncStartDate   string `json:"SyncStartDate"` // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type AddCredentialInput struct {
	Label    string `json:"Label"`
	CertPath string `json:"CertPath"`
}

type UpdateCredentialPathInput struct {
	CredentialID string `json:"CredentialID"`
	CertPath     string `json:"CertPath"`
}

type AssignCredentialInput struct {
	CompanyCNPJ  string `json:"CompanyCNPJ"`
	CredentialID string `json:"CredentialID"`
}

type UpdateCredentialDataInput struct {
	CredentialID string `json:"CredentialID"`
	Label        string `json:"Label"`
}

type ListInput struct {
	CNPJ       string `json:"CNPJ"`
	Competence string `json:"Competence"`
	Direction  string `json:"Direction"`
	OnlyUnread bool `json:"OnlyUnread"`
}

type PullInput struct {
	CNPJ string `json:"CNPJ"`
	Mode string `json:"Mode"`
}

type PullResult struct {
	CompanyName              string `json:"CompanyName"`
	CNPJ                     string `json:"CNPJ"`
	CredentialLabel          string `json:"CredentialLabel"`
	CredentialCNPJ           string `json:"CredentialCNPJ"`
	ConsultationBasis        string `json:"ConsultationBasis"`
	Status                   string `json:"Status"`
	StopReason               string `json:"StopReason"`
	LastProcessedNSU         int64 `json:"LastProcessedNSU"`
	LastFoundNSU             *int64 `json:"LastFoundNSU"`
	EmptyStreak              int `json:"EmptyStreak"`
	DocumentsFound           int `json:"DocumentsFound"`
	EventsFound              int `json:"EventsFound"`
	DocumentsSaved           int `json:"DocumentsSaved"`
	EventsSaved              int `json:"EventsSaved"`
	DocumentsSkippedByPolicy int `json:"DocumentsSkippedByPolicy"`
	EventsSkippedByPolicy    int `json:"EventsSkippedByPolicy"`
	Errors                   int `json:"Errors"`
	Duration                 time.Duration `json:"Duration"`
}

type QueryNFSeInput struct {
	CompanyCNPJ string `json:"CompanyCNPJ"`
	ChaveAcesso string `json:"ChaveAcesso"`
}

type ResetSyncInput struct {
	CompanyCNPJ string `json:"CompanyCNPJ"`
}

type ExportDocumentsInput struct {
	CNPJ        string `json:"CNPJ"`
	Competence  string `json:"Competence"`
	Direction   string `json:"Direction"`
	Format      string `json:"Format"`
	OutPath     string `json:"OutPath"`
	Incremental bool `json:"Incremental"`
}

type ExportDANFSeInput struct {
	CNPJ        string `json:"CNPJ"`
	ChaveAcesso string `json:"ChaveAcesso"`
	OutPath     string `json:"OutPath"`
}

type ExportXMLInput struct {
	CNPJ        string `json:"CNPJ"`
	ChaveAcesso string `json:"ChaveAcesso"`
	OutPath     string `json:"OutPath"`
}

type ExportResult struct {
	OutPath       string `json:"OutPath"`
	Format        string `json:"Format"`
	Incremental   bool `json:"Incremental"`
	ExportedCount int `json:"ExportedCount"`
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
			FirstSyncedAt:      document.FirstSyncedAt,
			LastSyncedAt:       document.LastSyncedAt,
			ViewedAt:           document.ViewedAt,
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

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type ConnectionTestResult struct {
	CertLoaded        bool   `json:"certLoaded"`
	CertSubject       string `json:"certSubject"`
	CertExpiration    string `json:"certExpiration"`
	MTLSAccepted      bool   `json:"mtlsAccepted"`
	EndpointReached   bool   `json:"endpointReached"`
	ResponseCode      string `json:"responseCode"`
	ResponseDetail    string `json:"responseDetail"`
	StatusExplanation string `json:"statusExplanation"`
}
