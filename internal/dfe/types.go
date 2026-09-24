package dfe

import "errors"

// ErrInvalidEnum is returned when an enum value fails to parse. The specific
// value is included in the message; match with errors.Is(err, ErrInvalidEnum).
var ErrInvalidEnum = errors.New("invalid enum value")

// CompanyID identifies a company registered in nanci.
type CompanyID string
