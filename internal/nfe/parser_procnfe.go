package nfe

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/xmlwalk"
)

// ParseProcNFe parses a procNFe (nfeProc: the signed NFe plus protNFe), the
// full NF-e in layout 4.00. Totals come only from total/ICMSTot, never from
// the per-item taxes under det/imposto.
func ParseProcNFe(data []byte) (Document, error) {
	doc := Document{Completeness: CompletenessCompleta}
	var warnings []string
	var infNFeID, protChave, cStat, xMotivo string

	onStart := func(path string, attrs []xml.Attr) {
		if strings.HasSuffix(path, "/NFe/infNFe") {
			infNFeID = xmlwalk.AttrValue(attrs, "Id")
			doc.LayoutVersion = xmlwalk.AttrValue(attrs, "versao")
		}
	}
	onText := func(path, value string) error {
		switch {
		// ide
		case strings.HasSuffix(path, "/infNFe/ide/mod"):
			doc.Modelo = value
		case strings.HasSuffix(path, "/infNFe/ide/serie"):
			doc.Serie = value
		case strings.HasSuffix(path, "/infNFe/ide/nNF"):
			doc.Numero = value
		case strings.HasSuffix(path, "/infNFe/ide/dhEmi"):
			if t := dfe.ParseDateTime("dhEmi", value, &warnings); t != nil {
				doc.IssueDate = *t
			}
		case strings.HasSuffix(path, "/infNFe/ide/tpNF"):
			doc.TpNF = value
		case strings.HasSuffix(path, "/infNFe/ide/finNFe"):
			doc.FinNFe = value
		case strings.HasSuffix(path, "/infNFe/ide/tpAmb"):
			doc.TpAmb = value
		case strings.HasSuffix(path, "/infNFe/ide/natOp"):
			doc.NatOp = value

		// parties
		case xmlwalk.HasAnySuffix(path, "/infNFe/emit/CNPJ", "/infNFe/emit/CPF"):
			doc.EmitenteCNPJ = value
		case strings.HasSuffix(path, "/infNFe/emit/xNome"):
			doc.EmitenteName = value
		case strings.HasSuffix(path, "/infNFe/emit/IE"):
			doc.EmitenteIE = value
		case strings.HasSuffix(path, "/infNFe/emit/enderEmit/UF"):
			doc.EmitenteUF = value
		case xmlwalk.HasAnySuffix(path, "/infNFe/dest/CNPJ", "/infNFe/dest/CPF", "/infNFe/dest/idEstrangeiro"):
			doc.DestinatarioCNPJ = value
		case strings.HasSuffix(path, "/infNFe/dest/xNome"):
			doc.DestinatarioName = value
		case xmlwalk.HasAnySuffix(path, "/infNFe/transp/transporta/CNPJ", "/infNFe/transp/transporta/CPF"):
			doc.TransportadorCNPJ = value
		case xmlwalk.HasAnySuffix(path, "/infNFe/autXML/CNPJ", "/infNFe/autXML/CPF"):
			doc.AutorizadosCNPJ = append(doc.AutorizadosCNPJ, value)

		// totals
		case strings.HasSuffix(path, "/infNFe/total/ICMSTot/vNF"):
			return dfe.ParseMoneyInto(&doc.TotalValue, "vNF", value)
		case strings.HasSuffix(path, "/infNFe/total/ICMSTot/vICMS"):
			return dfe.ParseMoneyInto(&doc.ICMSValue, "vICMS", value)
		case strings.HasSuffix(path, "/infNFe/total/ICMSTot/vIPI"):
			return dfe.ParseMoneyInto(&doc.IPIValue, "vIPI", value)

		// authorization protocol
		case strings.HasSuffix(path, "/protNFe/infProt/chNFe"):
			protChave = value
		case strings.HasSuffix(path, "/protNFe/infProt/nProt"):
			doc.Protocolo = value
		case strings.HasSuffix(path, "/protNFe/infProt/dhRecbto"):
			doc.AuthorizedAt = dfe.ParseDateTime("dhRecbto", value, &warnings)
		case strings.HasSuffix(path, "/protNFe/infProt/cStat"):
			cStat = value
		case strings.HasSuffix(path, "/protNFe/infProt/xMotivo"):
			xMotivo = value
		}
		return nil
	}
	if err := xmlwalk.Walk(data, onStart, onText); err != nil {
		return Document{}, err
	}

	key, err := dfe.KeyFromProtocol("NFe", protChave, infNFeID, &warnings)
	if err != nil {
		return Document{}, err
	}
	doc.ChaveAcesso = key
	if doc.Modelo == "" {
		doc.Modelo = key.Modelo()
	}
	if doc.EmitenteUF == "" {
		doc.EmitenteUF = key.UF()
	}

	switch cStat {
	case "100", "150":
		doc.Situacao = SituacaoAutorizada
	case "110", "205", "301", "302", "303":
		doc.Situacao = SituacaoDenegada
	case "":
		return Document{}, errors.New("missing essential field: protNFe/infProt/cStat")
	default:
		return Document{}, fmt.Errorf("unsupported protNFe cStat %s (%s)", cStat, xMotivo)
	}

	if doc.IssueDate.IsZero() {
		warnings = append(warnings, "missing dhEmi; competence is unknown")
	}
	doc.Competence = dfe.Competence(doc.IssueDate)
	doc.ParseWarnings = warnings
	return doc, nil
}
