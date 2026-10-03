package backlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A quote is found at the lines that hold it, not from an earlier line that
// only shares its first word (workline's BACKLOG.md, imported on a copy:
// an item's issue began with the end of the item before).
func TestLocate(t *testing.T) {
	repo := t.TempDir()
	text := "- **Releases.** Left: the sandbox,\n  then doctor - and init.\n- **Install.** Done: releases.\n  Left: a reusable action.\n"
	os.WriteFile(filepath.Join(repo, "BACKLOG.md"), []byte(text), 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // the machine's hooks stay out
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, c := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@x.invalid", "commit", "-qm", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, c...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	from, to, got, ok := Locate(repo, "BACKLOG.md", "- **Install.** Done: releases. Left: a reusable action.")
	if !ok || from != 3 || to != 4 || got != "- **Install.** Done: releases.\n  Left: a reusable action." {
		t.Fatalf("Locate = %d, %d, %q, %v", from, to, got, ok)
	}
}
