package httpclient

import (
	"strings"
	"unicode/utf8"
)

const (
	// MaxErrorBodyBytes caps how much of a rejected response body is read.
	MaxErrorBodyBytes = 64 * 1024 // 64 KiB
	// MaxErrorLogBodyBytes caps the error body attached to Error-level log
	// records and to StatusError.Error(); the full body (up to
	// MaxErrorBodyBytes) is only logged at trace.
	MaxErrorLogBodyBytes = 2 * 1024
	// maxTraceBodyBytes caps response bodies logged at trace level.
	maxTraceBodyBytes = 256 * 1024
)

// TruncateForLog shortens body for log output without splitting a UTF-8
// sequence, marking the cut so readers know the record is partial.
func TruncateForLog(body []byte, limit int) string {
	if len(body) <= limit {
		return string(body)
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return string(body[:cut]) + "... (truncated)"
}

// MaskIdentifier keeps the first and last two characters of v and stars the
// rest, so masked values stay recognisable without being reusable.
func MaskIdentifier(v string) string {
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}
