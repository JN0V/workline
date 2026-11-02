package productowner

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// A full read names the kind it asks for, on an issue no kind tells only
// (#895); the end to end is tests/conformance/cases/product-owner/
// kind-full-read-without-kind-is-said.yaml.
func TestKindAsked(t *testing.T) {
	none := forge.Issue{ID: 888}
	bug := forge.Issue{ID: 889, Labels: []string{"bug"}}
	set := &backlog.State{Kind: "bug"} // the role set it once: a person removed it since
	for _, c := range []struct {
		is   forge.Issue
		st   *backlog.State
		mode string
		want bool
	}{
		{none, nil, backlog.Act, true},
		{none, &backlog.State{}, backlog.Propose, true},
		{none, nil, backlog.Off, false},
		{bug, nil, backlog.Act, false},
		{none, set, backlog.Act, false},
	} {
		got := kindAsked(c.is, c.st, c.mode)
		if (got != "") != c.want || c.want && !strings.Contains(got, "`kind` on #888") {
			t.Errorf("#%d %v %s: %q, want asked %v", c.is.ID, c.is.Labels, c.mode, got, c.want)
		}
	}
}

// An issue at its tests is as near a merge as a ready one (#895).
func TestNearMergeTests(t *testing.T) {
	if n := nearMerge(forge.Issue{Labels: []string{backlog.LabelNeedsTests}}); n != 1 {
		t.Errorf("needs-tests: %d, want 1", n)
	}
}
