package cnpj

import (
	"errors"
	"regexp"
	"strings"
)

var (
	cnpjRegex = regexp.MustCompile(`^[A-Z0-9]{14}$`)

	ErrInvalidLength           = errors.New("CNPJ deve ter 14 caracteres")
	ErrInvalidFormat           = errors.New("CNPJ deve conter apenas letras e números")
	ErrInvalidCheckDigits      = errors.New("CNPJ numérico com dígitos verificadores inválidos")
	ErrAlphanumericUnsupported = errors.New("CNPJ alfanumérico ainda não é suportado nesta versão")
)

var (
	firstCheckDigitWeights  = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	secondCheckDigitWeights = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

// Clean removes punctuation from the CNPJ (dots, slash, dash) and normalizes letters to uppercase.
func Clean(cnpj string) string {
	cnpj = strings.ReplaceAll(cnpj, ".", "")
	cnpj = strings.ReplaceAll(cnpj, "/", "")
	cnpj = strings.ReplaceAll(cnpj, "-", "")
	return strings.ToUpper(strings.TrimSpace(cnpj))
}

// Validate enforces the current rollout policy:
// numeric CNPJ must pass DV validation; alphanumeric identifiers are rejected
// until official support is implemented end to end.
func Validate(cnpj string) error {
	cleaned := Clean(cnpj)
	if err := validateSyntax(cleaned); err != nil {
		return err
	}
	if isNumeric(cleaned) {
		if !isValidNumericCNPJ(cleaned) {
			return ErrInvalidCheckDigits
		}
		return nil
	}
	return ErrAlphanumericUnsupported
}

// Root extracts the first 8 characters from a syntactically valid CNPJ token.
func Root(cnpj string) (string, error) {
	cleaned := Clean(cnpj)
	if err := validateSyntax(cleaned); err != nil {
		return "", err
	}
	return cleaned[:8], nil
}

// Format applies the XX.XXX.XXX/XXXX-XX mask to the CNPJ token.
func Format(cnpj string) string {
	cleaned := Clean(cnpj)
	if len(cleaned) != 14 {
		return cnpj
	}
	return cleaned[:2] + "." + cleaned[2:5] + "." + cleaned[5:8] + "/" + cleaned[8:12] + "-" + cleaned[12:14]
}

func validateSyntax(cleaned string) error {
	if len(cleaned) != 14 {
		return ErrInvalidLength
	}
	if !cnpjRegex.MatchString(cleaned) {
		return ErrInvalidFormat
	}
	return nil
}

func isNumeric(cleaned string) bool {
	for _, r := range cleaned {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isValidNumericCNPJ(cleaned string) bool {
	if len(cleaned) != 14 {
		return false
	}
	if allSameDigits(cleaned) {
		return false
	}
	first := calculateCheckDigit(cleaned[:12], firstCheckDigitWeights)
	second := calculateCheckDigit(cleaned[:12]+string(rune('0'+first)), secondCheckDigitWeights)
	return int(cleaned[12]-'0') == first && int(cleaned[13]-'0') == second
}

func allSameDigits(cleaned string) bool {
	for i := 1; i < len(cleaned); i++ {
		if cleaned[i] != cleaned[0] {
			return false
		}
	}
	return true
}

func calculateCheckDigit(base string, weights []int) int {
	sum := 0
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * weights[i]
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}
