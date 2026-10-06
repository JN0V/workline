package role

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// on-block names only what the role may propose: a kind outside its
// intentions is refused when the role loads, never applied on a block.
func TestOnBlockOutsideIntentionsRefused(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "r"), 0o755)
	write := func(onBlock string) {
		os.WriteFile(filepath.Join(dir, "r", "role.yaml"), []byte("contract: 1\nname: r\nintentions: [comment, note]\non-block: "+onBlock+"\n"), 0o644)
	}
	write("[comment]")
	if r, err := Load(dir, "r"); err != nil || len(r.OnBlock) != 1 {
		t.Fatalf("on-block [comment]: %v, %v", r, err)
	}
	write("[patch]")
	if _, err := Load(dir, "r"); err == nil || !strings.Contains(err.Error(), `on-block names "patch"`) {
		t.Fatalf("on-block [patch]: %v, want it refused", err)
	}
}
