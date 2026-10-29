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
