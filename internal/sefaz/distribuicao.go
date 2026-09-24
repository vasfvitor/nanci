// Portions adapted from github.com/mschunke/gonfe (MIT License). See third_party/gonfe/LICENSE.

package sefaz

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

const (
	wsdlDistribuicao   = "http://www.portalfiscal.inf.br/nfe/wsdl/NFeDistribuicaoDFe"
	actionDistribuicao = wsdlDistribuicao + "/nfeDistDFeInteresse"
	versaoDistDFe      = "1.01"

	// maxNSU is the largest value that fits the 15-digit NSU field.
	maxNSU = 999_999_999_999_999
)

// distService names the parts of one distribution web service. The NF-e and
// CT-e services share the request and response layout and differ only here.
type distService struct {
	// name identifies the service in errors: "NF-e" or "CT-e".
	name      string
	namespace string
	wsdl      string
	action    string
	versao    string
	// wrapper is the operation element and dadosMsg the message element
	// around distDFeInt, both in the wsdl namespace.
	wrapper  string
	dadosMsg string
	endpoint func(Endpoints) string
}

var nfeDist = distService{
	name:      "NF-e",
	namespace: nfeNamespace,
	wsdl:      wsdlDistribuicao,
	action:    actionDistribuicao,
	versao:    versaoDistDFe,
	wrapper:   "nfeDistDFeInteresse",
	dadosMsg:  "nfeDadosMsg",
	endpoint:  func(e Endpoints) string { return e.Distribuicao },
}

// cStat values of NFeDistribuicaoDFe and CTeDistribuicaoDFe that are answers
// rather than errors.
const (
	// CStatNenhumDocumento: no document after the NSU sent; wait an hour
	// before asking again.
	CStatNenhumDocumento = 137
	// CStatDocumentoLocalizado: the answer carries docZips.
	CStatDocumentoLocalizado = 138
	// CStatConsumoIndevido: too many requests; the CNPJ is blocked for an hour.
	CStatConsumoIndevido = 656
)

// DistResult is a retDistDFeInt with cStat 137, 138 or 656. What each one
// means for the NSU cursor is up to the caller.
type DistResult struct {
	CStat   int
	XMotivo string
	// DhResp is zero when the response has no valid dhResp.
	DhResp time.Time
	UltNSU int64
	MaxNSU int64
	Docs   []DocZip
}

// DocZip is one document of the lote, still gzipped and base64 encoded.
type DocZip struct {
	NSU int64
	// Schema is the schema attribute as sent, with the .xsd suffix, for
	// example "resNFe_v1.01.xsd".
	Schema string
	// Content is the base64 of the gzipped XML.
	Content string
}

// DistNSU asks for the documents after ultNSU (distNSU), the normal way to
// walk the queue. cUFAutor is the IBGE code of the company's UF.
func (c *Client) DistNSU(ctx context.Context, cnpjValue string, cUFAutor int, ultNSU int64) (DistResult, error) {
	query, err := nsuQuery("distNSU", "ultNSU", ultNSU)
	if err != nil {
		return DistResult{}, err
	}
	return c.distribuicao(ctx, nfeDist, cnpjValue, cUFAutor, query)
}

// ConsNSU asks for the single document with the given NSU (consNSU), to fill
// a gap in the queue.
func (c *Client) ConsNSU(ctx context.Context, cnpjValue string, cUFAutor int, nsu int64) (DistResult, error) {
	query, err := nsuQuery("consNSU", "NSU", nsu)
	if err != nil {
		return DistResult{}, err
	}
	return c.distribuicao(ctx, nfeDist, cnpjValue, cUFAutor, query)
}

// ConsChNFe asks for the NF-e with the given access key (consChNFe).
func (c *Client) ConsChNFe(ctx context.Context, cnpjValue string, cUFAutor int, chave string) (DistResult, error) {
	key, err := dfe.ParseAccessKey(chave)
	if err != nil {
		return DistResult{}, err
	}
	return c.distribuicao(ctx, nfeDist, cnpjValue, cUFAutor, "<consChNFe><chNFe>"+key.String()+"</chNFe></consChNFe>")
}

func nsuQuery(group, field string, nsu int64) (string, error) {
	if nsu < 0 || nsu > maxNSU {
		return "", fmt.Errorf("NSU %d out of range", nsu)
	}
	return fmt.Sprintf("<%s><%s>%015d</%s></%s>", group, field, nsu, field, group), nil
}

