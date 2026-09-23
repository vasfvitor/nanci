// Portions adapted from github.com/mschunke/gonfe (MIT License). See third_party/gonfe/LICENSE.

package sefaz

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
)

const (
	wsdlRecepcaoEvento   = "http://www.portalfiscal.inf.br/nfe/wsdl/NFeRecepcaoEvento4"
	actionRecepcaoEvento = wsdlRecepcaoEvento + "/nfeRecepcaoEvento"
	versaoEvento         = "1.00"

	// COrgaoAN is the cOrgao of the Ambiente Nacional, which receives every
	// manifestação do destinatário.
	COrgaoAN = "91"
	// MaxEventosPorLote is the most eventos one envEvento may carry.
	MaxEventosPorLote = 20
	// maxNSeqEvento keeps the sequence within the two digits of the Id.
	maxNSeqEvento = 20
	// eventTransportRetries resends a lote once when no HTTP response
	// arrived. A lote that did reach SEFAZ comes back as 573 (duplicidade)
	// on the second try, which callers treat as already registered.
	eventTransportRetries = 1
	// dhEventoLayout has no fractional seconds and never uses Z.
	dhEventoLayout = "2006-01-02T15:04:05-07:00"
)

// cStat values of NFeRecepcaoEvento4. CStatLoteProcessado is the lote
// level; the others belong to one evento.
const (
	CStatLoteProcessado = 128

	CStatEventoVinculado             = 135 // registered and linked to the NF-e
	CStatEventoNaoVinculado          = 136 // registered, not linked to the NF-e
	CStatChaveInexistente            = 494
	CStatDuplicidadeEvento           = 573 // same evento already registered
	CStatAutorDiverge                = 575 // author is not the destinatário
	CStatDhEventoAntesEmissao        = 577
	CStatDhEventoPosterior           = 578 // dhEvento after SEFAZ time: fix the clock
	CStatDhEventoAntesAutorizacao    = 579
	CStatForaDoPrazo                 = 596
	CStatCNPJBaseDiverge             = 631 // CNPJ-Base differs from the certificate
	CStatCienciaNFeCancelada         = 650
	CStatDesconhecimentoNFeCancelada = 651
	CStatCienciaAposManifestacao     = 655 // ciência after a conclusive manifestação
)

// IsRegistered reports whether cStat means SEFAZ registered the evento now.
func IsRegistered(cStat int) bool {
	return cStat == CStatEventoVinculado || cStat == CStatEventoNaoVinculado
}

// IsAlreadyDone reports whether cStat means the same evento was registered
// before (573). A ciência sent after a conclusive manifestação (655) is not
// already done: SEFAZ did not register it, so it is a rejection.
func IsAlreadyDone(cStat int) bool {
	return cStat == CStatDuplicidadeEvento
}

// Evento is one manifestação do destinatário to sign and send.
type Evento struct {
	ChaveAcesso string
	// CNPJ is the destinatário, which must share its CNPJ-Base with the
	// signing certificate.
	CNPJ string
	// TpEvento and DescEvento come from nfe.ManifestationType.
	TpEvento   string
	DescEvento string
	// NSeqEvento is 1 for the first evento of a type on a key.
	NSeqEvento int
	// DhEvento is sent in America/Sao_Paulo time.
	DhEvento time.Time
	// XJust is the justificativa of Operação não Realizada (210240), already
	// cleaned by nfe.ValidateJustificativa; it must be empty otherwise.
	XJust string
}

// EventoResult is the answer SEFAZ gave to one evento of a lote.
type EventoResult struct {
	ChaveAcesso string
	TpEvento    string
	NSeqEvento  int
	// CStat is 0 when the answer had no retEvento for this evento.
	CStat       int
	XMotivo     string
	Protocolo   string
	DhRegEvento *time.Time
	// SignedEvento is the <evento> exactly as sent.
	SignedEvento []byte
	// RetEvento is the <retEvento> exactly as received; nil when missing.
	RetEvento []byte
}

// LoteResult is a processed lote (cStat 128) with one result per evento, in
// the order the eventos were given.
type LoteResult struct {
	IDLote  string
	CStat   int
	XMotivo string
	Eventos []EventoResult
}

// NewIDLote returns a 15-digit lote id from now: yyMMddHHmmss followed by the
// milliseconds.
func NewIDLote(now time.Time) string {
	return fmt.Sprintf("%s%03d", now.Format("060102150405"), now.Nanosecond()/int(time.Millisecond))
}

