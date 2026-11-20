package company

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
	// SyncStartPolicyLocked is true while the company has an NFS-e sync cursor
	// in any environment: UpdateCompany then refuses another start policy or
	// date. Set by ListCompanies only.
	SyncStartPolicyLocked bool
	LastRunStatus         syncstate.SyncStatus
	LastRunStopReason     syncstate.SyncStopReason
	LastRunError          string // message of the last NFS-e run, set only when it failed
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
