package engine

import "testing"

// A doc's header lines recording when it was read are left out, in a
// front matter or a README's header; the same words in its body, or in a
// doc with no header, are kept: they say what the doc says.
func TestUnrecorded(t *testing.T) {
	cases := []struct{ doc, want string }{
		{"---\nsources: [a.go]\nchecked: 111\njudged: 222\njudged-in-parts: 333\nverified: agent:x\n---\n# T\n\njudged: kept\n",
			"---\nsources: [a.go]\nchecked: 111\n---\n# T\n\njudged: kept\n"},
		{"<!-- workline\nsources: [a.go]\njudged: 222\n-->\n# T\n", "<!-- workline\nsources: [a.go]\n-->\n# T\n"},
		{"# T\n\njudged: 222\n", "# T\n\njudged: 222\n"},
		{"---\njudged: 222\nno end\n", "---\njudged: 222\nno end\n"},
	}
	for _, c := range cases {
		if got := unrecorded(c.doc); got != c.want {
			t.Errorf("unrecorded(%q) = %q, want %q", c.doc, got, c.want)
		}
	}
}

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
