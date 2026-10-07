package nfse

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

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