// EnviarEventos signs each evento, sends them in one envEvento and matches
// every retEvento back by chNFe and tpEvento. A lote cStat other than 128 is
// a *RejectionError.
func (c *Client) EnviarEventos(ctx context.Context, signer *Signer, idLote string, eventos []Evento) (LoteResult, error) {
	if signer == nil {
		return LoteResult{}, errors.New("signer is required")
	}
	if len(eventos) == 0 || len(eventos) > MaxEventosPorLote {
		return LoteResult{}, fmt.Errorf("a lote takes 1 to %d eventos, got %d", MaxEventosPorLote, len(eventos))
	}
	if !isDigits(idLote) || len(idLote) > 15 {
		return LoteResult{}, fmt.Errorf("invalid idLote %q: must have 1 to 15 digits", idLote)
	}

	results := make([]EventoResult, len(eventos))
	seen := make(map[string]bool, len(eventos))
	for i, e := range eventos {
		key := eventoMatchKey(e.ChaveAcesso, e.TpEvento)
		if seen[key] {
			return LoteResult{}, fmt.Errorf("evento %d repeats chave and tpEvento of an earlier evento in the lote", i+1)
		}
		seen[key] = true

		signed, err := signer.SignEvento(e, c.tpAmb)
		if err != nil {
			return LoteResult{}, fmt.Errorf("evento %d: %w", i+1, err)
		}
		results[i] = EventoResult{
			ChaveAcesso:  strings.ToUpper(strings.TrimSpace(e.ChaveAcesso)),
			TpEvento:     e.TpEvento,
			NSeqEvento:   e.NSeqEvento,
			SignedEvento: signed,
		}
	}

	var envEvento bytes.Buffer
	envEvento.WriteString(`<envEvento xmlns="` + nfeNamespace + `" versao="` + versaoEvento + `">`)
	envEvento.WriteString("<idLote>" + idLote + "</idLote>")
	for _, r := range results {
		envEvento.Write(r.SignedEvento)
	}
	envEvento.WriteString("</envEvento>")
	// NFeRecepcaoEvento4 takes nfeDadosMsg directly, without an operation
	// element around it.
	body := `<nfeDadosMsg xmlns="` + wsdlRecepcaoEvento + `">` + envEvento.String() + `</nfeDadosMsg>`

	respBody, err := c.post(ctx, c.endpoints.RecepcaoEvento, actionRecepcaoEvento, []byte(body), eventTransportRetries)
	if err != nil {
		return LoteResult{}, err
	}
	return parseRetEnvEvento(respBody, results)
}

type retEnvEvento struct {
	IDLote  string `xml:"idLote"`
	CStat   int    `xml:"cStat"`
	XMotivo string `xml:"xMotivo"`
}

type retEvento struct {
	CStat       int    `xml:"infEvento>cStat"`
	XMotivo     string `xml:"infEvento>xMotivo"`
	ChNFe       string `xml:"infEvento>chNFe"`
	TpEvento    string `xml:"infEvento>tpEvento"`
	DhRegEvento string `xml:"infEvento>dhRegEvento"`
	NProt       string `xml:"infEvento>nProt"`
}

// parseRetEnvEvento fills results with the retEvento answers in body.
func parseRetEnvEvento(body []byte, results []EventoResult) (LoteResult, error) {
	var lote retEnvEvento
	if err := decodeElement(body, "retEnvEvento", &lote); err != nil {
		return LoteResult{}, err
	}
	if lote.CStat != CStatLoteProcessado {
		return LoteResult{}, &RejectionError{CStat: lote.CStat, XMotivo: strings.TrimSpace(lote.XMotivo)}
	}

	rawEventos, err := findElements(body, "retEvento")
	if err != nil {
		return LoteResult{}, err
	}
	byKey := make(map[string]int, len(results))
	for i, r := range results {
		byKey[eventoMatchKey(r.ChaveAcesso, r.TpEvento)] = i
	}
	for _, rawEvento := range rawEventos {
		var ret retEvento
		if err := xml.Unmarshal(rawEvento, &ret); err != nil {
			return LoteResult{}, fmt.Errorf("parse retEvento: %w", err)
		}
		i, ok := byKey[eventoMatchKey(ret.ChNFe, ret.TpEvento)]
		if !ok {
			continue
		}
		r := &results[i]
		r.CStat = ret.CStat
		r.XMotivo = strings.TrimSpace(ret.XMotivo)
		r.Protocolo = strings.TrimSpace(ret.NProt)
		if dh, err := time.Parse(time.RFC3339, strings.TrimSpace(ret.DhRegEvento)); err == nil {
			r.DhRegEvento = &dh
		}
		r.RetEvento = rawEvento
	}
	for i := range results {
		if results[i].RetEvento == nil {
			results[i].XMotivo = "SEFAZ answer has no retEvento for this evento"
		}
	}

	return LoteResult{
		IDLote:  strings.TrimSpace(lote.IDLote),
		CStat:   lote.CStat,
		XMotivo: strings.TrimSpace(lote.XMotivo),
		Eventos: results,
	}, nil
}

func eventoMatchKey(chave, tpEvento string) string {
	return strings.ToUpper(strings.TrimSpace(chave)) + "/" + strings.TrimSpace(tpEvento)
}

