package engine

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// A pending head whose update's time cannot be read (its tip not in the
// clone) keeps its place: it is not taken for a wait past 30 minutes, and
// the next one in the queue is not updated in the same run.
func TestOwnQueueKeepsAPendingHeadWhoseTimeIsUnread(t *testing.T) {
	repo, _, _ := gitRepo(t)
	updated := false
	entry := func(n, checks, head string) *inLine {
		return &inLine{
			where: "#" + n, ready: true,
			mr: forge.MergeRequest{Here: true, Branch: devmark.Branch + "queue-" + n},
			st: forge.MergeState{Behind: true, AutoMerge: true, Checks: checks, Head: head, AutoMergeAt: "2026-11-28T10:0" + n + ":00Z"},
			up: func(bool, string) (bool, error) { updated = true; return true, nil },
		}
	}
	q := []*inLine{entry("1", "pending", strings.Repeat("d", 40)), entry("2", "success", "")}
	sum, err := ownQueue(nil, FollowOptions{Repo: repo, Base: "main"}, &Result{}, q, "")
	if err != nil {
		t.Fatal(err)
	}
	if updated || !strings.Contains(sum, "#1: head of the queue, its checks pending") || !strings.Contains(sum, "#2: queued, position 2") {
		t.Fatalf("the pending head lost its place: updated %v, %q", updated, sum)
	}
}
