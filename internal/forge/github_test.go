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
	want := MergeRequest{Branch: "feat", Base: "main", Here: true, Title: "Average over the count", Body: "Closes #4\n\nNo change in behaviour."}
	if mr != want {
		t.Errorf("merge request: %+v, want %+v", mr, want)
	}
}
