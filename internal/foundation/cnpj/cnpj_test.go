package cnpj

import (
	"errors"
	"strconv"
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
		{"12.abc.345/01de-35", "12ABC34501DE35"},
		{" 12abc34501de35 ", "12ABC34501DE35"},
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
		{"12.345.678/0001", ErrInvalidLength},
		{"123456780001999", ErrInvalidLength},
		{"12.345.678/0001-9@", ErrInvalidFormat},

		// Alfanumérico (IN RFB 2.229/2024). O exemplo da Receita Federal é
		// 12.ABC.345/01DE-35; a aritmética está detalhada em
		// TestCheckDigitsForAlphanumericExample.
		{"12.ABC.345/01DE-35", nil},
		{"12ABC34501DE35", nil},
		{"12.abc.345/01de-35", nil},
		{"ZZ.999.AB1/CD00-06", nil},
		{"A1.B2C.3D4/E5F6-68", nil},

		// DV alfanumérico incorreto: primeiro dígito errado, depois o segundo.
		{"12.ABC.345/01DE-45", ErrInvalidCheckDigits},
		{"12.ABC.345/01DE-36", ErrInvalidCheckDigits},

		// Os dois últimos caracteres precisam ser numéricos mesmo em CNPJ alfanumérico.
		{"AB.CDE.FGH/IJKL-MN", ErrNonNumericCheckDigit},
		{"12.ABC.345/01DE-3E", ErrNonNumericCheckDigit},
		{"12.ABC.345/01DE-E5", ErrNonNumericCheckDigit},
	}

	for _, tt := range tests {
		actual := Validate(tt.input)
		if !errors.Is(actual, tt.expected) {
			t.Errorf("Validate(%q) = %v; expected %v", tt.input, actual, tt.expected)
		}
	}
}

// TestCheckDigitsForAlphanumericExample documenta a conta do DV alfanumérico
// para a base 12ABC34501DE, conferida à mão.
//
// Cada caractere vale seu código ASCII menos 48, então '0'..'9' valem 0..9 e
// 'A'..'Z' valem 17..42:
//
//	1  2  A   B   C   3  4  5  0  1  D   E
//	1  2  17  18  19  3  4  5  0  1  20  21
//
// DV1, pesos 5,4,3,2,9,8,7,6,5,4,3,2:
//
//	1*5 + 2*4 + 17*3 + 18*2 + 19*9 + 3*8 + 4*7 + 5*6 + 0*5 + 1*4 + 20*3 + 21*2
//	= 5 + 8 + 51 + 36 + 171 + 24 + 28 + 30 + 0 + 4 + 60 + 42 = 459
//	459 % 11 = 8 (11*41 = 451); 8 >= 2, então DV1 = 11 - 8 = 3.
//
// DV2 sobre 12ABC34501DE3, pesos 6,5,4,3,2,9,8,7,6,5,4,3,2:
//
//	1*6 + 2*5 + 17*4 + 18*3 + 19*2 + 3*9 + 4*8 + 5*7 + 0*6 + 1*5 + 20*4 + 21*3 + 3*2
//	= 6 + 10 + 68 + 54 + 38 + 27 + 32 + 35 + 0 + 5 + 80 + 63 + 6 = 424
//	424 % 11 = 6 (11*38 = 418); 6 >= 2, então DV2 = 11 - 6 = 5.
//
// Logo o CNPJ completo é 12.ABC.345/01DE-35.
func TestCheckDigitsForAlphanumericExample(t *testing.T) {
	const base = "12ABC34501DE"

	first := calculateCheckDigit(base, firstCheckDigitWeights)
	if first != 3 {
		t.Fatalf("primeiro DV de %q = %d; esperado 3", base, first)
	}

	second := calculateCheckDigit(base+strconv.Itoa(first), secondCheckDigitWeights)
	if second != 5 {
		t.Fatalf("segundo DV de %q = %d; esperado 5", base, second)
	}
}

func TestRoot(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{"45.723.174/0001-10", "45723174", false},
		{"12.ABC.345/01DE-35", "12ABC345", false},
		{"12.abc.345/01de-35", "12ABC345", false},
		{"AB.CDE.FGH/IJKL-MN", "", true},
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
		{"12ABC34501DE35", "12.ABC.345/01DE-35"},
		{"12abc34501de35", "12.ABC.345/01DE-35"},
		{"12.ABC.345/01DE-35", "12.ABC.345/01DE-35"},
		{"123", "123"}, // Retorna original se inválido
	}

	for _, tt := range tests {
		actual := Format(tt.input)
		if actual != tt.expected {
			t.Errorf("Format(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}
