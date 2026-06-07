package nfse

import (
	"strings"

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

	companyRoot := getRootSafely(companyCNPJValue)
	if companyRoot != "" && (companyRoot == getRootSafely(doc.PrestadorCNPJ) ||
		companyRoot == getRootSafely(doc.TomadorCNPJ) ||
		companyRoot == getRootSafely(doc.IntermediarioCNPJ)) {
		return CompanyParticipation{CompanyRole: CompanyRole("none"), VisibilityReason: VisibilityReason("same_root_only")}
	}

	return CompanyParticipation{CompanyRole: CompanyRole("none"), VisibilityReason: VisibilityReason("unknown")}
}

func getRootSafely(c string) string {
	c = strings.ReplaceAll(c, ".", "")
	c = strings.ReplaceAll(c, "-", "")
	c = strings.ReplaceAll(c, "/", "")
	if len(c) >= 8 {
		return c[:8]
	}
	return ""
}
