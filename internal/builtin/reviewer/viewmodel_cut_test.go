package reviewer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// bigViewModel is a 900-line Kotlin file: a 100-line function `load`
// in the middle, one line of which the change edits.
func bigViewModel(edited bool) string {
	var b strings.Builder
	b.WriteString("class BigViewModel {\n")
	for i := 0; i < 350; i++ {
		fmt.Fprintf(&b, "    val a%d = %d\n", i, i)
	}
	b.WriteString("    fun load() {\n")
	for i := 1; i <= 100; i++ {
		v := i
		if edited && i == 50 {
			v = 5000
		}
		fmt.Fprintf(&b, "        val step%d = %d\n", i, v)
	}
	b.WriteString("    }\n")
	for i := 0; b.Len() > 0 && strings.Count(b.String(), "\n") < 899; i++ {
		fmt.Fprintf(&b, "    val z%d = %d\n", i, i)
	}
	b.WriteString("}\n")
	return b.String()
}

// askOnBigViewModel runs the lens input of a change to one function of a
// 900-line file, with the given code-lines-max, and returns the input
// the lens is given and the state's advisories.
func askOnBigViewModel(t *testing.T, codeLinesMax int) (string, []verdict.Finding) {
	t.Helper()
	t.Setenv("WORKLINE_ROLE", "reviewer")
	t.Setenv("WORKLINE_ROLES_DIR", t.TempDir())
	repo := gitRepo(t, map[string]string{"README.md": "x\n"})
	run, commit := gitSteps(t, repo)
	commit("BigViewModel.kt", bigViewModel(false))
	base := run("rev-parse", "HEAD")
	head := commit("BigViewModel.kt", bigViewModel(true))
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
	s := Settings{CodeLinesMax: codeLinesMax, PartsMax: 2}
	st := &state{Head: head, From: base, Base: base, Lenses: []string{"correctness"}}
	why, err := ask(runDir, repo, s, st, files)
	if err != nil || why != "" {
		t.Fatalf("ask: %q, %v", why, err)
	}
	task, err := os.ReadFile(filepath.Join(runDir, "in", "parts", "1-correctness", "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(task), st.Advisories
}

// Changed file whole: Given a changed file of 900 lines, When the
// reviewer reads it, Then its hunks' functions are whole in its input.
func TestChangedFileOfNineHundredLinesHasItsFunctionWhole(t *testing.T) {
	task, _ := askOnBigViewModel(t, 600)
	for _, line := range []string{"val step1 = 1", "val step50 = 5000", "val step100 = 100"} {
		if !strings.Contains(task, line) {
			t.Errorf("the function the change is in is not whole in the input: %q missing", line)
		}
	}
}

// Every lens: Given a cut input, When the reviewer runs, Then it says
// which lenses did not run and why.
func TestCutInputSaysWhichLensesDidNotRun(t *testing.T) {
	_, advisories := askOnBigViewModel(t, 20)
	for _, f := range advisories {
		if f.Rule == verdict.CutRule && strings.Contains(f.Message, "correctness") && strings.Contains(f.Message, "not run") {
			return
		}
	}
	t.Errorf("no cut warning says that the lens correctness did not run, and why: %+v", advisories)
}
