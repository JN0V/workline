package tester

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/kit/testmark"

	"github.com/JN0V/workline/internal/forge"
)

const parkedBody = "## Validation\n\n- **Two by three:** Given 2 and 3, When multiplied, Then 6.\n"

// parkedForge writes a fake forge holding two open issues, each with
// scenarios: #1 with the labels given, #2 ready.
func parkedForge(t *testing.T, labels []string, only bool) *forge.Fake {
	t.Helper()
	items := []forge.FakeItem{{ID: 1, Title: "one", Body: parkedBody, Labels: labels, Author: "reporter"}}
	if !only {
		items = append(items, forge.FakeItem{ID: 2, Title: "two", Body: parkedBody, Labels: []string{"workline:ready"}, Author: "reporter"})
	}
	data, err := json.Marshal(forge.FakeState{Issues: items})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return &forge.Fake{Path: path, Repo: t.TempDir()}
}

// Parked ready issue.
// Given an open issue labelled workline:ready and workline:parked, and
// another ready one not parked, when the tester picks, then it picks
// the one not parked; and the parked one is not waiting for its tests.
func TestPickLeavesAParkedReadyIssueToItsTests(t *testing.T) {
	f := parkedForge(t, []string{"workline:ready", "workline:parked"}, false)
	id, err := pick(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != 2 {
		t.Errorf("pick = #%d, want #2: the parked #1 left alone", id)
	}
	issues, err := f.Issues()
	if err != nil {
		t.Fatal(err)
	}
	for _, is := range issues {
		if is.ID == 1 && testmark.Untested(is, nil, nil) {
			t.Error("a parked ready issue is said to wait for its tests")
		}
	}
}

// Unparked.
// Given a ready issue parked, when a person removes workline:parked,
// then the tester's next pick takes it as before: left alone while
// parked, picked once the label is off.
func TestPickTakesAnIssueAgainOnceUnparkedToItsTests(t *testing.T) {
	id, err := pick(parkedForge(t, []string{"workline:ready", "workline:parked"}, true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != 0 {
		t.Fatalf("pick = #%d: the parked issue was picked", id)
	}
	id, err = pick(parkedForge(t, []string{"workline:ready"}, true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Errorf("pick = #%d, want #1 once workline:parked is off", id)
	}
}
