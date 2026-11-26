package documentalist

import (
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// A claim is what an agent says of one passage of a doc, quoted at the
// lines it cites: the reader's answer, and the citations of a fix.

// lineRange is a range of lines, written "12", "12-14", or as a number.
type lineRange struct{ from, to int }

func (r *lineRange) UnmarshalYAML(n *yaml.Node) error {
	v := strings.ReplaceAll(strings.TrimSpace(n.Value), "–", "-")
	a, b, found := strings.Cut(v, "-")
	from, err1 := strconv.Atoi(strings.TrimSpace(a))
	to, err2 := from, error(nil)
	if found {
		to, err2 = strconv.Atoi(strings.TrimSpace(b))
	}
	if err1 != nil || err2 != nil || from < 1 || to < from {
		return fmt.Errorf("%q is not a range of lines like 12-14", n.Value)
	}
	r.from, r.to = from, to
	return nil
}

func (r lineRange) String() string {
	if r.from == r.to {
		return fmt.Sprintf("line %d", r.from)
	}
	return fmt.Sprintf("lines %d-%d", r.from, r.to)
}

func (r lineRange) overlaps(o lineRange) bool { return r.from <= o.to && o.from <= r.to }

// claim is an answer about one passage of a doc.
type claim struct {
	Lines  lineRange `yaml:"lines"`
	Status string    `yaml:"status"`
	Quote  string    `yaml:"quote"`
	Source *struct {
		Path  string    `yaml:"path"`
		Lines lineRange `yaml:"lines"`
		Quote string    `yaml:"quote"`
	} `yaml:"source"`
	Why string `yaml:"why"`
}

// quotedAt says whether quote is found in the lines r cites, whitespace aside.
func quotedAt(content string, r lineRange, quote string) bool {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	q := strings.Join(strings.Fields(quote), " ")
	if q == "" || r.from < 1 || r.to > len(lines) {
		return false
	}
	return strings.Contains(strings.Join(strings.Fields(strings.Join(lines[r.from-1:r.to], " ")), " "), q)
}
