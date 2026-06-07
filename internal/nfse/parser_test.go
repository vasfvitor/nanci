package nfse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name            string
		xml             string
		wantType        string
		wantChave       string
		wantReplacement string
		wantEventAt     bool
		wantDescription string
	}{
		{
			name:            "cancelamento",
			xml:             `<pedCancNFSe><infPedidoCanc><chNFSe>CHAVE-CANC</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`,
			wantType:        "cancelamento",
			wantChave:       "CHAVE-CANC",
			wantEventAt:     true,
			wantDescription: "Erro emissao",
		},
		{
			name:            "substituicao",
			xml:             `<eventoSubstituicao><chNFSe>CHAVE-OLD</chNFSe><chNFSeSubst>CHAVE-NEW</chNFSeSubst><descEvento>Substituicao de nota</descEvento></eventoSubstituicao>`,
			wantType:        "substituicao",
			wantChave:       "CHAVE-OLD",
			wantReplacement: "CHAVE-NEW",
			wantDescription: "Substituicao de nota",
		},
		{
			name:            "unknown",
			xml:             `<eventoGenerico><chNFSe>CHAVE-UNK</chNFSe><xJust>Payload nao classificado</xJust></eventoGenerico>`,
			wantType:        "unknown",
			wantChave:       "CHAVE-UNK",
			wantDescription: "Payload nao classificado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseEvent([]byte(tt.xml))
			if err != nil {
				t.Fatalf("ParseEvent: %v", err)
			}
			if event.Type != tt.wantType {
				t.Fatalf("Type = %q, want %q", event.Type, tt.wantType)
			}
			if event.ChaveAcesso != tt.wantChave {
				t.Fatalf("ChaveAcesso = %q, want %q", event.ChaveAcesso, tt.wantChave)
			}
			if event.ReplacementChaveAcesso != tt.wantReplacement {
				t.Fatalf("ReplacementChaveAcesso = %q, want %q", event.ReplacementChaveAcesso, tt.wantReplacement)
			}
			if event.EventAtValid != tt.wantEventAt {
				t.Fatalf("EventAtValid = %v, want %v", event.EventAtValid, tt.wantEventAt)
			}
			if event.Description != tt.wantDescription {
				t.Fatalf("Description = %q, want %q", event.Description, tt.wantDescription)
			}
		})
	}
}

func TestParseEventRejectsMissingChave(t *testing.T) {
	_, err := ParseEvent([]byte(`<eventoSemChave><descEvento>Sem chave</descEvento></eventoSemChave>`))
	if err == nil {
		t.Fatal("ParseEvent error = nil, want failure")
	}
}

func TestParseEventParsesRFC3339Timestamp(t *testing.T) {
	event, err := ParseEvent([]byte(`<pedCancNFSe><infPedidoCanc><chNFSe>CHAVE-TS</chNFSe><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`))
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if !event.EventAtValid {
		t.Fatal("EventAtValid = false, want true")
	}
	if !event.EventAt.Equal(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("EventAt = %s", event.EventAt)
	}
}

func TestParseXML(t *testing.T) {
	tests := []struct {
		filename     string
		wantChave    string
		wantTotalRet float64
		wantErrors   bool
		wantWarning  bool
		wantVersion  string
		wantNumero   string
		wantDesc     string
	}{
		{
			filename:    "simple-prestada.xml",
			wantChave:   "1122334455667788990011223344556677889900112244",
			wantVersion: "1.0",
		},
		{
			filename:    "simple-tomada.xml",
			wantChave:   "9988776655443322110099887766554433221100998877",
			wantVersion: "1.0",
		},
		{
			filename:     "com-retencoes.xml",
			wantChave:    "5555555555555555555555555555555555555555555555",
			wantVersion:  "1.01",
			wantTotalRet: 1715.00, // 150(IRRF) + 1100(INSS) + 65(PIS) + 300(COFINS) + 100(CSLL)
		},
		{
			filename:    "sem-retencoes.xml",
			wantChave:   "4444444444444444444444444444444444444444444444",
			wantVersion: "1.0",
			wantWarning: true, // Missing tomador
		},
		{
			filename:   "invalid.xml",
			wantErrors: true,
		},
		{
			filename:    "ibscbs-extra-fields.xml",
			wantChave:   "7777777777777777777777777777777777777777777777",
			wantVersion: "1.01",
			wantWarning: true,
		},
		{
			filename:    "com-numero-descricao.xml",
			wantChave:   "8888888888888888888888888888888888888888888888",
			wantVersion: "1.0",
			wantNumero:  "12345",
			wantDesc:    "Serviços de desenvolvimento de software",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join("testdata", tt.filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read testdata %s: %v", tt.filename, err)
			}

			doc, err := ParseXML(data)
			if tt.wantErrors {
				if err == nil {
					t.Fatalf("Expected error for %s, got nil", tt.filename)
				}
				return // we expected it to fail, so nothing else to check
			}

			if err != nil {
				t.Fatalf("ParseXML failed for %s: %v", tt.filename, err)
			}

			if doc.ChaveAcesso != tt.wantChave {
				t.Errorf("ChaveAcesso = %q, want %q", doc.ChaveAcesso, tt.wantChave)
			}
			if doc.LayoutVersion != tt.wantVersion {
				t.Errorf("LayoutVersion = %q, want %q", doc.LayoutVersion, tt.wantVersion)
			}
			if doc.TotalRetentions != tt.wantTotalRet {
				t.Errorf("TotalRetentions = %f, want %f", doc.TotalRetentions, tt.wantTotalRet)
			}
			if tt.wantNumero != "" && doc.NFSeNumber != tt.wantNumero {
				t.Errorf("NFSeNumber = %q, want %q", doc.NFSeNumber, tt.wantNumero)
			}
			if tt.wantDesc != "" && doc.ServiceDescription != tt.wantDesc {
				t.Errorf("ServiceDescription = %q, want %q", doc.ServiceDescription, tt.wantDesc)
			}

			hasWarning := len(doc.ParseWarnings) > 0
			if hasWarning && !tt.wantWarning {
				// Only fail if we didn't explicitly want a warning and it's not the ISS warning
				// since our simple fixtures have ISS and thus get a warning about ISS retention.
				// Let's check if the only warning is the ISS one.
				nonIssWarning := false
				for _, w := range doc.ParseWarnings {
					if !strings.Contains(w, "ISS presente") {
						nonIssWarning = true
					}
				}
				if nonIssWarning {
					t.Errorf("Unexpected warnings for %s: %v", tt.filename, doc.ParseWarnings)
				}
			} else if !hasWarning && tt.wantWarning {
				t.Errorf("Expected warnings for %s, got none", tt.filename)
			}
		})
	}
}
