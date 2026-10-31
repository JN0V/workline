package developer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
	id, passed := pick(issues, nil, s, 0, false, map[int]bool{1: true}, nil)
	if id != 0 || len(passed) != 1 || passed[0].Why != "its claim was let go in this run: taken again at a next one" {
		t.Errorf("pick: %d, %v", id, passed)
	}
}
