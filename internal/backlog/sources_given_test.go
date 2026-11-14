package backlog

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// Sources already given: Given a product owner's answer naming in
// `sources` only files its task gave, When the line applies it, Then
// nothing is left to read again: no needs-code finding is left, so the
// issue moves on to its tests instead of being blocked as lacking code.
func TestSourcesAlreadyGivenLeaveNothingToRead(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // the machine's hooks stay out
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	if err := os.WriteFile(filepath.Join(repo, "export.go"), []byte("package export\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@x.invalid", "commit", "-qm", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, c...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	state := forge.FakeState{Issues: []forge.FakeItem{{
		ID: 7, Author: "ann", Insider: true, Title: "Export loses rows", Body: "Rows are lost on export.",
		Labels: []string{LabelToRefine},
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
		Acts:         map[string]Setting{"sources": {Mode: Act}},
		Scope:        []int{7},
		CodeGiven:    map[int][]string{7: {"export.go"}},
	}
	named := Proposal{Do: "sources", Issue: 7, Sources: []string{"export.go"}, Why: "the code the issue is about"}
	plan, err := Decide(f, repo, role, cfg, map[int]Proposal{0: named}, []int{7}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	acted := false
	for _, d := range plan.Decisions {
		if d.Act.Do == "sources" && d.Mode == Act {
			acted = true
		}
	}
	if !acted {
		t.Fatalf("setup: the sources was not planned as done: %+v %+v", plan.Decisions, plan.Findings)
	}
	for _, fd := range plan.Findings {
		if fd.Rule == verdict.NeedsCode {
			t.Errorf("files the task gave are named, yet the plan asks to read them again: %+v", fd)
		}
	}
}
