package cte

import (
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// parseDateTime parses the CT-e date-time format (RFC 3339 with offset). On
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

// parseMoneyInto parses an XSD decimal into dst, naming field in the error.
func parseMoneyInto(dst *dfe.Money, field, value string) error {
	m, err := dfe.ParseMoney(value)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = m
	return nil
}

// addMoneyInto parses an XSD decimal and adds it to dst, naming field in the
// error.
func addMoneyInto(dst *dfe.Money, field, value string) error {
	var m dfe.Money
	if err := parseMoneyInto(&m, field, value); err != nil {
		return err
	}
	sum, err := dst.Add(m)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = sum
	return nil
}
