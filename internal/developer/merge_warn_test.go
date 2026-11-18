package developer

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// findingsLine is the hidden record of a reviewer's comment, holding one
// finding.
func findingsLine(holds, lens, where, text string) string {
	return `<!-- workline:findings [{"lens":"` + lens + `","holds":"` + holds + `","where":"` + where +
		`","severity":"Important","text":"` + text + `"}] -->`
}

// mergeReadWith reads, as the merge pass does, a developer's pull request
// for a fix (its issue labelled `bug`) on a local forge, with the
// reviewer's summary comment and its comment of the lenses that warn.
func mergeReadWith(t *testing.T, act Act, summary, warn string) Merge {
	t.Helper()
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	repo := t.TempDir()
	cmd := exec.Command("git", "-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid", "init", "-q", "-b", "main")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s", out)
	}
	l := &forge.Local{Repo: repo}
	if _, err := l.Open("a bug", "A bug.", []string{"bug"}); err != nil {
		t.Fatal(err)
	}
	id, err := l.OpenMergeRequest("workline/developer/1", "main", "fix: the bug", "Closes #1", false)
	if err != nil {
		t.Fatal(err)
	}
	target := forge.Target{Kind: "merge-request", ID: id}
	if err := l.Comment(target, "**The developer's record**\n\n"+act.Line(), forge.Marker("sticky=developer/"+devmark.RecordKey)); err != nil {
		t.Fatal(err)
	}
	if summary != "" {
		if err := l.Comment(target, summary, forge.Marker("sticky=reviewer/summary")); err != nil {
			t.Fatal(err)
		}
	}
	if warn != "" {
		if err := l.Comment(target, warn, forge.Marker("sticky=reviewer/warn")); err != nil {
			t.Fatal(err)
		}
	}
	mr, err := l.MergeRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ReadMerge(l, mr)
	if err != nil {
		t.Fatal(err)
	}
	if m.Why != "" || m.Kind != devmark.KindFix {
		t.Fatalf("the pull request is not read as a developer's fix: why %q, kind %q", m.Why, m.Kind)
	}
	return m
}

// A warn finding does not hold: given a fix whose checks are green and
// whose verdict passes, when the only findings left are in the reviewer's
// comment of the lenses that warn, then the merge reads no finding open
// (so the merge policy merges it with no person's yes).
func TestReadMergeWarnFindingDoesNotHold(t *testing.T) {
	summary := "**workline reviewer** — verdict pass.\n\n" + `<!-- workline:findings [] -->`
	warn := "**workline reviewer** — the lenses that warn never hold the verdict.\n\n" +
		findingsLine("warns", "tests", "a_test.go:3", "Important: a case is not tested")
	m := mergeReadWith(t, Act{Act: "code", Issue: 1}, summary, warn)
	if len(m.Open) != 0 {
		t.Errorf("findings open %q: a finding of the lenses that warn holds nothing", m.Open)
	}
}

// The tester's answer counts: given a tests finding the developer sent to
// the tester's Draft (the record names it), when the tester answers it,
// then that finding is no longer counted open: the merge reads no finding
// open for it, and the verdict's own comment is left as it is.
func TestReadMergeTestsFindingSentToTesterIsNotOpen(t *testing.T) {
	act := Act{
		Act: "code", Issue: 1,
		Tester: []string{"tests|a_test.go|a case is not tested"},
		Rounds: []devmark.Round{{N: 1, Tester: []string{"warns tests `a_test.go:3`: Important: a case is not tested"}}},
	}
	summary := "**workline reviewer** — verdict pass.\n\n" + `<!-- workline:findings [] -->`
	warn := "**workline reviewer** — the lenses that warn never hold the verdict.\n\n" +
		findingsLine("warns", "tests", "a_test.go:3", "Important: a case is not tested")
	m := mergeReadWith(t, act, summary, warn)
	if len(m.Open) != 0 {
		t.Errorf("findings open %q: a tests finding sent to the tester is not counted open", m.Open)
	}
	if !strings.Contains(m.Summary, "verdict pass") {
		t.Errorf("the verdict's comment is not read as it is: %q", m.Summary)
	}
}
