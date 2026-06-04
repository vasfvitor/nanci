package nfse

import (
	"context"
	"time"
)

// Company represents a company that syncs documents.
type Company struct {
	ID          string
	CNPJ        string    // supports numeric (14 digits) and alphanumeric
	CNPJRoot    string    // first 8 chars - groups branches
	Name        string
	CertPath    string
	Environment string    // "producao_restrita" | "producao"
	LastNSU     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Document represents a synced fiscal document (NFS-e).
type Document struct {
	ID             string
	CompanyID      string
	ChaveAcesso    string
	NSU            int64
	Direction      string // "tomada" | "prestada" | "intermediario"
	IssueDate      time.Time
	Competence     string // "YYYY-MM"
	PrestadorCNPJ  string
	PrestadorName  string
	TomadorCNPJ    string
	TomadorName    string
	ServiceValue   float64
	ISSValue       float64
	IRRFValue      float64
	INSSValue      float64
	PISValue       float64
	COFINSValue    float64
	CSLLValue      float64
	Status         string // "normal" | "cancelada" | "substituida"
	XMLPath        string
	RawHash        string
	ParseError     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CompanyStats provides aggregated information about a company's synchronization state.
type CompanyStats struct {
	TotalDocuments int
	TotalService   float64
	LastSync       *time.Time
}

// SyncRun represents a synchronization execution for audit and control.
type SyncRun struct {
	ID             string
	CompanyID      string
	StartedAt      time.Time
	FinishedAt     *time.Time
	FromNSU        int64
	ToNSU          int64
	DocumentsFound int
	ErrorsCount    int
	Status         string // "running" | "completed" | "failed" | "interrupted"
}

// ProgressEvent contains information about the progress of a long-running operation.
type ProgressEvent struct {
	CurrentNSU  int64
	MaxNSU      int64
	DocsFound   int
	DocsInBatch int
	Errors      int
	Message     string
}

// ProgressFunc is a callback function to report progress.
type ProgressFunc func(event ProgressEvent)

// CredentialProvider defines how to obtain certificate passwords.
// Implemented differently in CLI and future UI.
type CredentialProvider interface {
	GetCertPassword(ctx context.Context, certPath string) ([]byte, error)
}
