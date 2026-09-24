package nfe

import (
	"fmt"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// parseDateTime parses the NF-e date-time format (RFC 3339 with offset). On
// failure it appends a warning naming the field and returns nil.
func parseDateTime(field, value string, warnings *[]string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("invalid %s format: %s", field, value))
		return nil
	}
	return &parsed
}

// competence returns "YYYY-MM" in the offset dhEmi was written with, or ""
// for a zero time.
func competence(issueDate time.Time) string {
	if issueDate.IsZero() {
		return ""
	}
	return issueDate.Format("2006-01")
}

// withoutLeadingZeros turns the zero-padded serie and nNF slots of the access
// key into the form used inside the NF-e ("001" -> "1", "000" -> "0").
func withoutLeadingZeros(s string) string {
	trimmed := strings.TrimLeft(s, "0")
	if trimmed == "" && s != "" {
		return "0"
	}
	return trimmed
}

// parseMoneyInto parses an XSD decimal into dst, naming field in the error.
func parseMoneyInto(dst *dfe.Money, field, value string) error {
	m, err := dfe.ParseMoney(value)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = m
	return nil
}
