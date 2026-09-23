// Package redact masks the identifiers of companies, people and documents
// so fiscal payloads can go to the log.
package redact

import (
	"regexp"
	"strings"
)

// MaskIdentifier keeps the first and last two characters of v and stars the
// rest, so masked values stay recognisable without being reusable.
func MaskIdentifier(v string) string {
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

// identifierElement matches the text of XML elements that identify a
// company, a person or a document in NF-e and NFS-e payloads, with or
// without a namespace prefix.
var identifierElement = regexp.MustCompile(`(<(?:[\w.-]+:)?(?:CNPJ|CPF|CNPJDest|CPFDest|NIF|IE|IM|chNFe|chNFSe|xNome|xFant|email|fone)(?:\s[^>]*)?>)([^<]*)(</)`)

// MaskXMLIdentifiers masks, with MaskIdentifier, the text of the CNPJ, CPF,
// CNPJDest, CPFDest, NIF, IE, IM, chNFe, chNFSe, xNome, xFant, email and fone
// elements of body, so XML can go to the log.
func MaskXMLIdentifiers(body []byte) []byte {
	return identifierElement.ReplaceAllFunc(body, func(match []byte) []byte {
		parts := identifierElement.FindSubmatch(match)
		masked := MaskIdentifier(string(parts[2]))
		out := make([]byte, 0, len(match))
		out = append(out, parts[1]...)
		out = append(out, masked...)
		return append(out, parts[3]...)
	})
}
