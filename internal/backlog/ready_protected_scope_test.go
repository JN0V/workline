package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Protected Scope: Given a Scope naming only a CI workflow, When the product
// owner would move it to ready, Then it waits on a person, saying a person
// makes that change: the ready is not done, and the report says why.
func TestReadyWaitsOnAPersonForACIOnlyScope(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	body := "## Need\n\nCI must run the linter.\n\n" +
		"## Validation\n\n- **Lint runs:** Given a push, When CI runs, Then the linter runs.\n\n" +
		"## Verification\n\nRead the workflow.\n\n" +
		"## Scope\n\n.github/workflows/ci.yml"
	st := State{Confirmed: "abc1234"}
	state := forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Author: "ann", Insider: true, Title: "Lint in CI", Body: body,
		Labels:   []string{LabelToRefine},
		Comments: []forge.FakeComment{{Body: FormatState(st) + "\n\n" + StateMarker(role)}},
	}}}
	path := filepath.Join(t.TempDir(), "forge.json")
	data, _ := json.Marshal(state)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}

	cfg := Config{
		Level:        Enterprising,
		Wait:         DefaultWait,
		MovedPercent: 100,
		Acts:         map[string]Setting{"ready": {Mode: Act}},
	}
	ready := Proposal{Do: "ready", Issue: 7, Why: "the spec is complete"}
	plan, err := Decide(f, "", role, cfg, map[int]Proposal{0: ready}, []int{7}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range plan.Decisions {
		if d.Act.Do == "ready" && d.Mode == Act {
			t.Errorf("#7 names only a CI workflow, yet the plan moves it to ready: %+v", d)
		}
	}
	said := false
	for _, fd := range plan.Findings {
		if strings.Contains(fd.Message, "a person makes") {
			said = true
		}
	}
	if !said {
		t.Errorf("no finding says a person makes that change: %+v", plan.Findings)
	}
}
