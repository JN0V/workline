package gitrange

import (
	"strings"
	"testing"
)

func TestRanges(t *testing.T) {
	for _, c := range []struct{ rng, args, head, base string }{
		{"a..b", "a..b", "b", "a"},
		{"b", "b", "b", ""},
		{"m ^x ^y", "m ^x ^y", "m", "x"},
	} {
		if got := strings.Join(Args(c.rng), " "); got != c.args || Head(c.rng) != c.head || Base(c.rng) != c.base {
			t.Errorf("%q: %q %q %q", c.rng, got, Head(c.rng), Base(c.rng))
		}
	}
}
