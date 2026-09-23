package nfse

import (
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

// ClassifyCompanyParticipation derives company-scoped role and visibility for a canonical document.
func ClassifyCompanyParticipation(doc *Document, companyCNPJ string) CompanyParticipation {
	companyCNPJValue := cnpj.Format(companyCNPJ) // Ensure normalized using foundation/cnpj

	switch companyCNPJValue {
	case cnpj.Format(doc.PrestadorCNPJ):
		return CompanyParticipation{CompanyRole: CompanyRole("prestada"), VisibilityReason: VisibilityReason("exact_prestador")}
	case cnpj.Format(doc.TomadorCNPJ):
		return CompanyParticipation{CompanyRole: CompanyRole("tomada"), VisibilityReason: VisibilityReason("exact_tomador")}
	case cnpj.Format(doc.IntermediarioCNPJ):
		return CompanyParticipation{CompanyRole: CompanyRole("intermediario"), VisibilityReason: VisibilityReason("exact_intermediario")}
	}

	companyRoot := cnpj.RootOrEmpty(companyCNPJValue)
	if companyRoot != "" && (companyRoot == cnpj.RootOrEmpty(doc.PrestadorCNPJ) ||
		companyRoot == cnpj.RootOrEmpty(doc.TomadorCNPJ) ||
		companyRoot == cnpj.RootOrEmpty(doc.IntermediarioCNPJ)) {
		return CompanyParticipation{CompanyRole: CompanyRole("none"), VisibilityReason: VisibilityReason("same_root_only")}
	}

	return CompanyParticipation{CompanyRole: CompanyRole("none"), VisibilityReason: VisibilityReason("unknown")}
}
