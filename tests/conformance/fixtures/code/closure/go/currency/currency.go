// Package currency names the currencies the shop takes.
package currency

import (
	"errors"

	"example.com/shop/rates"
)

// Code is a currency's ISO code.
type Code string

// None is an amount's currency before its first line.
const None Code = ""

// Parse reads a currency's code.
func Parse(s string) (Code, error) {
	if !rates.Known(s) {
		return None, errors.New("currency body: unknown " + s)
	}
	return Code(s), nil
}
