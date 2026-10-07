package nfse_test

import (
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// alphaChave is an NFS-e chave whose prestador has the alphanumeric CNPJ of
// the NT 2025.001 example, 12.ABC.345/01DE-35: cMun 3550308, ambGer 1,
// tpInsc 2, the CNPJ, nNFSe, AAMM 2608, cNum and a DV that nanci does not
// verify.
const alphaChave = "3550308" + "1" + "2" + "12ABC34501DE35" + "0000000000123" + "2608" + "123456789" + "7"

func TestParseAccessKey(t *testing.T) {
	digits := strings.Repeat("1", 50)
	tests := []struct {
		name    string
		raw     string
		want    nfse.AccessKey
		wantErr bool
	}{
		{name: "numeric", raw: digits, want: nfse.AccessKey(digits)},
		{name: "alphanumeric CNPJ in the inscrição federal", raw: alphaChave, want: alphaChave},
		{name: "lowercase and spaces are normalized", raw: "  " + strings.ToLower(alphaChave) + " ", want: alphaChave},
		{name: "letter in the first inscrição position", raw: digits[:9] + "A" + digits[10:], want: nfse.AccessKey(digits[:9] + "A" + digits[10:])},
		{name: "letter in the last inscrição position", raw: digits[:22] + "Z" + digits[23:], want: nfse.AccessKey(digits[:22] + "Z" + digits[23:])},
		{name: "letter in the tipo de inscrição", raw: digits[:8] + "A" + digits[9:], wantErr: true},
		{name: "letter in the número", raw: digits[:23] + "A" + digits[24:], wantErr: true},
		{name: "letter in the DV", raw: digits[:49] + "A", wantErr: true},
		{name: "punctuation", raw: digits[:9] + "-" + digits[10:], wantErr: true},
		{name: "49 characters", raw: digits[:49], wantErr: true},
		{name: "44-character NF-e key", raw: "35260911222333000181550010000012341123456787", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nfse.ParseAccessKey(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAccessKey(%q) = %q, want error", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAccessKey(%q): %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseAccessKey(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseInfNFSeID(t *testing.T) {
	digits := strings.Repeat("1", 50)
	tests := []struct {
		name    string
		id      string
		want    nfse.AccessKey
		wantErr bool
	}{
		{name: "prefixed", id: nfse.InfNFSeIDPrefix + digits, want: nfse.AccessKey(digits)},
		{name: "bare chave", id: digits, want: nfse.AccessKey(digits)},
		{name: "prefixed alphanumeric CNPJ", id: nfse.InfNFSeIDPrefix + alphaChave, want: alphaChave},
		{name: "prefix only", id: nfse.InfNFSeIDPrefix, wantErr: true},
		{name: "short", id: nfse.InfNFSeIDPrefix + digits[:49], wantErr: true},
		{name: "non-digit", id: nfse.InfNFSeIDPrefix + digits[:49] + "X", wantErr: true},
		{name: "other prefix", id: "NFE" + digits, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nfse.ParseInfNFSeID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseInfNFSeID(%q) = %q, want error", tt.id, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInfNFSeID(%q): %v", tt.id, err)
			}
			if got != tt.want {
				t.Errorf("ParseInfNFSeID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
