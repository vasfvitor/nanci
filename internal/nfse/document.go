package nfse

import (
	"time"
)

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
	RelationID       string
	CompanyID        CompanyID
	DocumentID       DocumentID
	CompanyRole      CompanyRole      // "tomada" | "prestada" | "intermediario" | "none"
	VisibilityReason VisibilityReason // "exact_prestador" | "exact_tomador" | "exact_intermediario" | "same_root_only" | "unknown"
	FirstSeenNSU     *int64
	LastSeenNSU      *int64
	FirstSyncedAt    time.Time
	LastSyncedAt     time.Time
	ViewedAt         *time.Time
}

// CompanyParticipation contains company-scoped role and visibility classification for one document.
type CompanyParticipation struct {
	CompanyRole      CompanyRole
	VisibilityReason VisibilityReason
}
