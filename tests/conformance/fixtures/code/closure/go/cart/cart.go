// Package cart holds an order's lines.
package cart

import "example.com/shop/money"

// Cart holds the lines of an order.
type Cart struct {
	Lines []money.Amount
}

// Total adds the cart's lines.
func (c Cart) Total() money.Amount {
	sum := money.Zero()
	for _, l := range c.Lines {
		sum = sum.Plus(l)
	}
	return sum
}
