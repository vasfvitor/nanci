package main

import (
	"regexp"
	"strings"
)

// Log files on disk keep CNPJs in clear for local troubleshooting; they are
// masked only when exported (see docs/specs/2026-06-20-diagnostics-and-secure-logging-design.md).
var (
	formattedCNPJPattern = regexp.MustCompile(`\b[A-Za-z0-9]{2}\.[A-Za-z0-9]{3}\.[A-Za-z0-9]{3}/[A-Za-z0-9]{4}-[A-Za-z0-9]{2}\b`)
	// NF-e or CT-e access key (same layout): UF(2) AAMM(4) CNPJ(14, alphanumeric allowed) and 24
	// more digits. Only the embedded CNPJ slot is masked.
	nfeAccessKeyPattern = regexp.MustCompile(`\b\d{6}[0-9A-Z]{14}\d{24}\b`)
	// NFS-e access key: cMun(7) ambGer(1) tpInsc(1) inscFed(14) nNFSe(13)
	// AnoMes(4) cod(9) DV(1). Only the inscrição federal slot is masked; a
	// CPF sits there padded with three leading zeros.
	nfseAccessKeyPattern = regexp.MustCompile(`\b\d{50}\b`)
	// Raw matches are restricted to digits so hashes and hex IDs are untouched.
	rawCNPJPattern = regexp.MustCompile(`\b\d{14}\b`)
)

// sanitizeLogContent masks every CNPJ in content as XX.***.***/****-XX,
// keeping the first and last two characters for partial traceability.
func sanitizeLogContent(content []byte) []byte {
	out := formattedCNPJPattern.ReplaceAllFunc(content, maskCNPJMatch)
	out = nfeAccessKeyPattern.ReplaceAllFunc(out, maskNFeAccessKeyCNPJ)
	out = nfseAccessKeyPattern.ReplaceAllFunc(out, maskNFSeAccessKeyInscricao)
	return rawCNPJPattern.ReplaceAllFunc(out, maskCNPJMatch)
}

// maskNFeAccessKeyCNPJ masks the CNPJ at positions 7-20 of an NF-e access key
// and keeps the rest of the key intact.
func maskNFeAccessKeyCNPJ(key []byte) []byte {
	masked := make([]byte, 0, len(key)+4)
	masked = append(masked, key[:6]...)
	masked = append(masked, maskCNPJMatch(key[6:20])...)
	return append(masked, key[20:]...)
}

// maskNFSeAccessKeyInscricao masks the inscrição federal at positions 10-23
// of an NFS-e access key and keeps the rest of the key intact.
func maskNFSeAccessKeyInscricao(key []byte) []byte {
	masked := make([]byte, 0, len(key)+4)
	masked = append(masked, key[:9]...)
	masked = append(masked, maskCNPJMatch(key[9:23])...)
	return append(masked, key[23:]...)
}

func maskCNPJMatch(match []byte) []byte {
	cleaned := strings.NewReplacer(".", "", "/", "", "-", "").Replace(string(match))
	if len(cleaned) != 14 {
		return match
	}
	return []byte(cleaned[:2] + ".***.***/****-" + cleaned[12:])
}
