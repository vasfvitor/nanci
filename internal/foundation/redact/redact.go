// Package redact masks the identifiers of companies, people and documents
// so fiscal payloads can go to the log.
package redact

import (
	"regexp"
	"strings"
)

// MaskIdentifier keeps the first and last two characters of v and stars the
// rest, so masked values stay recognisable without being reusable. It counts
// runes, so accented names keep whole characters at the edges.
func MaskIdentifier(v string) string {
	r := []rune(v)
	if len(r) <= 4 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:2]) + strings.Repeat("*", len(r)-4) + string(r[len(r)-2:])
}

// identifierElement matches the text of XML elements that identify a
// company, a person or a document in NF-e, CT-e and NFS-e payloads, with or
// without a namespace prefix. The name must match whole: chave (the NF-e key
// in a CT-e infDoc/infNFe) does not catch chaveTeste.
var identifierElement = regexp.MustCompile(`(<(?:[\w.-]+:)?(?:CNPJ|CPF|CNPJDest|CPFDest|NIF|IE|IM|chNFe|chNFSe|chCTe|chave|xNome|xFant|email|fone)(?:\s[^>]*)?>)([^<]*)(</)`)

// MaskXMLIdentifiers masks, with MaskIdentifier, the text of the CNPJ, CPF,
// CNPJDest, CPFDest, NIF, IE, IM, chNFe, chNFSe, chCTe, chave, xNome, xFant,
// email and fone elements of body, so XML can go to the log.
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
