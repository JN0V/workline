package tester

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/testmark"
)

// Green is kept:
// Given a bug's parent whose reproduction tests, on the tester's Draft, are
// green over main, main having moved since the Draft was written,
// When the tester proves its symptom,
// Then the tests reach main through a ready merge request, built on main's
// tip, as any test does: main itself is not written, and the parent's
// comment names the merge request.
func TestSymptomGreenKeepsItsTestsThroughAMergeRequest(t *testing.T) {
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI"} {
		t.Setenv(v, "")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	repo, run := gitRepo(t, map[string]string{"app.txt": "app\n"})
	commit := func(file, text, msg string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, file), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		run(repo, "add", file)
		run(repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", msg)
	}
	base := run(repo, "rev-parse", "HEAD")
	const branch = testmark.TestsBranch + "3"
	run(repo, "checkout", "-q", "-b", branch)
	commit("repro_test.sh", "echo repro\n", "test: the reproduction")
	blobID := run(repo, "rev-parse", "HEAD:repro_test.sh")
	run(repo, "checkout", "-q", "main")
	commit("fix.txt", "fixed\n", "fix: the bug")
	mainTip := run(repo, "rev-parse", "main")

	l := &forge.Local{Repo: repo}
	id, err := l.OpenMergeRequest(branch, "main", "test: the reproduction of #3", "Tests of #3", true)
	if err != nil {
		t.Fatal(err)
	}
	rec := record{Issue: 3, Base: base, Files: map[string]string{"repro_test.sh": blobID}, Tests: []string{"t.TestRepro"}}
	target := forge.Target{Kind: "merge-request", ID: id}
	if err := l.Sticky(target, rec.body(), forge.Marker("sticky=tester/"+testmark.DraftRecordKey), true); err != nil {
		t.Fatal(err)
	}
	s := Settings{Report: "junit.xml",
		Command: `if [ -f repro_test.sh ]; then n=TestRepro; else n=TestOther; fi; printf '<testsuite name="t"><testcase classname="t" name="%s"/></testsuite>' "$n" > {report}`}

	sym, err := proveSymptom(repo, l, 3, []int{11, 12}, s)
	if err != nil {
		t.Fatal(err)
	}
	if !sym.Green {
		t.Fatalf("the reproduction test is green over main, yet: %s", sym.Why)
	}
	if got := run(repo, "rev-parse", "main"); got != mainTip {
		t.Errorf("main moved to %s: the tests reach it through a merge request, never written there", got)
	}
	mrs, err := l.ListMergeRequests()
	if err != nil {
		t.Fatal(err)
	}
	var ready *forge.MergeRequest
	for i := range mrs {
		if !mrs[i].Draft {
			ready = &mrs[i]
		}
	}
	if ready == nil {
		t.Fatalf("no ready merge request holds the tests: %+v", mrs)
	}
	if _, err := git(repo, "cat-file", "-e", ready.Branch+":repro_test.sh"); err != nil {
		t.Errorf("the ready merge request's branch %s lacks the reproduction test: %v", ready.Branch, err)
	}
	if _, err := git(repo, "merge-base", "--is-ancestor", mainTip, ready.Branch); err != nil {
		t.Errorf("the ready merge request's branch %s is not built on main's tip %s: %v", ready.Branch, mainTip, err)
	}
	named := regexp.MustCompile(`(?i)(merge request|pull request|!|#)\s*` + strconv.Itoa(ready.ID) + `\b`)
	if !named.MatchString(sym.Why) {
		t.Errorf("the parent's comment does not name merge request %d: %q", ready.ID, sym.Why)
	}
}
