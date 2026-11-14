package documentalist

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"go.yaml.in/yaml/v3"
)

// A doc no agent can judge is left to a person, and said so once on a
// merge request: its line is not said again at a new head unless the doc or
// its sources changed (#1361). What was said is kept in a comment of its
// own, as the records of docs judged are, with the ids of the doc and its
// sources.

// personKey keys the documentalist's comment that keeps what was said.
const personKey = "too-large"

// personFile is what pre tells post of what it said (in/person-said.yaml).
const personFile = "person-said.yaml"

var personLine = regexp.MustCompile(`<!-- workline:doc-too-large (.*?) -->`)

const personMarker = "workline:doc-too-large"

// personSaid is what a run said of the docs left to a person.
type personSaid struct {
	repo  string
	on    bool
	prior map[string]keptRecord // said at an earlier head, by doc
	now   map[string]keptRecord // said or still standing, by doc
	head  string
	skip  bool // the run decided no doc's line: nothing is written
}

// unreconsidered says the run decided no doc's line, so what is kept stands.
func (p *personSaid) unreconsidered() { p.skip = true }

// keep carries a doc's earlier record, for a doc this run did not reconsider.
func (p *personSaid) keep(doc string) {
	if r, ok := p.prior[doc]; ok {
		if _, done := p.now[doc]; !done {
			p.now[doc] = r
		}
	}
}

// newPersonSaid reads, on a person's merge request, what the comment
// keeps; elsewhere nothing is kept, and every line is said.
func newPersonSaid(repo string) *personSaid {
	p := &personSaid{repo: repo, prior: map[string]keptRecord{}, now: map[string]keptRecord{}}
	t, ok := targetMergeRequest()
	if !ok || os.Getenv("WORKLINE_EVENT") != "merge-request" {
		return p
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		return p
	}
	if _, lists := f.(forge.Merged); !lists {
		return p
	}
	var u othersKept
	body, err := stickyOn(f, t, personKey, &u)
	if err != nil {
		return p
	}
	out, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return p
	}
	p.on, p.head, p.prior = true, strings.TrimSpace(out), parsePersonSaid(body)
	return p
}

// parsePersonSaid reads the records a comment keeps, by doc.
func parsePersonSaid(body string) map[string]keptRecord {
	out := map[string]keptRecord{}
	for _, m := range personLine.FindAllStringSubmatch(body, -1) {
		var r keptRecord
		if json.Unmarshal([]byte(m[1]), &r) == nil && r.Doc != "" && r.At != "" && len(r.IDs) > 0 {
			out[r.Doc] = r
		}
	}
	return out
}

// line is what a finding says of a doc left to a person, said as text: in
// full the first time, then, while neither the doc nor its sources change,
// that they are unchanged since. A text not leaving the doc to a person
// is returned as it is.
func (p *personSaid) line(doc string, d *Doc, text string) string {
	if !strings.Contains(text, personJudges) || !p.on || d == nil {
		return text
	}
	if r, ok := p.prior[doc]; ok {
		if same, _, err := asJudgedAt(p.repo, "HEAD", r); err == nil && same {
			p.now[doc] = r
			return fmt.Sprintf("(left to a person at %s on this merge request: the doc and its sources unchanged since, so not said again)", abbrev(r.At))
		}
	}
	if _, ok := p.now[doc]; !ok {
		if paths, local := docPaths(d); local {
			if ids, err := objectIDs(p.repo, "HEAD", paths); err == nil {
				p.now[doc] = keptRecord{Doc: doc, Key: personKey, At: p.head, IDs: ids}
			}
		}
	}
	return text
}

// write tells post, when what the comment keeps changes, what it keeps.
func (p *personSaid) write(runDir string) error {
	if !p.on || p.skip || hiddenJSON(p.sorted(p.now)) == hiddenJSON(p.sorted(p.prior)) {
		return nil
	}
	data, err := yaml.Marshal(p.sorted(p.now))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "in", personFile), data, 0o644)
}

func (p *personSaid) sorted(m map[string]keptRecord) []keptRecord {
	out := []keptRecord{}
	for _, k := range slices.Sorted(maps.Keys(m)) {
		out = append(out, m[k])
	}
	return out
}

// personSaidBody is the comment keeping what was said: what a person
// reads, then each record hidden.
func personSaidBody(records []keptRecord) string {
	if len(records) == 0 {
		return "**Docs left to a person**: none now."
	}
	var b strings.Builder
	b.WriteString("**Docs left to a person** — no agent can judge these docs, and the documentalist said so once. " +
		"It says it again only if the doc or its sources change.\n\n")
	for _, r := range records {
		fmt.Fprintf(&b, "- `%s`, said at %s\n", r.Doc, abbrev(r.At))
	}
	b.WriteString("\n")
	for _, r := range records {
		fmt.Fprintf(&b, "<!-- %s %s -->\n", personMarker, hiddenJSON(r))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// personComment is the intention proposing the comment, when pre said
// something that changes it; false otherwise.
func personComment(runDir string) (intent.Intention, bool, error) {
	data, err := os.ReadFile(filepath.Join(runDir, "in", personFile))
	if os.IsNotExist(err) {
		return intent.Intention{}, false, nil
	}
	if err != nil {
		return intent.Intention{}, false, err
	}
	var records []keptRecord
	if err := yaml.Unmarshal(data, &records); err != nil {
		return intent.Intention{}, false, err
	}
	value := map[string]any{"sticky": personKey, "body": personSaidBody(records)}
	if len(records) == 0 {
		value["update-only"] = true
	}
	return intent.Intention{Kind: "comment", Value: value}, true, nil
}
