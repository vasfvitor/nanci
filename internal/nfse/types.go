package nfse

import (
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// --- Identifiers ---

type AccessKey string

// InfNFSeIDPrefix starts the Id attribute of infNFSe, followed by the 50
// characters of the chave.
const InfNFSeIDPrefix = "NFS"

const (
	accessKeyLength = 50
	// The inscrição federal sits at positions 10-23 of the chave:
	// cMun(7) ambGer(1) tpInsc(1) inscFed(14) nNFSe(13) AAMM(4) cNum(9) DV(1).
	inscricaoStart = 9  // 0-based index of its first character
	inscricaoEnd   = 23 // 0-based index after its last character
)

// ParseAccessKey trims and upper-cases key and checks the character set of
// the 50-character NFS-e chave. Every position is a digit except the
// inscrição federal, which also accepts uppercase letters for the
// alphanumeric CNPJ, as the TSIdNFSe pattern NFS[0-9]{9}[0-9A-Z]{14}[0-9]{27}
// of the NFS-e schemas of 27/07/2026 (NT SE/CGNFS-e 009) says. The check
// digit is not verified: no NFS-e text defines it for a chave with letters.
func ParseAccessKey(key string) (AccessKey, error) {
	key = strings.ToUpper(strings.TrimSpace(key))
	if len(key) != accessKeyLength {
		return "", fmt.Errorf("access key must have exactly %d characters, got %d", accessKeyLength, len(key))
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		inInscricao := i >= inscricaoStart && i < inscricaoEnd
		if (c >= '0' && c <= '9') || (inInscricao && c >= 'A' && c <= 'Z') {
			continue
		}
		return "", fmt.Errorf("access key has invalid character %q at position %d", c, i+1)
	}
	return AccessKey(key), nil
}

// ParseInfNFSeID parses the chave in an infNFSe Id, "NFS" followed by the 50
// characters. The prefix is optional, so a bare chave parses too.
func ParseInfNFSeID(id string) (AccessKey, error) {
	return ParseAccessKey(strings.TrimPrefix(id, InfNFSeIDPrefix))
}

type (
	DocumentID string
	SyncRunID  string
)

// --- Enums ---

type (
	DocumentStatus    string
	CompanyRole       string
	VisibilityReason  string
	EventType         string
	SyncStatus        string
	ConsultationBasis string
	SyncMode          string
	SyncStopReason    string
	SyncStartPolicy   string
)

const (
	SyncStatusRunning     SyncStatus = "running"
	SyncStatusCompleted   SyncStatus = "completed"
	SyncStatusFailed      SyncStatus = "failed"
	SyncStatusInterrupted SyncStatus = "interrupted"
)

func ParseDocumentStatus(val string) (DocumentStatus, error) {
	status := DocumentStatus(val)
	if !status.Valid() {
		return "", fmt.Errorf("invalid document status %q: %w", val, dfe.ErrInvalidEnum)
	}
	return status, nil
}

func (e DocumentStatus) Valid() bool {
	switch e {
	case DocumentStatusNormal, DocumentStatusCancelada, DocumentStatusSubstituida:
		return true
	default:
		return false
	}
}

func (e DocumentStatus) String() string {
	return string(e)
}

const (
	CompanyRoleTomada        CompanyRole = "tomada"
	CompanyRolePrestada      CompanyRole = "prestada"
	CompanyRoleIntermediario CompanyRole = "intermediario"
)

const (
	DocumentStatusNormal      DocumentStatus = "normal"
	DocumentStatusCancelada   DocumentStatus = "cancelada"
	DocumentStatusSubstituida DocumentStatus = "substituida"
)

func ParseCompanyRole(val string) (CompanyRole, error) {
	role := CompanyRole(val)
	if !role.Valid() {
		return "", fmt.Errorf("invalid company role %q: %w", val, dfe.ErrInvalidEnum)
	}
	return role, nil
}

func (e CompanyRole) Valid() bool {
	switch e {
	case CompanyRoleTomada, CompanyRolePrestada, CompanyRoleIntermediario:
		return true
	default:
		return false
	}
}

func (e CompanyRole) String() string {
	return string(e)
}

const (
	VisibilityReasonExactPrestador     VisibilityReason = "exact_prestador"
	VisibilityReasonExactTomador       VisibilityReason = "exact_tomador"
	VisibilityReasonExactIntermediario VisibilityReason = "exact_intermediario"
	VisibilityReasonSameRootOnly       VisibilityReason = "same_root_only"
	VisibilityReasonUnknown            VisibilityReason = "unknown"
)

func ParseVisibilityReason(val string) (VisibilityReason, error) {
	reason := VisibilityReason(val)
	if !reason.Valid() {
		return "", fmt.Errorf("invalid visibility reason %q: %w", val, dfe.ErrInvalidEnum)
	}
	return reason, nil
}

func (e VisibilityReason) Valid() bool {
	switch e {
	case VisibilityReasonExactPrestador, VisibilityReasonExactTomador, VisibilityReasonExactIntermediario, VisibilityReasonSameRootOnly, VisibilityReasonUnknown:
		return true
	default:
		return false
	}
}

func (e VisibilityReason) String() string {
	return string(e)
}

const (
	EventTypeCancelamento EventType = "cancelamento"
	EventTypeSubstituicao EventType = "substituicao"
	EventTypeUnknown      EventType = "unknown"
)

func ParseEventType(val string) (EventType, error) {
	evtType := EventType(val)
	if !evtType.Valid() {
		return "", fmt.Errorf("invalid event type %q: %w", val, dfe.ErrInvalidEnum)
	}
	return evtType, nil
}

func (e EventType) Valid() bool {
	switch e {
	case EventTypeCancelamento, EventTypeSubstituicao, EventTypeUnknown:
		return true
	default:
		return false
	}
}

func (e EventType) String() string {
	return string(e)
}

func ParseSyncStatus(val string) (SyncStatus, error) {
	status := SyncStatus(val)
	if !status.Valid() {
		return "", fmt.Errorf("invalid sync status %q: %w", val, dfe.ErrInvalidEnum)
	}
	return status, nil
}

func (e SyncStatus) Valid() bool {
	switch e {
	case SyncStatusRunning, SyncStatusCompleted, SyncStatusFailed, SyncStatusInterrupted:
		return true
	default:
		return false
	}
}

func (e SyncStatus) String() string {
	return string(e)
}

const (
	ConsultationBasisExactCertificateCNPJ ConsultationBasis = "exact_certificate_cnpj"
	ConsultationBasisSameRootCertificate  ConsultationBasis = "same_root_certificate"
)

func ParseConsultationBasis(val string) (ConsultationBasis, error) {
	basis := ConsultationBasis(val)
	if !basis.Valid() {
		return "", fmt.Errorf("invalid consultation basis %q: %w", val, dfe.ErrInvalidEnum)
	}
	return basis, nil
}

func (e ConsultationBasis) Valid() bool {
	switch e {
	case ConsultationBasisExactCertificateCNPJ, ConsultationBasisSameRootCertificate:
		return true
	default:
		return false
	}
}

func (e ConsultationBasis) String() string {
	return string(e)
}

const (
	SyncModeNormal     SyncMode = "normal"
	SyncModeFirstSetup SyncMode = "first_setup"
)

func ParseSyncMode(val string) (SyncMode, error) {
	mode := SyncMode(val)
	if !mode.Valid() {
		return "", fmt.Errorf("invalid sync mode %q: %w", val, dfe.ErrInvalidEnum)
	}
	return mode, nil
}

func (m SyncMode) Valid() bool {
	switch m {
	case SyncModeNormal, SyncModeFirstSetup:
		return true
	default:
		return false
	}
}

func (m SyncMode) String() string {
	return string(m)
}

const (
	SyncStopReasonEmptyLimit      SyncStopReason = "empty_limit"
	SyncStopReasonContextCanceled SyncStopReason = "context_canceled"
	SyncStopReasonFetchError      SyncStopReason = "fetch_error"
	SyncStopReasonProcessError    SyncStopReason = "process_error"
	// SyncStopReasonCaughtUp means the source reported no documents beyond the cursor.
	SyncStopReasonCaughtUp SyncStopReason = "caught_up"
	// SyncStopReasonConsumoIndevido is the tax authority's rejection for querying too often.
	SyncStopReasonConsumoIndevido SyncStopReason = "consumo_indevido"
	// SyncStopReasonRateBudget means the local hourly request budget is exhausted.
	SyncStopReasonRateBudget SyncStopReason = "rate_budget"
)

func ParseSyncStopReason(val string) (SyncStopReason, error) {
	reason := SyncStopReason(val)
	if !reason.Valid() {
		return "", fmt.Errorf("invalid sync stop reason %q: %w", val, dfe.ErrInvalidEnum)
	}
	return reason, nil
}

func (r SyncStopReason) Valid() bool {
	switch r {
	case SyncStopReasonEmptyLimit, SyncStopReasonContextCanceled, SyncStopReasonFetchError, SyncStopReasonProcessError,
		SyncStopReasonCaughtUp, SyncStopReasonConsumoIndevido, SyncStopReasonRateBudget:
		return true
	default:
		return false
	}
}

func (r SyncStopReason) String() string {
	return string(r)
}

const (
	SyncStartPolicyAll       SyncStartPolicy = "all"
	SyncStartPolicySinceDate SyncStartPolicy = "since_date"
	SyncStartPolicyFromNow   SyncStartPolicy = "from_now"
)

func ParseSyncStartPolicy(val string) (SyncStartPolicy, error) {
	policy := SyncStartPolicy(val)
	if !policy.Valid() {
		return "", fmt.Errorf("invalid sync start policy %q: %w", val, dfe.ErrInvalidEnum)
	}
	return policy, nil
}

func (p SyncStartPolicy) Valid() bool {
	switch p {
	case SyncStartPolicyAll, SyncStartPolicySinceDate, SyncStartPolicyFromNow:
		return true
	default:
		return false
	}
}

func (p SyncStartPolicy) String() string {
	return string(p)
}
