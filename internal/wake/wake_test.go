package wake

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// A bug is told by its label, scoped or not, whatever its case; a title
// is never read.
func TestIsBug(t *testing.T) {
	for labels, want := range map[string]bool{
		"bug": true, "Bug": true, "type::bug": true, "kind/bug": true, "type: Bug": true,
		"enhancement": false, "debug": false, "bugfix": false, "": false,
	} {
		if got := isBug([]string{labels}); got != want {
			t.Errorf("isBug(%q) = %v, want %v", labels, got, want)
		}
	}
}

// A decision record is a Markdown file in a folder named adr, adrs or
// decisions; its README is the index, none.
func TestIsADR(t *testing.T) {
	for p, want := range map[string]bool{
		"docs/adr/0001-x.md": true, "adr/0002.md": true, "doc/decisions/use-go.md": true, "docs/adrs/1.md": true,
		"docs/adr/README.md": false, "docs/adr/img/a.png": false, "docs/radr/1.md": false, "docs/adr.md": false,
	} {
		if got := isADR(p); got != want {
			t.Errorf("isADR(%q) = %v, want %v", p, got, want)
		}
	}
}

// The issue keeps each trigger 7 days, the newest read of each, and is
// unchanged when a read finds what it lists already.
func TestIssueKeepsAWeek(t *testing.T) {
	first := &Result{Head: "aaaaaaa1", Day: "2026-01-01", Entries: []Entry{
		{Kind: Contract, Key: "contract:adr:a", Text: "ADR added: `a`", At: "aaaaaaa1", Day: "2026-01-01"},
		{Kind: Symptom, Key: "symptom:area:x", Text: "x, by 3", At: "aaaaaaa1", Day: "2026-01-01"},
	}}
	body, changed, any := Issue(first, "")
	if !changed || !any {
		t.Fatalf("first: changed %v any %v", changed, any)
	}
	again := &Result{Head: "bbbbbbb2", Day: "2026-01-05", Entries: []Entry{
		{Kind: Symptom, Key: "symptom:area:x", Text: "x, by 4", At: "bbbbbbb2", Day: "2026-01-05"},
	}}
	body, changed, _ = Issue(again, body)
	if !changed || !strings.Contains(body, "ADR added: `a`") || !strings.Contains(body, "x, by 4") || strings.Contains(body, "x, by 3") {
		t.Errorf("again: changed %v\n%s", changed, body)
	}
	if _, changed, _ = Issue(&Result{Head: "ccccccc3", Day: "2026-01-05", Entries: again.Entries}, body); changed {
		t.Error("the same read changed the issue")
	}
	late := &Result{Head: "ddddddd4", Day: "2026-01-09", Entries: []Entry{}}
	body, changed, _ = Issue(late, body)
	if !changed || strings.Contains(body, "ADR added: `a`") || !strings.Contains(body, "x, by 4") {
		t.Errorf("a week on: changed %v\n%s", changed, body)
	}
}

// stubForge lists one merge, its closing links unread.
type stubForge struct {
	forge.Forge
	as string
}

func (s stubForge) MergedSince(time.Time) ([]forge.MergeRequest, error) {
	return []forge.MergeRequest{{ID: 12, MergedAs: s.as, Author: "alice"}}, nil
}
func (stubForge) Closes([]int) (map[int][]forge.Issue, error) {
	return nil, errors.New("rate limited")
}
func (stubForge) Labelled(string) ([]forge.Target, error) { return nil, nil }

// The issues each merge closes unread, no merge is said to close none,
// nor a fix counted: the read is said unread instead.
func TestClosesUnreadAccusesNone(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"commit", "-q", "--allow-empty", "-m", "a"}, {"commit", "-q", "--allow-empty", "-m", "b"}} {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "core.hooksPath=/dev/null"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	head, _ := gitIn(repo).commit("HEAD")
	r, err := Read(Options{Repo: repo, After: "HEAD", Forge: stubForge{as: head}})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Entries {
		if e.Kind == Need {
			t.Errorf("accused: %+v", e)
		}
	}
	if !strings.Contains(strings.Join(r.Unread, "\n"), "rate limited") || strings.Contains(r.Facts, "closing a bug") {
		t.Errorf("unread %q, facts %q", r.Unread, r.Facts)
	}
}
