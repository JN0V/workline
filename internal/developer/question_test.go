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
	code := scopeCode(repo, head, b)
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
