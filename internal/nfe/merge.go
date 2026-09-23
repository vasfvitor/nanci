package nfe

import "slices"

// MergeDocument returns the row to persist when a document with the same
// chave already exists.
//
//   - A completa is never replaced by a resumo. A resumo arriving over a
//     completa only raises Situacao and fills AuthorizedAt and Protocolo when
//     the completa lacks them.
//   - Otherwise the incoming document replaces the existing one.
//   - Situacao is always the more severe of the two:
//     cancelada > denegada > autorizada.
//   - ResumoRawHash keeps the hash of whichever side was the resumo.
//   - ID is always the existing one.
func MergeDocument(existing, incoming Document) Document {
	var merged Document
	if existing.Completeness == CompletenessCompleta && incoming.Completeness == CompletenessResumo {
		merged = existing
		if merged.AuthorizedAt == nil {
			merged.AuthorizedAt = incoming.AuthorizedAt
		}
		if merged.Protocolo == "" {
			merged.Protocolo = incoming.Protocolo
		}
		merged.ResumoRawHash = incoming.RawHash
	} else {
		merged = incoming
		merged.ID = existing.ID
		switch {
		case existing.Completeness == CompletenessResumo && incoming.Completeness == CompletenessCompleta:
			merged.ResumoRawHash = existing.RawHash
		case merged.ResumoRawHash == "":
			merged.ResumoRawHash = existing.ResumoRawHash
		}
	}
	merged.Situacao = moreSevere(existing.Situacao, incoming.Situacao)
	return merged
}

// SituacaoFromEvents applies registered events to a base situação: any
// cancelamento makes the document cancelada. Pass only registered events.
func SituacaoFromEvents(base Situacao, events []EventType) Situacao {
	if slices.Contains(events, EventTypeCancelamento) {
		return SituacaoCancelada
	}
	return base
}

func moreSevere(a, b Situacao) Situacao {
	if severity(b) > severity(a) {
		return b
	}
	return a
}

func severity(s Situacao) int {
	switch s {
	case SituacaoCancelada:
		return 3
	case SituacaoDenegada:
		return 2
	case SituacaoAutorizada:
		return 1
	default:
		return 0
	}
}
