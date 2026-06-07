package nfse

import (
	"time"
)

// Company represents a company that syncs documents.
type Company struct {
	ID                 CompanyID
	CNPJ               string // stored as a 14-char identifier; current input policy accepts validated numeric CNPJ only
	CNPJRoot           string // first 8 chars - groups branches
	Name               string
	CredentialID       CredentialID
	CredentialLabel    string
	CredentialCertPath string
	Environment        Environment // derived from the assigned credential
	LastNSU            int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Credential represents a reusable mTLS credential that can be assigned to multiple companies.
type Credential struct {
	ID                CredentialID
	Label             string
	CertPath          string
	Environment       Environment
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
	ID                 DocumentID
	ChaveAcesso        AccessKey
	IssueDate          time.Time
	Competence         string // "YYYY-MM"
	PrestadorCNPJ      string
	PrestadorName      string
	TomadorCNPJ        string
	TomadorName        string
	IntermediarioCNPJ  string
	IntermediarioName  string
	ServiceValue       Money
	ISSValue           Money
	IRRFValue          Money
	INSSValue          Money
	PISValue           Money
	COFINSValue        Money
	CSLLValue          Money
	TotalRetentions    Money
	Status             DocumentStatus // "normal" | "cancelada" | "substituida"
	LayoutVersion      string
	XMLPath            string
	RawHash            string
	ParseWarnings      []string
	NFSeNumber         string
	ServiceDescription string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CompanyDocument represents the participation of one managed company in a canonical document.
type CompanyDocument struct {
	Document
	RelationID        string
	CompanyID         CompanyID
	DocumentID        DocumentID
	CompanyRole       CompanyRole      // "tomada" | "prestada" | "intermediario" | "none"
	VisibilityReason  VisibilityReason // "exact_prestador" | "exact_tomador" | "exact_intermediario" | "same_root_only" | "unknown"
	FirstSeenNSU      int64
	LastSeenNSU       int64
	FirstSeenNSUValid bool
	LastSeenNSUValid  bool
	FirstSyncedAt     time.Time
	LastSyncedAt      time.Time
}

// CompanyParticipation contains company-scoped role and visibility classification for one document.
type CompanyParticipation struct {
	CompanyRole      CompanyRole
	VisibilityReason VisibilityReason
}

// SyncRun represents a synchronization execution for audit and control.
type SyncRun struct {
	ID                SyncRunID
	CompanyID         CompanyID
	CredentialID      CredentialID
	CredentialCNPJ    string
	ConsultationCNPJ  string
	ConsultationBasis ConsultationBasis // "exact_certificate_cnpj" | "same_root_certificate"
	StartedAt         time.Time
	FinishedAt        *time.Time
	FromNSU           int64
	ToNSU             int64
	DocumentsFound    int
	ErrorsCount       int
	Status            SyncStatus // "running" | "completed" | "failed" | "interrupted"
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
