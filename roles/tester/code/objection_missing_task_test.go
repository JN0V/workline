package tester

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// missingTaskText writes the answer task of a "test missing" objection on
// an issue whose tests are the frozen files given, and reads it back.
func missingTaskText(t *testing.T, frozen ...string) string {
	t.Helper()
	runDir, repo := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range frozen {
		files[f] = "blob-" + f
		if err := os.WriteFile(filepath.Join(repo, f), []byte("def test_old(): pass\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := answer{Issue: 7, File: frozen[0], Kind: "missing",
		Scenarios: []work.Scenario{{Name: "Restart", Text: "Given …"}},
		Objection: objection{Test: frozen[0], Kind: "missing", Why: "no test for the restart"},
		Record:    record{Issue: 7, Files: files}}
	if code := writeAnswerTask(runDir, repo, a, forge.Thread{}, &forge.Issue{Title: "Console"}, Settings{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Told where the test goes:
// Given a "test missing" objection on a Draft whose issue has one frozen test file,
// When the tester's answer task is written,
// Then it says the test goes in that file, or in another frozen test file it
// lists, never in a new file.
func TestAnswerTaskForMissingTestSaysWhereItGoes(t *testing.T) {
	task := missingTaskText(t, "test_console.py")
	for _, want := range []string{"`test_console.py`", "never a new file"} {
		if !strings.Contains(task, want) {
			t.Errorf("the task lacks %q:\n%s", want, task)
		}
	}
}

// Told how to patch it:
// Given the same objection,
// When the task is written,
// Then it says the patch format for an existing file: the file whole with
// `file` and `content`, or a diff of that file.
func TestAnswerTaskForMissingTestSaysHowToPatch(t *testing.T) {
	task := missingTaskText(t, "test_console.py")
	for _, want := range []string{"the file whole with `file` and `content`", "a diff of that file"} {
		if !strings.Contains(task, want) {
			t.Errorf("the task lacks %q:\n%s", want, task)
		}
	}
}
