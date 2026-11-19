package backlog

import (
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// No issue zero: Given a local forge's issues, one whose state record keeps
// an act with no issue number (as an issue's own state does), and an act of
// the run that names no issue, When reconcile runs the backlog pass, Then
// it names no #0: no finding, in its place or its message, speaks of an
// issue that does not exist.
func TestReconcileNamesNoIssueZero(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // the machine's hooks and settings stay out
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "") // in CI, the local forge refuses writes
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	l := &forge.Local{Repo: repo}
	id, err := l.Open("Export loses rows", "Rows are lost on export.", []string{LabelToRefine})
	if err != nil {
		t.Fatal(err)
	}
	st := State{Confirmed: "abc1234", Did: []Did{{Act: "sources", Level: Normal, Day: "2026-11-01"}}}
	if err := l.Comment(forge.Target{Kind: "issue", ID: id}, FormatState(st), StateMarker(role)); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Open("Export to JSON", "Please add a JSON export.", nil); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Level:        Normal,
		Wait:         DefaultWait,
		MovedPercent: 100,
		Acts:         map[string]Setting{"unready": {Mode: Act}},
	}
	// An act that holds no issue number: the issue's own state leaves it out.
	noIssue := Proposal{Do: "unready", Why: "its spec changed"}
	plan, err := Decide(l, repo, role, cfg, map[int]Proposal{0: noIssue}, []int{id}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	zero := regexp.MustCompile(`#0\b`)
	for _, f := range plan.Findings {
		if zero.MatchString(f.Where) || zero.MatchString(f.Message) {
			t.Errorf("a finding names #0, an issue that does not exist: %+v", f)
		}
	}
	for _, d := range plan.Decisions {
		if d.Mode != Off && d.Act.Issue == 0 {
			t.Errorf("an act on no issue is planned: %+v", d)
		}
	}
}
