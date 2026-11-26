// Package docmark is what the documentalist keeps of a doc and of its
// weekly sample's settings, as the sample reads them: a doc's sources and
// its checked commits, the sample's judge and caps (ADR-0014, ADR-0046,
// ADR-0070 §6 e). The documentalist reads and writes them the same.
package docmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
	"go.yaml.in/yaml/v3"
)

// Doc is a documentation file that declares its sources.
type Doc struct {
	Path    string
	Sources []string
	Checked map[string]string // repository name ("" = this one) -> commit
	// Judged is the commit an agent judged the doc at without vouching for
	// every sentence: it is not put before an agent again until a source
	// changes after it. A `judged-in-parts` a doc still records is read,
	// never written again, and holds nothing (docs/adr/0074).
	Judged string
}

// SampleSettings is the `sample` setting: the judge reading the sample, and
// the least independence from the model that vouched it may stand at.
type SampleSettings struct {
	Judge   string `json:"judge"`    // an --ai value: claude:opus, cmd:…
	AtLeast string `json:"at-least"` // provider, model or context (ADR-0005)
	// After: a commit (a tag) before which nothing is sampled — the
	// release whose engine earns `checked` (ADR-0014, step 0).
	After Ref `json:"after"`
	// AIMaxTokens: the tokens a week's sample may spend, every call
	// together (in, cache included, and out); past it, the acts left are
	// not read, said. 0: SampleTokens (ADR-0046).
	AIMaxTokens int `json:"ai-max-tokens"`
	// Panel: the three judges that settle a dispute (ADR-0046), as --ai
	// values; unset, Claude's three models when the judge is Claude's.
	Panel []string `json:"panel"`
}

// SampleTokens is the weekly sample's cap on tokens when the setting
// names none (ADR-0046).
const SampleTokens = 400000

// TokensCap is the weekly sample's cap on tokens.
func (s SampleSettings) TokensCap() int {
	if s.AIMaxTokens > 0 {
		return s.AIMaxTokens
	}
	return SampleTokens
}

// Ref is a setting naming a commit or a tag. Written unquoted, a commit of
// digits only (7515148) is a number to YAML: it is read as the text it is.
// One YAML would read otherwise than written (0123456, octal) never gets
// here: the config is refused, saying to quote it (role.CheckConfig).
type Ref string

func (r *Ref) UnmarshalJSON(data []byte) error {
	text := string(data)
	if text != "" && strings.Trim(text, "0123456789") == "" {
		*r = Ref(text)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("a commit or a tag, as text: %s", text)
	}
	*r = Ref(s)
	return nil
}

// ErrNotInClone is a `checked` naming a commit the clone does not have: a
// squash may have left it off the branch (ADR-0046).
var ErrNotInClone = errors.New("a commit this clone does not have")

// SampleView is what the weekly sample reads of the role's settings: the
// sample's own, the docs' globs, and the characters a doc's sources may
// take to be judged whole.
type SampleView struct {
	Sample   SampleSettings
	Docs     []string
	WholeCap int
}

// IsDoc says whether a path is one of the docs the settings name.
func (v SampleView) IsDoc(p string) bool {
	for _, g := range v.Docs {
		if pathglob.Match(g, p) {
			return true
		}
	}
	return false
}

// Frontmatter is the YAML block at the top of a doc.
type Frontmatter struct {
	Sources []string `yaml:"sources"`
	// A node, not a string: read as YAML, a commit like 11180e1 is a number.
	Checked yaml.Node `yaml:"checked"`
	Judged  yaml.Node `yaml:"judged"`
}

// Header returns a doc's header — YAML between `---` lines, or, where a
// frontmatter would show on the forge (a README), between `<!-- workline` and
// `-->` — and how many lines it takes. A doc without one has no header.
func Header(content string) (meta string, lines int) {
	all := strings.Split(content, "\n")
	var end string
	switch {
	case len(all) > 0 && all[0] == "---":
		end = "---"
	case len(all) > 0 && strings.TrimSpace(all[0]) == "<!-- workline":
		end = "-->"
	default:
		return "", 0
	}
	for i := 1; i < len(all); i++ {
		if strings.TrimSpace(all[i]) == end {
			return strings.Join(all[1:i], "\n"), i + 1
		}
	}
	return "", 0
}

// ParseDoc reads a doc's header. A doc without sources is not tracked.
func ParseDoc(path string, content []byte) (*Doc, error) {
	block, n := Header(string(content))
	if n == 0 {
		return nil, nil
	}
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		return nil, fmt.Errorf("%s: frontmatter: %w", path, err)
	}
	if len(fm.Sources) == 0 {
		return nil, nil
	}
	d := &Doc{Path: path, Sources: fm.Sources, Checked: map[string]string{}, Judged: fm.Judged.Value}
	switch c := fm.Checked; c.Kind {
	case yaml.ScalarNode:
		d.Checked[""] = c.Value
	case yaml.MappingNode:
		for i := 0; i+1 < len(c.Content); i += 2 {
			d.Checked[c.Content[i].Value] = c.Content[i+1].Value
		}
	}
	return d, nil
}
