package cnpj

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	cnpjRegex = regexp.MustCompile(`^[A-Z0-9]{14}$`)

	ErrInvalidLength        = errors.New("CNPJ deve ter 14 caracteres")
	ErrInvalidFormat        = errors.New("CNPJ deve conter apenas letras e números")
	ErrNonNumericCheckDigit = errors.New("os dois últimos caracteres do CNPJ devem ser numéricos")
	ErrInvalidCheckDigits   = errors.New("CNPJ com dígitos verificadores inválidos")
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

// Validate checks the syntax and the two check digits of a CNPJ.
// Numeric and alphanumeric CNPJs are both accepted, following the rule from
// Instrução Normativa RFB 2.229/2024: the first 12 characters may be letters or
// digits and the last 2 are always numeric check digits.
func Validate(cnpj string) error {
	cleaned := Clean(cnpj)
	if err := validateSyntax(cleaned); err != nil {
		return err
	}
	if !hasValidCheckDigits(cleaned) {
		return ErrInvalidCheckDigits
	}
	return nil
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
	if !isDigit(cleaned[12]) || !isDigit(cleaned[13]) {
		return ErrNonNumericCheckDigit
	}
	return nil
}

// hasValidCheckDigits recomputes both check digits and compares them with the
// ones carried by cleaned, which must already have passed validateSyntax.
func hasValidCheckDigits(cleaned string) bool {
	if allSameCharacters(cleaned) {
		return false
	}
	first := calculateCheckDigit(cleaned[:12], firstCheckDigitWeights)
	second := calculateCheckDigit(cleaned[:12]+strconv.Itoa(first), secondCheckDigitWeights)
	return int(cleaned[12]-'0') == first && int(cleaned[13]-'0') == second
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func allSameCharacters(cleaned string) bool {
	for i := 1; i < len(cleaned); i++ {
		if cleaned[i] != cleaned[0] {
			return false
		}
	}
	return true
}

// calculateCheckDigit applies the mod-11 rule over the character values defined
// by the tax authority: each character is worth its ASCII code minus 48, so
// '0'..'9' are worth 0..9 and 'A'..'Z' are worth 17..42. Numeric CNPJs are just
// the special case where every character is a digit.
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
