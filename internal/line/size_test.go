package line

import (
	"reflect"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
)

// The closed issues are read the last updated first until 100 the line
// touched: those with no record are skipped, not counted, and no page
// past the hundredth with a record is asked.
func TestDoneRecordsStopsAtAHundredTouched(t *testing.T) {
	asked := 0
	page := func(n int) ([]forge.Issue, bool, error) {
		asked = n
		var list []forge.Issue
		for i := range forge.ClosedPageSize {
			list = append(list, forge.Issue{ID: 10000 - (n-1)*forge.ClosedPageSize - i, Closed: true})
		}
		return list, true, nil
	}
	recordOf := func(is forge.Issue) (Record, bool, error) {
		if is.ID%2 == 0 {
			return Record{}, false, nil // half before the line
		}
		return Record{Spent: map[string][]int{"developer": {1000}}}, true, nil
	}
	got, pages, err := doneRecords(page, recordOf, func(forge.Issue) (bool, error) { return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != sizeWindow || pages != 2 || asked != 2 {
		t.Errorf("read %d done in %d pages (asked up to %d), want %d in 2", len(got), pages, asked, sizeWindow)
	}
}

// The record's comment, a run counted, says the new total; the issue a
// merge request closes is the one its text closes, else its branch's.
func TestWithRecordSaysTheNewTotal(t *testing.T) {
	body := "**workline's line** last woke the developer. 5k fresh tokens spent on it so far. Edited by the engine, not by hand.\n\n<!-- workline:line-record {\"state\":\"ready\"} -->"
	got := withRecord(body, Record{State: "ready", Spent: map[string][]int{"developer": {5000}, "reviewer": {1600}}})
	if !strings.Contains(got, " 6.6k fresh tokens spent on it so far.") || !strings.Contains(got, `"reviewer":[1600]`) || strings.Count(got, "line-record") != 1 {
		t.Errorf("withRecord = %q", got)
	}
	if n := ClosedIssue(forge.MergeRequest{Branch: "workline/tester/issue-7"}); n != 7 {
		t.Errorf("closedIssue of the tester's branch = %d, want 7", n)
	}
	if n := ClosedIssue(forge.MergeRequest{Branch: "workline/developer/3", Body: "Fixes #4"}); n != 4 {
		t.Errorf("closedIssue = %d, want 4, what its text closes", n)
	}
}

// A run's fresh tokens are added by role; a run that called no agent, or
// whose calls said no tokens, adds none; the record read is not changed.
func TestSpendCountsFreshTokensByRole(t *testing.T) {
	was := map[string][]int{"tester": {7000}}
	steps := []Step{
		{Name: "product-owner", Result: &engine.Result{Calls: []agent.Call{{TokensIn: 3000, TokensCached: 1000, TokensOut: 500}, {TokensIn: 100}}}},
		{Name: "reviewer", Result: &engine.Result{}},
		{Name: "tester", Result: &engine.Result{Calls: []agent.Call{{Failed: "unreachable"}}}},
		{Name: "tester", Result: &engine.Result{Calls: []agent.Call{{TokensIn: 2000}}}},
	}
	got := spend(was, steps)
	want := map[string][]int{"product-owner": {2600}, "tester": {7000, 2000}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("spend = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(was, map[string][]int{"tester": {7000}}) {
		t.Errorf("the record read was changed: %v", was)
	}
	if n := (Record{Spent: got}).Fresh(); n != 11600 {
		t.Errorf("Fresh = %d, want 11600", n)
	}
}

func TestTokensSaidShort(t *testing.T) {
	for n, want := range map[int]string{950: "950", 10500: "10.5k", 400000: "400k", 1600000: "1.6M", 156200: "156.2k"} {
		if got := Tokens(n); got != want {
			t.Errorf("Tokens(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestP85ByNearestRank(t *testing.T) {
	if got := p85([]int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}); got != 9 {
		t.Errorf("p85 of 1..10 = %d, want 9", got)
	}
	if got := p85([]int{42}); got != 42 {
		t.Errorf("p85 of one = %d, want 42", got)
	}
}
