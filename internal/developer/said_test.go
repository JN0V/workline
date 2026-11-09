package developer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// A claim is let go at claim-hours exactly, not an hour before; one an
// open pull request closes is kept, however old; one whose day the forge
// does not say is kept and said.
func TestStaleAtClaimHours(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	data := `{"issues": [
	  {"id": 1, "title": "Young", "body": "", "labels": ["workline:developing"], "labeled": {"workline:developing": "2026-10-30T10:00:00Z"}},
	  {"id": 2, "title": "At the limit", "body": "", "labels": ["workline:developing"], "labeled": {"workline:developing": "2026-10-30T09:00:00Z"}},
	  {"id": 3, "title": "Old, a pull request", "body": "", "labels": ["workline:developing"], "labeled": {"workline:developing": "2026-01-01T09:00:00Z"}},
	  {"id": 4, "title": "No day", "body": "", "labels": ["workline:developing"]},
	  {"id": 5, "title": "Not claimed", "body": "", "labels": ["workline:ready"], "labeled": {"workline:developing": "2026-01-01T09:00:00Z"}}
	], "merge-requests": []}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &forge.Fake{Path: path}
	issues, err := f.Issues()
	if err != nil {
		t.Fatal(err)
	}
	mrs := []forge.MergeRequest{{ID: 9, Title: "feat: old", Body: "Closes #3"}}
	now := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		hours int
		let   []int
	}{{24, []int{2}}, {23, []int{1, 2}}, {25, nil}} {
		said, found, err := stale(f, issues, mrs, SettingsFrom(map[string]any{"claim-hours": c.hours}), now, nil)
		if err != nil {
			t.Fatal(err)
		}
		var let []int
		for _, in := range said {
			if in.Kind == "label" {
				let = append(let, in.Value.(map[string]any)["issue"].(int))
			}
		}
		if len(let) != len(c.let) || len(let) > 0 && (let[0] != c.let[0] || let[len(let)-1] != c.let[len(c.let)-1]) {
			t.Errorf("claim-hours %d: let go %v, want %v", c.hours, let, c.let)
		}
		unknown := 0
		for _, fd := range found {
			if fd.Rule == "claim-age-unknown" && fd.Where == "#4" {
				unknown++
			}
		}
		if unknown != 1 {
			t.Errorf("claim-hours %d: #4's unknown day said %d times, want once: %v", c.hours, unknown, found)
		}
	}
}

// A claim let go in a run is passed over as let go, not as taken.
func TestPickPassesALetGoClaim(t *testing.T) {
	s := SettingsFrom(nil)
	issues := []forge.Issue{{ID: 1, Labels: []string{"workline:ready", s.ClaimLabel}}}
	id, passed := pick(issues, nil, s, 0, false, map[int]bool{1: true}, nil, nil, nil)
	if id != 0 || len(passed) != 1 || passed[0].Why != "its claim was let go in this run: taken again at a next one" {
		t.Errorf("pick: %d, %v", id, passed)
	}
}

// The line reads the developer's comment for an overlap it passed an
// issue over for (#998): that reason alone, as OverlapWhy says it.
func TestPassedForOverlap(t *testing.T) {
	for c, want := range map[string]bool{
		backlog.Said("**The developer**", "waits on this issue", OverlapWhy(5, "lib.sh"), "#5 merged or let go"): true,
		backlog.Said("**The developer**", "waits on this issue", "its acceptance tests are not written yet", "the tester writes them"): false,
		"**The developer** took this issue. Passed over: #2: " + OverlapWhy(5, "lib.sh"): false,
	} {
		if got := PassedForOverlap(c); got != want {
			t.Errorf("PassedForOverlap(%q) = %v, want %v", c, got, want)
		}
	}
}

// A claim whose agent wrote no code is let go at once when the issue's
// body changed since the claim read it (#1084); the same body, or a
// claim that wrote code, or a record from before Act.Body, waits for
// claim-hours.
func TestNoCodeOnOtherBody(t *testing.T) {
	body := "## Need\n\nI want to multiply.\n"
	for _, c := range []struct {
		a    Act
		want bool
	}{
		{Act{NoCode: true, Body: BodyMark(body)}, false},
		{Act{NoCode: true, Body: BodyMark("## Need\n\nI want to add.")}, true},
		{Act{Body: BodyMark("other")}, false},
		{Act{NoCode: true}, false},
	} {
		if got := NoCodeOnOtherBody(c.a, body); got != c.want {
			t.Errorf("%+v: %v, want %v", c.a, got, c.want)
		}
	}
	// The same input: no code on this body, or on one it did not keep.
	for _, c := range []struct {
		a    Act
		want bool
	}{
		{Act{NoCode: true, Body: BodyMark(body)}, true},
		{Act{NoCode: true}, true},
		{Act{NoCode: true, Body: BodyMark("## Need\n\nI want to add.")}, false},
		{Act{Body: BodyMark(body)}, false},
	} {
		if got := NoCodeOnSameBody(c.a, body); got != c.want {
			t.Errorf("same body %+v: %v, want %v", c.a, got, c.want)
		}
	}
	if BodyMark(body) != BodyMark("  "+body+"\n") {
		t.Error("a body's mark reads its spaces around")
	}
}

// TestBodyChangedEngineLine: the engine's own "Blocked by" line is no
// change of a person's (#1271), and a record's mark kept of the whole
// body before it was left out still reads the same body.
func TestBodyChangedEngineLine(t *testing.T) {
	body := "## Need\n\nI want to add."
	blocked := body + "\n\nBlocked by #9. " + backlog.BlockedByMarker
	if BodyChanged(BodyMark(body), blocked) {
		t.Error("the engine's line read as a change")
	}
	if BodyChanged(digest(blocked), blocked) {
		t.Error("a record's whole-body mark read as a change")
	}
	if !BodyChanged(BodyMark(body), blocked+"\n\nAnd subtract.") {
		t.Error("a person's change not read")
	}
}
