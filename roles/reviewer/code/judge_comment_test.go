package reviewer

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The LED fixture: the code at the base, then the change that adds a guard
// under a comment calling the behaviour intended.
const ledBase = `package led

// LED is a lamp.
type LED struct {
	On    bool
	Level int
}

// SetBrightness sets the lamp's level.
func SetBrightness(l *LED, level int) {
	l.On, l.Level = true, level
}
`

const ledHead = `package led

// LED is a lamp.
type LED struct {
	On    bool
	Level int
}

// SetBrightness sets the lamp's level. A level of 0 switches the LED off:
// this is intended.
func SetBrightness(l *LED, level int) {
	if level == 0 {
		l.On = false
		return
	}
	l.On, l.Level = true, level
}
`

// The issue the change closes: its Need and the person's answer say a level
// of 0 keeps the LED on, which the code's comment contradicts.
const ledIssue = `### Issue #12: Brightness

#### Need

A level of 0 must never switch the LED off: it stays on at its lowest light.

The person answered: "Brightness 0 never switches the LED off, whatever the code says."

#### Verification

- SetBrightness(l, 0) leaves l.On true.
`

// Comment is not the spec. Given a finding against the issue's spec, when
// the judge reads a code comment saying the same behaviour is intended, then
// the finding is kept: the judge is given the issue's spec and the person's
// answer, and told they outrank the comment, the code's words, not the spec.
func TestJudgeKeepsAFindingAgainstTheSpecWhenACommentCallsItIntended(t *testing.T) {
	t.Setenv("WORKLINE_ROLE", "reviewer")
	t.Setenv("WORKLINE_ROLES_DIR", t.TempDir())
	lens := ".workline/roles/reviewer/lenses/correctness.md"
	repo := gitRepo(t, map[string]string{"led/led.go": ledBase, lens: "Find what the change breaks.\n"})
	run, commit := gitSteps(t, repo)
	base := run("rev-parse", "HEAD")
	head := commit("led/led.go", ledHead)
	at := 0
	for i, l := range strings.Split(ledHead, "\n") {
		if strings.Contains(l, "l.On = false") {
			at = i + 1
		}
	}
	files, err := changedFiles(repo, base, head)
	if err != nil {
		t.Fatal(err)
	}
	change, err := readChange(repo, base, head, files)
	if err != nil {
		t.Fatal(err)
	}
	st := state{Head: head, Base: base, Change: change, Issues: []ClosedIssue{{ID: 12, Text: ledIssue}}}
	f := Finding{Lens: "correctness", Severity: "important", Title: "A level of 0 switches the LED off", Why: "the issue's Need says a level of 0 keeps it on", Where: "led/led.go:" + strconv.Itoa(at), Related: true, On: at, Cause: Quote{Path: "led/led.go", Quote: "l.On = false"}}
	c := candidates{Related: []Finding{f}}
	runDir, n := t.TempDir(), 0
	if err := askJudges(runDir, repo, Settings{JudgeLinesMax: 200}, st, &c, &n); err != nil {
		t.Fatal(err)
	}
	if len(c.Asked) != 1 {
		t.Fatalf("judges asked: %v, want one", c.Asked)
	}
	var asked struct{ Question, Material string }
	if err := readJSON(filepath.Join(runDir, "in", "judge", "01", "question.yaml"), &asked); err != nil {
		t.Fatal(err)
	}
	given := asked.Question + "\n" + asked.Material
	for _, want := range []string{"A level of 0 must never switch the LED off", "Brightness 0 never switches the LED off", "this is intended"} {
		if !strings.Contains(given, want) {
			t.Errorf("the judge is not given %q:\n%s", want, given)
		}
	}
	if !strings.Contains(strings.ToLower(given), "outrank") {
		t.Errorf("the judge is not told the issue's spec and the person's answer outrank a code comment:\n%s", given)
	}
}
