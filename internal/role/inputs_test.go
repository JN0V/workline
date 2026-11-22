package role

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inputs names only what the shared pre can read: a role declaring
// `inputs: [mail]` is refused when it loads, mail named, with the list
// of inputs allowed.
func TestUnknownInputRefused(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "r"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(inputs string) {
		t.Helper()
		body := []byte("contract: 1\nname: r\nintentions: [note]\ninputs: " + inputs + "\n")
		if err := os.WriteFile(filepath.Join(dir, "r", "role.yaml"), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("[target, commits]")
	if _, err := Load(dir, "r"); err != nil {
		t.Fatalf("inputs [target, commits]: %v, want it loaded", err)
	}
	write("[mail]")
	_, err := Load(dir, "r")
	if err == nil {
		t.Fatal("inputs [mail]: loaded, want it refused")
	}
	for _, want := range []string{"mail", "target", "commits", "issues", "wake", "sample", "runs"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("inputs [mail]: the refusal %q does not hold %q", err.Error(), want)
		}
	}
}
