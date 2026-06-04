package cnpj

import (
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12.345.678/0001-99", "12345678000199"},
		{" 12.345.678/0001-99 ", "12345678000199"},
		{"AB.CDE.FGH/IJKL-MN", "ABCDEFGHIJKLMN"},
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
		{"12.345.678/0001-99", nil},
		{"12345678000199", nil},
		{"AB.CDE.FGH/IJKL-MN", nil}, // Alfanumérico válido
		{"12.345.678/0001", ErrInvalidLength},
		{"123456780001999", ErrInvalidLength},
		{"12.345.678/0001-9@", ErrInvalidFormat}, // Caractere especial
	}

	for _, tt := range tests {
		actual := Validate(tt.input)
		if actual != tt.expected {
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
		{"12.345.678/0001-99", "12345678", false},
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
		{"12345678000199", "12.345.678/0001-99"},
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
