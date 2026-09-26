// Package report writes findings in the formats forges read: SARIF 2.1.0,
// for GitHub code scanning (and GitLab Ultimate), and GitLab's Code Quality
// report, which every GitLab tier shows on a merge request. Both place a
// finding on a file and a line, so a finding whose `where` names no file of
// the repository (a commit's subject, a gate, a folder) is left out: it stays
// in the verdict and in --json.
package report

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

// Item is a finding with the role that made it and how its run ended.
type Item struct {
	Role, Status string
	verdict.Finding
}

// FromRole lists the findings of one role's run.
func FromRole(role string, r *engine.Result) []Item {
	var out []Item
	for _, f := range r.Findings {
		out = append(out, Item{role, r.Status, f})
	}
	return out
}

// FromLine lists the findings of every step of a line.
func FromLine(r *line.Result) []Item {
	var out []Item
	for _, s := range r.Steps {
		if s.Result != nil {
			out = append(out, FromRole(strings.TrimSuffix(s.Name, " (handoff)"), s.Result)...)
		}
		if s.Gate != nil {
			for _, f := range s.Gate.Findings {
				out = append(out, Item{s.Name, s.Gate.Status, f})
			}
		}
	}
	return out
}

// level is the SARIF level: a finding that blocks is an error, one lowered
// to a warning is a warning, one reported by a run that passed is a note.
func (i Item) level() string {
	switch {
	case i.Level == "warn":
		return "warning"
	case i.Level == "block" || i.Status != verdict.Pass:
		return "error"
	}
	return "note"
}

func (i Item) rule() string { return i.Role + "/" + i.Rule }

// fingerprint stays the same while the rule and the place do, whatever the
// message says, so a forge sees the same finding from one run to the next.
func (i Item) fingerprint() string {
	sum := sha256.Sum256([]byte(i.rule() + "\x00" + i.Where))
	return hex.EncodeToString(sum[:])
}

type placed struct {
	Item
	path string
	line int
}

// place finds the file and line a finding names: a file of the repository,
// at the heading its #anchor names if any, else at its first line.
func place(repo string, items []Item) []placed {
	var out []placed
	for _, i := range items {
		path, anchor, _ := strings.Cut(i.Where, "#")
		path = filepath.ToSlash(filepath.Clean(path))
		if i.Where == "" || filepath.IsAbs(path) || strings.HasPrefix(path, "../") {
			continue
		}
		if st, err := os.Stat(filepath.Join(repo, path)); err != nil || !st.Mode().IsRegular() {
			continue
		}
		out = append(out, placed{i, path, headingLine(filepath.Join(repo, path), anchor)})
	}
	return out
}

// headingLine is the line of the heading whose anchor is given, or 1.
func headingLine(file, anchor string) int {
	if anchor == "" {
		return 1
	}
	f, err := os.Open(file)
	if err != nil {
		return 1
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for n := 1; s.Scan(); n++ {
		if t := s.Text(); strings.HasPrefix(t, "#") && slug(strings.TrimLeft(t, "# ")) == anchor {
			return n
		}
	}
	return 1
}

// slug is the anchor a heading gets on GitHub and GitLab.
func slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r > 127:
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// SARIF writes the findings as one SARIF 2.1.0 run of the tool "workline",
// each rule named <role>/<rule>.
func SARIF(repo string, items []Item) ([]byte, error) {
	type text struct {
		Text string `json:"text"`
	}
	type rule struct {
		ID               string `json:"id"`
		ShortDescription text   `json:"shortDescription"`
		HelpURI          string `json:"helpUri"`
	}
	type region struct {
		StartLine int `json:"startLine"`
	}
	type location struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region region `json:"region"`
		} `json:"physicalLocation"`
	}
	type result struct {
		RuleID              string            `json:"ruleId"`
		Level               string            `json:"level"`
		Message             text              `json:"message"`
		Locations           []location        `json:"locations"`
		PartialFingerprints map[string]string `json:"partialFingerprints"`
	}
	rules := map[string]rule{}
	results := []result{}
	for _, p := range place(repo, items) {
		id := p.rule()
		help := "https://github.com/JN0V/workline/blob/main/roles/" + p.Role + "/README.md"
		if strings.HasPrefix(p.Role, "gate:") {
			help = "https://github.com/JN0V/workline/blob/main/docs/spec/gates.md"
		}
		rules[id] = rule{ID: id, ShortDescription: text{p.Rule + ", from " + p.Role}, HelpURI: help}
		var l location
		l.PhysicalLocation.ArtifactLocation.URI = p.path
		l.PhysicalLocation.Region.StartLine = p.line
		msg := p.Message
		if msg == "" {
			msg = p.Rule
		}
		results = append(results, result{RuleID: id, Level: p.level(), Message: text{msg},
			Locations: []location{l}, PartialFingerprints: map[string]string{"workline/v1": p.fingerprint()}})
	}
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ordered := make([]rule, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, rules[id])
	}
	doc := map[string]any{
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"version": "2.1.0",
		"runs": []any{map[string]any{
			"tool":    map[string]any{"driver": map[string]any{"name": "workline", "informationUri": "https://github.com/JN0V/workline", "rules": ordered}},
			"results": results,
		}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// CodeQuality writes the findings as a GitLab Code Quality report.
func CodeQuality(repo string, items []Item) ([]byte, error) {
	severity := map[string]string{"error": "blocker", "warning": "minor", "note": "info"}
	type lines struct {
		Begin int `json:"begin"`
	}
	type issue struct {
		Description string `json:"description"`
		CheckName   string `json:"check_name"`
		Fingerprint string `json:"fingerprint"`
		Severity    string `json:"severity"`
		Location    struct {
			Path  string `json:"path"`
			Lines lines  `json:"lines"`
		} `json:"location"`
	}
	out := []issue{}
	for _, p := range place(repo, items) {
		i := issue{Description: p.Message, CheckName: p.rule(), Fingerprint: p.fingerprint(), Severity: severity[p.level()]}
		if i.Description == "" {
			i.Description = p.Rule
		}
		i.Location.Path, i.Location.Lines.Begin = p.path, p.line
		out = append(out, i)
	}
	return json.MarshalIndent(out, "", "  ")
}

// Write writes the reports asked for; an empty file name skips that one.
func Write(repo string, items []Item, sarifFile, codeQualityFile string) error {
	for _, w := range []struct {
		file string
		make func(string, []Item) ([]byte, error)
	}{{sarifFile, SARIF}, {codeQualityFile, CodeQuality}} {
		if w.file == "" {
			continue
		}
		data, err := w.make(repo, items)
		if err != nil {
			return err
		}
		if err := os.WriteFile(w.file, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}
