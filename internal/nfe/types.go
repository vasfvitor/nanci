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

func (t EventType) Valid() bool {
	switch t {
	case EventTypeCancelamento, EventTypeCartaCorrecao, EventTypeCiencia, EventTypeConfirmacao,
		EventTypeDesconhecimento, EventTypeNaoRealizada, EventTypeUnknown:
		return true
	default:
		return false
	}
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

// Label returns the manifestação state shown to the user.
func (m Manifestacao) Label() string {
	switch m {
	case ManifestacaoNenhuma:
		return "Sem manifestação"
	case ManifestacaoCiencia:
		return "Ciência"
	case ManifestacaoConfirmada:
		return "Confirmada"
	case ManifestacaoDesconhecida:
		return "Desconhecida"
	case ManifestacaoNaoRealizada:
		return "Operação não realizada"
	default:
		return string(m)
	}
}

// Conclusive reports whether m comes from a conclusive manifestação:
// confirmação, desconhecimento or operação não realizada.
func (m Manifestacao) Conclusive() bool {
	switch m {
	case ManifestacaoConfirmada, ManifestacaoDesconhecida, ManifestacaoNaoRealizada:
		return true
	default:
		return false
	}
}
