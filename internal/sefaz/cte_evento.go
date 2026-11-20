package sefaz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// CTeRecepcaoEventoV4 (MOC CT-e 4.00, section 5). The WSDL namespace and the
// action follow the pattern of the other CT-e services; the official WSDL
// answers only with a client certificate and was not read (see
// docs/pesquisa/2026-11-19-cte-desacordo.md, "Não encontrado").
const (
	wsdlRecepcaoEventoCTe   = "http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4"
	actionRecepcaoEventoCTe = wsdlRecepcaoEventoCTe + "/cteRecepcaoEvento"
	versaoEventoCTe         = "4.00"

	// TpEventoPrestacaoDesacordo is the prestação do serviço em desacordo,
	// sent by the tomador.
	TpEventoPrestacaoDesacordo = "610110"
	// descEventoDesacordo is the form without accents the schema accepts
	// next to "Prestação do Serviço em Desacordo".
	descEventoDesacordo = "Prestacao do Servico em Desacordo"
	// maxNSeqEventoCTe is the highest nSeqEvento of the event (cStat 636).
	maxNSeqEventoCTe = 20
	// XObsDesacordoMin and XObsDesacordoMax bound the observation, in
	// characters (evPrestDesacordo_v4.00.xsd).
	XObsDesacordoMin = 15
	XObsDesacordoMax = 255
)

// cStat values of CTeRecepcaoEventoV4 for one event (MOC CT-e 4.00, 5.1.2,
// 5.1.5 and 6.5.1). A cStat can have 4 digits since NT 2025.001.
const (
	CStatEventoCTeCTeNaoAutorizado = 134 // registered; the CT-e is not authorized
	CStatEventoCTeVinculado        = 135 // registered and linked to the CT-e
	CStatEventoCTeSemCTe           = 136 // registered; the CT-e is not in the SEFAZ base
	CStatCTeDenegado               = 205
	CStatCTeInexistente            = 217
	CStatCTeCancelado              = 218
	CStatDuplicidadeEventoCTe      = 631 // same event already registered
	CStatCTeJaSubstituido          = 664
	CStatAutorNaoTomador           = 755
	CStatDesacordoForaDoPrazo      = 787 // more than 45 days after the authorization
)

// IsRegisteredCTe reports whether cStat means SEFAZ registered the CT-e
// event now.
func IsRegisteredCTe(cStat int) bool {
	return cStat == CStatEventoCTeVinculado || cStat == CStatEventoCTeCTeNaoAutorizado || cStat == CStatEventoCTeSemCTe
}

// IsAlreadyDoneCTe reports whether cStat means the same CT-e event was
// registered before (631).
func IsAlreadyDoneCTe(cStat int) bool {
	return cStat == CStatDuplicidadeEventoCTe
}

// EventoDesacordo is one prestação do serviço em desacordo to sign and send.
type EventoDesacordo struct {
	// ChaveAcesso is a CT-e (model 57) or CT-e OS (67).
	ChaveAcesso string
	// CNPJ is the tomador, which must share its CNPJ-Base with the signing
	// certificate.
	CNPJ string
	// NSeqEvento is 1 for the first desacordo of the CT-e.
	NSeqEvento int
	// DhEvento is sent in America/Sao_Paulo time.
	DhEvento time.Time
	// XObs is the observation, XObsDesacordoMin to XObsDesacordoMax
	// characters (CheckXObsDesacordo).
	XObs string
}

// EventoCTeResult is the answer SEFAZ gave to one CT-e event.
type EventoCTeResult struct {
	CStat       int
	XMotivo     string
	Protocolo   string
	DhRegEvento *time.Time
	// SignedEvento is the <eventoCTe> exactly as sent.
	SignedEvento []byte
	// RetEvento is the <retEventoCTe> exactly as received.
	RetEvento []byte
}

