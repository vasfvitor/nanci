package nfse

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Money represents monetary values in integer cents (e.g., 1000 = R$ 10,00).
// The XSD allows 15 integer digits plus exactly two decimals, which safely fits in int64.
type Money int64

var (
	ErrInvalidMoneyFormat = errors.New("invalid money format")
	ErrMoneyOverflow      = errors.New("money overflow")
	// XML monetary format: optional decimals, no thousands separators, exactly two decimals if present
	moneyRegex = regexp.MustCompile(`^[0-9]{1,15}(\.[0-9]{1,2})?$`)
)

// ParseMoney parses a string in XSD decimal format into Money.
func ParseMoney(value string) (Money, error) {
	value = strings.TrimSpace(value)
	if !moneyRegex.MatchString(value) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMoneyFormat, value)
	}

	parts := strings.Split(value, ".")
	integerPart := parts[0]
	fractionalPart := "00"

	if len(parts) == 2 {
		fractionalPart = parts[1]
		if len(fractionalPart) == 1 {
			fractionalPart += "0" // pad to two decimals
		}
	}

	centsStr := integerPart + fractionalPart
	cents, err := strconv.ParseInt(centsStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrMoneyOverflow, err)
	}

	return Money(cents), nil
}

// NewMoneyFromCents creates a Money value from raw cents.
func NewMoneyFromCents(cents int64) Money {
	return Money(cents)
}

// Cents returns the integer cents value.
func (m Money) Cents() int64 {
	return int64(m)
}

// Add adds two Money values.
func (m Money) Add(other Money) (Money, error) {
	// Simple overflow check for int64 addition
	result := m.Cents() + other.Cents()
	if (result > m.Cents()) != (other.Cents() > 0) && other.Cents() != 0 {
		return 0, ErrMoneyOverflow
	}
	return Money(result), nil
}

// Sub subtracts another Money value.
func (m Money) Sub(other Money) (Money, error) {
	// Simple overflow check for int64 subtraction
	result := m.Cents() - other.Cents()
	if (result < m.Cents()) != (other.Cents() > 0) && other.Cents() != 0 {
		return 0, ErrMoneyOverflow
	}
	return Money(result), nil
}

// FormatBRL formats the Money value into Brazilian Real representation (e.g., "1.234,56").
func (m Money) FormatBRL() string {
	cents := m.Cents()
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}

	fractional := cents % 100
	integer := cents / 100

	intStr := strconv.FormatInt(integer, 10)
	var formattedInt strings.Builder
	for i, c := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			formattedInt.WriteRune('.')
		}
		formattedInt.WriteRune(c)
	}

	return fmt.Sprintf("%s%s,%02d", sign, formattedInt.String(), fractional)
}

// --- Identifiers ---

type AccessKey string

func ParseAccessKey(key string) (AccessKey, error) {
	key = strings.TrimSpace(key)
	if len(key) != 50 {
		return "", fmt.Errorf("access key must be exactly 50 digits, got %d", len(key))
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("access key contains non-digit character: %c", r)
		}
	}
	return AccessKey(key), nil
}

type (
	DocumentID   string
	CompanyID    string
	CredentialID string
	SyncRunID    string
)

// --- Enums ---

type Environment string

const (
	EnvironmentProduction Environment = "producao"
	EnvironmentRestricted Environment = "producao_restrita"
)

func ParseEnvironment(val string) (Environment, error) {
	switch Environment(val) {
	case EnvironmentProduction, EnvironmentRestricted:
		return Environment(val), nil
	default:
		return "", fmt.Errorf("invalid environment: %s", val)
	}
}

func (e Environment) Valid() bool {
	_, err := ParseEnvironment(string(e))
	return err == nil
}

func (e Environment) String() string {
	return string(e)
}

type (
	DocumentStatus    string
	CompanyRole       string
	VisibilityReason  string
	EventType         string
	SyncStatus        string
	ConsultationBasis string
	SyncMode          string
	SyncStopReason    string
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
		return "", fmt.Errorf("invalid document status: %s", val)
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
		return "", fmt.Errorf("invalid company role: %s", val)
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
		return "", fmt.Errorf("invalid visibility reason: %s", val)
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
		return "", fmt.Errorf("invalid event type: %s", val)
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
		return "", fmt.Errorf("invalid sync status: %s", val)
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
		return "", fmt.Errorf("invalid consultation basis: %s", val)
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
		return "", fmt.Errorf("invalid sync mode: %s", val)
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
)

func ParseSyncStopReason(val string) (SyncStopReason, error) {
	reason := SyncStopReason(val)
	if !reason.Valid() {
		return "", fmt.Errorf("invalid sync stop reason: %s", val)
	}
	return reason, nil
}

func (r SyncStopReason) Valid() bool {
	switch r {
	case SyncStopReasonEmptyLimit, SyncStopReasonContextCanceled, SyncStopReasonFetchError, SyncStopReasonProcessError:
		return true
	default:
		return false
	}
}

func (r SyncStopReason) String() string {
	return string(r)
}
