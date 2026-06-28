package nfse_test

import (
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestDocument_Instantiation(t *testing.T) {
	doc := nfse.Document{
		ID:                 "doc123",
		ChaveAcesso:        "12345678901234567890123456789012345678901234567890",
		IssueDate:          time.Now(),
		Competence:         "2023-01",
		PrestadorCNPJ:      "11.111.111/0001-11",
		PrestadorName:      "Prestador Test",
		TomadorCNPJ:        "22.222.222/0001-22",
		TomadorName:        "Tomador Test",
		Status:             nfse.DocumentStatusNormal,
		ServiceDescription: "Dev Services",
	}

	if doc.ID != "doc123" {
		t.Errorf("Expected ID doc123, got %s", doc.ID)
	}
	if doc.Status != nfse.DocumentStatusNormal {
		t.Errorf("Expected Status normal, got %s", doc.Status)
	}
}

func TestDocumentStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.DocumentStatus
		valid    bool
	}{
		{"normal", nfse.DocumentStatusNormal, true},
		{"cancelada", nfse.DocumentStatusCancelada, true},
		{"substituida", nfse.DocumentStatusSubstituida, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			status, err := nfse.ParseDocumentStatus(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if status != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, status)
				}
				if !status.Valid() {
					t.Errorf("Expected status %s to be valid", status)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %s, got nil", tt.input)
				}
				if status.Valid() {
					t.Errorf("Expected status %s to be invalid", status)
				}
			}
		})
	}
}

func TestCompanyRole(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.CompanyRole
		valid    bool
	}{
		{"tomada", nfse.CompanyRoleTomada, true},
		{"prestada", nfse.CompanyRolePrestada, true},
		{"intermediario", nfse.CompanyRoleIntermediario, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			role, err := nfse.ParseCompanyRole(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if role != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, role)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %s, got nil", tt.input)
				}
			}
		})
	}
}

func TestVisibilityReason(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.VisibilityReason
		valid    bool
	}{
		{"exact_prestador", nfse.VisibilityReasonExactPrestador, true},
		{"exact_tomador", nfse.VisibilityReasonExactTomador, true},
		{"exact_intermediario", nfse.VisibilityReasonExactIntermediario, true},
		{"same_root_only", nfse.VisibilityReasonSameRootOnly, true},
		{"unknown", nfse.VisibilityReasonUnknown, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reason, err := nfse.ParseVisibilityReason(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if reason != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, reason)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %s, got nil", tt.input)
				}
			}
		})
	}
}

func TestClassifyCompanyParticipation(t *testing.T) {
	doc := &nfse.Document{
		PrestadorCNPJ:     "11111111000111",
		TomadorCNPJ:       "22222222000122",
		IntermediarioCNPJ: "33333333000133",
	}

	tests := []struct {
		name           string
		cnpj           string
		expectedRole   nfse.CompanyRole
		expectedReason nfse.VisibilityReason
	}{
		{"Exact Prestador", "11.111.111/0001-11", nfse.CompanyRolePrestada, nfse.VisibilityReasonExactPrestador},
		{"Exact Tomador", "22.222.222/0001-22", nfse.CompanyRoleTomada, nfse.VisibilityReasonExactTomador},
		{"Exact Intermediario", "33.333.333/0001-33", nfse.CompanyRoleIntermediario, nfse.VisibilityReasonExactIntermediario},
		{"Same Root Prestador", "11.111.111/0002-22", nfse.CompanyRole("none"), nfse.VisibilityReasonSameRootOnly},
		{"Same Root Tomador", "22.222.222/0002-22", nfse.CompanyRole("none"), nfse.VisibilityReasonSameRootOnly},
		{"Unknown", "44.444.444/0001-44", nfse.CompanyRole("none"), nfse.VisibilityReasonUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			participation := nfse.ClassifyCompanyParticipation(doc, tt.cnpj)
			if participation.CompanyRole != tt.expectedRole {
				t.Errorf("Expected role %s, got %s", tt.expectedRole, participation.CompanyRole)
			}
			if participation.VisibilityReason != tt.expectedReason {
				t.Errorf("Expected reason %s, got %s", tt.expectedReason, participation.VisibilityReason)
			}
		})
	}
}
