package main

import (
	"regexp"
	"strings"
)

// Log files on disk keep CNPJs in clear for local troubleshooting; they are
// masked only when exported (see docs/specs/2026-06-20-diagnostics-and-secure-logging-design.md).
var (
	formattedCNPJPattern = regexp.MustCompile(`\b[A-Za-z0-9]{2}\.[A-Za-z0-9]{3}\.[A-Za-z0-9]{3}/[A-Za-z0-9]{4}-[A-Za-z0-9]{2}\b`)
	// Raw matches are restricted to digits so hashes and hex IDs are untouched.
	rawCNPJPattern = regexp.MustCompile(`\b\d{14}\b`)
)

// sanitizeLogContent masks every CNPJ in content as XX.***.***/****-XX,
// keeping the first and last two characters for partial traceability.
func sanitizeLogContent(content []byte) []byte {
	out := formattedCNPJPattern.ReplaceAllFunc(content, maskCNPJMatch)
	return rawCNPJPattern.ReplaceAllFunc(out, maskCNPJMatch)
}

func maskCNPJMatch(match []byte) []byte {
	cleaned := strings.NewReplacer(".", "", "/", "", "-", "").Replace(string(match))
	if len(cleaned) != 14 {
		return match
	}
	return []byte(cleaned[:2] + ".***.***/****-" + cleaned[12:])
}
