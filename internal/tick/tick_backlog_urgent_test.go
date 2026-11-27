package tick

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// unreadLabels is a forge whose issues' label events cannot be read.
type unreadLabels struct{ *forge.Fake }

func (unreadLabels) LabelEvents(int) ([]forge.LabelEvent, error) {
	return nil, errors.New("the forge is down")
}

// TestDueUrgentLabelsUnread: Given an Urgent issue with no state, Ready
// full, and its label events unread, When the tick runs, Then no backlog
// run is listed on a guess, and the failed read is said in Unread, naming
// the issue (#2322).
func TestDueUrgentLabelsUnread(t *testing.T) {
	po := "routing: {events: {schedule: [product-owner], issue: [tester, developer]}, wip: {ready: 10}}\n"
	o := setUp(t, po, 10, slices.Clone(done), "")
	fake := o.Forge.(*forge.Fake)
	data, err := os.ReadFile(fake.Path)
	if err != nil {
		t.Fatal(err)
	}
	var s forge.FakeState
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	s.Issues = append(s.Issues, forge.FakeItem{ID: 41, Title: "urgent", Labels: []string{"workline:priority/1"}})
	if data, err = json.Marshal(s); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fake.Path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	o.Forge = unreadLabels{fake}
	r := Due(o)
	if got := jobsOf(r); slices.Contains(got, "backlog") {
		t.Errorf("jobs %v, want no backlog listed on a guess", got)
	}
	if !slices.ContainsFunc(r.Unread, func(s string) bool { return strings.Contains(s, "#41") && strings.Contains(s, "the forge is down") }) {
		t.Errorf("unread %v, want the Urgent #41's labels said unread; said %v", r.Unread, r.Said)
	}
}
