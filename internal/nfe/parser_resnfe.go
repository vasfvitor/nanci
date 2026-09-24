package nfe

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// ParseResNFe parses a resNFe (schema resNFe_v1.01), the summary SEFAZ
// distributes to the destinatário before the manifestação. Serie, Numero,
// Modelo and EmitenteUF come from the access key because the summary does not
// carry them.
func ParseResNFe(data []byte) (Document, error) {
	doc := Document{Completeness: CompletenessResumo}
	var warnings []string
	var chave, cSitNFe string

	onStart := func(path string, attrs []xml.Attr) {
		if strings.HasSuffix(path, "/resNFe") {
			doc.LayoutVersion = attrValue(attrs, "versao")
		}
	}
	onText := func(path, value string) error {
		switch {
		case strings.HasSuffix(path, "/resNFe/chNFe"):
			chave = value
		case hasAnySuffix(path, "/resNFe/CNPJ", "/resNFe/CPF"):
			doc.EmitenteCNPJ = value
		case strings.HasSuffix(path, "/resNFe/xNome"):
			doc.EmitenteName = value
		case strings.HasSuffix(path, "/resNFe/IE"):
			doc.EmitenteIE = value
		case strings.HasSuffix(path, "/resNFe/dhEmi"):
			if t := parseDateTime("dhEmi", value, &warnings); t != nil {
				doc.IssueDate = *t
			}
		case strings.HasSuffix(path, "/resNFe/tpNF"):
			doc.TpNF = value
		case strings.HasSuffix(path, "/resNFe/vNF"):
			return parseMoneyInto(&doc.TotalValue, "vNF", value)
		case strings.HasSuffix(path, "/resNFe/dhRecbto"):
			doc.AuthorizedAt = parseDateTime("dhRecbto", value, &warnings)
		case strings.HasSuffix(path, "/resNFe/nProt"):
			doc.Protocolo = value
		case strings.HasSuffix(path, "/resNFe/cSitNFe"):
			cSitNFe = value
		}
		return nil
	}
	if err := walkXML(data, onStart, onText); err != nil {
		return Document{}, err
	}

	if chave == "" {
		return Document{}, errors.New("missing essential field: chNFe")
	}
	key, err := dfe.ParseAccessKey(chave)
	if err != nil {
		return Document{}, fmt.Errorf("chNFe: %w", err)
	}
	doc.ChaveAcesso = key
	doc.Modelo = key.Modelo()
	doc.Serie = withoutLeadingZeros(key.Serie())
	doc.Numero = withoutLeadingZeros(key.Numero())
	doc.EmitenteUF = key.UF()

	switch cSitNFe {
	case "1":
		doc.Situacao = SituacaoAutorizada
	case "2":
		doc.Situacao = SituacaoDenegada
	case "3":
		doc.Situacao = SituacaoCancelada
	case "":
		return Document{}, errors.New("missing essential field: cSitNFe")
	default:
		return Document{}, fmt.Errorf("unsupported cSitNFe %q", cSitNFe)
	}

	if doc.IssueDate.IsZero() {
		warnings = append(warnings, "missing dhEmi; competence is unknown")
	}
	doc.Competence = competence(doc.IssueDate)
	doc.ParseWarnings = warnings
	return doc, nil
}
