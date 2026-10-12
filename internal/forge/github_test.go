package forge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GitHub's comments come with their author, of the project by their
// author_association as an issue's author, a bot by the user's type. gh is
// replaced by a script printing what its --jq prints, one object a line.
func TestGitHubNotes(t *testing.T) {
	bin := t.TempDir()
	args := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + "\n" + `cat <<'EOF'
{"body":"Rows lost.","author":"zed","association":"NONE","bot":false}
{"body":"agreed","author":"dev","association":"COLLABORATOR","bot":false}
{"body":"agreed","author":"ci[bot]","association":"NONE","bot":true}
{"body":"me too","author":"mia","association":"CONTRIBUTOR","bot":false}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	notes, err := g.Notes(Target{Kind: "issue", ID: 9})
	if err != nil {
		t.Fatal(err)
	}
	want := []Note{
		{Body: "Rows lost.", Author: "zed"},
		{Body: "agreed", Author: "dev", Insider: true},
		{Body: "agreed", Author: "ci[bot]", Bot: true},
		{Body: "me too", Author: "mia"},
	}
	if fmt.Sprint(notes) != fmt.Sprint(want) {
		t.Errorf("notes:\n%+v\nwant\n%+v", notes, want)
	}
	asked, _ := os.ReadFile(args)
	for _, w := range []string{"repos/{owner}/{repo}/issues/9/comments", "author_association", `.user.type == "Bot"`} {
		if !strings.Contains(string(asked), w) {
			t.Errorf("gh not asked for %q: %s", w, asked)
		}
	}
	bodies, _ := g.Comments(Target{Kind: "issue", ID: 9})
	if len(bodies) != 4 || bodies[1] != "agreed" {
		t.Errorf("comments: %q", bodies)
	}
}

// A pull request closed by a role: found by its branch among the open
// ones, one comment saying why, then closed; one from another branch, or
// none open, is left alone. A close GitHub refuses after the comment is
// said, never passed; run again, it closes without a second comment.
func TestGitHubCloseMergeRequest(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\nd=" + bin + "\n" + `case "$*" in
*pulls?state=open*) printf '6\tworkline/documentalist/suspect\n7\tfeat/other\n' ;;
"api -X POST "*/comments*) touch "$d/commented" ;;
*/issues/6/comments*) [ ! -f "$d/commented" ] || echo '<!-- workline:closed -->' ;;
"api -X PATCH "*) [ ! -f "$d/fail" ] || { echo "HTTP 502: Bad Gateway" >&2; exit 1; } ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	os.WriteFile(filepath.Join(bin, "fail"), nil, 0o644)
	if err := g.CloseMergeRequest("workline/documentalist/suspect", "Closed: nothing is left to merge.", "<!-- workline:closed -->"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("a close GitHub refused: %v, want it said", err)
	}
	os.Remove(filepath.Join(bin, "fail"))
	if err := g.CloseMergeRequest("workline/documentalist/suspect", "Closed: nothing is left to merge.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	if err := g.CloseMergeRequest("workline/documentalist/gone", "Closed.", "<!-- workline:closed -->"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := string(data)
	for _, w := range []string{"-X POST repos/{owner}/{repo}/issues/6/comments -f body=Closed: nothing is left to merge.\n\n<!-- workline:closed -->",
		"-X PATCH repos/{owner}/{repo}/pulls/6 -f state=closed"} {
		if !strings.Contains(calls, w) {
			t.Errorf("gh not asked %q: %s", w, calls)
		}
	}
	if strings.Count(calls, "-X POST") != 1 || strings.Count(calls, "state=closed") != 2 {
		t.Errorf("want one comment and two closes, the first refused: %s", calls)
	}
	if strings.Contains(calls, "/7") {
		t.Errorf("another pull request touched: %s", calls)
	}
}

// A pull request says where it comes from and goes, and what its author
// says of it: its title and body, a review's testimony (ADR-0020).
func TestGitHubMergeRequest(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\n" + `cat <<'EOF'
{"title":"Average over the count","body":"Closes #4\n\nNo change in behaviour.","head":{"ref":"feat","repo":{"full_name":"a/b"}},"base":{"ref":"main","repo":{"full_name":"a/b"}}}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	mr, err := (&github{repo: t.TempDir()}).MergeRequest(5)
	if err != nil {
		t.Fatal(err)
	}
	want := MergeRequest{ID: 5, Branch: "feat", Base: "main", Here: true, Title: "Average over the count", Body: "Closes #4\n\nNo change in behaviour."}
	if mr != want {
		t.Errorf("merge request: %+v, want %+v", mr, want)
	}
}

// The open pull requests, each with where it comes from and who opened it
// (ADR-0041): a fork's, or one whose fork is gone, is not here; an app's
// is a bot's; each head where GitHub keeps it.
func TestGitHubListMergeRequests(t *testing.T) {
	bin := t.TempDir()
	args := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + args + "\n" + `cat <<'EOF'
{"id":9,"branch":"patch-1","base":"main","here":false,"author":"zed","bot":false,"title":"Fix a typo"}
{"id":7,"branch":"feat/sessions","base":"main","here":true,"author":"dana","bot":false,"title":"Two hours"}
{"id":8,"branch":"release-please--branches--main","base":"main","here":true,"author":"workline-app[bot]","bot":true,"title":"chore: release 1.1.0"}
EOF
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := (&github{repo: t.TempDir()}).ListMergeRequests()
	if err != nil {
		t.Fatal(err)
	}
	want := []MergeRequest{
		{ID: 7, Branch: "feat/sessions", Base: "main", Here: true, Title: "Two hours", Author: "dana", Head: "refs/pull/7/head"},
		{ID: 8, Branch: "release-please--branches--main", Base: "main", Here: true, Title: "chore: release 1.1.0", Author: "workline-app[bot]", Bot: true, Head: "refs/pull/8/head"},
		{ID: 9, Branch: "patch-1", Base: "main", Title: "Fix a typo", Author: "zed", Head: "refs/pull/9/head"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("merge requests:\n%+v\nwant\n%+v", got, want)
	}
	asked, _ := os.ReadFile(args)
	for _, w := range []string{"pulls?state=open&per_page=100", `.head.repo.full_name // ""`, `.user.type == "Bot"`} {
		if !strings.Contains(string(asked), w) {
			t.Errorf("gh not asked for %q: %s", w, asked)
		}
	}
	if !(MergeRequest{Branch: "renovate/go-1.x"}).ByBot() || !(MergeRequest{Branch: "dependabot/go_modules/x"}).ByBot() || (MergeRequest{Branch: "feat/renovate"}).ByBot() {
		t.Error("a dependency bot's branch is told by its name, whoever opened it; another is not")
	}
}
