package nfse

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// Company represents a company that syncs documents.
type Company struct {
	ID                 dfe.CompanyID
	CNPJ               string // stored as a 14-char identifier; current input policy accepts validated numeric CNPJ only
	CNPJRoot           string // first 8 chars - groups branches
	Name               string
	CredentialID       dfe.CredentialID
	CredentialLabel    string
	CredentialCertPath string
	Environment        dfe.Environment // derived from the assigned credential
	UF                 string          // optional state sigla, e.g. "SP"; empty when unknown
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	SyncStartPolicy    syncstate.SyncStartPolicy
	SyncStartDate      *time.Time
	InitialSyncDoneAt  *time.Time // NFS-e source; read from company_sync_sources
	LastRunStatus      syncstate.SyncStatus
	LastRunStopReason  syncstate.SyncStopReason
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Credential represents a reusable mTLS credential that can be assigned to multiple companies.
type Credential struct {
	ID                dfe.CredentialID
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