// CheckXObsDesacordo validates the observation of a desacordo the way the
// schema does: 15 to 255 characters, all within Latin-1 (U+0020 to U+00FF,
// so no curly quotes or emoji), not starting or ending with a space.
func CheckXObsDesacordo(xObs string) error {
	if !utf8.ValidString(xObs) {
		return errors.New("a observação não é UTF-8 válido")
	}
	n := utf8.RuneCountInString(xObs)
	if n < XObsDesacordoMin || n > XObsDesacordoMax {
		return fmt.Errorf("a observação deve ter de %d a %d caracteres, tem %d", XObsDesacordoMin, XObsDesacordoMax, n)
	}
	if strings.TrimSpace(xObs) != xObs {
		return errors.New("a observação não pode começar nem terminar com espaço")
	}
	for _, r := range xObs {
		if r < 0x20 || r > 0xFF || (r >= 0x7F && r < 0xA0) {
			return fmt.Errorf("a observação tem um caractere que a SEFAZ não aceita: %q", r)
		}
	}
	return nil
}

// SignEventoDesacordo returns the standalone signed eventoCTe of e.
func (s *Signer) SignEventoDesacordo(e EventoDesacordo, tpAmb string) ([]byte, error) {
	id, infEvento, err := buildInfEventoDesacordo(e, tpAmb)
	if err != nil {
		return nil, err
	}
	return s.signInfEvento("eventoCTe", cteNamespace, versaoEventoCTe, id, infEvento)
}

// EnviarEventoDesacordo signs e and sends it alone to the autorizadora of the
// chave's UF: CT-e events have no lote. Every cStat SEFAZ answers comes back
// in the result; only a missing or malformed answer is an error.
func (c *Client) EnviarEventoDesacordo(ctx context.Context, signer *Signer, e EventoDesacordo) (EventoCTeResult, error) {
	if signer == nil {
		return EventoCTeResult{}, errors.New("signer is required")
	}
	key, err := dfe.ParseAccessKey(e.ChaveAcesso)
	if err != nil {
		return EventoCTeResult{}, err
	}
	url := c.endpoints.RecepcaoEventoCTe
	if url == "" {
		if url, err = CTeRecepcaoEventoURL(key.UFCode(), c.tpAmb); err != nil {
			return EventoCTeResult{}, err
		}
	}

	signed, err := signer.SignEventoDesacordo(e, c.tpAmb)
	if err != nil {
		return EventoCTeResult{}, err
	}
	// Like NFeRecepcaoEvento4, the service takes cteDadosMsg directly, with
	// the eventoCTe uncompressed.
	body := `<cteDadosMsg xmlns="` + wsdlRecepcaoEventoCTe + `">` + string(signed) + `</cteDadosMsg>`

	// A resend after a lost answer comes back as 631, which callers treat as
	// already registered.
	respBody, err := c.post(ctx, url, actionRecepcaoEventoCTe, []byte(body), eventTransportRetries)
	if err != nil {
		return EventoCTeResult{}, err
	}
	result, err := parseRetEventoCTe(respBody, key)
	if err != nil {
		return EventoCTeResult{}, err
	}
	result.SignedEvento = signed
	return result, nil
}

type retEventoCTe struct {
	CStat       int    `xml:"infEvento>cStat"`
	XMotivo     string `xml:"infEvento>xMotivo"`
	ChCTe       string `xml:"infEvento>chCTe"`
	DhRegEvento string `xml:"infEvento>dhRegEvento"`
	NProt       string `xml:"infEvento>nProt"`
}

// parseRetEventoCTe reads the retEventoCTe in body. An answer about another
// chave is an error.
func parseRetEventoCTe(body []byte, key dfe.AccessKey) (EventoCTeResult, error) {
	raws, err := findElements(body, "retEventoCTe")
	if err != nil {
		return EventoCTeResult{}, err
	}
	if len(raws) != 1 {
		return EventoCTeResult{}, fmt.Errorf("SEFAZ response has %d retEventoCTe elements, want 1", len(raws))
	}
	var ret retEventoCTe
	if err := decodeElement(raws[0], "retEventoCTe", &ret); err != nil {
		return EventoCTeResult{}, err
	}
	if ch := strings.TrimSpace(ret.ChCTe); ch != "" && !strings.EqualFold(ch, key.String()) {
		return EventoCTeResult{}, fmt.Errorf("SEFAZ answered for chave %s, not %s", ch, key)
	}
	result := EventoCTeResult{
		CStat:     ret.CStat,
		XMotivo:   strings.TrimSpace(ret.XMotivo),
		Protocolo: strings.TrimSpace(ret.NProt),
		RetEvento: raws[0],
	}
	if dh, err := time.Parse(time.RFC3339, strings.TrimSpace(ret.DhRegEvento)); err == nil {
		result.DhRegEvento = &dh
	}
	return result, nil
}

