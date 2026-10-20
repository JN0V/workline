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
		{"French: à partir de", "Signé à partir de v0.38.0.\n", "v0.38.0", false},
		{"French: avant", "Non signé avant v0.38.0.\n", "v0.38.0", false},
		{"French: après", "La première version après v0.38.0.\n", "v0.38.0", false},
		{"French: ou plus récent", "Il faut v0.38.0 ou plus récent.\n", "v0.38.0", false},
		{"French: ou ultérieure", "Il faut v0.38.0 ou ultérieure.\n", "v0.38.0", false},
		{"before", "Unsigned before v0.38.0.\n", "v0.38.0", false},
		{"at least", "It needs at least v0.38.0.\n", "v0.38.0", false},
		{"minimum", "Minimum v0.38.0.\n", "v0.38.0", false},
		{"fixed in", "The leak, fixed in v0.38.0.\n", "v0.38.0", false},
		{"onwards", "Signed from now on: v0.38.0 onwards.\n", "v0.38.0", false},
		{"or higher", "It needs v0.38.0 or higher.\n", "v0.38.0", false},
		{"in code quotes after a marker", "The image holds it from `v0.38.0`.\n", "v0.38.0", false},
		{"in code quotes, added in", "The name, added in `v0.38.0`.\n", "v0.38.0", false},
		{"in code quotes before a minimum", "It needs `v0.38.0` or later.\n", "v0.38.0", false},
		{"a plus holding more is a pin", "Require `example.com/tool v0.38.0+incompatible`.\n", "v0.38.0", true},
		{"a word between from and the version", "Install from tag v0.38.0.\n", "v0.38.0", true},
		{"a marker closing the line before", "The templates need the first release after\nv0.38.0, its image too.\n", "v0.38.0", false},
		{"a minimum opening the next line", "The templates need v0.38.0\nor later.\n", "v0.38.0", false},
		{"a sentence ended on the line before", "It signs.\nv0.38.0 is the version to pin.\n", "v0.38.0", true},
		{"a heading is no sentence before", "## Since\n\nv0.38.0 is the version to pin.\n", "v0.38.0", true},
		{"a code block's comment, since", "```yaml\n# since v0.38.0\nNAME: tool\n```\n", "v0.38.0", false},
		{"a code block's comment, a minimum", "```sh\n# needs 0.38.0 or later\nrun\n```\n", "v0.38.0", false},
		{"minimum version", "Minimum version 0.38.0.\n", "v0.38.0", false},
		{"at least version", "It needs at least version v0.38.0.\n", "v0.38.0", false},
		{"a code line is no sentence before", "```\nsigned after\nv0.38.0\n```\n", "v0.38.0", true},
		{"a pin under a key since", "```yaml\nsince: v0.38.0\n```\n", "v0.38.0", true},
		{"a pin under a key from", "```yaml\nfrom: v0.38.0\n```\n", "v0.38.0", true},
		{"a pin under a key after", "```yaml\nafter: v0.38.0\n```\n", "v0.38.0", true},
		{"a pin under a key, no language", "```\nsince: v0.38.0\n```\n", "v0.38.0", true},
		{"a pin in a JSON example", "```json\n{\"since\": \"v0.38.0\"}\n```\n", "v0.38.0", true},
		{"a flag's value", "```sh\ntool --since v0.38.0\n```\n", "v0.38.0", true},
		{"a hash in a string is no comment", "```yaml\nnote: \"x # since v0.38.0\"\n```\n", "v0.38.0", true},
		{"a pin before a comment", "```yaml\nsince: v0.38.0 # the version to pin\n```\n", "v0.38.0", true},
		{"a comment after code, since", "```yaml\nNAME: tool # since v0.38.0\n```\n", "v0.38.0", false},
		{"a comment by the block's language", "```go\n// Since v0.38.0, it signs.\nrun()\n```\n", "v0.38.0", false},
		{"a comment, no language", "```\n// since v0.38.0\n```\n", "v0.38.0", false},
		{"a range in code", "```toml\ntool = \">=0.38.0\"\n```\n", "v0.38.0", false},
		{"a block comment over lines", "```go\n/*\n since v0.38.0\n*/\nrun()\n```\n", "v0.38.0", false},
		{"code after a block comment closed", "```go\n/* the pin */\nconst Version = \"v0.38.0\"\n```\n", "v0.38.0", true},
		{"a comment in a block before is not this one's", "```go\n/* open\n```\n\n```yaml\nsince: v0.38.0\n```\n", "v0.38.0", true},
	} {
		if got := saysAsNow(doc(c.body), c.v); got != c.want {
			t.Errorf("%s: saysAsNow = %v, want %v", c.name, got, c.want)
		}
	}
}
