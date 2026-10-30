package board

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
)

func TestKey(t *testing.T) {
	for in, want := range map[string]string{"workflow::In progress": "inprogress", "To-do": "todo", "WIP 🚧": "wip", "Done": "done"} {
		if got := Key(in); got != want {
			t.Errorf("Key(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	m, un := Match([]string{"Backlog", "Todo", "Doing", "Closed"}, role.Board{})
	if m != (Slots{"Todo", "Doing", "Closed"}) || len(un) != 0 {
		t.Errorf("Match = %+v, %v", m, un)
	}
	m, un = Match([]string{"Next", "Build"}, role.Board{Ready: "next", Doing: "Ship"})
	if m != (Slots{Ready: "Next"}) || len(un) != 1 || un[0] != "doing: Ship" {
		t.Errorf("Match with a setting = %+v, %v", m, un)
	}
}

func TestChoose(t *testing.T) {
	gh := []forge.Board{{Forge: "github", Number: 1, Columns: []string{"Todo"}}, {Forge: "github", Number: 2, Columns: []string{"Todo"}}, {Forge: "github", Number: 5}}
	if b, c := Choose(gh, role.Board{}); b != nil || len(c) != 2 {
		t.Errorf("two projects: %v, %v", b, c)
	}
	if b, _ := Choose(gh, role.Board{Project: 2}); b == nil || b.Number != 2 {
		t.Errorf("project 2: %v", b)
	}
	if b, c := Choose(gh, role.Board{Project: 5}); b != nil || len(c) != 0 {
		t.Errorf("a project with no Status: %v, %v", b, c)
	}
	gl := []forge.Board{{Forge: "gitlab", Number: 7, Columns: []string{"A", "B"}}, {Forge: "gitlab", Number: 9, Columns: []string{"B", "A"}}}
	if b, _ := Choose(gl, role.Board{Project: 3}); b == nil || b.Number != 7 {
		t.Errorf("GitLab, the same labels, project: set: %v", b)
	}
	gl[1].Columns = []string{"C", "D"}
	if b, c := Choose(gl, role.Board{}); b != nil || len(c) != 2 {
		t.Errorf("GitLab, two boards matching nothing: %v, %v", b, c)
	}
}

// TestToGitHub: a GitHub project's Status is written by the role, but on
// a user's project, or at the claim when its workflow moves the card.
func TestToGitHub(t *testing.T) {
	to := func(owner string, flows map[string]bool, unread, slot string) *Move {
		t.Helper()
		st := forge.FakeState{Boards: []forge.Board{{Forge: "github", Number: 3, Name: "P", Owner: owner,
			Columns: []string{"Todo", "In progress", "Done"}, Workflows: flows}}, BoardsUnread: unread}
		data, _ := json.Marshal(st)
		path := filepath.Join(t.TempDir(), "forge.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		mv, _ := To(&forge.Fake{Path: path}, slot, role.Board{})
		return mv
	}
	linked := map[string]bool{LinkedWorkflow: true, ClosedWorkflow: true}
	if mv := to("organization", linked, "", "ready"); !mv.Writes() || mv.Column != "Todo" || mv.Project != 3 || mv.Labels() || mv.Said != "" {
		t.Errorf("ready, an organisation's: %+v", mv)
	}
	if mv := to("organization", linked, "", "doing"); mv.Writes() || mv.Left != "workflow" {
		t.Errorf("doing, its workflow on: %+v", mv)
	}
	if mv := to("organization", map[string]bool{ClosedWorkflow: false}, "", "doing"); !mv.Writes() || mv.Column != "In progress" || !strings.Contains(mv.Said, "*Item closed*") {
		t.Errorf("doing, its workflows off: %+v", mv)
	}
	if mv := to("user", map[string]bool{ClosedWorkflow: true}, "", "ready"); mv.Writes() || mv.Left != "user" {
		t.Errorf("ready, a user's: %+v", mv)
	}
	if f := to("user", linked, "", "doing").Findings(); len(f) != 1 || f[0].Rule != "board-user-project" || strings.Contains(f[0].Message, "turn on") {
		t.Errorf("a user's, its workflows on: %+v", f)
	}
	if f := to("user", map[string]bool{LinkedWorkflow: true}, "", "ready").Findings(); len(f) != 1 || !strings.Contains(f[0].Message, "turn on its workflow *Item closed* to move cards to `Done`") {
		t.Errorf("a user's, Item closed off: %+v", f)
	}
	unread := forge.Board{Name: "P", Number: 3, Owner: "user", WorkflowsUnread: "403"}
	if s := userSaid(unread, Slots{Doing: "In progress"}, func(string) bool { return false }); !strings.Contains(s, "its workflows not read (403)") || strings.Contains(s, "move the cards to the other columns;") {
		t.Errorf("a user's, its workflows unread: %s", s)
	}
	if f := to("organization", linked, "", "ready").Findings(); f != nil {
		t.Errorf("an organisation's, its workflows on, says nothing: %+v", f)
	}
	if mv := to("organization", nil, "502 Bad Gateway", "ready"); mv != nil {
		t.Errorf("unread: %+v", mv)
	}
}
