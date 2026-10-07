package syncstate

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/dfe"
)

type SyncRunID string

type (
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