// ProcEventoNFe joins a signed evento and its retEvento into the
// procEventoNFe document kept for storage and export. Both parts inherit the
// NF-e namespace from the wrapper.
func ProcEventoNFe(signedEvento, retEvento []byte) []byte {
	var b bytes.Buffer
	b.WriteString(`<procEventoNFe xmlns="` + nfeNamespace + `" versao="` + versaoEvento + `">`)
	b.Write(signedEvento)
	b.Write(retEvento)
	b.WriteString(`</procEventoNFe>`)
	return b.Bytes()
}

// buildInfEvento validates e and writes the infEvento element already in
// canonical form (C14N 1.0): no whitespace between tags, no empty-element
// tags, Id as the only attribute. It returns the Id and the element without
// a namespace declaration, as embedded in the evento.
func buildInfEvento(e Evento, tpAmb string) (id, infEvento string, err error) {
	key, err := nfe.ParseAccessKey(e.ChaveAcesso)
	if err != nil {
		return "", "", err
	}
	if err := cnpj.Validate(e.CNPJ); err != nil {
		return "", "", fmt.Errorf("invalid CNPJ: %w", err)
	}
	if len(e.TpEvento) != 6 || !isDigits(e.TpEvento) {
		return "", "", fmt.Errorf("invalid tpEvento %q", e.TpEvento)
	}
	if e.NSeqEvento < 1 || e.NSeqEvento > maxNSeqEvento {
		return "", "", fmt.Errorf("nSeqEvento must be 1 to %d, got %d", maxNSeqEvento, e.NSeqEvento)
	}
	if e.DhEvento.IsZero() {
		return "", "", errors.New("dhEvento is required")
	}
	if e.DescEvento == "" {
		return "", "", errors.New("descEvento is required")
	}
	if err := checkEventoText("descEvento", e.DescEvento); err != nil {
		return "", "", err
	}
	if err := checkEventoText("xJust", e.XJust); err != nil {
		return "", "", err
	}
	needsXJust := e.TpEvento == nfe.TpEventoNaoRealizada
	if needsXJust && e.XJust == "" {
		return "", "", fmt.Errorf("xJust is required for tpEvento %s", e.TpEvento)
	}
	if !needsXJust && e.XJust != "" {
		return "", "", fmt.Errorf("xJust is only accepted for tpEvento %s", nfe.TpEventoNaoRealizada)
	}

	id = fmt.Sprintf("ID%s%s%02d", e.TpEvento, key, e.NSeqEvento)

	var b strings.Builder
	b.WriteString(`<infEvento Id="` + id + `">`)
	b.WriteString("<cOrgao>" + COrgaoAN + "</cOrgao>")
	b.WriteString("<tpAmb>" + tpAmb + "</tpAmb>")
	b.WriteString("<CNPJ>" + cnpj.Clean(e.CNPJ) + "</CNPJ>")
	b.WriteString("<chNFe>" + key.String() + "</chNFe>")
	b.WriteString("<dhEvento>" + formatDhEvento(e.DhEvento) + "</dhEvento>")
	b.WriteString("<tpEvento>" + e.TpEvento + "</tpEvento>")
	b.WriteString("<nSeqEvento>" + strconv.Itoa(e.NSeqEvento) + "</nSeqEvento>")
	b.WriteString("<verEvento>" + versaoEvento + "</verEvento>")
	b.WriteString(`<detEvento versao="` + versaoEvento + `">`)
	b.WriteString("<descEvento>" + escapeText(e.DescEvento) + "</descEvento>")
	if e.XJust != "" {
		b.WriteString("<xJust>" + escapeText(e.XJust) + "</xJust>")
	}
	b.WriteString("</detEvento></infEvento>")
	return id, b.String(), nil
}

// checkEventoText rejects text SEFAZ would refuse (cStat 588 for
// surrounding whitespace) or that would not survive canonicalization
// unchanged.
func checkEventoText(field, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s is not valid UTF-8", field)
	}
	if strings.TrimSpace(s) != s {
		return fmt.Errorf("%s has leading or trailing whitespace", field)
	}
	if strings.ContainsFunc(s, unicode.IsControl) {
		return fmt.Errorf("%s has control characters", field)
	}
	return nil
}

// escapeText escapes element text the way C14N 1.0 writes it. Quotes stay
// literal, unlike xml.EscapeText, so the bytes sent are the bytes SEFAZ
// canonicalizes and hashes.
func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#xD;").Replace(s)
}

// saoPaulo is the time zone of dhEvento. Brazil has had no daylight saving
// since 2019, so the fixed offset is right when the zone database is
// missing.
var saoPaulo = loadSaoPaulo()

func loadSaoPaulo() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("-03", -3*60*60)
	}
	return loc
}

func formatDhEvento(t time.Time) string {
	return t.In(saoPaulo).Format(dhEventoLayout)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
