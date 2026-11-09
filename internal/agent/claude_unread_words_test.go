package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// Unread words still posted: Given an agent whose JSON result holds words
// that are no YAML, When the developer opens its pull request, Then those
// words are kept for a person, as today.
func TestUnreadResultWordsKept(t *testing.T) {
	bin := t.TempDir()
	words := "I changed the parser and all tests pass."
	fake := "#!/bin/sh\ncat > /dev/null\necho '{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":\"" + words + "\"}'\n"
	must(t, os.WriteFile(filepath.Join(bin, "claude"), []byte(fake), 0o755))
	for _, tool := range claudeShellTools {
		must(t, os.WriteFile(filepath.Join(bin, tool), []byte("#!/bin/sh\n"), 0o755))
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir, runDir := t.TempDir(), t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Code."), 0o644))
	for _, d := range []string{"in", "out", WorktreeDir} {
		must(t, os.MkdirAll(filepath.Join(runDir, d), 0o755))
	}
	must(t, os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("The issue."), 0o644))
	if _, err := (claude{}).Propose(Request{RunDir: runDir, Repo: t.TempDir(), MaxTurns: 7,
		Role: &role.Role{Name: "developer", Dir: dir, Intentions: []string{"patch", "comment"}}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(runDir, "out", ClosingUnread))
	if err != nil || !strings.Contains(string(data), words) {
		t.Errorf("the words are not kept for a person: %q, %v", data, err)
	}
}
