package engine

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
)

// A round on the developer's own branch may move the issue its merge
// request closes to Blocked or Doing, and comment on it; nothing else on
// any other issue (#923).
func TestOutOfLineRoundMoves(t *testing.T) {
	r := &role.Role{Name: "developer", Intentions: []string{"patch", "comment", "label", "note"}}
	mr := forge.MergeRequest{Branch: "workline/developer/3", Title: "Multiply", Body: "Closes #3"}
	moves := roundMovesOf(t.TempDir(), "developer", &mr)
	if moves == nil || moves.issue != 3 {
		t.Fatalf("roundMovesOf = %+v, want issue 3", moves)
	}
	label := func(issue int, add, remove []any, extra map[string]any) intent.Intention {
		v := map[string]any{"issue": issue, "add": add, "remove": remove}
		for k, x := range extra {
			v[k] = x
		}
		return intent.Intention{Kind: "label", Value: v}
	}
	states := []any{"workline:ready", "workline:developing", "workline:to-refine", "workline:needs-tests"}
	for _, c := range []struct {
		name string
		in   intent.Intention
		want string
	}{
		{"blocked on its own", label(3, []any{"workline:blocked"}, states, map[string]any{"card": "blocked"}), ""},
		{"doing on its own", label(3, []any{"workline:developing"}, []any{"workline:ready", "workline:blocked"}, map[string]any{"card": "doing"}), ""},
		{"comment on its own", intent.Intention{Kind: "comment", Value: map[string]any{"issue": 3, "sticky": "claim", "body": "x"}}, ""},
		{"blocked on another", label(9, []any{"workline:blocked"}, states, nil), "another issue"},
		{"comment on another", intent.Intention{Kind: "comment", Value: map[string]any{"issue": 9, "body": "x"}}, "another issue"},
		{"another label on its own", label(3, []any{"bug"}, nil, nil), "no move of its state"},
		{"two labels on its own", label(3, []any{"workline:blocked", "workline:accepted"}, nil, nil), "no move of its state"},
		{"a label off that is no state's", label(3, []any{"workline:blocked"}, []any{"workline:accepted"}, nil), "no move of its state"},
		{"a board's status", label(3, []any{"workline:blocked"}, nil, map[string]any{"status": map[string]any{"project": 1}}), "no move of its state"},
	} {
		got := outOfLine(r, c.in, 0, moves)
		if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: outOfLine = %q, want %q", c.name, got, c.want)
		}
	}
	if m := roundMovesOf(t.TempDir(), "developer", &forge.MergeRequest{Branch: "workline/developer/3", Body: "Closes #4"}); m != nil {
		t.Errorf("a merge request closing another issue: moves %+v, want none", m)
	}
	if got := outOfLine(r, label(3, []any{"workline:blocked"}, nil, nil), 0, nil); !strings.Contains(got, "another issue") {
		t.Errorf("no round: outOfLine = %q, want refused", got)
	}
}
