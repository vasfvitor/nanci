package cnpj

import (
	"errors"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12.345.678/0001-99", "12345678000199"},
		{" 12.345.678/0001-99 ", "12345678000199"},
		{"ab.cde.fgh/ijkl-mn", "ABCDEFGHIJKLMN"},
		{"12345678000199", "12345678000199"},
	}

	for _, tt := range tests {
		actual := Clean(tt.input)
		if actual != tt.expected {
			t.Errorf("Clean(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		input    string
		expected error
	}{
		{"45.723.174/0001-10", nil},
		{"45723174000110", nil},
		{"12.345.678/0001-99", ErrInvalidCheckDigits},
		{"11.111.111/1111-11", ErrInvalidCheckDigits},
		{"AB.CDE.FGH/IJKL-MN", ErrAlphanumericUnsupported},
		{"12.345.678/0001", ErrInvalidLength},
		{"123456780001999", ErrInvalidLength},
		{"12.345.678/0001-9@", ErrInvalidFormat},
	}

	for _, tt := range tests {
		actual := Validate(tt.input)
		if !errors.Is(actual, tt.expected) {
			t.Errorf("Validate(%q) = %v; expected %v", tt.input, actual, tt.expected)
		}
	}
}

func TestRoot(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{"45.723.174/0001-10", "45723174", false},
		{"AB.CDE.FGH/IJKL-MN", "ABCDEFGH", false},
		{"12.345.678/0001", "", true},
	}

	for _, tt := range tests {
		actual, err := Root(tt.input)
		if (err != nil) != tt.expectError {
			t.Errorf("Root(%q) expected error: %v, got: %v", tt.input, tt.expectError, err)
		}
		if actual != tt.expected {
			t.Errorf("Root(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"45723174000110", "45.723.174/0001-10"},
		{"ABCDEFGHIJKLMN", "AB.CDE.FGH/IJKL-MN"},
		{"123", "123"}, // Retorna original se inválido
	}

	for _, tt := range tests {
		actual := Format(tt.input)
		if actual != tt.expected {
			t.Errorf("Format(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}
