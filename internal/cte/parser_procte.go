package cte

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/xmlwalk"
)

// tipoByElement maps the signed element that wraps infCte to the kind of
// document.
var tipoByElement = map[string]TipoDocumento{
	"CTe":     TipoDocumentoCTe,
	"CTeOS":   TipoDocumentoCTeOS,
	"GTVe":    TipoDocumentoGTVe,
	"CTeSimp": TipoDocumentoCTeSimplificado,
}

// ParseProcCTe parses a transport document with its authorization protocol:
// procCTe (cteProc), procCTeOS (cteOSProc), procGTVe and procCTeSimp
// (cteSimpProc), in layouts 2.00 to 4.00. The four share the infCte paths
// read here, and the kind of document comes from the element that wraps
// infCte, so a CT-e Simplificado delivered under the procCTe schema is still
// recognized. Fields a kind does not carry stay empty.
//
// The tomador is resolved after the walk (see resolveTomador). Only the
// access key and the protocol cStat are essential; everything else that is
// missing or unexpected becomes a parse warning.
func ParseProcCTe(data []byte) (Document, error) {
	var doc Document
	var warnings []string
	var wrapper, infCteID, protChave, protTpAmb, cStat, xMotivo string
	var named Party // the tomador as named in toma4, tomaTerceiro or infCte/toma
	var nfeChaves []string

	parties := []struct {
		party  *Party
		prefix string // path suffix of the party element
		ender  string // name of its address element
	}{
		{&doc.Emitente, "/infCte/emit", "enderEmit"},
		{&doc.Remetente, "/infCte/rem", "enderReme"},
		{&doc.Expedidor, "/infCte/exped", "enderExped"},
		{&doc.Recebedor, "/infCte/receb", "enderReceb"},
		{&doc.Destinatario, "/infCte/dest", "enderDest"},
		{&named, "/infCte/ide/toma4", "enderToma"},        // CT-e
		{&named, "/infCte/ide/tomaTerceiro", "enderToma"}, // GTV-e
		{&named, "/infCte/toma", "enderToma"},             // CT-e OS and CT-e Simplificado
	}

	onStart := func(p string, attrs []xml.Attr) {
		if strings.HasSuffix(p, "/infCte") && infCteID == "" {
			infCteID = xmlwalk.AttrValue(attrs, "Id")
			doc.LayoutVersion = xmlwalk.AttrValue(attrs, "versao")
			wrapper = path.Base(path.Dir(p))
		}
	}
	onText := func(p, value string) error {
		for _, pt := range parties {
			if setPartyField(pt.party, p, pt.prefix, pt.ender, value) {
				return nil
			}
		}

		switch {
		// ide
		case strings.HasSuffix(p, "/infCte/ide/tpAmb"):
			doc.TpAmb = value
		case strings.HasSuffix(p, "/infCte/ide/mod"):
			doc.Modelo = value
		case strings.HasSuffix(p, "/infCte/ide/serie"):
			doc.Serie = value
		case strings.HasSuffix(p, "/infCte/ide/nCT"):
			doc.Numero = value
		case strings.HasSuffix(p, "/infCte/ide/CFOP"):
			doc.CFOP = value
		case strings.HasSuffix(p, "/infCte/ide/natOp"):
			doc.NatOp = value
		case strings.HasSuffix(p, "/infCte/ide/dhEmi"):
			if t := dfe.ParseDateTime("dhEmi", value, &warnings); t != nil {
				doc.IssueDate = *t
			}
		case strings.HasSuffix(p, "/infCte/ide/tpCTe"):
			doc.TpCTe = value
		case strings.HasSuffix(p, "/infCte/ide/tpServ"):
			doc.TpServ = value
		case strings.HasSuffix(p, "/infCte/ide/modal"):
			doc.Modal = value
		case strings.HasSuffix(p, "/infCte/ide/UFIni"):
			doc.MunIni.UF = value
		case strings.HasSuffix(p, "/infCte/ide/UFFim"):
			doc.MunFim.UF = value
		case xmlwalk.HasAnySuffix(p,
			"/infCte/ide/toma3/toma", "/infCte/ide/toma03/toma", "/infCte/ide/toma4/toma",
			"/infCte/ide/toma/toma", "/infCte/ide/tomaTerceiro/toma", "/infCte/toma/toma"):
			doc.TomadorIndicador = value

		// where the transport starts and ends: ide on the CT-e and CT-e OS,
		// the first det on the CT-e Simplificado, origem and destino on the
		// GTV-e
		case xmlwalk.HasAnySuffix(p, "/infCte/ide/cMunIni", "/infCte/det/cMunIni", "/infCte/origem/cMun"):
			setOnce(&doc.MunIni.Codigo, value)
		case xmlwalk.HasAnySuffix(p, "/infCte/ide/xMunIni", "/infCte/det/xMunIni", "/infCte/origem/xMun"):
			setOnce(&doc.MunIni.Nome, value)
		case strings.HasSuffix(p, "/infCte/origem/UF"):
			doc.MunIni.UF = value
		case xmlwalk.HasAnySuffix(p, "/infCte/ide/cMunFim", "/infCte/det/cMunFim", "/infCte/destino/cMun"):
			setOnce(&doc.MunFim.Codigo, value)
		case xmlwalk.HasAnySuffix(p, "/infCte/ide/xMunFim", "/infCte/det/xMunFim", "/infCte/destino/xMun"):
			setOnce(&doc.MunFim.Nome, value)
		case strings.HasSuffix(p, "/infCte/destino/UF"):
			doc.MunFim.UF = value

		case xmlwalk.HasAnySuffix(p, "/infCte/autXML/CNPJ", "/infCte/autXML/CPF"):
			doc.AutorizadosCNPJ = append(doc.AutorizadosCNPJ, value)

		// values: vPrest on the CT-e and CT-e OS, total on the CT-e
		// Simplificado. Only one ICMS group exists per document; vICMSOutraUF
		// is the ICMS due to another UF.
		case xmlwalk.HasAnySuffix(p, "/infCte/vPrest/vTPrest", "/infCte/total/vTPrest"):
			return dfe.ParseMoneyInto(&doc.TotalValue, "vTPrest", value)
		case xmlwalk.HasAnySuffix(p, "/infCte/vPrest/vRec", "/infCte/total/vTRec"):
			return dfe.ParseMoneyInto(&doc.ReceivableValue, "vRec", value)
		case isICMSGroupField(p):
			return addMoneyInto(&doc.ICMSValue, "vICMS", value)
		case strings.HasSuffix(p, "/infCte/imp/vTotTrib"):
			return dfe.ParseMoneyInto(&doc.TotTribValue, "vTotTrib", value)
		case xmlwalk.HasAnySuffix(p, "/infCTeNorm/infCarga/vCarga", "/infCte/infCarga/vCarga"):
			return dfe.ParseMoneyInto(&doc.CargaValue, "vCarga", value)
		case xmlwalk.HasAnySuffix(p, "/infCTeNorm/infCarga/proPred", "/infCte/infCarga/proPred"):
			doc.ProdutoPredominante = value
		case strings.HasSuffix(p, "/infCte/detGTV/infEspecie/vEspecie"):
			// The GTV-e carries cash, not goods: the carga value is the sum
			// of the species transported.
			return addMoneyInto(&doc.CargaValue, "vEspecie", value)

		// NF-e transported: infDoc on layouts 3.00 and 4.00, rem on 2.00,
		// det on the CT-e Simplificado
		case xmlwalk.HasAnySuffix(p, "/infCTeNorm/infDoc/infNFe/chave", "/infCte/rem/infNFe/chave", "/infCte/det/infNFe/chNFe"):
			nfeChaves = append(nfeChaves, value)

		// authorization protocol: protCTe, or the protocol of the other kinds
		case strings.HasSuffix(p, "/infProt/chCTe"):
			protChave = value
		case strings.HasSuffix(p, "/infProt/tpAmb"):
			protTpAmb = value
		case strings.HasSuffix(p, "/infProt/nProt"):
			doc.Protocolo = value
		case strings.HasSuffix(p, "/infProt/dhRecbto"):
			doc.AuthorizedAt = dfe.ParseDateTime("dhRecbto", value, &warnings)
		case strings.HasSuffix(p, "/infProt/cStat"):
			cStat = value
		case strings.HasSuffix(p, "/infProt/xMotivo"):
			xMotivo = value
		}
		return nil
	}
	if err := xmlwalk.Walk(data, onStart, onText); err != nil {
		return Document{}, err
	}

	tipo, ok := tipoByElement[wrapper]
	if !ok {
		return Document{}, fmt.Errorf("unsupported document element %q around infCte", wrapper)
	}
	doc.TipoDocumento = tipo

	key, err := dfe.KeyFromProtocol("CTe", protChave, infCteID, &warnings)
	if err != nil {
		return Document{}, err
	}
	doc.ChaveAcesso = key
	if doc.Modelo == "" {
		doc.Modelo = key.Modelo()
	}
	if doc.Modelo != tipo.Modelo() {
		warnings = append(warnings, fmt.Sprintf("modelo %s does not match document kind %s", doc.Modelo, tipo))
	}
	if doc.Emitente.UF == "" {
		doc.Emitente.UF = key.UF()
	}

	// A missing or invalid tpAmb is not an error here: the sync decides the
	// tpAmb to store against the environment it queried.
	switch {
	case doc.TpAmb != "":
	case protTpAmb != "":
		warnings = append(warnings, "missing ide/tpAmb; using protocol tpAmb")
		doc.TpAmb = protTpAmb
	default:
		warnings = append(warnings, "missing ide/tpAmb")
	}

	switch cStat {
	case "100", "150":
		doc.Situacao = SituacaoAutorizada
	case "110", "205", "301", "302", "303":
		doc.Situacao = SituacaoDenegada
	case "":
		return Document{}, errors.New("missing essential field: infProt/cStat")
	default:
		return Document{}, fmt.Errorf("unsupported protocol cStat %s (%s)", cStat, xMotivo)
	}

	resolveTomador(&doc, named, &warnings)
	doc.NFeChaves, doc.MaskedKeys = validNFeChaves(nfeChaves, &warnings)

	if doc.IssueDate.IsZero() {
		warnings = append(warnings, "missing dhEmi; competence is unknown")
	}
	doc.Competence = dfe.Competence(doc.IssueDate)
	doc.ParseWarnings = warnings
	return doc, nil
}

