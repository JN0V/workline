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

// A parent's task list is added once, after its text, and rewritten in
// place when a resumed split lists its children again.
func TestListChildren(t *testing.T) {
	body := ListChildren("Two needs.\n\n## Scope\n\nsrc.", []int{10})
	want := "Two needs.\n\n## Scope\n\nsrc.\n\n## Sub-issues\n\n- [ ] #10"
	if body != want {
		t.Fatalf("first list = %q", body)
	}
	if again := ListChildren(body, []int{10, 11}); again != want+"\n- [ ] #11" {
		t.Fatalf("list rewritten = %q", again)
	}
}

// Two children are one when their titles differ only in case or spaces;
// distinct titles, or one title under two parents, are not.
func TestSplitKey(t *testing.T) {
	same := SplitKey(9, "Keep the last row")
	for _, title := range []string{"Keep the last row", "keep the LAST row", "  Keep  the last\trow "} {
		if SplitKey(9, title) != same {
			t.Errorf("SplitKey(9, %q) differs from %q's", title, "Keep the last row")
		}
	}
	if SplitKey(9, "Quote commas") == same || SplitKey(10, "Keep the last row") == same {
		t.Error("another title, or another parent, gives the same key")
	}
}
