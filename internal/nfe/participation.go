package nfe

import (
	"slices"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// Participation is a company's role in an NF-e and the reason it can see it.
type Participation struct {
	CompanyRole      CompanyRole
	VisibilityReason VisibilityReason
}

// ClassifyParticipation decides the role of companyCNPJ in doc. The rules
// are checked in order and the first match wins:
//
//  1. destinatário
//  2. emitente
//  3. transportador
//  4. autorizado (autXML)
//  5. a resumo not emitted by the company: resumos are only distributed to
//     the destinatário, so the company is the destinatário
//  6. same CNPJ root as any party: none/same_root_only
//  7. none/unknown
//
// Call it again when a resumo is upgraded to completa.
func ClassifyParticipation(doc *Document, companyCNPJ string) Participation {
	company := cnpj.Clean(companyCNPJ)

	switch {
	case sameParty(company, doc.DestinatarioCNPJ):
		return Participation{CompanyRoleDestinatario, VisibilityReasonExactDestinatario}
	case sameParty(company, doc.EmitenteCNPJ):
		return Participation{CompanyRoleEmitente, VisibilityReasonExactEmitente}
	case sameParty(company, doc.TransportadorCNPJ):
		return Participation{CompanyRoleTransportador, VisibilityReasonExactTransportador}
	case slices.ContainsFunc(doc.AutorizadosCNPJ, func(a string) bool { return sameParty(company, a) }):
		return Participation{CompanyRoleAutorizado, VisibilityReasonExactAutorizado}
	case doc.Completeness == CompletenessResumo:
		return Participation{CompanyRoleDestinatario, VisibilityReasonResumoDestinatario}
	}

	parties := append([]string{doc.DestinatarioCNPJ, doc.EmitenteCNPJ, doc.TransportadorCNPJ}, doc.AutorizadosCNPJ...)
	if companyRoot := cnpj.RootOrEmpty(company); companyRoot != "" {
		for _, party := range parties {
			if cnpj.RootOrEmpty(party) == companyRoot {
				return Participation{CompanyRoleNone, VisibilityReasonSameRootOnly}
			}
		}
	}
	return Participation{CompanyRoleNone, VisibilityReasonUnknown}
}

// sameParty reports whether two CNPJ/CPF values are the same non-empty
// document number after removing punctuation.
func sameParty(company, party string) bool {
	return company != "" && company == cnpj.Clean(party)
}
