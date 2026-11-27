package developer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// stalledBeat runs one beat of the reconcile on the developer's pull
// request 2 of f: when a round is due, the round runs with an agent that
// could not be reached (no call spent, nothing said, nothing changed) and
// its answer is applied to the pull request as the engine applies it.
// It says whether a round ran, the record on the pull request after the
// beat, and whether the issue was blocked for a person.
func stalledBeat(t *testing.T, f *forge.Fake) (ran bool, rec devmark.Act, blocked bool) {
	t.Helper()
	target := forge.Target{Kind: "merge-request", ID: 2}
	mr, err := f.MergeRequest(2)
	if err != nil {
		t.Fatal(err)
	}
	kept, err := forge.ReadKept(f, target)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := kept.Last(forge.Marker("sticky=developer/" + devmark.RecordKey))
	rec, _ = devmark.ReadAct(body)
	due, err := RoundDue(f, t.TempDir(), mr)
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		return false, rec, false
	}
	r, _, err := readAsked(f, kept, t.TempDir(), rec, mr, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Branch, r.At = mr.Branch, "HEAD"
	repo := gitRepo(t, map[string]string{"src/lib.go": "package lib\n"})
	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeIn(runDir, roundFile, r); err != nil {
		t.Fatal(err)
	}
	if err := writeIn(runDir, boundsFile, Bounds{Issue: 1, Scope: []string{"src/lib.go"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "calls.jsonl"), []byte(`{"agent":"cmd","failed":"agent unavailable: login expired"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), nil); err != nil {
		t.Fatal(err)
	}
	if code := reviewPost(runDir, repo, Settings{Rounds: 3, MaxTurns: 40}); code != 0 {
		t.Fatalf("reviewPost = %d", code)
	}
	out, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range out {
		switch m, _ := in.Value.(map[string]any); in.Kind {
		case "comment":
			key, _ := m["sticky"].(string)
			text, _ := m["body"].(string)
			if err := f.Sticky(target, text, forge.Marker("sticky=developer/"+key), true); err != nil {
				t.Fatal(err)
			}
		case "label":
			if strings.Contains(fmt.Sprint(m), "blocked") {
				blocked = true
			}
		}
	}
	kept, err = forge.ReadKept(f, target)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = kept.Last(forge.Marker("sticky=developer/" + devmark.RecordKey))
	rec, _ = devmark.ReadAct(body)
	return true, rec, blocked
}

// stalledForge is the developer's pull request holding a person's reply
// not heard yet: a round is due on it.
func stalledForge(t *testing.T) *forge.Fake {
	return threadForge(t, nil, []forge.FakeComment{
		{Body: "Please also handle the empty list.", Author: "dana", Insider: true, ID: "505"},
	})
}

// The round counts toward the round
// Given a pull request whose round cannot progress (its agent is called
// and spends nothing, says nothing, changes nothing),
// When the develop dispatch runs on the next beat,
// Then the round counts toward the round limit in the record, or the
// issue becomes blocked for a person.
func TestStalledRoundCountsTowardTheRoundLimit(t *testing.T) {
	f := stalledForge(t)
	ran, rec, blocked := stalledBeat(t, f)
	if !ran {
		t.Fatal("no round was due on the first beat: the state is not the stalled one")
	}
	if rec.Round < 1 && len(rec.Rounds) == 0 && !blocked {
		t.Errorf("a round that cannot progress is counted nowhere: record round %d, %d rounds listed, issue not blocked", rec.Round, len(rec.Rounds))
	}
}

// The round is not dispatched again
// Given a round that ran without calling an agent and changed nothing,
// When the next beat comes,
// Then the round is not dispatched again with the same inputs.
func TestNoAgentRoundIsNotDispatchedAgain(t *testing.T) {
	f := stalledForge(t)
	if ran, _, _ := stalledBeat(t, f); !ran {
		t.Fatal("no round was due on the first beat: the state is not the stalled one")
	}
	again, _, blocked := stalledBeat(t, f)
	if again && !blocked {
		t.Error("the same round was dispatched again on the same inputs, its issue not blocked")
	}
}

// The count has advanced or the
// Given a pull request stuck at "review round 1 of 5",
// When 5 beats pass,
// Then the count has advanced or the issue is blocked.
func TestStuckRoundAdvancesOrBlocksWithinFiveBeats(t *testing.T) {
	f := stalledForge(t)
	var rec devmark.Act
	blocked := false
	for beat := 0; beat < 5 && !blocked; beat++ {
		_, rec, blocked = stalledBeat(t, f)
	}
	if !blocked && rec.Round < 1 && len(rec.Rounds) == 0 {
		t.Errorf("after 5 beats the count is still %d (%d rounds listed) and the issue is not blocked", rec.Round, len(rec.Rounds))
	}
}
