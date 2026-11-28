package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ghClosing puts a gh in place of the real one: it logs each call, and
// answers the list of the open pull requests, number and head branch a
// line, with what the file it gives holds. It gives the forge, a way to
// set that list, and what gh was asked so far.
func ghClosing(t *testing.T, pulls string) (g *github, setPulls func(string), calls func() string) {
	t.Helper()
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	list := filepath.Join(bin, "pulls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\n" + `case "$*" in
*pulls?state=open*) cat ` + list + ` ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setPulls = func(s string) {
		if err := os.WriteFile(list, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setPulls(pulls)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &github{repo: t.TempDir()}, setPulls, func() string {
		data, _ := os.ReadFile(log)
		return string(data)
	}
}

// deletedRefs is the calls of gh that delete something.
func deletedRefs(calls string) []string {
	var out []string
	for _, line := range strings.Split(calls, "\n") {
		if strings.Contains(line, "-X DELETE") {
			out = append(out, line)
		}
	}
	return out
}

// Superseded tester branch: given a workline/tester/42 branch with an open
// pull request, when the line closes that pull request without merging,
// then the branch is deleted on the forge, and no other branch is.
func TestGitHubCloseMergeRequestDeletesTheBranch(t *testing.T) {
	g, _, calls := ghClosing(t, "6\tworkline/tester/42\n7\tfeat/other\n")
	if err := g.CloseMergeRequest("workline/tester/42", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	got := calls()
	if !strings.Contains(got, "-X PATCH repos/{owner}/{repo}/pulls/6 -f state=closed") {
		t.Errorf("the pull request not closed: %s", got)
	}
	del := deletedRefs(got)
	if len(del) != 1 || !strings.HasPrefix(del[0], "api -X DELETE ") || !strings.Contains(del[0], "refs/heads/workline/tester/42") {
		t.Errorf("want one delete, of the branch refs/heads/workline/tester/42: %q", del)
	}
}

// Branch still in use: given a workline/ branch that another open pull
// request uses, when the line closes one pull request on it, then the
// branch is kept; once no other uses it, a close deletes it.
func TestGitHubCloseMergeRequestKeepsABranchInUse(t *testing.T) {
	g, setPulls, calls := ghClosing(t, "6\tworkline/tester/42\n9\tworkline/tester/42\n")
	if err := g.CloseMergeRequest("workline/tester/42", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls(), "state=closed") {
		t.Fatalf("the pull request not closed: %s", calls())
	}
	if del := deletedRefs(calls()); len(del) != 0 {
		t.Fatalf("the branch another open pull request uses was deleted: %q", del)
	}
	setPulls("6\tworkline/tester/42\n")
	if err := g.CloseMergeRequest("workline/tester/42", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	if del := deletedRefs(calls()); len(del) != 1 || !strings.Contains(del[0], "refs/heads/workline/tester/42") {
		t.Errorf("no other pull request uses the branch now, want it deleted: %q", del)
	}
}

// Not the line's branch: given a person's branch alice/fix with a pull
// request the line closes, when it closes, then the branch is kept; the
// line's own branch, closed the same way, is deleted.
func TestGitHubCloseMergeRequestKeepsAPersonsBranch(t *testing.T) {
	g, _, calls := ghClosing(t, "5\talice/fix\n6\tworkline/tester/42\n")
	if err := g.CloseMergeRequest("alice/fix", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(calls(), "-X PATCH repos/{owner}/{repo}/pulls/5 -f state=closed") {
		t.Fatalf("the pull request not closed: %s", calls())
	}
	if del := deletedRefs(calls()); len(del) != 0 {
		t.Fatalf("a person's branch was deleted: %q", del)
	}
	if err := g.CloseMergeRequest("workline/tester/42", "Closed by workline: superseded.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	del := deletedRefs(calls())
	if len(del) != 1 || strings.Contains(del[0], "alice") || !strings.Contains(del[0], "refs/heads/workline/tester/42") {
		t.Errorf("want the line's branch deleted, and it alone: %q", del)
	}
}
