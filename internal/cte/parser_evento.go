package cte

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/xmlwalk"
)

// detEventoPrefix is the path inside which each event type has its own
// group, such as evCancCTe or evCCeCTe.
const detEventoPrefix = "/eventoCTe/infEvento/detEvento/"

// correcao is one infCorrecao group of a carta de correção.
type correcao struct {
	grupo, campo, valor, item string
}

// ParseProcEventoCTe parses a procEventoCTe: the signed eventoCTe plus the
// retEventoCTe SEFAZ answered with. The event counts as registered only when
// retEventoCTe cStat is 134, 135 or 136; otherwise the registration fields
// stay empty and a warning is added.
func ParseProcEventoCTe(data []byte) (Event, error) {
	var ev Event
	var warnings []string
	var chave, tpEvento, nSeqEvento, retTpAmb string
	var cStat, xMotivo, xEvento, nProt, dhRegEvento string
	var correcoes []correcao

	onStart := func(p string, _ []xml.Attr) {
		if strings.Contains(p, detEventoPrefix) && strings.HasSuffix(p, "/infCorrecao") {
			correcoes = append(correcoes, correcao{})
		}
	}
	onText := func(p, value string) error {
		if group, field, ok := detEventoField(p); ok {
			setDetEventoField(&ev, correcoes, group, field, value)
			return nil
		}

		switch {
		// eventoCTe: what the author sent
		case strings.HasSuffix(p, "/eventoCTe/infEvento/cOrgao"):
			ev.COrgao = value
		case strings.HasSuffix(p, "/eventoCTe/infEvento/tpAmb"):
			ev.TpAmb = value
		case xmlwalk.HasAnySuffix(p, "/eventoCTe/infEvento/CNPJ", "/eventoCTe/infEvento/CPF"):
			ev.AutorCNPJ = value
		case strings.HasSuffix(p, "/eventoCTe/infEvento/chCTe"):
			chave = value
		case strings.HasSuffix(p, "/eventoCTe/infEvento/dhEvento"):
			ev.EventAt = parseDateTime("dhEvento", value, &warnings)
		case strings.HasSuffix(p, "/eventoCTe/infEvento/tpEvento"):
			tpEvento = value
		case strings.HasSuffix(p, "/eventoCTe/infEvento/nSeqEvento"):
			nSeqEvento = value

		// retEventoCTe: what SEFAZ answered
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/tpAmb"):
			retTpAmb = value
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/cStat"):
			cStat = value
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/xMotivo"):
			xMotivo = value
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/xEvento"):
			xEvento = value
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/nProt"):
			nProt = value
		case strings.HasSuffix(p, "/retEventoCTe/infEvento/dhRegEvento"):
			dhRegEvento = value
		}
		return nil
	}
	if err := xmlwalk.Walk(data, onStart, onText); err != nil {
		return Event{}, err
	}

	if err := setEventIdentity(&ev, chave, tpEvento, nSeqEvento); err != nil {
		return Event{}, err
	}
	switch {
	case ev.TpAmb != "":
	case retTpAmb != "":
		warnings = append(warnings, "missing eventoCTe tpAmb; using retEventoCTe tpAmb")
		ev.TpAmb = retTpAmb
	default:
		return Event{}, errors.New("missing essential field: infEvento/tpAmb")
	}
	if ev.TpAmb != "1" && ev.TpAmb != "2" {
		return Event{}, fmt.Errorf("invalid tpAmb %q", ev.TpAmb)
	}
	if ev.Description == "" {
		ev.Description = xEvento
	}
	ev.Correcao = formatCorrecoes(correcoes)

	ev.CStat = cStat
	ev.XMotivo = xMotivo
	switch cStat {
	case "134", "135", "136":
		ev.Registered = true
		ev.Protocolo = nProt
		if dhRegEvento != "" {
			ev.RegisteredAt = parseDateTime("dhRegEvento", dhRegEvento, &warnings)
		}
	case "":
		warnings = append(warnings, "missing retEventoCTe cStat; event not registered")
	default:
		warnings = append(warnings, fmt.Sprintf("retEventoCTe cStat %s (%s); event not registered", cStat, xMotivo))
	}

	ev.ParseWarnings = warnings
	return ev, nil
}

// detEventoField splits a path inside detEvento into the event group's
// child ("descEvento", or "infCorrecao" for its children) and the field
// name. Deeper paths, such as the NF-e list of a comprovante de entrega, are
// not read.
func detEventoField(p string) (group, field string, ok bool) {
	i := strings.Index(p, detEventoPrefix)
	if i < 0 {
		return "", "", false
	}
	parts := strings.Split(p[i+len(detEventoPrefix):], "/")
	switch len(parts) {
	case 2: // evCancCTe/xJust
		return "", parts[1], true
	case 3: // evCCeCTe/infCorrecao/grupoAlterado
		return parts[1], parts[2], true
	default:
		return "", "", false
	}
}

func setDetEventoField(ev *Event, correcoes []correcao, group, field, value string) {
	if group == "infCorrecao" {
		if len(correcoes) == 0 {
			return
		}
		c := &correcoes[len(correcoes)-1]
		switch field {
		case "grupoAlterado":
			c.grupo = value
		case "campoAlterado":
			c.campo = value
		case "valorAlterado":
			c.valor = value
		case "nroItemAlterado":
			c.item = value
		}
		return
	}
	if group != "" {
		return
	}
	switch field {
	case "descEvento":
		ev.Description = value
	case "xJust":
		ev.Justificativa = value
	case "xObs":
		ev.Observacao = value
	case "xCondUso":
		ev.CondicaoUso = value
	}
}

// formatCorrecoes writes the infCorrecao groups as
// "grupo.campo=valor; grupo.campo[item]=valor".
func formatCorrecoes(correcoes []correcao) string {
	parts := make([]string, 0, len(correcoes))
	for _, c := range correcoes {
		field := c.grupo + "." + c.campo
		if c.item != "" {
			field += "[" + c.item + "]"
		}
		parts = append(parts, field+"="+c.valor)
	}
	return strings.Join(parts, "; ")
}

// setEventIdentity validates and sets the fields that identify an event:
// chave, tpEvento and nSeqEvento.
func setEventIdentity(ev *Event, chave, tpEvento, nSeqEvento string) error {
	if chave == "" {
		return errors.New("missing essential field: chCTe")
	}
	key, err := dfe.ParseAccessKey(chave)
	if err != nil {
		return fmt.Errorf("chCTe: %w", err)
	}
	if tpEvento == "" {
		return errors.New("missing essential field: tpEvento")
	}
	if nSeqEvento == "" {
		return errors.New("missing essential field: nSeqEvento")
	}
	seq, err := strconv.Atoi(nSeqEvento)
	if err != nil || seq < 1 {
		return fmt.Errorf("invalid nSeqEvento %q", nSeqEvento)
	}

	ev.ChaveAcesso = key
	ev.TpEvento = tpEvento
	ev.Type = EventTypeFromTpEvento(tpEvento)
	ev.NSeqEvento = seq
	return nil
}
