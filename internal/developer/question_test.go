package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// TestScopeCodeShowsWhatTheAgentMayPatch: the agent has no tools, so its
// question holds the Scope's code, a folder's files, a path not there yet
// said, the frozen tests left to their own section (measured: without the
// code, a real agent wrote no patch).
func TestScopeCodeShowsWhatTheAgentMayPatch(t *testing.T) {
	repo := gitRepo(t, map[string]string{"lib.sh": "sub() { echo $(($2 - $1)); }\n", "calc/a.sh": "a=1\n",
		"tests/test_sub.sh": "frozen\n", "other.sh": "not in scope\n"})
	head, _ := gitOut(repo, "rev-parse", "HEAD")
	b := Bounds{Issue: 5, Scope: []string{"lib.sh", "calc", "new.sh", "tests/test_sub.sh"}, Frozen: map[string]int{"tests/test_sub.sh": 5}}
	code, err := scopeCode(repo, head, b)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### `lib.sh`", "sub() { echo $(($2 - $1)); }", "### `calc/a.sh`", "`new.sh`: not there yet"} {
		if !strings.Contains(code, want) {
			t.Errorf("the question lacks %q:\n%s", want, code)
		}
	}
	for _, not := range []string{"not in scope", "frozen"} {
		if strings.Contains(code, not) {
			t.Errorf("the question shows %q:\n%s", not, code)
		}
	}
	task := taskText(forge.Issue{ID: 5, Title: "Sub"}, b, nil, code)
	if strings.Index(task, "sub() {") > strings.Index(task, "## Your answer") {
		t.Errorf("the code comes before the answer's rules:\n%s", task)
	}
}

// TestScopeCodeFailsOnAGitError: a commit git cannot read is an error,
// never "not there yet": the agent would write over a file it was not shown.
func TestScopeCodeFailsOnAGitError(t *testing.T) {
	repo := gitRepo(t, map[string]string{"lib.sh": "x=1\n"})
	b := Bounds{Issue: 5, Scope: []string{"lib.sh"}}
	code, err := scopeCode(repo, "0123456789abcdef0123456789abcdef01234567", b)
	if err == nil || !strings.Contains(err.Error(), "git cannot list lib.sh") {
		t.Fatalf("an unreadable commit gave no error saying why: %v", err)
	}
	if strings.Contains(code, "not there yet") {
		t.Errorf("an unreadable commit read as a path not there yet:\n%s", code)
	}
}

// TestScopeCodeNamesAFileOverTheRoom: past codeRoom, a file is named as
// not shown, its text left out.
func TestScopeCodeNamesAFileOverTheRoom(t *testing.T) {
	big := strings.Repeat("y", codeRoom+1)
	repo := gitRepo(t, map[string]string{"big.txt": big, "small.sh": "s=1\n"})
	head, _ := gitOut(repo, "rev-parse", "HEAD")
	code, err := scopeCode(repo, head, Bounds{Issue: 5, Scope: []string{"big.txt", "small.sh"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "`big.txt`: not shown, over the question's room.") || strings.Contains(code, big) {
		t.Errorf("the file over the room is not named, or shown whole")
	}
	if !strings.Contains(code, "### `small.sh`") {
		t.Errorf("the file within the room is not shown:\n%s", code)
	}
}

// TestTaskTextShowsTheObjectionAsYAML: the objection's form is shown whole,
// a YAML item, when the tester froze a test (measured: shown inline, a
// real agent wrote it as text, and it was refused).
func TestTaskTextShowsTheObjectionAsYAML(t *testing.T) {
	b := Bounds{Issue: 5, Scope: []string{"lib.sh"}, Frozen: map[string]int{"tests/test_sub.sh": 5}}
	task := taskText(forge.Issue{ID: 5, Title: "Sub"}, b, map[string]string{"tests/test_sub.sh": "frozen\n"}, "")
	for _, want := range []string{"```yaml\n- comment:\n    objection: {test: <file>", "never a text", "### `tests/test_sub.sh`"} {
		if !strings.Contains(task, want) {
			t.Errorf("the question lacks %q:\n%s", want, task)
		}
	}
	if strings.Contains(taskText(forge.Issue{ID: 5}, Bounds{Issue: 5, Scope: []string{"lib.sh"}}, nil, ""), "objection:") {
		t.Errorf("an objection form shown with no frozen test")
	}
}

// TestRoundCodeShowsTheBranchCode: the review round's question holds the
// Scope's code as its branch holds it, before the answer's rules.
func TestRoundCodeShowsTheBranchCode(t *testing.T) {
	repo := gitRepo(t, map[string]string{"lib.sh": "fixed() { :; }\n"})
	b := Bounds{Issue: 5, Scope: []string{"lib.sh"}}
	code, err := roundCode(repo, b)
	if err != nil {
		t.Fatal(err)
	}
	task := roundTask(round{MR: 9}, forge.Issue{ID: 5, Title: "Sub"}, b, Settings{Rounds: 3}, code)
	at, answer := strings.Index(task, "fixed() { :; }"), strings.Index(task, "## Your answer")
	if at < 0 || !strings.Contains(task, "## The code in scope, as it reads") || at > answer {
		t.Errorf("the round's question lacks its branch's code before the answer:\n%s", task)
	}
}

// TestRoundCodeFailsWithNoHead: a head git cannot read fails the round,
// saying why, never a question with no code.
func TestRoundCodeFailsWithNoHead(t *testing.T) {
	code, err := roundCode(t.TempDir(), Bounds{Issue: 5, Scope: []string{"lib.sh"}})
	if err == nil || !strings.Contains(err.Error(), "git cannot read its branch's head") || code != "" {
		t.Fatalf("no head gave %q, %v: want an error saying why", code, err)
	}
}
