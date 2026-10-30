package developer

import (
	"errors"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// TestTaskTextNamesTheCodeNotPastes: the issue's question names its
// Scope, the frozen tests' paths, the test command and the bounds, and
// holds no code: its agent reads it in its worktree (ADR-0056, step 5).
func TestTaskTextNamesTheCodeNotPastes(t *testing.T) {
	b := Bounds{Issue: 5, Scope: []string{"lib.sh", "calc"}, Frozen: map[string]int{"tests/test_sub.sh": 5, "tests/other.sh": 4}}
	task := taskText(forge.Issue{ID: 5, Title: "Sub", Body: "## Need\n\nSubtract."}, b, "sh run-tests.sh {report}", nil, Settings{MaxTurns: 40, MaxSeconds: 900})
	for _, want := range []string{"# Issue #5: Sub", "Subtract.", "- `lib.sh`", "- `calc`", "- `tests/test_sub.sh`",
		"`sh run-tests.sh {report}`", "40 turns at most", "900 seconds at most", "Edit the files in your worktree"} {
		if !strings.Contains(task, want) {
			t.Errorf("the question lacks %q:\n%s", want, task)
		}
	}
	for _, not := range []string{"tests/other.sh", "### `", "as it reads", "Patches"} {
		if strings.Contains(task, not) {
			t.Errorf("the question holds %q:\n%s", not, task)
		}
	}
	if none := taskText(forge.Issue{ID: 5}, b, "", nil, Settings{}); !strings.Contains(none, "sets no test command") || strings.Contains(none, "turns at most") {
		t.Errorf("no test command nor caps, not said so:\n%s", none)
	}
}

// TestTaskTextSaysTestCommandUnread: tester settings it cannot read are
// said so, with why, never "the project sets no test command": the agent
// would not run tests the project has.
func TestTaskTextSaysTestCommandUnread(t *testing.T) {
	unread := errors.New("the tester's settings: json: cannot unmarshal number into Go struct field Settings.test-command of type string")
	task := taskText(forge.Issue{ID: 5}, Bounds{Issue: 5, Scope: []string{"lib.sh"}}, "", unread, Settings{})
	if !strings.Contains(task, "could not be read: "+unread.Error()) || strings.Contains(task, "sets no test command") {
		t.Errorf("an unread test command not said so:\n%s", task)
	}
}

// TestTaskTextShowsTheObjectionAsYAML: the objection's form is shown whole,
// a YAML item, when the tester froze a test (measured: shown inline, a
// real agent wrote it as text, and it was refused).
func TestTaskTextShowsTheObjectionAsYAML(t *testing.T) {
	b := Bounds{Issue: 5, Scope: []string{"lib.sh"}, Frozen: map[string]int{"tests/test_sub.sh": 5}}
	task := taskText(forge.Issue{ID: 5, Title: "Sub"}, b, "", nil, Settings{})
	for _, want := range []string{"```yaml\n- comment:\n    objection: {test: <file>", "never a text", "- `tests/test_sub.sh`"} {
		if !strings.Contains(task, want) {
			t.Errorf("the question lacks %q:\n%s", want, task)
		}
	}
	if strings.Contains(taskText(forge.Issue{ID: 5}, Bounds{Issue: 5, Scope: []string{"lib.sh"}}, "", nil, Settings{}), "objection:") {
		t.Errorf("an objection form shown with no frozen test")
	}
}

// TestRoundTaskNamesTheCodeNotPastes: the review round's question names
// the findings, the Scope, the frozen tests and the test command, as an
// issue run's does, and asks a note a finding: its agent reads its branch
// in its worktree (ADR-0056, step 6).
func TestRoundTaskNamesTheCodeNotPastes(t *testing.T) {
	b := Bounds{Issue: 5, Scope: []string{"lib.sh"}, Frozen: map[string]int{"tests/test_sub.sh": 5}}
	r := round{MR: 9, Asked: []Asked{{Lens: "correctness", Where: "lib.sh:1", Text: "Important: sub swaps its operands", Holds: "blocks"}}}
	task := roundTask(r, forge.Issue{ID: 5, Title: "Sub"}, b, Settings{Rounds: 3, MaxTurns: 40}, "sh run-tests.sh {report}", nil)
	for _, want := range []string{"# Review round 1 of 3 on your pull request 9", "sub swaps its operands", "# Issue #5: Sub", "- `lib.sh`",
		"- `tests/test_sub.sh`", "`sh run-tests.sh {report}`", "40 turns at most", "Edit the files in your worktree", "Its diff is your fix", "a `note` a finding"} {
		if !strings.Contains(task, want) {
			t.Errorf("the round's question lacks %q:\n%s", want, task)
		}
	}
	if strings.Contains(task, "Its diff is your code") || strings.Contains(task, "Patches") {
		t.Errorf("the round's question asks an issue run's answer, or patches:\n%s", task)
	}
}
