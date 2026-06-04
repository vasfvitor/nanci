package cnpj

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// cnpjRegex checks if the string has exactly 14 alphanumeric characters.
	cnpjRegex = regexp.MustCompile(`^[a-zA-Z0-9]{14}$`)

	ErrInvalidLength = errors.New("CNPJ deve ter 14 caracteres")
	ErrInvalidFormat = errors.New("CNPJ deve conter apenas letras e números")
)

// Clean removes punctuation from the CNPJ (dots, slash, dash).
func Clean(cnpj string) string {
	cnpj = strings.ReplaceAll(cnpj, ".", "")
	cnpj = strings.ReplaceAll(cnpj, "/", "")
	cnpj = strings.ReplaceAll(cnpj, "-", "")
	return strings.TrimSpace(cnpj)
}

// Validate checks if the CNPJ has 14 alphanumeric characters.
// The new RFB CNPJ format is alphanumeric, so we focus on length
// and allowed characters.
func Validate(cnpj string) error {
	cleaned := Clean(cnpj)
	if len(cleaned) != 14 {
		return ErrInvalidLength
	}
	if !cnpjRegex.MatchString(cleaned) {
		return ErrInvalidFormat
	}
	return nil
}

// Root extracts the first 8 characters from the CNPJ (Base/Root CNPJ).
func Root(cnpj string) (string, error) {
	cleaned := Clean(cnpj)
	if err := Validate(cleaned); err != nil {
		return "", err
	}
	return cleaned[:8], nil
}

// Format applies the XX.XXX.XXX/XXXX-XX mask to the CNPJ.
func Format(cnpj string) string {
	cleaned := Clean(cnpj)
	if len(cleaned) != 14 {
		return cnpj
	}
	return cleaned[:2] + "." + cleaned[2:5] + "." + cleaned[5:8] + "/" + cleaned[8:12] + "-" + cleaned[12:14]
}
