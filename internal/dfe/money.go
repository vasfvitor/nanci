package dfe

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Money represents monetary values in integer cents (e.g., 1000 = R$ 10,00).
// The XSD allows 15 integer digits plus exactly two decimals, which safely fits in int64.
type Money int64

var (
	ErrInvalidMoneyFormat = errors.New("invalid money format")
	ErrMoneyOverflow      = errors.New("money overflow")
	// XML monetary format: optional decimals, no thousands separators, exactly two decimals if present
	moneyRegex = regexp.MustCompile(`^[0-9]{1,15}(\.[0-9]{1,2})?$`)
)

// ParseMoney parses a string in XSD decimal format into Money.
func ParseMoney(value string) (Money, error) {
	value = strings.TrimSpace(value)
	if !moneyRegex.MatchString(value) {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMoneyFormat, value)
	}

	parts := strings.Split(value, ".")
	integerPart := parts[0]
	fractionalPart := "00"

	if len(parts) == 2 {
		fractionalPart = parts[1]
		if len(fractionalPart) == 1 {
			fractionalPart += "0" // pad to two decimals
		}
	}

	centsStr := integerPart + fractionalPart
	cents, err := strconv.ParseInt(centsStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrMoneyOverflow, err)
	}

	return Money(cents), nil
}

// NewMoneyFromCents creates a Money value from raw cents.
func NewMoneyFromCents(cents int64) Money {
	return Money(cents)
}

// Cents returns the integer cents value.
func (m Money) Cents() int64 {
	return int64(m)
}

// Add adds two Money values.
func (m Money) Add(other Money) (Money, error) {
	// Simple overflow check for int64 addition
	result := m.Cents() + other.Cents()
	if (result > m.Cents()) != (other.Cents() > 0) && other.Cents() != 0 {
		return 0, ErrMoneyOverflow
	}
	return Money(result), nil
}

// Sub subtracts another Money value.
func (m Money) Sub(other Money) (Money, error) {
	// Simple overflow check for int64 subtraction
	result := m.Cents() - other.Cents()
	if (result < m.Cents()) != (other.Cents() > 0) && other.Cents() != 0 {
		return 0, ErrMoneyOverflow
	}
	return Money(result), nil
}

// FormatBRL formats the Money value into Brazilian Real representation (e.g., "1.234,56").
func (m Money) FormatBRL() string {
	cents := m.Cents()
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}

	fractional := cents % 100
	integer := cents / 100

	intStr := strconv.FormatInt(integer, 10)
	var formattedInt strings.Builder
	for i, c := range intStr {
		if i > 0 && (len(intStr)-i)%3 == 0 {
			formattedInt.WriteRune('.')
		}
		formattedInt.WriteRune(c)
	}

	return fmt.Sprintf("%s%s,%02d", sign, formattedInt.String(), fractional)
}
