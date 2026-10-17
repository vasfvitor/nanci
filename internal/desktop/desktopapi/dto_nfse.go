// NFS-e DTOs: status, document rows and events, sync, direct query, reset
// and export inputs, and their mappers.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
)

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
	FirstSeenNSU       *int64
	LastSeenNSU        *int64
	FirstSyncedAt      time.Time
	LastSyncedAt       time.Time
	ViewedAt           *time.Time
}

type DocumentEvent struct {
	ID                     string
	Type                   string
	EventAt                *time.Time
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
}

type ListInput struct {
	CNPJ       string
	Competence string
	Direction  string
	OnlyUnread bool
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

type QueryNFSeInput struct {
	CompanyCNPJ string
	ChaveAcesso string
}

type ResetSyncInput struct {
	CompanyCNPJ string
}

type ExportDocumentsInput struct {
	CNPJ         string
	Competence   string
	Direction    string
	Format       string
	OutPath      string
	Incremental  bool
	ChavesAcesso []string
}

type ExportDANFSeInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

type ExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
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
