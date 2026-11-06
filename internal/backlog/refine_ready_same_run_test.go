package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// Report matches apply: Given a run holding a refine and a ready on the same
// issue (the refine rewrites a Validation the role drafted before, so the
// rewrite is a draft again once applied), When the plan is written, Then its
// line for the ready says what apply will do: apply refuses a ready on a
// draft no person accepted and no assumption takes, so the plan never says
// the issue is moved to ready.
func TestReportMatchesApply(t *testing.T) {
	need, verif, scope := work.Name(work.Need), work.Name(work.Verification), work.Name(work.Scope)
	valid := work.Name(work.Validation)
	body := "## " + need + "\n\nRows are lost on export.\n\n---\n\n## " + valid + "\n\n" + DraftMarker +
		"\n\nIt must export every row.\n\n## " + verif + "\n\nRun the export.\n\n## " + scope + "\n\nexport.go"
	st := State{Confirmed: "abc1234"}
	st.SetWrote(valid, SectionDigest(body, valid))
	state := forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Author: "ann", Insider: true, Title: "Export loses rows", Body: body,
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
		Tester:       true,
		Wait:         DefaultWait,
		MovedPercent: 100,
		Acts: map[string]Setting{
			"refine": {Mode: Act, Drafts: Act},
			"ready":  {Mode: Act},
		},
	}
	refine := Proposal{Do: "refine", Issue: 7, Why: "scenarios in the form the tester reads",
		Validation: "- **Export every row:** Given a table, When it is exported, Then every row is written."}
	ready := Proposal{Do: "ready", Issue: 7, Why: "the spec is complete"}
	plan, err := Decide(f, "", role, cfg, map[int]Proposal{0: refine, 1: ready}, []int{7}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	refined := false
	for _, d := range plan.Decisions {
		if d.Act.Do == "refine" && d.Mode == Act {
			refined = true
		}
	}
	if !refined {
		t.Fatalf("setup: the refine was not planned as done: %+v %+v", plan.Decisions, plan.Findings)
	}
	for _, d := range plan.Decisions {
		if d.Act.Do == "ready" && d.Mode == Act && len(d.Act.Assumed) == 0 {
			t.Errorf("the plan moves #7 to ready, but apply refuses it: the refined Validation is a draft no person accepted: %+v", d)
		}
	}
	for _, fd := range plan.Findings {
		if fd.Rule == "done" && strings.Contains(fd.Message, "to ready") && !strings.Contains(fd.Message, "taken as written") {
			t.Errorf("the report says %q, apply refuses it", fd.Message)
		}
	}
}
