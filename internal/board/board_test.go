package board

import (
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
