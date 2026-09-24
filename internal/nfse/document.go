package nfse

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
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
	ServiceValue       dfe.Money
	ISSValue           dfe.Money
	IRRFValue          dfe.Money
	INSSValue          dfe.Money
	PISValue           dfe.Money
	COFINSValue        dfe.Money
	CSLLValue          dfe.Money
	TotalRetentions    dfe.Money
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
	CompanyID        dfe.CompanyID
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
