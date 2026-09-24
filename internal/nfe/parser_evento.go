package nfe

import (
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/xmlwalk"
)

// ParseResEvento parses a resEvento (schema resEvento_v1.01), the summary of
// an event distributed by SEFAZ. A distributed summary is always of a
// registered event.
func ParseResEvento(data []byte) (Event, error) {
	ev := Event{Completeness: CompletenessResumo, Registered: true}
	var warnings []string
	var chave, tpEvento, nSeqEvento string

	onText := func(path, value string) error {
		switch {
		case strings.HasSuffix(path, "/resEvento/chNFe"):
			chave = value
		case xmlwalk.HasAnySuffix(path, "/resEvento/CNPJ", "/resEvento/CPF"):
			ev.AutorCNPJ = value
		case strings.HasSuffix(path, "/resEvento/dhEvento"):
			ev.EventAt = dfe.ParseDateTime("dhEvento", value, &warnings)
		case strings.HasSuffix(path, "/resEvento/tpEvento"):
			tpEvento = value
		case strings.HasSuffix(path, "/resEvento/nSeqEvento"):
			nSeqEvento = value
		case strings.HasSuffix(path, "/resEvento/xEvento"):
			ev.Description = value
		case strings.HasSuffix(path, "/resEvento/dhRecbto"):
			ev.RegisteredAt = dfe.ParseDateTime("dhRecbto", value, &warnings)
		case strings.HasSuffix(path, "/resEvento/nProt"):
			ev.Protocolo = value
		}
		return nil
	}
	if err := xmlwalk.Walk(data, nil, onText); err != nil {
		return Event{}, err
	}

	key, seq, err := dfe.ParseEventIdentity("NFe", chave, tpEvento, nSeqEvento)
	if err != nil {
		return Event{}, err
	}
	ev.ChaveAcesso, ev.TpEvento, ev.NSeqEvento = key, tpEvento, seq
	ev.Type = EventTypeFromTpEvento(tpEvento)
	ev.ParseWarnings = warnings
	return ev, nil
}

// ParseProcEventoNFe parses a procEventoNFe (schema procEventoNFe_v1.00): the
// signed evento plus the retEvento SEFAZ answered with. The event counts as
// registered only when retEvento cStat is 135, 136 or 155; otherwise the
// registration fields stay empty and a warning is added.
func ParseProcEventoNFe(data []byte) (Event, error) {
	ev := Event{Completeness: CompletenessCompleta}
	var warnings []string
	var chave, tpEvento, nSeqEvento string
	var cStat, xMotivo, xEvento, nProt, dhRegEvento string

	onText := func(path, value string) error {
		switch {
		// evento: what the author sent
		case strings.HasSuffix(path, "/evento/infEvento/chNFe"):
			chave = value
		case xmlwalk.HasAnySuffix(path, "/evento/infEvento/CNPJ", "/evento/infEvento/CPF"):
			ev.AutorCNPJ = value
		case strings.HasSuffix(path, "/evento/infEvento/dhEvento"):
			ev.EventAt = dfe.ParseDateTime("dhEvento", value, &warnings)
		case strings.HasSuffix(path, "/evento/infEvento/tpAmb"):
			ev.TpAmb = value
		case strings.HasSuffix(path, "/evento/infEvento/tpEvento"):
			tpEvento = value
		case strings.HasSuffix(path, "/evento/infEvento/nSeqEvento"):
			nSeqEvento = value
		case strings.HasSuffix(path, "/evento/infEvento/detEvento/descEvento"):
			ev.Description = value
		case strings.HasSuffix(path, "/evento/infEvento/detEvento/xJust"):
			ev.Justificativa = value
		case strings.HasSuffix(path, "/evento/infEvento/detEvento/xCorrecao"):
			ev.Correcao = value

		// retEvento: what SEFAZ answered
		case strings.HasSuffix(path, "/retEvento/infEvento/cStat"):
			cStat = value
		case strings.HasSuffix(path, "/retEvento/infEvento/xMotivo"):
			xMotivo = value
		case strings.HasSuffix(path, "/retEvento/infEvento/xEvento"):
			xEvento = value
		case strings.HasSuffix(path, "/retEvento/infEvento/nProt"):
			nProt = value
		case strings.HasSuffix(path, "/retEvento/infEvento/dhRegEvento"):
			dhRegEvento = value
		}
		return nil
	}
	if err := xmlwalk.Walk(data, nil, onText); err != nil {
		return Event{}, err
	}

	key, seq, err := dfe.ParseEventIdentity("NFe", chave, tpEvento, nSeqEvento)
	if err != nil {
		return Event{}, err
	}
	ev.ChaveAcesso, ev.TpEvento, ev.NSeqEvento = key, tpEvento, seq
	ev.Type = EventTypeFromTpEvento(tpEvento)
	if ev.Description == "" {
		ev.Description = xEvento
	}

	ev.CStat = cStat
	ev.XMotivo = xMotivo
	switch cStat {
	case "135", "136", "155":
		ev.Registered = true
		ev.Protocolo = nProt
		if dhRegEvento != "" {
			ev.RegisteredAt = dfe.ParseDateTime("dhRegEvento", dhRegEvento, &warnings)
		}
	case "":
		warnings = append(warnings, "missing retEvento cStat; event not registered")
	default:
		warnings = append(warnings, fmt.Sprintf("retEvento cStat %s (%s); event not registered", cStat, xMotivo))
	}

	ev.ParseWarnings = warnings
	return ev, nil
}
