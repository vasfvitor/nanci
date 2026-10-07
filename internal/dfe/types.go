package dfe

import (
	"errors"
	"fmt"
)

// ErrInvalidEnum is returned when an enum value fails to parse. The specific
// value is included in the message; match with errors.Is(err, ErrInvalidEnum).
var ErrInvalidEnum = errors.New("invalid enum value")

// CompanyID identifies a company registered in nanci.
type CompanyID string

type CredentialID string

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
		return "", fmt.Errorf("invalid environment %q: %w", val, ErrInvalidEnum)
	}
}

func (e Environment) Valid() bool {
	_, err := ParseEnvironment(string(e))
	return err == nil
}

func (e Environment) String() string {
	return string(e)
}