// ProcEventoCTe joins a signed eventoCTe and its retEventoCTe into the
// procEventoCTe document kept for storage and export.
func ProcEventoCTe(signedEvento, retEvento []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<procEventoCTe xmlns="` + cteNamespace + `" versao="` + versaoEventoCTe + `">`)
	b.Write(signedEvento)
	b.Write(retEvento)
	b.WriteString(`</procEventoCTe>`)
	return b.Bytes()
}

// buildInfEventoDesacordo validates e and writes its infEvento already in
// canonical form, like buildInfEvento. The CT-e infEvento has no verEvento,
// takes cOrgao from the chave's UF, and pads nSeqEvento to 3 digits in the
// Id (MOC CT-e 4.00, 5.1.1).
func buildInfEventoDesacordo(e EventoDesacordo, tpAmb string) (id, infEvento string, err error) {
	key, err := dfe.ParseAccessKey(e.ChaveAcesso)
	if err != nil {
		return "", "", err
	}
	if m := key.Modelo(); m != "57" && m != "67" {
		return "", "", fmt.Errorf("a prestação em desacordo vale para o CT-e (57) e o CT-e OS (67), não para o modelo %s", m)
	}
	if err := cnpj.Validate(e.CNPJ); err != nil {
		return "", "", fmt.Errorf("invalid CNPJ: %w", err)
	}
	if tpAmb != TpAmbProducao && tpAmb != TpAmbHomologacao {
		return "", "", fmt.Errorf("invalid tpAmb %q", tpAmb)
	}
	if e.NSeqEvento < 1 || e.NSeqEvento > maxNSeqEventoCTe {
		return "", "", fmt.Errorf("nSeqEvento must be 1 to %d, got %d", maxNSeqEventoCTe, e.NSeqEvento)
	}
	if e.DhEvento.IsZero() {
		return "", "", errors.New("dhEvento is required")
	}
	if err := CheckXObsDesacordo(e.XObs); err != nil {
		return "", "", err
	}

	id = fmt.Sprintf("ID%s%s%03d", TpEventoPrestacaoDesacordo, key, e.NSeqEvento)

	var b strings.Builder
	b.WriteString(`<infEvento Id="` + id + `">`)
	b.WriteString("<cOrgao>" + key.UFCode() + "</cOrgao>")
	b.WriteString("<tpAmb>" + tpAmb + "</tpAmb>")
	b.WriteString("<CNPJ>" + cnpj.Clean(e.CNPJ) + "</CNPJ>")
	b.WriteString("<chCTe>" + key.String() + "</chCTe>")
	b.WriteString("<dhEvento>" + formatDhEvento(e.DhEvento) + "</dhEvento>")
	b.WriteString("<tpEvento>" + TpEventoPrestacaoDesacordo + "</tpEvento>")
	b.WriteString("<nSeqEvento>" + strconv.Itoa(e.NSeqEvento) + "</nSeqEvento>")
	b.WriteString(`<detEvento versaoEvento="` + versaoEventoCTe + `">`)
	b.WriteString("<evPrestDesacordo>")
	b.WriteString("<descEvento>" + descEventoDesacordo + "</descEvento>")
	b.WriteString("<indDesacordoOper>1</indDesacordoOper>")
	b.WriteString("<xObs>" + escapeText(e.XObs) + "</xObs>")
	b.WriteString("</evPrestDesacordo></detEvento></infEvento>")
	return id, b.String(), nil
}
