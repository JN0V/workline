package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// queued is an accepted request held behind, ready, green, on a developer's
// branch; up is its update.
func queued(id int, at string, up func(bool, string) (bool, error)) *inLine {
	return &inLine{
		where: fmt.Sprintf("#%d", id),
		up:    up,
		mr:    forge.MergeRequest{ID: id, Here: true, Branch: "workline/developer/" + fmt.Sprint(id)},
		st:    forge.MergeState{AutoMerge: true, Behind: true, Checks: "success", AutoMergeAt: at},
	}
}

// An error at the head's update other than an unreachable forge passes
// that head over, said, and the next one is updated: the run goes on.
func TestOwnQueueUpdateErrorPassesTheHeadOver(t *testing.T) {
	failed := func(bool, string) (bool, error) { return true, errors.New("its comments could not be read") }
	updated := false
	res := &Result{}
	merged := func(bool, string) (bool, error) {
		updated = true
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merged-from-base", Where: "#2"})
		return true, nil
	}
	q := []*inLine{queued(1, "2026-11-20T10:00:00Z", failed), queued(2, "2026-11-20T11:00:00Z", merged)}
	if err := ownQueue(nil, FollowOptions{Base: "main"}, res, q, nil); err != nil {
		t.Fatalf("ownQueue ended the run: %v", err)
	}
	if !updated {
		t.Fatalf("the next one was not updated: %+v", res.Findings)
	}
	lines := map[string]string{}
	for _, x := range res.Findings {
		if x.Rule == "behind-not-updated" {
			lines[x.Where] = x.Message
		}
	}
	if lines["#1"] != "passed over, main not merged in, said above" || lines["#2"] != "head of the queue, main merged in" {
		t.Fatalf("lines: %v; findings: %+v", lines, res.Findings)
	}
}

// An unreachable forge at the head's update ends the run.
func TestOwnQueueUnreachableEndsTheRun(t *testing.T) {
	down := func(bool, string) (bool, error) { return true, fmt.Errorf("%w: timeout", forge.ErrUnreachable) }
	q := []*inLine{queued(1, "2026-11-20T10:00:00Z", down)}
	if err := ownQueue(nil, FollowOptions{Base: "main"}, &Result{}, q, nil); !errors.Is(err, forge.ErrUnreachable) {
		t.Fatalf("err = %v, want the forge unreachable", err)
	}
}

// lines are the queue's lines ownQueue said, by merge request.
func lines(res *Result) map[string]string {
	l := map[string]string{}
	for _, x := range res.Findings {
		if x.Rule == "behind-not-updated" {
			l[x.Where] = x.Message
		}
	}
	return l
}

// Where no accepted request workline may update is held behind clean — a
// base that requires no branch up to date holds one behind only for a
// conflict —, there is no queue: nothing is said, nothing updated.
func TestOwnQueueSaysNothingWithNoQueue(t *testing.T) {
	updated := false
	up := func(bool, string) (bool, error) { updated = true; return true, nil }
	conflicting := queued(1, "2026-11-20T10:00:00Z", up)
	conflicting.st = forge.MergeState{} // its conflict said by its merge round: not weighed ready
	persons := queued(2, "2026-11-20T11:00:00Z", up)
	persons.mr.Branch = "feat/sessions"
	res := &Result{}
	fs := []verdict.Finding{{Rule: "merge-conflict", Level: "warn", Where: "#1", Message: "it conflicts with main in a.go"}}
	if err := ownQueue(nil, FollowOptions{Base: "main"}, res, []*inLine{conflicting, persons}, fs); err != nil {
		t.Fatal(err)
	}
	if updated || len(res.Findings) > 0 {
		t.Fatalf("no queue, yet updated %v, said %+v", updated, res.Findings)
	}
}

// A branch not a developer's is passed over, said by what it is: another
// role's of workline, a person's, a fork's.
func TestOwnQueueSaysWhoseBranch(t *testing.T) {
	none := func(bool, string) (bool, error) { return false, nil }
	tester, persons, fork := queued(1, "2026-11-20T10:00:00Z", none), queued(2, "2026-11-20T10:01:00Z", none), queued(3, "2026-11-20T10:02:00Z", none)
	tester.mr.Branch, persons.mr.Branch, fork.mr.Here = "workline/tester/7", "feat/sessions", false
	res := &Result{}
	if err := ownQueue(nil, FollowOptions{Base: "main"}, res, []*inLine{tester, persons, fork, queued(4, "2026-11-20T10:03:00Z", none)}, nil); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"#1": "passed over, workline's tester branch, not a developer's",
		"#2": "passed over, a person's branch",
		"#3": "passed over, a person's branch",
		"#4": "passed over, main not merged in",
	}
	if got := lines(res); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("lines: %v, want %v", got, want)
	}
}

// The order is the priority, then when auto-merge was turned on, read as
// times, not as text; one with no time after every one with one.
func TestOwnQueueOrdersByPriorityThenTime(t *testing.T) {
	var order []string
	up := func(id string) func(bool, string) (bool, error) {
		return func(bool, string) (bool, error) { order = append(order, id); return false, nil }
	}
	none, late, early, urgent := queued(1, "", up("#1")), queued(2, "2026-11-20T09:30:00Z", up("#2")), queued(3, "2026-11-20T10:00:00+01:00", up("#3")), queued(4, "2026-11-20T23:00:00Z", up("#4"))
	urgent.mr.Labels = []string{"workline:priority/1"}
	// None merges in: each is tried in turn, the order they are taken in.
	if err := ownQueue(nil, FollowOptions{Base: "main"}, &Result{}, []*inLine{none, late, early, urgent}, nil); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(order) != "[#4 #3 #2 #1]" {
		t.Fatalf("order: %v, want [#4 #3 #2 #1]", order)
	}
}
