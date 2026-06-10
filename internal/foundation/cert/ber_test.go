package cert

import (
	"bytes"
	"encoding/asn1"
	"os"
	"strings"
	"testing"
)

func TestBer2Der(t *testing.T) {
	tests := []struct {
		name    string
		ber     []byte
		wantDER []byte
		wantErr bool
	}{
		{
			name:    "empty input",
			wantErr: true,
		},
		{
			name:    "simple DER sequence",
			ber:     []byte{0x30, 0x03, 0x02, 0x01, 0x05},
			wantDER: []byte{0x30, 0x03, 0x02, 0x01, 0x05},
		},
		{
			name:    "empty indefinite sequence",
			ber:     []byte{0x30, 0x80, 0x00, 0x00},
			wantDER: []byte{0x30, 0x00},
		},
		{
			name: "nested indefinite sequences",
			ber: []byte{
				0x30, 0x80,
				0x30, 0x80,
				0x02, 0x01, 0x05,
				0x00, 0x00,
				0x00, 0x00,
			},
			wantDER: []byte{
				0x30, 0x05,
				0x30, 0x03,
				0x02, 0x01, 0x05,
			},
		},
		{
			name: "constructed OCTET STRING",
			ber: []byte{
				0x30, 0x80,
				0x24, 0x80,
				0x04, 0x02, 0x01, 0x02,
				0x04, 0x01, 0x03,
				0x00, 0x00,
				0x00, 0x00,
			},
			wantDER: []byte{
				0x30, 0x05,
				0x04, 0x03, 0x01, 0x02, 0x03,
			},
		},
		{
			name:    "indefinite primitive",
			ber:     []byte{0x30, 0x03, 0x02, 0x80, 0x00},
			wantErr: true,
		},
		{
			name:    "constructed OCTET STRING with invalid child",
			ber:     []byte{0x30, 0x05, 0x24, 0x03, 0x02, 0x01, 0x01},
			wantErr: true,
		},
		{
			name:    "unsupported constructed primitive",
			ber:     []byte{0x30, 0x02, 0x23, 0x00},
			wantErr: true,
		},
		{
			name:    "trailing data",
			ber:     []byte{0x30, 0x00, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDER, err := ber2der(tt.ber)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ber2der() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !bytes.Equal(gotDER, tt.wantDER) {
				t.Fatalf("ber2der() = %x, want %x", gotDER, tt.wantDER)
			}
		})
	}
}

func TestBer2DerMalformedInputDoesNotPanic(t *testing.T) {
	tests := [][]byte{
		{0x30},
		{0x30, 0x81},
		{0x30, 0x82, 0x01},
		{0x30, 0x03, 0x02, 0x02, 0x01},
		{0x30, 0x80},
		{0x30, 0x80, 0x02, 0x01, 0x01},
		{0x30, 0x02, 0x1f, 0x80},
	}

	for _, input := range tests {
		input := input
		t.Run(strings.ToUpper(fmtHex(input)), func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("ber2der(%x) panicked: %v", input, recovered)
				}
			}()
			if _, err := ber2der(input); err == nil {
				t.Fatalf("ber2der(%x) returned no error", input)
			}
		})
	}
}

func TestBer2DerRejectsExcessiveNesting(t *testing.T) {
	input := []byte{0x30, 0x00}
	for range maxASN1Depth + 1 {
		input = append([]byte{0x30, 0x80}, append(input, 0x00, 0x00)...)
	}

	if _, err := ber2der(input); err == nil {
		t.Fatal("ber2der() returned no error for excessive nesting")
	}
}

func TestNormalizePKCS12BER_ExternalStructure(t *testing.T) {
	path := os.Getenv("NANCI_TEST_PFX_PATH")
	if path == "" {
		t.Skip("set NANCI_TEST_PFX_PATH to test an external certificate structure")
	}

	ber, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	der, changed, err := normalizePKCS12BER(ber)
	if err != nil {
		t.Fatalf("normalizePKCS12BER failed: %v", err)
	}
	if !changed {
		t.Fatal("expected external BER certificate to require normalization")
	}

	var root asn1.RawValue
	rest, err := asn1.Unmarshal(der, &root)
	if err != nil {
		t.Fatalf("normalized certificate is not valid DER: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("normalized certificate has %d trailing bytes", len(rest))
	}
	if root.Tag != asn1.TagSequence || !root.IsCompound {
		t.Fatal("normalized certificate root is not a SEQUENCE")
	}
}

func fmtHex(input []byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(input)*2)
	for i, b := range input {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out)
}
