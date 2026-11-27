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
		ready: true,
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
