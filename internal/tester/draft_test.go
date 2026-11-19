package tester

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/testmark"
)

// The record other roles read of the tester's Draft (testmark.ReadDraft)
// is the tester's own, as it reads it: read where it reads it, its
// fields the same, none where it reads none.
func TestTheDraftOthersReadIsTheTestersRecord(t *testing.T) {
	full := record{Issue: 7, Base: "abc", Files: map[string]string{"a_test.go": "b1"}, Tests: []string{"TestA"},
		Scenarios: []Scenario{{Name: "adds"}}, Covers: map[string][]string{"adds": {"TestA"}}, RedUnproven: "a build broke"}
	own := func(body string) forge.FakeComment {
		return forge.FakeComment{Body: body + "\n<!-- workline:sticky=tester/" + recordKey + " -->", Own: true}
	}
	bodies := map[int][]forge.FakeComment{
		1: {own(full.body())},
		2: {own("no record here")},
		3: {own("<!-- workline:tester-record {\"issue\":7,\"files\":{}} -->")},
		4: {own("<!-- workline:tester-record {not json} -->")},
		5: nil,
	}
	var st forge.FakeState
	for id, cs := range bodies {
		st.MergeRequests = append(st.MergeRequests, forge.FakeItem{ID: id, Branch: testmark.TestsBranch + "7", Comments: cs})
	}
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path, Repo: t.TempDir()}
	for id := range bodies {
		r, missing, err := readRecord(f, id)
		d, ok, derr := testmark.ReadDraft(f, id)
		if (err != nil) != (derr != nil) || (err == nil && missing == nil) != ok {
			t.Errorf("#%d: the tester reads %v, %v; the kit %v, %v", id, missing, err, ok, derr)
			continue
		}
		want := testmark.Draft{Issue: r.Issue, Base: r.Base, Files: r.Files, Scenarios: r.Scenarios, Covers: r.Covers, RedUnproven: r.RedUnproven}
		if ok && !reflect.DeepEqual(d, want) {
			t.Errorf("#%d: the kit reads %+v, the tester %+v", id, d, want)
		}
	}
	if d, ok, _ := testmark.ReadDraft(f, 1); !ok || d.Issue != 7 || d.RedUnproven == "" || len(d.Covers) != 1 {
		t.Errorf("a whole record read as %+v, %v", d, ok)
	}
}
