package sefaz

import (
	"regexp"

	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
)

// identifierElement matches the text of elements that identify a company, a
// person or a document, with or without a namespace prefix.
var identifierElement = regexp.MustCompile(`(<(?:[\w.-]+:)?(?:CNPJ|CPF|CNPJDest|CPFDest|chNFe|xNome|IE)(?:\s[^>]*)?>)([^<]*)(</)`)

// RedactForLog masks the text of CNPJ, CPF, chNFe, xNome and IE elements
// (and CNPJDest/CPFDest of event answers) so response bodies can go to the
// log.
func RedactForLog(body []byte) []byte {
	return identifierElement.ReplaceAllFunc(body, func(match []byte) []byte {
		parts := identifierElement.FindSubmatch(match)
		masked := httpclient.MaskIdentifier(string(parts[2]))
		out := make([]byte, 0, len(match))
		out = append(out, parts[1]...)
		out = append(out, masked...)
		return append(out, parts[3]...)
	})
}
