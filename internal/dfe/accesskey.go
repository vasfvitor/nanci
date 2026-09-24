// Package dfe holds the vocabulary shared by the documentos fiscais
// eletrônicos distributed by SEFAZ (NF-e, CT-e): the 44-character access key,
// the value types their domains have in common and the helpers their XML
// parsers share. It has no network or database code.
package dfe

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/foundation/uf"
)

// ErrInvalidAccessKey is returned by ParseAccessKey. The message carries the
// specific problem; match with errors.Is.
var ErrInvalidAccessKey = errors.New("invalid access key")

// AccessKey is the 44-character chave de acesso shared by NF-e and CT-e:
//
//	cUF(2) AAMM(4) CNPJ(14) mod(2) serie(3) nNF(9) tpEmis(1) cNF(8) cDV(1)
//
// Every position is a digit except the CNPJ slot, which also accepts
// uppercase letters for the alphanumeric CNPJ (NT 2025.001). The model is not
// validated here: each document parser checks its own.
type AccessKey string

const (
	accessKeyLength = 44
	cnpjSlotStart   = 6  // 0-based index of the first CNPJ character
	cnpjSlotEnd     = 20 // 0-based index after the last CNPJ character
)

// ParseAccessKey trims and upper-cases raw, checks the character set of each
// position and verifies the mod-11 check digit. Characters are worth their
// ASCII code minus 48, the same rule foundation/cnpj uses, so an all-digit key
// is just the special case of the alphanumeric one.
//
// A CNPJ slot with letters must hold a valid alphanumeric CNPJ. An all-digit
// slot is not checked further because it may be a zero-padded CPF.
func ParseAccessKey(raw string) (AccessKey, error) {
	key := strings.ToUpper(strings.TrimSpace(raw))
	if len(key) != accessKeyLength {
		return "", fmt.Errorf("%w: must have %d characters, got %d", ErrInvalidAccessKey, accessKeyLength, len(key))
	}
	for i := 0; i < len(key); i++ {
		c := key[i]
		inCNPJSlot := i >= cnpjSlotStart && i < cnpjSlotEnd
		if isDigit(c) || (inCNPJSlot && isLetter(rune(c))) {
			continue
		}
		return "", fmt.Errorf("%w: invalid character %q at position %d", ErrInvalidAccessKey, c, i+1)
	}
	slot := key[cnpjSlotStart:cnpjSlotEnd]
	if strings.ContainsFunc(slot, isLetter) {
		if err := cnpj.Validate(slot); err != nil {
			return "", fmt.Errorf("%w: emitente CNPJ: %w", ErrInvalidAccessKey, err)
		}
	}
	if want := accessKeyCheckDigit(key[:accessKeyLength-1]); int(key[accessKeyLength-1]-'0') != want {
		return "", fmt.Errorf("%w: check digit should be %d", ErrInvalidAccessKey, want)
	}
	return AccessKey(key), nil
}

// accessKeyCheckDigit applies the mod-11 rule with weights 2..9 repeating
// from the rightmost character. Remainders 0 and 1 give check digit 0.
func accessKeyCheckDigit(base string) int {
	sum := 0
	weight := 2
	for i := len(base) - 1; i >= 0; i-- {
		sum += int(base[i]-'0') * weight
		weight++
		if weight > 9 {
			weight = 2
		}
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLetter(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func (k AccessKey) String() string {
	return string(k)
}

// UFCode returns the IBGE code of the issuing state (cUF), e.g. "35".
func (k AccessKey) UFCode() string {
	return k.slice(0, 2)
}

// UF returns the state abbreviation of the issuing state, e.g. "SP", or an
// empty string when the code is unknown.
func (k AccessKey) UF() string {
	code, err := strconv.Atoi(k.UFCode())
	if err != nil {
		return ""
	}
	sigla, _ := uf.Sigla(code)
	return sigla
}

// EmitenteCNPJ returns the CNPJ slot. For an emitente identified by CPF the
// slot holds the CPF left-padded with zeros.
func (k AccessKey) EmitenteCNPJ() string {
	return k.slice(cnpjSlotStart, cnpjSlotEnd)
}

// Modelo returns the document model: "55" for NF-e, "57" for CT-e, "64" for
// GTV-e and "67" for CT-e OS.
func (k AccessKey) Modelo() string {
	return k.slice(20, 22)
}

// Serie returns the series as written in the key, zero-padded to 3 digits.
func (k AccessKey) Serie() string {
	return k.slice(22, 25)
}

// Numero returns the document number as written in the key, zero-padded to 9
// digits.
func (k AccessKey) Numero() string {
	return k.slice(25, 34)
}

func (k AccessKey) slice(start, end int) string {
	if len(k) != accessKeyLength {
		return ""
	}
	return string(k[start:end])
}
