package codepick

import (
	"strings"
	"testing"
)

// A file named past the budget is said cut, its lines given of its
// lines; one that fits is not (#1159).
func TestCutSaysWhatTheBudgetLeftOut(t *testing.T) {
	dir := repo(t, map[string]string{"big.sh": filler(300, "# line %d"), "small.sh": filler(5, "# line %d")})
	pk, _ := pick(t, dir, Ask{Text: "`big.sh`", Budget: 100})
	if c := pk.Cut(); !strings.Contains(c, "of the 300 lines of the files named given (`big.sh` ") || !strings.Contains(c, " of 300 lines") {
		t.Errorf("a file past the budget: %q", c)
	}
	if pk, _ := pick(t, dir, Ask{Text: "`small.sh`", Budget: 100}); pk.Cut() != "" {
		t.Errorf("a file that fits said cut: %q", pk.Cut())
	}
}
