// Package cte holds the CT-e domain (modelos 57 and 67, the GTV-e modelo 64
// and the CT-e Simplificado): documents, events, participation rules and the
// parsers for the payloads distributed by CTeDistribuicaoDFe. It has no
// database or network code.
package cte

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// Situacao is the fiscal state of a CT-e.
type Situacao string

const (
	SituacaoAutorizada Situacao = "autorizada"
	SituacaoDenegada   Situacao = "denegada"
	SituacaoCancelada  Situacao = "cancelada"
)

func ParseSituacao(val string) (Situacao, error) {
	s := Situacao(val)
	if !s.Valid() {
		return "", fmt.Errorf("invalid situacao %q: %w", val, dfe.ErrInvalidEnum)
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

// CompanyRole is the part a managed company plays in a CT-e. The constants
// are listed in priority order: the first role the company matches is its
// primary role.
type CompanyRole string

const (
	CompanyRoleTomador      CompanyRole = "tomador"
	CompanyRoleDestinatario CompanyRole = "destinatario"
	CompanyRoleRemetente    CompanyRole = "remetente"
	CompanyRoleExpedidor    CompanyRole = "expedidor"
	CompanyRoleRecebedor    CompanyRole = "recebedor"
	CompanyRoleEmitente     CompanyRole = "emitente"
	CompanyRoleAutorizado   CompanyRole = "autorizado"
	CompanyRoleNone         CompanyRole = "none"
)

func ParseCompanyRole(val string) (CompanyRole, error) {
	r := CompanyRole(val)
	if !r.Valid() {
		return "", fmt.Errorf("invalid company role %q: %w", val, dfe.ErrInvalidEnum)
	}
	return r, nil
}

func (r CompanyRole) Valid() bool {
	switch r {
	case CompanyRoleTomador, CompanyRoleDestinatario, CompanyRoleRemetente, CompanyRoleExpedidor,
		CompanyRoleRecebedor, CompanyRoleEmitente, CompanyRoleAutorizado, CompanyRoleNone:
		return true
	default:
		return false
	}
}

// VisibilityReason explains why a company can see a CT-e.
type VisibilityReason string

const (
	VisibilityReasonExactTomador      VisibilityReason = "exact_tomador"
	VisibilityReasonExactDestinatario VisibilityReason = "exact_destinatario"
	VisibilityReasonExactRemetente    VisibilityReason = "exact_remetente"
	VisibilityReasonExactExpedidor    VisibilityReason = "exact_expedidor"
	VisibilityReasonExactRecebedor    VisibilityReason = "exact_recebedor"
	VisibilityReasonExactEmitente     VisibilityReason = "exact_emitente"
	VisibilityReasonExactAutorizado   VisibilityReason = "exact_autorizado"
	VisibilityReasonSameRootOnly      VisibilityReason = "same_root_only"
	VisibilityReasonUnknown           VisibilityReason = "unknown"
)

func (r VisibilityReason) Valid() bool {
	switch r {
	case VisibilityReasonExactTomador, VisibilityReasonExactDestinatario, VisibilityReasonExactRemetente,
		VisibilityReasonExactExpedidor, VisibilityReasonExactRecebedor, VisibilityReasonExactEmitente,
		VisibilityReasonExactAutorizado, VisibilityReasonSameRootOnly, VisibilityReasonUnknown:
		return true
	default:
		return false
	}
}

// TipoDocumento is the kind of transport document.
type TipoDocumento string

const (
	TipoDocumentoCTe             TipoDocumento = "cte"              // CT-e, modelo 57
	TipoDocumentoCTeOS           TipoDocumento = "cte_os"           // CT-e OS, modelo 67
	TipoDocumentoGTVe            TipoDocumento = "gtve"             // GTV-e, modelo 64
	TipoDocumentoCTeSimplificado TipoDocumento = "cte_simplificado" // CT-e Simplificado, modelo 57
)

func ParseTipoDocumento(val string) (TipoDocumento, error) {
	t := TipoDocumento(val)
	if !t.Valid() {
		return "", fmt.Errorf("invalid tipo de documento %q: %w", val, dfe.ErrInvalidEnum)
	}
	return t, nil
}

func (t TipoDocumento) Valid() bool {
	switch t {
	case TipoDocumentoCTe, TipoDocumentoCTeOS, TipoDocumentoGTVe, TipoDocumentoCTeSimplificado:
		return true
	default:
		return false
	}
}

// Modelo returns the modelo carried by the access key of this kind of
// document.
func (t TipoDocumento) Modelo() string {
	switch t {
	case TipoDocumentoCTe, TipoDocumentoCTeSimplificado:
		return "57"
	case TipoDocumentoGTVe:
		return "64"
	case TipoDocumentoCTeOS:
		return "67"
	default:
		return ""
	}
}

// tpEvento codes of the events CTeDistribuicaoDFe distributes.
const (
	TpEventoCCe                    = "110110"
	TpEventoCancelamento           = "110111"
	TpEventoEPEC                   = "110113"
	TpEventoRegistroMultimodal     = "110160"
	TpEventoGTV                    = "110170"
	TpEventoComprovanteEntrega     = "110180"
	TpEventoCancComprovanteEntrega = "110181"
	TpEventoInsucessoEntrega       = "110190"
	TpEventoCancInsucessoEntrega   = "110191"
	TpEventoMDFeAutorizado         = "310610"
	TpEventoMDFeCancelado          = "310611"
	TpEventoPrestacaoDesacordo     = "610110"
	TpEventoCancPrestacaoDesacordo = "610111"
)

// EventType is nanci's name for a CT-e event.
type EventType string

const (
	EventTypeCancelamento                   EventType = "cancelamento"
	EventTypeCartaCorrecao                  EventType = "carta_correcao"
	EventTypeEPEC                           EventType = "epec"
	EventTypeRegistroMultimodal             EventType = "registro_multimodal"
	EventTypeGTV                            EventType = "gtv"
	EventTypeComprovanteEntrega             EventType = "comprovante_entrega"
	EventTypeCancelamentoComprovanteEntrega EventType = "cancelamento_comprovante_entrega"
	EventTypeInsucessoEntrega               EventType = "insucesso_entrega"
	EventTypeCancelamentoInsucessoEntrega   EventType = "cancelamento_insucesso_entrega"
	EventTypePrestacaoDesacordo             EventType = "prestacao_desacordo"
	EventTypeCancelamentoDesacordo          EventType = "cancelamento_desacordo"
	EventTypeMDFeAutorizado                 EventType = "mdfe_autorizado"
	EventTypeMDFeCancelado                  EventType = "mdfe_cancelado"
	EventTypeUnknown                        EventType = "unknown"
)

// EventTypeFromTpEvento maps a tpEvento code to an EventType. Codes nanci
// does not handle map to EventTypeUnknown.
func EventTypeFromTpEvento(tp string) EventType {
	switch tp {
	case TpEventoCancelamento:
		return EventTypeCancelamento
	case TpEventoCCe:
		return EventTypeCartaCorrecao
	case TpEventoEPEC:
		return EventTypeEPEC
	case TpEventoRegistroMultimodal:
		return EventTypeRegistroMultimodal
	case TpEventoGTV:
		return EventTypeGTV
	case TpEventoComprovanteEntrega:
		return EventTypeComprovanteEntrega
	case TpEventoCancComprovanteEntrega:
		return EventTypeCancelamentoComprovanteEntrega
	case TpEventoInsucessoEntrega:
		return EventTypeInsucessoEntrega
	case TpEventoCancInsucessoEntrega:
		return EventTypeCancelamentoInsucessoEntrega
	case TpEventoPrestacaoDesacordo:
		return EventTypePrestacaoDesacordo
	case TpEventoCancPrestacaoDesacordo:
		return EventTypeCancelamentoDesacordo
	case TpEventoMDFeAutorizado:
		return EventTypeMDFeAutorizado
	case TpEventoMDFeCancelado:
		return EventTypeMDFeCancelado
	default:
		return EventTypeUnknown
	}
}

func (t EventType) Valid() bool {
	switch t {
	case EventTypeCancelamento, EventTypeCartaCorrecao, EventTypeEPEC, EventTypeRegistroMultimodal, EventTypeGTV,
		EventTypeComprovanteEntrega, EventTypeCancelamentoComprovanteEntrega, EventTypeInsucessoEntrega,
		EventTypeCancelamentoInsucessoEntrega, EventTypePrestacaoDesacordo, EventTypeCancelamentoDesacordo,
		EventTypeMDFeAutorizado, EventTypeMDFeCancelado, EventTypeUnknown:
		return true
	default:
		return false
	}
}
