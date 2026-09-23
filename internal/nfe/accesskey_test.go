package nfe

import (
	"errors"
	"testing"
)

func TestParseAccessKey(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    AccessKey
		wantErr bool
	}{
		{name: "numeric", raw: "35260911222333000181550010000012341123456787", want: "35260911222333000181550010000012341123456787"},
		{name: "numeric with remainder below 2", raw: "35260911222333000181550010000012361345678900", want: "35260911222333000181550010000012361345678900"},
		{name: "real sample", raw: "35170349607369000156550010000229481398694060", want: "35170349607369000156550010000229481398694060"},
		{name: "alphanumeric CNPJ", raw: "41260912ABC34501DE35550020000000771456789019", want: "41260912ABC34501DE35550020000000771456789019"},
		{name: "lowercase and spaces are normalized", raw: "  41260912abc34501de35550020000000771456789019 ", want: "41260912ABC34501DE35550020000000771456789019"},
		{name: "bad check digit", raw: "35260911222333000181550010000012341123456788", wantErr: true},
		{name: "alphanumeric bad check digit", raw: "41260912ABC34501DE35550020000000771456789018", wantErr: true},
		{name: "letter outside the CNPJ slot", raw: "35260911222333000181550010000012341A23456787", wantErr: true},
		// Key check digit is right, but the alphanumeric CNPJ in the slot is not.
		{name: "letters in slot with invalid CNPJ", raw: "41260912ABC34501DE36550020000000771456789010", wantErr: true},
		{name: "43 characters", raw: "3526091122233300018155001000001234112345678", wantErr: true},
		{name: "50-digit NFS-e key", raw: "12345678901234567890123456789012345678901234567890", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAccessKey(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidAccessKey) {
					t.Fatalf("ParseAccessKey(%q) error = %v, want ErrInvalidAccessKey", tt.raw, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAccessKey(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseAccessKey(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestAccessKeyAccessors(t *testing.T) {
	key, err := ParseAccessKey("41260912ABC34501DE35550020000000771456789019")
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string][2]string{
		"UFCode":       {key.UFCode(), "41"},
		"UF":           {key.UF(), "PR"},
		"EmitenteCNPJ": {key.EmitenteCNPJ(), "12ABC34501DE35"},
		"Modelo":       {key.Modelo(), "55"},
		"Serie":        {key.Serie(), "002"},
		"Numero":       {key.Numero(), "000000077"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s() = %q, want %q", name, c[0], c[1])
		}
	}

	var empty AccessKey
	if empty.UF() != "" || empty.Numero() != "" {
		t.Errorf("accessors on an empty key should return empty strings")
	}
}
