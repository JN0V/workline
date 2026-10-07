package reviewer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The tests a tests-lens judge is shown: those in the cause's folder and
// those naming its file elsewhere, whole, up to the cap; the rest named;
// never a file that is not a test.
func TestTestsTouching(t *testing.T) {
	repo := t.TempDir()
	files := map[string]string{
		"shop/shipping.go":        "package shop\n\nfunc Shipping() int { return 490 }\n",
		"shop/cart_test.go":       "package shop\n\nfunc TestTotal() {}\n",
		"it/test_shipping.py":     "from shop import shipping\n",
		"it/test_cart.py":         "from shop import cart\n",
		"docs/shipping.md":        "Shipping costs.\n",
		"shop/big/zz_test.go":     "package big\n",
		"e2e/tests/shipping.spec": strings.Repeat("shipping\n", 50),
	}
	for name, text := range files {
		p := filepath.Join(repo, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(text), 0o644)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "start"}} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1") // the user's hooks stay out
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	patterns := []string{"**/*_test.*", "**/test_*", "**/tests/**"}
	got := testsTouching(repo, "HEAD", "shop/shipping.go", patterns, 20)
	for _, want := range []string{"### shop/cart_test.go", "### it/test_shipping.py", "Not shown, past 20 lines: e2e/tests/shipping.spec."} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, not := range []string{"test_cart.py", "docs/shipping.md", "zz_test.go", "### shop/shipping.go"} {
		if strings.Contains(got, not) {
			t.Errorf("holds %q:\n%s", not, got)
		}
	}
	if strings.Index(got, "cart_test.go") > strings.Index(got, "test_shipping.py") {
		t.Errorf("the cause's folder's tests come first:\n%s", got)
	}
	if got := testsTouching(repo, "HEAD", "docs/shipping.md", nil, 20); !strings.Contains(got, "no `tests` setting") {
		t.Errorf("no setting, not said:\n%s", got)
	}
}

// A lens's front matter is read empty, closed on the last line, or not at
// all; left open, it is refused.
func TestReadLensFrontMatter(t *testing.T) {
	roles := t.TempDir()
	t.Setenv("WORKLINE_ROLES_DIR", roles)
	dir := filepath.Join(roles, roleName(), "lenses")
	os.MkdirAll(dir, 0o755)
	for _, c := range []struct{ text, body, question, err string }{
		{"**Plain.**\n", "**Plain.**\n", "", ""},
		{"---\n---\n**Empty.**\n", "**Empty.**\n", "", ""},
		{"---\r\njudge:\r\n  question: Asked?\r\n---\r\n**CRLF.**\r\n", "**CRLF.**\n", "Asked?", ""},
		{"---\njudge:\n  question: Asked?\n---\n**Lens.**\n", "**Lens.**\n", "Asked?", ""},
		{"---\njudge:\n  question: Asked?\n---", "", "Asked?", ""},
		{"---\njudge:\n  question: Asked?\n", "", "", "not closed"},
		{"---\njudge: {reads: lint}\n---\n", "", "", "only `tests`"},
		{"---\nsubject: plan\n---\n", "", "", "only `spec`"},
	} {
		os.WriteFile(filepath.Join(dir, "l.md"), []byte(c.text), 0o644)
		l, err := readLens(t.TempDir(), "l")
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: error %v, want %q", c.text, err, c.err)
			}
			continue
		}
		if err != nil || strings.TrimSpace(l.Text) != strings.TrimSpace(c.body) || l.Judge.Question != c.question {
			t.Errorf("%q: %q, %q, %v; want %q, %q", c.text, l.Text, l.Judge.Question, err, c.body, c.question)
		}
	}
}
