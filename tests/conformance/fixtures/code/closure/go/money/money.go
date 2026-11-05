// Package money adds amounts of one currency.
package money

import (
	"fmt"

	"example.com/shop/currency"
)

// Amount is a sum in cents of one currency.
type Amount struct {
	Cents int64
	Unit  currency.Code
}

// Zero is no money, in no currency yet.
func Zero() Amount {
	return Amount{Unit: currency.None} // money body: zero
}

// Plus adds two amounts of the same currency.
func (a Amount) Plus(b Amount) Amount {
	if a.Unit == currency.None {
		a.Unit = b.Unit
	}
	if a.Unit != b.Unit {
		panic(fmt.Sprintf("money body: %s plus %s", a.Unit, b.Unit))
	}
	return Amount{Cents: a.Cents + b.Cents, Unit: a.Unit}
}
