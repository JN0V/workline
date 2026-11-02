package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGateConfig(t *testing.T, text string) string {
	t.Helper()
	repo := t.TempDir()
	dir := filepath.Join(repo, ".workline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// Wrong name
// Given a config declaring the gates `lint` and `tests`,
// When a person runs `workline gate lnt`,
// Then the finding says `no gate named "lnt"` and lists `lint, tests`.
func TestWrongName(t *testing.T) {
	repo := writeGateConfig(t, `gates:
  tests:
    checks:
      - id: t
        run: "true"
        output: exit
  lint:
    checks:
      - id: l
        run: "true"
        output: exit
`)
	_, err := Load(repo, "lnt")
	if err == nil {
		t.Fatal("Load of a wrong name returned no error")
	}
	if !strings.Contains(err.Error(), `no gate named "lnt"`) || !strings.Contains(err.Error(), "lint, tests") {
		t.Errorf("error = %q; want it to say no gate named \"lnt\" and list lint, tests", err)
	}
}

// No gate
// Given a config declaring no gate,
// When a person runs `workline gate lint`,
// Then the finding says no gate is declared.
func TestNoGate(t *testing.T) {
	repo := writeGateConfig(t, "gates: {}\n")
	_, err := Load(repo, "lint")
	if err == nil {
		t.Fatal("Load with no gate declared returned no error")
	}
	if !strings.Contains(err.Error(), "no gate is declared") {
		t.Errorf("error = %q; want it to say no gate is declared", err)
	}
}
