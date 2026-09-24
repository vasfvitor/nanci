package cte

import (
	"slices"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// Participation is a company's role in a CT-e and the reason it can see it.
type Participation struct {
	CompanyRole CompanyRole
	// Papeis are all the roles the company plays, in priority order; empty
	// when CompanyRole is none.
	Papeis           []CompanyRole
	VisibilityReason VisibilityReason
}

// PapeisStrings returns papeis as plain strings, in the same order.
func PapeisStrings(papeis []CompanyRole) []string {
	out := make([]string, len(papeis))
	for i, p := range papeis {
		out[i] = string(p)
	}
	return out
}

// ClassifyParticipation decides the roles of companyCNPJ in doc. Every party
// the company matches exactly adds its role, in this order, and the first
// one is the primary role:
//
//  1. tomador
//  2. destinatário
//  3. remetente
//  4. expedidor
//  5. recebedor
//  6. emitente
//  7. autorizado (autXML)
//
// With no exact match, the same CNPJ root as any party gives
// none/same_root_only, and anything else none/unknown.
func ClassifyParticipation(doc *Document, companyCNPJ string) Participation {
	company := cnpj.Clean(companyCNPJ)

	parties := []struct {
		role CompanyRole
		cnpj string
	}{
		{CompanyRoleTomador, doc.Tomador.CNPJ},
		{CompanyRoleDestinatario, doc.Destinatario.CNPJ},
		{CompanyRoleRemetente, doc.Remetente.CNPJ},
		{CompanyRoleExpedidor, doc.Expedidor.CNPJ},
		{CompanyRoleRecebedor, doc.Recebedor.CNPJ},
		{CompanyRoleEmitente, doc.Emitente.CNPJ},
	}
	var papeis []CompanyRole
	allCNPJs := slices.Clone(doc.AutorizadosCNPJ)
	for _, p := range parties {
		if sameParty(company, p.cnpj) {
			papeis = append(papeis, p.role)
		}
		allCNPJs = append(allCNPJs, p.cnpj)
	}
	if slices.ContainsFunc(doc.AutorizadosCNPJ, func(a string) bool { return sameParty(company, a) }) {
		papeis = append(papeis, CompanyRoleAutorizado)
	}
	if len(papeis) > 0 {
		return Participation{CompanyRole: papeis[0], Papeis: papeis, VisibilityReason: exactVisibility[papeis[0]]}
	}

	if companyRoot := cnpj.RootOrEmpty(company); companyRoot != "" {
		if slices.ContainsFunc(allCNPJs, func(party string) bool { return cnpj.RootOrEmpty(party) == companyRoot }) {
			return Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonSameRootOnly}
		}
	}
	return Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonUnknown}
}

var exactVisibility = map[CompanyRole]VisibilityReason{
	CompanyRoleTomador:      VisibilityReasonExactTomador,
	CompanyRoleDestinatario: VisibilityReasonExactDestinatario,
	CompanyRoleRemetente:    VisibilityReasonExactRemetente,
	CompanyRoleExpedidor:    VisibilityReasonExactExpedidor,
	CompanyRoleRecebedor:    VisibilityReasonExactRecebedor,
	CompanyRoleEmitente:     VisibilityReasonExactEmitente,
	CompanyRoleAutorizado:   VisibilityReasonExactAutorizado,
}

// sameParty reports whether two CNPJ/CPF values are the same non-empty
// document number after removing punctuation.
func sameParty(company, party string) bool {
	return company != "" && company == cnpj.Clean(party)
}
