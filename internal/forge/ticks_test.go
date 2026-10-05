package forge

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// GitHub keeps no event for a box ticked: each version of the body in its
// edit history, newest first, is read against the one before, the tick
// given to its editor — of the project when GitHub gives them write.
func TestGitHubTicks(t *testing.T) {
	bin := t.TempDir()
	script := `#!/bin/sh
case "$*" in
*graphql*) cat <<'EOF'
{"data":{"repository":{"issue":{"userContentEdits":{"nodes":[
{"editedAt":"2026-10-05T10:03:00Z","diff":"- [x] Rename #9 <!-- workline:proposal=9/rename -->\n- [x] Close #7 <!-- workline:proposal=7/close-obsolete -->\n- [x] Order #4 <!-- workline:proposal=4/order -->","editor":{"login":"ci","__typename":"Bot"}},
{"editedAt":"2026-10-05T10:02:00Z","diff":"- [x] Rename #9 <!-- workline:proposal=9/rename -->\n- [x] Close #7 <!-- workline:proposal=7/close-obsolete -->\n- [ ] Order #4 <!-- workline:proposal=4/order -->","editor":{"login":"zed","__typename":"User"}},
{"editedAt":"2026-10-05T10:01:00Z","diff":"- [x] Rename #9 <!-- workline:proposal=9/rename -->\n- [ ] Close #7 <!-- workline:proposal=7/close-obsolete -->\n- [ ] Order #4 <!-- workline:proposal=4/order -->","editor":{"login":"dev","__typename":"User"}},
{"editedAt":"2026-10-05T10:00:00Z","diff":"- [ ] Rename #9 <!-- workline:proposal=9/rename -->\n- [ ] Close #7 <!-- workline:proposal=7/close-obsolete -->\n- [ ] Order #4 <!-- workline:proposal=4/order -->","editor":{"login":"bot-user","__typename":"User"}}
]}}}}}
EOF
;;
*collaborators/dev/permission*) echo write ;;
*collaborators/bot-user/permission*) echo admin ;;
*) echo read ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	ticks, err := g.Ticks(10)
	if err != nil {
		t.Fatal(err)
	}
	want := []Tick{
		{Item: "Rename #9 <!-- workline:proposal=9/rename -->", Done: true, Note: Note{Author: "dev", Insider: true}},
		{Item: "Close #7 <!-- workline:proposal=7/close-obsolete -->", Done: true, Note: Note{Author: "zed"}},
		{Item: "Order #4 <!-- workline:proposal=4/order -->", Done: true, Note: Note{Author: "ci", Bot: true}},
	}
	if !reflect.DeepEqual(ticks, want) {
		t.Errorf("ticks = %+v\nwant %+v", ticks, want)
	}
}

// GitLab writes a system note for each box ticked or unticked, the item's
// markdown escaped and its hidden comment's text kept (as read live on
// gitlab.com, 2026-10-05).
func TestGitLabTaskNote(t *testing.T) {
	m := taskNote.FindStringSubmatch(`marked the checklist item **Close \#4 as obsolete: src/x\.go says a \= \"1\" — why it is so\.  workline:proposal\=4/close\-obsolete** as completed`)
	if m == nil {
		t.Fatal("not read")
	}
	if got := escaped.ReplaceAllString(m[1], "$1"); got != `Close #4 as obsolete: src/x.go says a = "1" — why it is so.  workline:proposal=4/close-obsolete` || m[2] != "completed" {
		t.Errorf("item %q, %q", got, m[2])
	}
	if taskNote.MatchString("changed the description") {
		t.Error("another system note read as a tick")
	}
}
