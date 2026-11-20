package reviewer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// manyLines is a file of 800 lines; edited, every 20th line from the 10th
// is changed: 40 one-line hunks, too far apart to merge at 5 lines of
// context.
func manyLines(edited bool) string {
	var b strings.Builder
	for i := 1; i <= 800; i++ {
		if edited && i%20 == 10 {
			fmt.Fprintf(&b, "changed %03d\n", i)
			continue
		}
		fmt.Fprintf(&b, "line %03d\n", i)
	}
	return b.String()
}

// Context shrunk: Given a change of many one-line hunks, When the reviewer
// would read it in more parts than allowed, Then it first gives less
// context per hunk, and reads the change whole if that fits.
func TestManyOneLineHunksAreReadWholeWithLessContext(t *testing.T) {
	t.Setenv("WORKLINE_ROLE", "reviewer")
	t.Setenv("WORKLINE_ROLES_DIR", t.TempDir())
	repo := gitRepo(t, map[string]string{"README.md": "x\n"})
	run, commit := gitSteps(t, repo)
	commit("wide.txt", manyLines(false))
	base := run("rev-parse", "HEAD")
	head := commit("wide.txt", manyLines(true))
	lensDir := filepath.Join(repo, ".workline", "roles", "reviewer", "lenses")
	if err := os.MkdirAll(lensDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lensDir, "correctness.md"), []byte("Look for defects.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := changedFiles(repo, base, head)
	if err != nil || len(files) != 1 {
		t.Fatalf("changed files: %v, %v", files, err)
	}
	runDir := t.TempDir()
	// 40 hunks of 5 context lines either side are 520 lines: 3 parts of 220,
	// past parts-max; with 1 line of context they are 204 lines, with none 124.
	s := Settings{CodeLinesMax: 0, PartsMax: 2, DiffLinesMax: 220}
	st := &state{Head: head, From: base, Base: base, Lenses: []string{"correctness"}}
	why, err := ask(runDir, repo, s, st, files)
	if err != nil {
		t.Fatal(err)
	}
	if why != "" {
		t.Fatalf("the change is not read whole: %s", why)
	}
	if st.Parts > 1 {
		t.Errorf("the change is read in %d parts, want it whole in one call", st.Parts)
	}
	task, err := os.ReadFile(filepath.Join(runDir, "in", "parts", "1-correctness", "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(task), "This part") {
		t.Errorf("the call is told it is a part of the change")
	}
	for i := 10; i <= 800; i += 20 {
		if want := fmt.Sprintf("+changed %03d\n", i); !strings.Contains(string(task), want) {
			t.Errorf("the hunk of line %d is not in the one call: %q missing", i, want)
		}
	}
}