// setPartyField fills one field of party when p is that field under prefix,
// and reports whether it did.
func setPartyField(party *Party, p, prefix, ender, value string) bool {
	switch {
	case xmlwalk.HasAnySuffix(p, prefix+"/CNPJ", prefix+"/CPF"):
		party.CNPJ = value
	case strings.HasSuffix(p, prefix+"/xNome"):
		party.Name = value
	case strings.HasSuffix(p, prefix+"/IE"):
		party.IE = value
	case strings.HasSuffix(p, prefix+"/"+ender+"/UF"):
		party.UF = value
	default:
		return false
	}
	return true
}

// isICMSGroupField reports whether p is the vICMS (or vICMSOutraUF) of any
// ICMS group, as in /infCte/imp/ICMS/ICMS00/vICMS.
func isICMSGroupField(p string) bool {
	if !strings.HasSuffix(path.Dir(path.Dir(p)), "/infCte/imp/ICMS") {
		return false
	}
	name := path.Base(p)
	return name == "vICMS" || name == "vICMSOutraUF"
}

// setOnce sets dst to value unless dst is already set, so the first det of a
// CT-e Simplificado gives the start and end of the transport.
func setOnce(dst *string, value string) {
	if *dst == "" {
		*dst = value
	}
}

// resolveTomador copies the tomador into doc.Tomador. The CT-e points to
// one of its parties with toma3 (toma03 on layout 2.00) or names a third
// party in toma4; the GTV-e points to the remetente or destinatário with
// toma, or names a third party in tomaTerceiro; the CT-e OS and the CT-e
// Simplificado always name the tomador in infCte/toma. A tomador that cannot
// be resolved, such as a pointer to a party missing from the document,
// leaves Tomador empty with a warning.
func resolveTomador(doc *Document, named Party, warnings *[]string) {
	switch {
	case doc.TipoDocumento == TipoDocumentoCTeOS || doc.TipoDocumento == TipoDocumentoCTeSimplificado:
		doc.Tomador = named
	case doc.TomadorIndicador == "4":
		doc.Tomador = named
	case doc.TipoDocumento == TipoDocumentoGTVe:
		switch doc.TomadorIndicador {
		case "0":
			doc.Tomador = doc.Remetente
		case "1":
			doc.Tomador = doc.Destinatario
		}
	default:
		switch doc.TomadorIndicador {
		case "0":
			doc.Tomador = doc.Remetente
		case "1":
			doc.Tomador = doc.Expedidor
		case "2":
			doc.Tomador = doc.Recebedor
		case "3":
			doc.Tomador = doc.Destinatario
		}
	}
	if doc.Tomador.CNPJ == "" {
		*warnings = append(*warnings, fmt.Sprintf("tomador not resolved (toma %q)", doc.TomadorIndicador))
	}
}

// validNFeChaves keeps the NF-e keys that pass validation, without
// duplicates. Keys masked with 9s, which the Ambiente Nacional sends to
// autXML parties, are dropped with a single warning counting them, and
// masked reports that there were any; any other invalid key gets its own
// warning.
func validNFeChaves(raw []string, warnings *[]string) (chaves []string, masked bool) {
	maskedCount := 0
	for _, r := range raw {
		if strings.Trim(r, "9") == "" {
			maskedCount++
			continue
		}
		key, err := dfe.ParseAccessKey(r)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("invalid NF-e chave %s: %v", r, err))
			continue
		}
		if !slices.Contains(chaves, string(key)) {
			chaves = append(chaves, string(key))
		}
	}
	if maskedCount > 0 {
		*warnings = append(*warnings, fmt.Sprintf("%d NF-e chaves masked with 9s were dropped", maskedCount))
	}
	return chaves, maskedCount > 0
}
