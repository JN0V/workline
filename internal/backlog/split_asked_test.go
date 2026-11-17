package backlog

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// askedSplitFixture is a repository holding export.go and a fake forge with
// issue #7, opened by ann, a person of the project, whose one comment is
// the given text.
func askedSplitFixture(t *testing.T, comment string) (*forge.Fake, string) {
	t.Helper()
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
	item := forge.FakeItem{
		ID: 7, Author: "ann", Insider: true, Title: "Export in two formats",
		Body: "The shop needs its orders exported as CSV and as JSON.",
	}
	if comment != "" {
		item.Comments = []forge.FakeComment{{Body: comment, Author: "ann", Insider: true}}
	}
	data, _ := json.Marshal(forge.FakeState{Issues: []forge.FakeItem{item}})
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return &forge.Fake{Path: path}, repo
}

// askedSplitProposal is the product owner's split of #7 into two parts,
// not past its size.
func askedSplitProposal() Proposal {
	return Proposal{Do: "split", Issue: 7, Why: "the person asked for it", Into: []Child{
		{Title: "Export CSV", Need: "Orders are exported as CSV.", Validation: "- **CSV:** Given orders, When exported, Then a CSV is written.", Scope: "`export.go`", Sources: []string{"export.go"}},
		{Title: "Export JSON", Need: "Orders are exported as JSON.", Validation: "- **JSON:** Given orders, When exported, Then a JSON is written.", Scope: "`export.go`", Sources: []string{"export.go"}},
	}}
}

// Split asked, split done: Given an issue whose person writes "split" with
// the parts, When the product owner runs on that comment, Then the split
// is done (it opens the parts, each linked to the issue, and says so),
// though `split` is off by default and the issue is not past its size.
func TestSplitAskedByAPersonIsDone(t *testing.T) {
	f, repo := askedSplitFixture(t, "split: one issue for the CSV export, one for the JSON export")
	cfg := Config{
		Level:        Normal,
		Wait:         DefaultWait,
		MovedPercent: 100,
		Acts:         map[string]Setting{"split": {Mode: Off, Max: 2}}, // the role's default
		Scope:        []int{7},
	}
	plan, err := Decide(f, repo, role, cfg, map[int]Proposal{0: askedSplitProposal()}, []int{7}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range plan.Decisions {
		if d.Act.Do == "split" && d.Mode == Act {
			return
		}
	}
	t.Fatalf("a person asked a split on #7, yet it is not planned as done: %+v %+v", plan.Decisions, plan.Findings)
}

// Split asked, split off: Given a project that set `split` off, When a
// person asks a split, Then the product owner says on the issue that the
// project allows no split, and leaves the text as it is.
func TestSplitAskedWhereTheProjectSetSplitOff(t *testing.T) {
	f, repo := askedSplitFixture(t, "split: one issue for the CSV export, one for the JSON export")
	cfg := Config{
		Level:        Normal,
		Wait:         DefaultWait,
		MovedPercent: 100,
		Acts:         map[string]Setting{"split": {Mode: Off, Max: 2}},
		ModeSet:      []string{"split"}, // the project wrote the mode itself
		Scope:        []int{7},
	}
	plan, err := Decide(f, repo, role, cfg, map[int]Proposal{0: askedSplitProposal()}, []int{7}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range plan.Decisions {
		if d.Mode == Act || d.Mode == Propose {
			t.Errorf("the project set split off, yet %s is planned as %s: %+v", d.Act.Do, d.Mode, d)
		}
	}
	said := strings.Join(plan.drops[7], "\n") + "\n" + plan.Silent[7]
	for _, fd := range plan.Findings {
		if fd.Where == "#7" || strings.Contains(fd.Where, "7") {
			said += "\n" + fd.Message
		}
	}
	if !strings.Contains(strings.ToLower(said), "no split") {
		t.Errorf("nothing on #7 says the project allows no split; said: %q", said)
	}
	is, err := f.Issue(7)
	if err != nil {
		t.Fatal(err)
	}
	if is.Body != "The shop needs its orders exported as CSV and as JSON." {
		t.Errorf("the issue's text changed: %q", is.Body)
	}
}
