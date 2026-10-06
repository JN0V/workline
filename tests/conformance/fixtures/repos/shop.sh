#!/bin/sh
# A small Go shop on main, with no known defect and a test for each
# function: the reviewer's evaluation plants one change on it a case
# (tests/evaluation/cases/reviewer), each on a branch `feature` of its own.
set -eu
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
git init -q -b main .
git config user.name "Fixture"
git config user.email "fixture@example.invalid"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z" GIT_COMMITTER_DATE="2026-01-01T00:00:00Z"
mkdir -p shop
printf 'module example.com/shop\n\ngo 1.22\n' > go.mod
cat > shop/cart.go <<'GO'
package shop

// Item is a line of a cart: a product, its unit price in cents, and how
// many of it.
type Item struct {
	SKU   string
	Price int
	Qty   int
}

// Total is what the cart's items cost, in cents.
func Total(items []Item) int {
	sum := 0
	for _, it := range items {
		sum += it.Price * it.Qty
	}
	return sum
}

// Discount takes percent off a total, rounded down to the cent; percent
// is held between 0 and 100.
func Discount(total, percent int) int {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return total - total*percent/100
}
GO
cat > shop/cart_test.go <<'GO'
package shop

import "testing"

func TestTotal(t *testing.T) {
	items := []Item{{"tea", 450, 2}, {"cup", 1200, 1}}
	if got := Total(items); got != 2100 {
		t.Errorf("Total = %d, want 2100", got)
	}
	if got := Total(nil); got != 0 {
		t.Errorf("Total(nil) = %d, want 0", got)
	}
}

func TestDiscount(t *testing.T) {
	for _, c := range []struct{ total, percent, want int }{
		{1000, 10, 900}, {1000, 0, 1000}, {1000, 100, 0}, {1000, 150, 0}, {1000, -5, 1000},
	} {
		if got := Discount(c.total, c.percent); got != c.want {
			t.Errorf("Discount(%d, %d) = %d, want %d", c.total, c.percent, got, c.want)
		}
	}
}
GO
cat > shop/coupon.go <<'GO'
package shop

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseCoupon reads a coupon such as "SAVE15": the percent it takes off,
// from 1 to 90.
func ParseCoupon(code string) (int, error) {
	rest, ok := strings.CutPrefix(code, "SAVE")
	if !ok {
		return 0, fmt.Errorf("coupon %q: not a SAVE code", code)
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, fmt.Errorf("coupon %q: %w", code, err)
	}
	if n < 1 || n > 90 {
		return 0, fmt.Errorf("coupon %q: %d%% is out of range", code, n)
	}
	return n, nil
}
GO
cat > shop/coupon_test.go <<'GO'
package shop

import "testing"

func TestParseCoupon(t *testing.T) {
	if n, err := ParseCoupon("SAVE15"); err != nil || n != 15 {
		t.Errorf("SAVE15 = %d, %v", n, err)
	}
	for _, bad := range []string{"", "SAVE", "SAVEx", "SAVE0", "SAVE91", "GIFT10"} {
		if _, err := ParseCoupon(bad); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}
GO
cat > shop/stock.go <<'GO'
package shop

import "sync"

// Stock counts the units left of each product; it is safe for concurrent
// use.
type Stock struct {
	mu   sync.Mutex
	left map[string]int
}

// NewStock is an empty stock.
func NewStock() *Stock {
	return &Stock{left: map[string]int{}}
}

// Add puts n units of sku in stock; n of zero or less adds nothing.
func (s *Stock) Add(sku string, n int) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.left[sku] += n
}

// Take removes n units of sku, and says false, removing nothing, when
// fewer are left.
func (s *Stock) Take(sku string, n int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 || s.left[sku] < n {
		return false
	}
	s.left[sku] -= n
	return true
}

// Left is how many units of sku are left.
func (s *Stock) Left(sku string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.left[sku]
}
GO
cat > shop/stock_test.go <<'GO'
package shop

import "testing"

func TestStock(t *testing.T) {
	s := NewStock()
	s.Add("tea", 3)
	s.Add("tea", -1)
	if !s.Take("tea", 2) || s.Left("tea") != 1 {
		t.Errorf("taking two of three: left %d", s.Left("tea"))
	}
	if s.Take("tea", 2) || s.Take("cup", 1) || s.Take("tea", 0) {
		t.Error("took what is not there")
	}
}
GO
cat > shop/page.go <<'GO'
package shop

// Page returns page p of the items, counted from 1, size items a page;
// nil past the last page, or for a page or a size below 1.
func Page(items []Item, p, size int) []Item {
	if p < 1 || size < 1 {
		return nil
	}
	from := (p - 1) * size
	if from >= len(items) {
		return nil
	}
	return items[from:min(from+size, len(items))]
}
GO
cat > shop/page_test.go <<'GO'
package shop

import "testing"

func TestPage(t *testing.T) {
	items := []Item{{"a", 1, 1}, {"b", 1, 1}, {"c", 1, 1}}
	if got := Page(items, 1, 2); len(got) != 2 || got[1].SKU != "b" {
		t.Errorf("page 1: %v", got)
	}
	if Page(items, 3, 2) != nil {
		t.Error("past the end: not nil")
	}
}
GO
printf '# Shop\n\nCarts, coupons, stock.\n' > README.md
git add . && git commit -q -m "feat: start the shop"
