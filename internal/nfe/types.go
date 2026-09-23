// Package nfe holds the NF-e (modelo 55) domain: access keys, documents,
// events, participation rules and the parsers for the payloads distributed by
// NFeDistribuicaoDFe. It has no database or network code.
package nfe

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// Situacao is the fiscal state of an NF-e.
type Situacao string

const (
	SituacaoAutorizada Situacao = "autorizada"
	SituacaoDenegada   Situacao = "denegada"
	SituacaoCancelada  Situacao = "cancelada"
)

func ParseSituacao(val string) (Situacao, error) {
	s := Situacao(val)
	if !s.Valid() {
		return "", fmt.Errorf("invalid situacao %q: %w", val, nfse.ErrInvalidEnum)
	}
	return s, nil
}

func (s Situacao) Valid() bool {
	switch s {
	case SituacaoAutorizada, SituacaoDenegada, SituacaoCancelada:
		return true
	default:
		return false
	}
}

func (s Situacao) String() string {
	return string(s)
}

// Completeness tells whether nanci holds only the summary (resNFe/resEvento)
// or the full signed XML (procNFe/procEventoNFe).
type Completeness string

const (
	CompletenessResumo   Completeness = "resumo"
	CompletenessCompleta Completeness = "completa"
)

func ParseCompleteness(val string) (Completeness, error) {
	c := Completeness(val)
	if !c.Valid() {
		return "", fmt.Errorf("invalid completeness %q: %w", val, nfse.ErrInvalidEnum)
	}
	return c, nil
}

func (c Completeness) Valid() bool {
	switch c {
	case CompletenessResumo, CompletenessCompleta:
		return true
	default:
		return false
	}
}

func (c Completeness) String() string {
	return string(c)
}

// CompanyRole is the part a managed company plays in an NF-e.
type CompanyRole string

const (
	CompanyRoleDestinatario  CompanyRole = "destinatario"
	CompanyRoleEmitente      CompanyRole = "emitente"
	CompanyRoleTransportador CompanyRole = "transportador"
	CompanyRoleAutorizado    CompanyRole = "autorizado"
	CompanyRoleNone          CompanyRole = "none"
)

func ParseCompanyRole(val string) (CompanyRole, error) {
	r := CompanyRole(val)
	if !r.Valid() {
		return "", fmt.Errorf("invalid company role %q: %w", val, nfse.ErrInvalidEnum)
	}
	return r, nil
}

func (r CompanyRole) Valid() bool {
	switch r {
	case CompanyRoleDestinatario, CompanyRoleEmitente, CompanyRoleTransportador, CompanyRoleAutorizado, CompanyRoleNone:
		return true
	default:
		return false
	}
}

func (r CompanyRole) String() string {
	return string(r)
}

// VisibilityReason explains why a company can see an NF-e.
type VisibilityReason string

const (
	VisibilityReasonExactDestinatario  VisibilityReason = "exact_destinatario"
	VisibilityReasonExactEmitente      VisibilityReason = "exact_emitente"
	VisibilityReasonExactTransportador VisibilityReason = "exact_transportador"
	VisibilityReasonExactAutorizado    VisibilityReason = "exact_autorizado"
	VisibilityReasonResumoDestinatario VisibilityReason = "resumo_destinatario"
	VisibilityReasonSameRootOnly       VisibilityReason = "same_root_only"
	VisibilityReasonUnknown            VisibilityReason = "unknown"
)

func ParseVisibilityReason(val string) (VisibilityReason, error) {
	r := VisibilityReason(val)
	if !r.Valid() {
		return "", fmt.Errorf("invalid visibility reason %q: %w", val, nfse.ErrInvalidEnum)
	}
	return r, nil
}

func (r VisibilityReason) Valid() bool {
	switch r {
	case VisibilityReasonExactDestinatario, VisibilityReasonExactEmitente, VisibilityReasonExactTransportador,
		VisibilityReasonExactAutorizado, VisibilityReasonResumoDestinatario, VisibilityReasonSameRootOnly,
		VisibilityReasonUnknown:
		return true
	default:
		return false
	}
}

func (r VisibilityReason) String() string {
	return string(r)
}

// tpEvento codes used by nanci.
const (
	TpEventoCCe              = "110110"
	TpEventoCancelamento     = "110111"
	TpEventoCancSubstituicao = "110112"
	TpEventoConfirmacao      = "210200"
	TpEventoCiencia          = "210210"
	TpEventoDesconhecimento  = "210220"
	TpEventoNaoRealizada     = "210240"
)

// EventType is nanci's name for an NF-e event.
type EventType string

const (
	EventTypeCancelamento    EventType = "cancelamento"
	EventTypeCartaCorrecao   EventType = "carta_correcao"
	EventTypeCiencia         EventType = "ciencia"
	EventTypeConfirmacao     EventType = "confirmacao"
	EventTypeDesconhecimento EventType = "desconhecimento"
	EventTypeNaoRealizada    EventType = "nao_realizada"
	EventTypeUnknown         EventType = "unknown"
)

// EventTypeFromTpEvento maps a tpEvento code to an EventType. Codes nanci
// does not handle map to EventTypeUnknown.
func EventTypeFromTpEvento(tp string) EventType {
	switch tp {
	case TpEventoCancelamento, TpEventoCancSubstituicao:
		return EventTypeCancelamento
	case TpEventoCCe:
		return EventTypeCartaCorrecao
	case TpEventoCiencia:
		return EventTypeCiencia
	case TpEventoConfirmacao:
		return EventTypeConfirmacao
	case TpEventoDesconhecimento:
		return EventTypeDesconhecimento
	case TpEventoNaoRealizada:
		return EventTypeNaoRealizada
	default:
		return EventTypeUnknown
	}
}

func ParseEventType(val string) (EventType, error) {
	t := EventType(val)
	if !t.Valid() {
		return "", fmt.Errorf("invalid event type %q: %w", val, nfse.ErrInvalidEnum)
	}
	return t, nil
}

func (t EventType) Valid() bool {
	switch t {
	case EventTypeCancelamento, EventTypeCartaCorrecao, EventTypeCiencia, EventTypeConfirmacao,
		EventTypeDesconhecimento, EventTypeNaoRealizada, EventTypeUnknown:
		return true
	default:
		return false
	}
}

func (t EventType) String() string {
	return string(t)
}

// Manifestacao is the current manifestação do destinatário state of an NF-e
// for one company, derived from the events that company authored.
type Manifestacao string

const (
	ManifestacaoNenhuma      Manifestacao = "nenhuma"
	ManifestacaoCiencia      Manifestacao = "ciencia"
	ManifestacaoConfirmada   Manifestacao = "confirmada"
	ManifestacaoDesconhecida Manifestacao = "desconhecida"
	ManifestacaoNaoRealizada Manifestacao = "nao_realizada"
)

func ParseManifestacao(val string) (Manifestacao, error) {
	m := Manifestacao(val)
	if !m.Valid() {
		return "", fmt.Errorf("invalid manifestacao %q: %w", val, nfse.ErrInvalidEnum)
	}
	return m, nil
}

func (m Manifestacao) Valid() bool {
	switch m {
	case ManifestacaoNenhuma, ManifestacaoCiencia, ManifestacaoConfirmada, ManifestacaoDesconhecida, ManifestacaoNaoRealizada:
		return true
	default:
		return false
	}
}

func (m Manifestacao) String() string {
	return string(m)
}
