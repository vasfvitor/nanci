package sefaz

import (
	"github.com/vasfvitor/nanci/internal/foundation/httpclient"
)

// RedactForLog masks the identifiers of body (CNPJ, CPF, chNFe, xNome, IE,
// the CNPJDest/CPFDest of event answers and the other elements
// httpclient.MaskXMLIdentifiers covers) so response bodies can go to the log.
func RedactForLog(body []byte) []byte {
	return httpclient.MaskXMLIdentifiers(body)
}
