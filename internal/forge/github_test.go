package forge

import (
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
