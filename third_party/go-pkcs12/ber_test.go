package pkcs12

import (
	"bytes"
	"testing"
)

func TestNormalizeBERNestedAuthenticatedSafe(t *testing.T) {
	input := []byte{
		0x30, 0x80,
		0x30, 0x80,
		0xa0, 0x80,
		0x24, 0x80,
		0x04, 0x02, 0x01, 0x02,
		0x04, 0x01, 0x03,
		0x00, 0x00,
		0x00, 0x00,
		0x00, 0x00,
		0x00, 0x00,
	}
	want := []byte{
		0x30, 0x09,
		0x30, 0x07,
		0xa0, 0x05,
		0x04, 0x03, 0x01, 0x02, 0x03,
	}

	got, err := normalizeBER(input)
	if err != nil {
		t.Fatalf("normalizeBER failed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("normalizeBER() = %x, want %x", got, want)
	}
}

func TestNormalizeBERMalformedDoesNotPanic(t *testing.T) {
	inputs := [][]byte{
		{0x30},
		{0x30, 0x81},
		{0x30, 0x80},
		{0x24, 0x80, 0x02, 0x00, 0x00, 0x00},
	}

	for _, input := range inputs {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("normalizeBER(%x) panicked: %v", input, recovered)
				}
			}()
			if _, err := normalizeBER(input); err == nil {
				t.Fatalf("normalizeBER(%x) returned no error", input)
			}
		}()
	}
}
