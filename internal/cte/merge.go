package cte

import "slices"

// MergeDocument returns the row to persist when a document with the same
// chave already exists. CT-e distribution has no resumo, so the incoming
// document replaces the existing one, except that:
//
//   - ID is always the existing one.
//   - Situacao is the more severe of the two:
//     cancelada > denegada > autorizada.
//   - A masked copy (the one autXML parties receive, see
//     Document.MaskedKeys) never replaces a full one: the existing document
//     is kept whole and only its Situacao can change.
func MergeDocument(existing, incoming Document) Document {
	merged := incoming
	if incoming.MaskedKeys && !existing.MaskedKeys {
		merged = existing
	}
	merged.ID = existing.ID
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
