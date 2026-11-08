package simulator

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
)

// refiner answers the product owner with a refine that writes the
// issue's Verification, the rest left as the person wrote it:
// the body edited, the issue kept in refine.
type refiner struct{ calls int }

func (r *refiner) Propose(req agent.Request) (agent.Call, error) {
	r.calls++
	out := `- refine:
    issue: 7
    verification: "A test publishes two events and counts two deliveries."
    sources: [DomoticsCore-Core/include/DomoticsCore/EventBus.h]
    why: "The issue says what is wrong, not how it will be proved."
`
	return agent.Call{Agent: "fake", Model: "sim"}, os.WriteFile(filepath.Join(req.RunDir, "out", "intentions.yaml"), []byte(out), 0o644)
}

// TestOwnWriteThenAccepted replays round 6's refine on the trial copy
// (#1196, the product owner's 3/15/7 calls): the product owner's run
// edits the body, which must not wake it again on the next tick; a
// person's workline:accepted put on afterwards is new input, and wakes
// it (ADR-0063, §3).
func TestOwnWriteThenAccepted(t *testing.T) {
	work := t.TempDir()
	hermetic(t, work)
	repo := filepath.Join(work, "repo")
	if out, err := exec.Command("cp", "-a", repoTemplate, repo).CombinedOutput(); err != nil {
		t.Fatalf("the repository: %v\n%s", err, out)
	}
	// The template's line is the trial's (config): no reviewer after the
	// product owner, so its run is the only writer.
	at := time.Date(2026, 11, 8, 7, 0, 0, 0, time.UTC)
	f := filepath.Join(work, "forge.json")
	state := forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Title: "Events lost", Author: "ann", Insider: true,
		Body:     "## Need\n\nAnn needs every event delivered, so no reading is lost.\n\n## Validation\n\n- **Two events:** Given two events published, When the bus runs, Then both are delivered.\n\n## Scope\n\nEventBus in DomoticsCore-Core/include/DomoticsCore/EventBus.h\n",
		Labels:   []string{"workline:to-refine"},
		Comments: []forge.FakeComment{},
		Labeled:  map[string]string{"workline:to-refine": at.Add(-time.Hour).Format(time.RFC3339)},
	}}}
	write := func() {
		data, _ := json.Marshal(state)
		if err := os.WriteFile(f, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func() {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatal(err)
		}
	}
	write()
	roles, _ := filepath.Abs(filepath.Join("..", "..", "roles"))
	t.Setenv(clock.Fixed, at.Format(time.RFC3339))
	po := &refiner{}
	defer agent.Hook("refiner", po)()
	base := engine.Options{Repo: repo, RolesDir: roles, Forge: "fake:" + f, OpenMergeRequest: true,
		Target: &forge.Target{Kind: "issue", ID: 7}}
	ask := base
	ask.AI = "none"
	ask.Inputs = map[string]string{line.ReconcileInput: "list"}
	why := ""
	due := func() bool {
		r := line.RunWith(line.IssueEvent, ask, &line.Gates{})
		why = r.Summary
		for _, f := range r.Findings {
			why += "\n" + f.Rule + ": " + f.Message
		}
		return slices.Contains(r.Due, 7)
	}

	if !due() {
		t.Fatal("a fresh issue in refine is not due")
	}
	run := base
	run.AI = "hook:refiner"
	res := line.RunWith(line.IssueEvent, run, &line.Gates{})
	read()
	if po.calls != 1 || state.Issues[0].Body == "" || !slices.Contains(state.Issues[0].Labels, "workline:to-refine") {
		t.Fatalf("the product owner's run did not refine #7 in refine (%d calls): %s", po.calls, res.Summary)
	}
	// Tick 2: the body is the product owner's own edit; nothing new.
	t.Setenv(clock.Fixed, at.Add(10*time.Minute).Format(time.RFC3339))
	if due() {
		t.Fatalf("the product owner's own edit wakes it again on the next tick: %s", why)
	}
	// A person's yes, put on after the run: new input (#1196).
	read()
	state.Issues[0].Labels = append(state.Issues[0].Labels, "workline:accepted")
	state.Issues[0].Labeled["workline:accepted"] = at.Add(20 * time.Minute).Format(time.RFC3339)
	write()
	t.Setenv(clock.Fixed, at.Add(25*time.Minute).Format(time.RFC3339))
	if !due() {
		t.Fatalf("a person's workline:accepted wakes no role: %s", why)
	}
}
