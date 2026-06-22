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
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	SyncStartPolicy    SyncStartPolicy
	SyncStartDate      *time.Time
	InitialSyncDoneAt  *time.Time
	LastRunStatus      SyncStatus
	LastRunStopReason  SyncStopReason
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Credential represents a reusable mTLS credential that can be assigned to multiple companies.
type Credential struct {
	ID                CredentialID
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
