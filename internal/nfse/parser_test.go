package nfse

import "testing"

func TestClassifyCompanyParticipation(t *testing.T) {
	doc := &Document{
		PrestadorCNPJ:     "12345678000199",
		TomadorCNPJ:       "99887766000155",
		IntermediarioCNPJ: "11223344000177",
	}

	tests := []struct {
		name        string
		companyCNPJ string
		role        string
		visibility  string
	}{
		{name: "exact prestador", companyCNPJ: "12345678000199", role: "prestada", visibility: "exact_prestador"},
		{name: "exact tomador", companyCNPJ: "99887766000155", role: "tomada", visibility: "exact_tomador"},
		{name: "same root only", companyCNPJ: "12345678000270", role: "none", visibility: "same_root_only"},
		{name: "unknown", companyCNPJ: "55555555000155", role: "none", visibility: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCompanyParticipation(doc, tt.companyCNPJ)
			if got.CompanyRole != tt.role {
				t.Fatalf("role = %q, want %q", got.CompanyRole, tt.role)
			}
			if got.VisibilityReason != tt.visibility {
				t.Fatalf("visibility = %q, want %q", got.VisibilityReason, tt.visibility)
			}
		})
	}
}
