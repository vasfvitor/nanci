package dfe

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The helpers below are shared by the NF-e and CT-e parsers. kind is "NFe"
// or "CTe": the suffix of the chave element (chNFe, chCTe) and the prefix of
// the signed element's Id (Id="NFe…", Id="CTe…").

// ParseDateTime parses the NF-e and CT-e date-time format (RFC 3339 with
// offset). On failure it appends a warning naming the field and returns nil.
func ParseDateTime(field, value string, warnings *[]string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("invalid %s format: %s", field, value))
		return nil
	}
	return &parsed
}

// Competence returns "YYYY-MM" in the offset dhEmi was written with, or ""
// for a zero time.
func Competence(issueDate time.Time) string {
	if issueDate.IsZero() {
		return ""
	}
	return issueDate.Format("2006-01")
}

// ParseMoneyInto parses an XSD decimal into dst, naming field in the error.
func ParseMoneyInto(dst *Money, field, value string) error {
	m, err := ParseMoney(value)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = m
	return nil
}

// KeyFromProtocol picks the access key of a document from its protocol
// (infProt/chNFe or infProt/chCTe), falling back to the Id of the signed
// element (kind + key). When both are present they must agree.
func KeyFromProtocol(kind, protChave, id string, warnings *[]string) (AccessKey, error) {
	keyField := "ch" + kind
	idChave := strings.TrimPrefix(id, kind)
	switch {
	case protChave == "" && idChave == "":
		return "", errors.New("missing essential field: " + keyField)
	case protChave == "":
		*warnings = append(*warnings, fmt.Sprintf("missing infProt/%s; using Id %s", keyField, id))
		protChave = idChave
	case idChave != "" && idChave != protChave:
		return "", fmt.Errorf("infProt/%s %s does not match Id %s", keyField, protChave, id)
	}
	key, err := ParseAccessKey(protChave)
	if err != nil {
		return "", fmt.Errorf("%s: %w", keyField, err)
	}
	return key, nil
}

// ParseEventIdentity validates the fields that identify an event: its
// chave, its tpEvento (six digits, as it also names exported files) and its
// nSeqEvento.
func ParseEventIdentity(kind, chave, tpEvento, nSeqEvento string) (AccessKey, int, error) {
	keyField := "ch" + kind
	if chave == "" {
		return "", 0, errors.New("missing essential field: " + keyField)
	}
	key, err := ParseAccessKey(chave)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", keyField, err)
	}
	if tpEvento == "" {
		return "", 0, errors.New("missing essential field: tpEvento")
	}
	if !isTpEvento(tpEvento) {
		return "", 0, fmt.Errorf("invalid tpEvento %q", tpEvento)
	}
	if nSeqEvento == "" {
		return "", 0, errors.New("missing essential field: nSeqEvento")
	}
	seq, err := strconv.Atoi(nSeqEvento)
	if err != nil || seq < 1 {
		return "", 0, fmt.Errorf("invalid nSeqEvento %q", nSeqEvento)
	}
	return key, seq, nil
}

// isTpEvento reports whether s is a tpEvento code: exactly six ASCII digits.
func isTpEvento(s string) bool {
	if len(s) != 6 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}
