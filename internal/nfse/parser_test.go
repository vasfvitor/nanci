package nfse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseEventXML(t *testing.T) {
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
			xml:             "<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><cMotivo>Erro emissao</cMotivo><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>",
			wantType:        "cancelamento",
			wantChave:       "12345678901234567890123456789012345678901234567890",
			wantEventAt:     true,
			wantDescription: "Erro emissao",
		},
		{
			name:            "substituicao",
			xml:             "<substituicaoNfse><substituicao><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><chNFSeSubstituida>09876543210987654321098765432109876543210987654321</chNFSeSubstituida><xMotivo>Valor incorreto</xMotivo></substituicao></substituicaoNfse>",
			wantType:        "substituicao",
			wantChave:       "12345678901234567890123456789012345678901234567890",
			wantReplacement: "09876543210987654321098765432109876543210987654321",
			wantEventAt:     false,
			wantDescription: "Valor incorreto",
		},
		{
			name:            "unknown",
			xml:             "<eventoDesconhecido><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe></eventoDesconhecido>",
			wantType:        "unknown",
			wantChave:       "12345678901234567890123456789012345678901234567890",
			wantDescription: "Evento sincronizado via NSU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, _, err := ParseEventXML([]byte(tt.xml))
			if err != nil {
				t.Fatalf("ParseEvent: %v", err)
			}
			if string(event.Type) != tt.wantType {
				t.Fatalf("Type = %q, want %q", event.Type, tt.wantType)
			}
			if string(event.ChaveAcesso) != tt.wantChave {
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
	_, _, err := ParseEventXML([]byte(`<eventoSemChave><descEvento>Sem chave</descEvento></eventoSemChave>`))
	if err == nil {
		t.Fatal("ParseEvent error = nil, want failure")
	}
}

func TestParseEventParsesRFC3339Timestamp(t *testing.T) {
	event, _, err := ParseEventXML([]byte(`"<pedCancNFSe><infPedidoCanc><chNFSe>12345678901234567890123456789012345678901234567890</chNFSe><dhEvento>2026-06-04T12:00:00Z</dhEvento></infPedidoCanc></pedCancNFSe>`))
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
		wantTotalRet int64
		wantErrors   bool
		wantWarning  bool
		wantVersion  string
		wantNumero   string
		wantDesc     string
	}{
		{
			filename:    "simple-prestada.xml",
			wantChave:   "11223344556677889900112233445566778899001122441111",
			wantVersion: "1.0",
		},
		{
			filename:    "simple-tomada.xml",
			wantChave:   "99887766554433221100998877665544332211009988779999",
			wantVersion: "1.0",
		},
		{
			filename:     "com-retencoes.xml",
			wantChave:    "55555555555555555555555555555555555555555555555555",
			wantVersion:  "1.01",
			wantTotalRet: 171500, // 150(IRRF) + 1100(INSS) + 65(PIS) + 300(COFINS) + 100(CSLL)
		},
		{
			filename:    "sem-retencoes.xml",
			wantChave:   "44444444444444444444444444444444444444444444444444",
			wantVersion: "1.0",
			wantWarning: false,
		},
		{
			filename:   "invalid.xml",
			wantErrors: true,
		},
		{
			filename:    "ibscbs-extra-fields.xml",
			wantChave:   "77777777777777777777777777777777777777777777777777",
			wantVersion: "1.01",
			wantWarning: false,
		},
		{
			filename:    "com-numero-descricao.xml",
			wantChave:   "88888888888888888888888888888888888888888888888888",
			wantVersion: "1.0",
			wantNumero:  "12345",
			wantDesc:    "Serviços de desenvolvimento de software",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join("testdata", tt.filename)
			data, err := os.ReadFile(path) // #nosec G304 -- fixed testdata path.
			if err != nil {
				t.Fatalf("Failed to read testdata %s: %v", tt.filename, err)
			}

			doc, _, err := ParseDocumentXML(data)
			if tt.wantErrors {
				if err == nil {
					t.Fatalf("Expected error for %s, got nil", tt.filename)
				}
				return // we expected it to fail, so nothing else to check
			}

			if err != nil {
				t.Fatalf("ParseXML failed for %s: %v", tt.filename, err)
			}

			if string(doc.ChaveAcesso) != tt.wantChave {
				t.Errorf("ChaveAcesso = %q, want %q", doc.ChaveAcesso, tt.wantChave)
			}
			if doc.LayoutVersion != tt.wantVersion {
				t.Errorf("LayoutVersion = %q, want %q", doc.LayoutVersion, tt.wantVersion)
			}
			if int64(doc.TotalRetentions) != tt.wantTotalRet && tt.wantTotalRet > 0 {
				t.Errorf("TotalRetentions = %d, want %d", int64(doc.TotalRetentions), tt.wantTotalRet)
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
