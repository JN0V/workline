package cart

import "testing"

func TestEmptyCart(t *testing.T) {
	if (Cart{}).Total().Cents != 0 {
		t.Fatal("an empty cart costs nothing")
	}
}
