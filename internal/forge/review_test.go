package forge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A forge's patch read as the lines its diff shows: an added line with no
// old one, a kept line with its old one, a removed line not at all; a
// renamed file keeps its old path; git's own output read the same.
func TestShown(t *testing.T) {
	s := newShown()
	s.add("calc/mean.go", "calc/calc.go", "@@ -9,4 +9,4 @@ func Average(values []int) int {\n \t\ttotal += v\n \t}\n-\treturn total / len(values)\n+\treturn total / (len(values) - 1)\n }\n")
	for _, c := range []struct {
		line, old int
		shows     bool
	}{{9, 9, true}, {10, 10, true}, {11, 0, true}, {12, 12, true}, {8, 0, false}, {13, 0, false}} {
		old, ok := s.at("calc/mean.go", c.line)
		if ok != c.shows || old != c.old {
			t.Errorf("line %d: shown %v old %d, want %v %d", c.line, ok, old, c.shows, c.old)
		}
	}
	if s.oldPath("calc/mean.go") != "calc/calc.go" || s.oldPath("x.go") != "x.go" {
		t.Errorf("old paths: %v", s.old)
	}
	g := newShown()
	g.addGit("diff --git a/a.go b/a.go\nindex 1..2 100644\n--- a/a.go\n+++ b/a.go\n@@ -1,2 +1,3 @@\n x\n+--- y\n z\ndiff --git a/gone.go b/gone.go\n--- a/gone.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-w\n")
	if _, ok := g.at("a.go", 2); !ok {
		t.Errorf("a line added reading `--- y` is a line, not a header: %v", g.lines)
	}
	if _, ok := g.lines["/dev/null"]; ok {
		t.Errorf("a deleted file shows no new line: %v", g.lines)
	}
}

// A review region replaced whole, what it held for an engine posting no
// comment on a line gone; the marker alone replaced too; nothing to say,
// the region gone, the text around it kept apart.
func TestFillReview(t *testing.T) {
	region := "top\n\n" + ReviewHere + "\nin full\n" + ReviewEnd + "\n\nrest"
	if got := FillReview(region, "said"); got != "top\n\nsaid\n\nrest" {
		t.Errorf("region: %q", got)
	}
	if got := FillReview("top\n\n"+ReviewHere+"\n\nrest", "said"); got != "top\n\nsaid\n\nrest" {
		t.Errorf("marker alone: %q", got)
	}
	if got := FillReview(region, ""); got != "top\n\nrest" {
		t.Errorf("nothing to say: %q", got)
	}
	if got := FillReview("no region", "said"); got != "no region" {
		t.Errorf("no region: %q", got)
	}
}

// GitHub: the pull request's files read for the lines its diff shows, its
// threads not resolved for the keys open, then one review, event COMMENT,
// at head, holding only the comments to post, each on the right side with
// its key hidden after it.
func TestGitHubReview(t *testing.T) {
	bin := t.TempDir()
	posted := filepath.Join(bin, "posted.json")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*pulls/7/files*) cat <<'EOF'\n" + `{"filename":"calc/calc.go","previous":"","patch":"@@ -9,4 +9,4 @@\n \t\ttotal += v\n \t}\n-\treturn total / len(values)\n+\treturn total / (len(values) - 1)\n }"}` + "\nEOF\n;;\n" +
		"*graphql*) echo '{\"body\":\"old finding <!-- workline:line=correctness/open -->\"}' ;;\n" +
		"*pulls/7/reviews*) cat > " + posted + "; echo '{\"id\":1,\"html_url\":\"https://github.com/a/b/pull/7#pullrequestreview-1\"}' ;;\n" +
		"*) echo unexpected \"$*\" >&2; exit 1 ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	r, err := g.Review(7, "abc123", "the review", []LineComment{
		{Path: "calc/calc.go", Line: 11, Key: "correctness/open", Body: "still open"},
		{Path: "calc/calc.go", Line: 11, Key: "edge-cases/new", Body: "a new one"},
		{Path: "calc/calc.go", Line: 3, Key: "tests/far", Body: "outside the diff"},
	})
	if err != nil {
		t.Fatal(err)
	}
	states := []string{}
	for _, p := range r.Placed {
		states = append(states, p.State)
	}
	if strings.Join(states, ",") != "open,posted,outside" || r.URL != "https://github.com/a/b/pull/7#pullrequestreview-1" {
		t.Errorf("placed %v at %s", states, r.URL)
	}
	data, _ := os.ReadFile(posted)
	var req struct {
		Commit   string `json:"commit_id"`
		Event    string `json:"event"`
		Body     string `json:"body"`
		Comments []struct {
			Path, Side, Body string
			Line             int
		}
	}
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("the review posted does not read: %v: %s", err, data)
	}
	if req.Commit != "abc123" || req.Event != "COMMENT" || req.Body != "the review" || len(req.Comments) != 1 {
		t.Fatalf("review posted: %s", data)
	}
	if c := req.Comments[0]; c.Path != "calc/calc.go" || c.Line != 11 || c.Side != "RIGHT" || c.Body != "a new one\n\n<!-- workline:line=edge-cases/new -->" {
		t.Errorf("comment posted: %+v", c)
	}
}

// A token that may not review (HTTP 403) refuses the review: each comment
// to post is refused, said with what GitHub said and the right it needs;
// never an error.
func TestGitHubReviewRefused(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*pulls/7/files*) printf '%s\\n' '{\"filename\":\"a.go\",\"previous\":\"\",\"patch\":\"@@ -1 +1 @@\\n-x\\n+y\"}' ;;\n" +
		"*graphql*) ;;\n" +
		"*pulls/7/reviews*) echo 'gh: Resource not accessible by integration (HTTP 403)' >&2; exit 1 ;;\n" +
		"*html_url*) echo https://github.com/a/b ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r, err := (&github{repo: t.TempDir()}).Review(7, "", "b", []LineComment{{Path: "a.go", Line: 1, Key: "k/1", Body: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	p := r.Placed[0]
	if p.State != LineRefused || !strings.Contains(p.Why, "Resource not accessible by integration (HTTP 403)") || !strings.Contains(p.Why, "pull-requests: write") {
		t.Errorf("placed: %+v", p)
	}
	if r.URL != "https://github.com/a/b/pull/7/files" {
		t.Errorf("url: %s", r.URL)
	}
}
