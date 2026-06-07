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

type DocumentID string
type CompanyID string
type CredentialID string
type SyncRunID string

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

type DocumentStatus string
type CompanyRole string
type VisibilityReason string
type EventType string
type SyncStatus string
type ConsultationBasis string

const (
	SyncStatusRunning   SyncStatus = "running"
	SyncStatusCompleted SyncStatus = "completed"
	SyncStatusFailed    SyncStatus = "failed"
)

func ParseDocumentStatus(val string) (DocumentStatus, error) {
	return DocumentStatus(val), nil // TODO: Add valid constants later
}

func (e DocumentStatus) Valid() bool {
	return true // TODO
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
	return CompanyRole(val), nil // TODO: Add valid constants later
}

func (e CompanyRole) Valid() bool {
	return true // TODO
}

func (e CompanyRole) String() string {
	return string(e)
}

func ParseVisibilityReason(val string) (VisibilityReason, error) {
	return VisibilityReason(val), nil // TODO: Add valid constants later
}

func (e VisibilityReason) Valid() bool {
	return true // TODO
}

func (e VisibilityReason) String() string {
	return string(e)
}

func ParseEventType(val string) (EventType, error) {
	return EventType(val), nil // TODO: Add valid constants later
}

func (e EventType) Valid() bool {
	return true // TODO
}

func (e EventType) String() string {
	return string(e)
}

func ParseSyncStatus(val string) (SyncStatus, error) {
	return SyncStatus(val), nil // TODO: Add valid constants later
}

func (e SyncStatus) Valid() bool {
	return true // TODO
}

func (e SyncStatus) String() string {
	return string(e)
}

func ParseConsultationBasis(val string) (ConsultationBasis, error) {
	return ConsultationBasis(val), nil // TODO: Add valid constants later
}

func (e ConsultationBasis) Valid() bool {
	return true // TODO
}

func (e ConsultationBasis) String() string {
	return string(e)
}
