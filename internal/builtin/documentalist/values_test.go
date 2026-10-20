package documentalist

import "testing"

// A pin follows a release's bump; the release a feature came in, or a
// minimum, does not (#346).
func TestSaysAsNow(t *testing.T) {
	doc := func(body string) string { return "---\nsources: [ci/job.yml]\nchecked: abc1234\n---\n# The job\n\n" + body }
	for _, c := range []struct {
		name, body, v string
		want          bool
	}{
		{"a pin in an example", "```yaml\nWORKLINE_VERSION: v0.38.0\n```\n", "v0.38.0", true},
		{"an install line", "Run `go install example.com/tool@v0.38.0`.\n", "v0.38.0", true},
		{"an image's tag", "The image `ghcr.io/jn0v/workline:v0.38.0` holds it.\n", "v0.38.0", true},
		{"a version stated", "This is Clock **1.4.1**.\n", "1.4.1", true},
		{"a pin in code after a word that marks history in prose", "```\nFROM tool:v0.38.0\n```\n", "v0.38.0", true},
		{"the release a feature came in", "The image holds `ssh-keygen` from v0.38.0.\n", "v0.38.0", false},
		{"the first release after it", "The templates need the first release after v0.38.0.\n", "v0.38.0", false},
		{"added in", "The name, added in v0.38.0.\n", "v0.38.0", false},
		{"since", "Signed since v0.38.0.\n", "v0.38.0", false},
		{"a minimum", "It needs v0.38.0 or later.\n", "v0.38.0", false},
		{"a minimum, and newer", "It works with 0.38.0 and newer.\n", "v0.38.0", false},
		{"a minimum, a plus", "It needs `v0.38.0+`.\n", "v0.38.0", false},
		{"a range", "Declare `tool >=0.38.0`.\n", "v0.38.0", false},
		{"a longer number holding it", "Pin `v10.38.0`, or `v0.38.0.1`.\n", "v0.38.0", false},
		{"a pre-release", "Pin `v1.2.0-rc.1`.\n", "v1.2.0-rc.1", true},
		{"in the header only", "Nothing pinned here.\n", "abc1234", false},
		{"history beside a pin", "From v0.38.0, it signs.\n\n```yaml\nVERSION: v0.38.0\n```\n", "v0.38.0", true},
		{"a derived block", "<!-- workline:derive version -->v0.38.0<!-- workline:end -->\n", "v0.38.0", false},
		{"French: depuis", "Disponible depuis v0.38.0.\n", "v0.38.0", false},
	} {
		if got := saysAsNow(doc(c.body), c.v); got != c.want {
			t.Errorf("%s: saysAsNow = %v, want %v", c.name, got, c.want)
		}
	}
}
