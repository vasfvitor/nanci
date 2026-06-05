package nfse

import (
	"time"
)

// Company represents a company that syncs documents.
type Company struct {
	ID                 string
	CNPJ               string // supports numeric (14 digits) and alphanumeric
	CNPJRoot           string // first 8 chars - groups branches
	Name               string
	CredentialID       string
	CredentialLabel    string
	CredentialCertPath string
	Environment        string // derived from the assigned credential
	LastNSU            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Credential represents a reusable mTLS credential that can be assigned to multiple companies.
type Credential struct {
	ID                string
	Label             string
	CertPath          string
	Environment       string
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

// Document represents a synced fiscal document (NFS-e).
type Document struct {
	ID                string
	ChaveAcesso       string
	IssueDate         time.Time
	Competence        string // "YYYY-MM"
	PrestadorCNPJ     string
	PrestadorName     string
	TomadorCNPJ       string
	TomadorName       string
	IntermediarioCNPJ string
	IntermediarioName string
	ServiceValue      float64
	ISSValue          float64
	IRRFValue         float64
	INSSValue         float64
	PISValue          float64
	COFINSValue       float64
	CSLLValue         float64
	Status            string // "normal" | "cancelada" | "substituida"
	XMLPath           string
	RawHash           string
	ParseError        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CompanyDocument represents the participation of one managed company in a canonical document.
type CompanyDocument struct {
	Document
	RelationID        string
	CompanyID         string
	DocumentID        string
	CompanyRole       string // "tomada" | "prestada" | "intermediario" | "none"
	VisibilityReason  string // "exact_prestador" | "exact_tomador" | "exact_intermediario" | "same_root_only" | "unknown"
	FirstSeenNSU      int64
	LastSeenNSU       int64
	FirstSeenNSUValid bool
	LastSeenNSUValid  bool
	FirstSyncedAt     time.Time
	LastSyncedAt      time.Time
}

// CompanyParticipation contains company-scoped role and visibility classification for one document.
type CompanyParticipation struct {
	CompanyRole      string
	VisibilityReason string
}

// CompanyStats provides aggregated information about a company's synchronization state.
type CompanyStats struct {
	TotalDocuments int
	TotalService   float64
	LastSync       *time.Time
}

// SyncRun represents a synchronization execution for audit and control.
type SyncRun struct {
	ID                string
	CompanyID         string
	CredentialID      string
	CredentialCNPJ    string
	ConsultationCNPJ  string
	ConsultationBasis string // "exact_certificate_cnpj" | "same_root_certificate"
	StartedAt         time.Time
	FinishedAt        *time.Time
	FromNSU           int64
	ToNSU             int64
	DocumentsFound    int
	ErrorsCount       int
	Status            string // "running" | "completed" | "failed" | "interrupted"
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
