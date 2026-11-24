package simulator

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// failedDeveloperRun runs the trial-3 trace until an issue's labels say
// ready, then sets its line record back to refine with 2 hops (the
// state and count the record held before the labels moved on), and runs
// the line once more with the developer's agent failing. It returns the
// record's fields as set and the issue as that run left it.
func failedDeveloperRun(t *testing.T) (set map[string]any, after forge.FakeItem) {
	t.Helper()
	i := slices.IndexFunc(traces, func(tr trace) bool { return tr.name == "trial-3-error-code" })
	if i < 0 {
		t.Fatal("no trace named trial-3-error-code")
	}
	a := &simAgent{}
	w := newWorld(t, traces[i], a)
	id, n := 0, 0
	for ; n < w.k() && id == 0; n++ {
		tick = n
		w.run(start.Add(time.Duration(n) * period))
		for _, is := range w.state(t).Issues {
			if !is.Closed && slices.Contains(is.Labels, "workline:ready") && !slices.Contains(is.Labels, "workline:developing") {
				id = is.ID
				break
			}
		}
	}
	if id == 0 {
		t.Fatal("no issue came to ready within the trace's ticks")
	}
	if slices.ContainsFunc(a.calls, func(c call) bool { return c.Role == "developer" }) {
		t.Fatal("the developer ran before the issue's record was set back")
	}
	s := w.state(t)
	is := issueOf(s, id)
	for j := range is.Comments {
		body := is.Comments[j].Body
		m := recordRe.FindStringSubmatchIndex(body)
		if m == nil {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(body[m[2]:m[3]]), &rec); err != nil {
			t.Fatal(err)
		}
		rec["state"], rec["hops"], rec["reached"] = "refine", 2, "refine"
		for _, k := range []string{"errors", "needs", "claim", "ran"} {
			delete(rec, k)
		}
		data, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		is.Comments[j].Body = body[:m[2]] + string(data) + body[m[3]:]
		set = rec
	}
	if set == nil {
		t.Fatalf("#%d has no line record to set back", id)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(w.forge, data, 0o644); err != nil {
		t.Fatal(err)
	}
	tick = n
	w.run(start.Add(time.Duration(n) * period))
	if !slices.ContainsFunc(a.calls, func(c call) bool { return c.Role == "developer" && c.Kind == machineError }) {
		t.Fatalf("the developer's agent run did not fail on #%d: calls %v", id, a.calls)
	}
	return set, *issueOf(w.state(t), id)
}

// recordFields is the fields of an issue's line record, nil when it has none.
func recordFields(is forge.FakeItem) map[string]any {
	var out map[string]any
	for _, c := range is.Comments {
		if m := recordRe.FindStringSubmatch(c.Body); m != nil {
			var rec map[string]any
			if json.Unmarshal([]byte(m[1]), &rec) == nil {
				out = rec
			}
		}
	}
	return out
}

// TestDeveloperFailureKeepsHops - Hops kept: Given an issue whose record
// says refine with 2 hops while its labels say ready, When the
// developer's agent run fails, Then its record keeps 2 hops, or 3 if it
// catches up with ready, never fewer.
func TestDeveloperFailureKeepsHops(t *testing.T) {
	_, after := failedDeveloperRun(t)
	state, hops, had := record(after)
	if !had {
		t.Fatalf("#%d has no line record after the developer's failed run", after.ID)
	}
	if hops < 2 || hops > 3 {
		t.Errorf("#%d: record %s with %d hops after the developer's failed run, want 2 (kept) or 3 (caught up with ready)", after.ID, state, hops)
	}
}

// TestDeveloperFailureKeepsItsRecord - Record not rewritten: Given the
// same issue, When the developer's run fails, Then its record is not
// written anew as a first record: it is not at one hop, and what the
// record kept of earlier runs is still in it.
func TestDeveloperFailureKeepsItsRecord(t *testing.T) {
	set, after := failedDeveloperRun(t)
	_, hops, had := record(after)
	if !had {
		t.Fatalf("#%d has no line record after the developer's failed run", after.ID)
	}
	if hops == 1 {
		t.Errorf("#%d: record written anew as a first record (1 hop) after the developer's failed run, was 2 hops", after.ID)
	}
	got := recordFields(after)
	for _, k := range []string{"last-run", "spent"} {
		if _, was := set[k]; !was {
			continue
		}
		if _, kept := got[k]; !kept {
			t.Errorf("#%d: record lost its %q after the developer's failed run", after.ID, k)
		}
	}
}

// TestDeveloperBrokenAfterNeedsRoleMovesNothing: Given an issue the
// developer takes after a wait and a needs-role, When the developer's
// run ends broken, Then the issue's labels, state and hops are as before
// (#1875).
func TestDeveloperBrokenAfterNeedsRoleMovesNothing(t *testing.T) {
	holdsUnder(t, "workline-1003", pI3, []string{"progress,waiting,needs-role,progress,broken"})
}
