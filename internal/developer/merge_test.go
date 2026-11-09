package developer

import "testing"

func TestIssueKind(t *testing.T) {
	for _, c := range []struct {
		labels      []string
		kind, label string
	}{
		{[]string{"bug", "workline:ready"}, KindFix, "bug"},
		{[]string{"Type::Bug"}, KindFix, "Type::Bug"},
		{[]string{"enhancement"}, KindFeature, "enhancement"},
		{[]string{"task"}, KindTask, "task"},
		{[]string{"bug", "enhancement"}, KindFeature, "enhancement"}, // a feature first: the safe side
		{[]string{"task", "bug"}, KindFix, "bug"},
		{[]string{"workline:ready", "debug"}, "", ""}, // whole names only
		{nil, "", ""},
	} {
		kind, label := IssueKind(c.labels)
		if kind != c.kind || label != c.label {
			t.Errorf("IssueKind(%q) = %q, %q; want %q, %q", c.labels, kind, label, c.kind, c.label)
		}
	}
}

// TestUnreviewed: the merge reads the reviewer's verdict at the pull
// request's head only, its lenses having read it (ADR-0065, change 2).
func TestUnreviewed(t *testing.T) {
	const head = "4feb95cd88448f137d793775c4ad7b591c1ac175"
	for _, c := range []struct {
		name string
		m    Merge
		want string
	}{
		{"none", Merge{}, "no reviewer's verdict is on it"},
		{"older", Merge{ReviewedAt: "1111111111111111111111111111111111111111"}, "the reviewer's verdict is on 1111111, not on its head 4feb95c"},
		{"owed", Merge{ReviewedAt: head, Owed: true}, "the reviewer's lenses have not read its head 4feb95c yet: only its rules ran"},
		{"head", Merge{ReviewedAt: head}, ""},
	} {
		if got := c.m.Unreviewed(head); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		comment string
		owed    bool
	}{
		{"<!-- workline:record reviewed= runs=0 owed=all -->", true},
		{"<!-- workline:record reviewed=abc patches=def runs=2 owed=all read=all -->", true},
		{"<!-- workline:record reviewed=abc runs=1 -->", false},
	} {
		if got := reviewerOwed.MatchString(c.comment); got != c.owed {
			t.Errorf("owed %q: %v, want %v", c.comment, got, c.owed)
		}
	}
}
