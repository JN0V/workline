package line

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// raced is a forge whose record comment is changed by a line step right
// after the count reads the issue's comments, before it writes.
type raced struct {
	*forge.Fake
	step func()
	done bool
}

func (r *raced) Comments(t forge.Target) ([]string, error) {
	got, err := r.Fake.Comments(t)
	if t.Kind == "issue" && !r.done {
		r.done = true
		r.step()
	}
	return got, err
}

// Both kept: Given a record written by a line step between the count's
// read and its write, When the count writes, Then the record holds the
// step's state and the run's tokens.
func TestCountRunKeepsTheStepsRecordWrittenMeanwhile(t *testing.T) {
	dir := t.TempDir()
	state := forge.FakeState{
		Issues:        []forge.FakeItem{{ID: 7, Title: "issue", Labels: []string{}, Comments: []forge.FakeComment{}}},
		MergeRequests: []forge.FakeItem{{ID: 3, Title: "feat: seven", Body: "Closes #7", Labels: []string{}, Comments: []forge.FakeComment{}}},
	}
	data, _ := json.Marshal(state)
	path := filepath.Join(dir, "forge.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &forge.Fake{Path: path}
	issue := forge.Target{Kind: "issue", ID: 7}
	marker := forge.Marker("sticky=" + recordKey)
	if err := fake.Sticky(issue, backlog.Inert(recordBody(Record{V: recordV, State: "developing", Hops: 1}, "developer")), marker, true); err != nil {
		t.Fatal(err)
	}

	run := filepath.Join(dir, "runs", "20261101T100000-reviewer")
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "out", "run.yaml"), []byte("role: reviewer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	call := `{"agent":"fake","tokens-in":100,"tokens-out":20}` + "\n"
	if err := os.WriteFile(filepath.Join(run, "out", "calls.jsonl"), []byte(call), 0o644); err != nil {
		t.Fatal(err)
	}

	f := &raced{Fake: fake, step: func() {
		step := Record{V: recordV, State: "ready", Hops: 2}
		if err := fake.Sticky(issue, backlog.Inert(recordBody(step, "developer")), marker, false); err != nil {
			t.Error(err)
		}
	}}
	if err := countRun(f, 3, run); err != nil {
		t.Fatal(err)
	}

	comments, err := fake.Comments(issue)
	if err != nil {
		t.Fatal(err)
	}
	rec, had := recordIn(comments)
	if !had {
		t.Fatal("no record left on the issue")
	}
	if rec.State != "ready" || rec.Hops != 2 {
		t.Errorf("record state %q, hops %d; want the step's: ready, 2", rec.State, rec.Hops)
	}
	if got := rec.Spent["reviewer"]; !slices.Equal(got, []int{120}) {
		t.Errorf("spent by the reviewer %v, want [120]", got)
	}
	if len(rec.Counted) != 1 {
		t.Errorf("counted %v, want the one run", rec.Counted)
	}
}
