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

// The cap a cut names is the one that cut: the lines', or the bytes'
// when the lines would fit (#1159).
func TestCutByTheCapThatCut(t *testing.T) {
	dir := repo(t, map[string]string{"big.sh": filler(300, "# line %d")})
	byLines, _ := pick(t, dir, Ask{Text: "`big.sh`", Budget: 100, BudgetName: "code-lines-max"})
	if got := strings.Join(byLines.CutBy(), ","); got != "code-lines-max" {
		t.Errorf("cut by its lines: %q", got)
	}
	byBytes, _ := pick(t, dir, Ask{Text: "`big.sh`", Budget: 1000, Bytes: 1500, BudgetName: "code-lines-max"})
	if got := strings.Join(byBytes.CutBy(), ","); got != BytesName || byBytes.Cut() == "" {
		t.Errorf("cut by its bytes: %q, %q", got, byBytes.Cut())
	}
}
