package engine

import "testing"

func TestMergeDocFrontMatterByKey(t *testing.T) {
	old := "---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n"
	cases := []struct {
		name, ours, theirs, want string
		ok                       bool
	}{
		{"base moved checked, the branch added judged beside it",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 111\njudged: 333\n---\n# T\n\nTwo hours.\n",
			"---\nsources: [a.go]\nchecked: 222\njudged: 333\n---\n# T\n\nTwo hours.\n", true},
		{"both moved checked: base's, the later judgement",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 333\n---\n# T\n\nOne hour.\n",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n", true},
		{"the branch added a key first",
			"---\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n",
			"---\ntype: reference\nsources: [a.go]\nchecked: 111\n---\n# T\n\nOne hour.\n",
			"---\ntype: reference\nsources: [a.go]\nchecked: 222\n---\n# T\n\nOne hour.\n", true},
		{"both changed the same sentence",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nNinety minutes.\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\nTwo hours.\n",
			"", false},
	}
	for _, c := range cases {
		got, ok := mergeDoc(t.TempDir(), old, c.ours, c.theirs)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: got %v %q, want %v %q", c.name, ok, got, c.ok, c.want)
		}
	}
}
