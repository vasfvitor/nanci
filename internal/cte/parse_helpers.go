package cte

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// addMoneyInto parses an XSD decimal and adds it to dst, naming field in the
// error.
func addMoneyInto(dst *dfe.Money, field, value string) error {
	var m dfe.Money
	if err := dfe.ParseMoneyInto(&m, field, value); err != nil {
		return err
	}
	sum, err := dst.Add(m)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	*dst = sum
	return nil
}
