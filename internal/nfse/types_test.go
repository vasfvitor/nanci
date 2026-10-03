package nfse_test

import (
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

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