func (c *Client) distribuicao(ctx context.Context, svc distService, cnpjValue string, cUFAutor int, query string) (DistResult, error) {
	url := svc.endpoint(c.endpoints)
	if url == "" {
		return DistResult{}, fmt.Errorf("%s distribution URL is not configured", svc.name)
	}
	request, err := buildDistDFeInt(svc, c.tpAmb, cUFAutor, cnpjValue, query)
	if err != nil {
		return DistResult{}, err
	}
	// The distribution services are the only ones that want the operation
	// element around the message element.
	body := `<` + svc.wrapper + ` xmlns="` + svc.wsdl + `"><` + svc.dadosMsg + ` xmlns="` + svc.wsdl + `">` +
		request + `</` + svc.dadosMsg + `></` + svc.wrapper + `>`

	respBody, err := c.post(ctx, url, svc.action, []byte(body), 0)
	if err != nil {
		return DistResult{}, err
	}
	return parseRetDistDFeInt(respBody)
}

// buildDistDFeInt writes the request without whitespace between tags:
// SEFAZ rejects formatting characters with cStat 588.
func buildDistDFeInt(svc distService, tpAmb string, cUFAutor int, cnpjValue, query string) (string, error) {
	if err := cnpj.Validate(cnpjValue); err != nil {
		return "", fmt.Errorf("invalid CNPJ: %w", err)
	}
	// IBGE UF codes go from 11 (RO) to 53 (DF).
	if cUFAutor < 11 || cUFAutor > 53 {
		return "", fmt.Errorf("invalid cUFAutor %d", cUFAutor)
	}
	return `<distDFeInt xmlns="` + svc.namespace + `" versao="` + svc.versao + `">` +
		"<tpAmb>" + tpAmb + "</tpAmb>" +
		"<cUFAutor>" + strconv.Itoa(cUFAutor) + "</cUFAutor>" +
		"<CNPJ>" + cnpj.Clean(cnpjValue) + "</CNPJ>" +
		query +
		"</distDFeInt>", nil
}

type retDistDFeInt struct {
	CStat   int    `xml:"cStat"`
	XMotivo string `xml:"xMotivo"`
	DhResp  string `xml:"dhResp"`
	UltNSU  string `xml:"ultNSU"`
	MaxNSU  string `xml:"maxNSU"`
	DocZips []struct {
		NSU     string `xml:"NSU,attr"`
		Schema  string `xml:"schema,attr"`
		Content string `xml:",chardata"`
	} `xml:"loteDistDFeInt>docZip"`
}

// parseRetDistDFeInt reads the retDistDFeInt element out of the SOAP
// response, whatever the prefixes around it.
func parseRetDistDFeInt(body []byte) (DistResult, error) {
	var ret retDistDFeInt
	err := decodeElement(body, "retDistDFeInt", &ret)
	if err != nil {
		return DistResult{}, err
	}

	switch ret.CStat {
	case CStatNenhumDocumento, CStatDocumentoLocalizado, CStatConsumoIndevido:
	default:
		return DistResult{}, &RejectionError{CStat: ret.CStat, XMotivo: strings.TrimSpace(ret.XMotivo)}
	}

	result := DistResult{
		CStat:   ret.CStat,
		XMotivo: strings.TrimSpace(ret.XMotivo),
	}
	if dhResp, err := time.Parse(time.RFC3339, strings.TrimSpace(ret.DhResp)); err == nil {
		result.DhResp = dhResp
	}
	if result.UltNSU, err = parseNSU(ret.UltNSU); err != nil {
		return DistResult{}, fmt.Errorf("parse ultNSU: %w", err)
	}
	if result.MaxNSU, err = parseNSU(ret.MaxNSU); err != nil {
		return DistResult{}, fmt.Errorf("parse maxNSU: %w", err)
	}
	for _, doc := range ret.DocZips {
		nsu, err := parseNSU(doc.NSU)
		if err != nil {
			return DistResult{}, fmt.Errorf("parse docZip NSU: %w", err)
		}
		result.Docs = append(result.Docs, DocZip{
			NSU:     nsu,
			Schema:  strings.TrimSpace(doc.Schema),
			Content: strings.TrimSpace(doc.Content),
		})
	}
	return result, nil
}

// parseNSU reads a zero-padded NSU; an empty value is 0.
func parseNSU(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	nsu, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if nsu < 0 {
		return 0, errors.New("negative NSU")
	}
	return nsu, nil
}
