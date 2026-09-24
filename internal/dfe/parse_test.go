package dfe

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testKey = "35260911222333000181550010000012341123456787"

func TestParseDateTimeAndCompetence(t *testing.T) {
	var warnings []string
	got := ParseDateTime("dhEmi", "2026-09-30T23:30:00-03:00", &warnings)
	if got == nil || len(warnings) != 0 {
		t.Fatalf("ParseDateTime = %v with %v", got, warnings)
	}
	// The competence keeps the offset the date was written with, even when
	// the UTC month is already the next one.
	if c := Competence(*got); c != "2026-09" {
		t.Errorf("Competence = %q, want 2026-09", c)
	}
	if c := Competence(time.Time{}); c != "" {
		t.Errorf("Competence(zero) = %q, want empty", c)
	}

	if got := ParseDateTime("dhEmi", "30/09/2026", &warnings); got != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "dhEmi") {
		t.Errorf("ParseDateTime of a bad value = %v with %v, want nil and a warning naming dhEmi", got, warnings)
	}
}

func TestParseMoneyInto(t *testing.T) {
	var m Money
	if err := ParseMoneyInto(&m, "vNF", "1500.5"); err != nil || m.Cents() != 150050 {
		t.Errorf("ParseMoneyInto = %d, %v, want 150050", m.Cents(), err)
	}
	err := ParseMoneyInto(&m, "vNF", "1.500,50")
	if !errors.Is(err, ErrInvalidMoneyFormat) || !strings.HasPrefix(err.Error(), "vNF: ") {
		t.Errorf("ParseMoneyInto of a bad value error = %v, want ErrInvalidMoneyFormat naming vNF", err)
	}
	if m.Cents() != 150050 {
		t.Errorf("a failed parse changed dst to %d", m.Cents())
	}
}

func TestKeyFromProtocol(t *testing.T) {
	tests := []struct {
		name         string
		prot, id     string
		wantErr      bool
		wantWarnings int
	}{
		{"both agree", testKey, "NFe" + testKey, false, 0},
		{"protocol only", testKey, "", false, 0},
		{"Id fallback", "", "NFe" + testKey, false, 1},
		{"both missing", "", "", true, 0},
		{"disagree", testKey, "NFe35260911222333000181550010000012351234567894", true, 0},
		{"bad check digit", testKey[:43] + "0", "", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warnings []string
			key, err := KeyFromProtocol("NFe", tt.prot, tt.id, &warnings)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("KeyFromProtocol = %q, want an error", key)
				}
				return
			}
			if err != nil || key != testKey || len(warnings) != tt.wantWarnings {
				t.Errorf("KeyFromProtocol = %q, %v with %v, want the key with %d warnings", key, err, warnings, tt.wantWarnings)
			}
		})
	}
}

func TestParseEventIdentity(t *testing.T) {
	key, seq, err := ParseEventIdentity("CTe", testKey, "110111", "01")
	if err != nil || key != testKey || seq != 1 {
		t.Fatalf("ParseEventIdentity = %q, %d, %v", key, seq, err)
	}

	tests := []struct {
		name, chave, tpEvento, nSeq, wantErr string
	}{
		{"missing chave", "", "110111", "1", "missing essential field: chCTe"},
		{"bad chave", testKey[:43] + "0", "110111", "1", "chCTe: invalid access key"},
		{"missing tpEvento", testKey, "", "1", "missing essential field: tpEvento"},
		{"tpEvento with a path", testKey, "../../x", "1", "invalid tpEvento"},
		{"short tpEvento", testKey, "11011", "1", "invalid tpEvento"},
		{"missing nSeqEvento", testKey, "110111", "", "missing essential field: nSeqEvento"},
		{"zero nSeqEvento", testKey, "110111", "0", "invalid nSeqEvento"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseEventIdentity("CTe", tt.chave, tt.tpEvento, tt.nSeq)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}
