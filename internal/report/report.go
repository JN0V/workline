// Package report writes findings in the formats forges read: SARIF 2.1.0,
// for GitHub code scanning (and GitLab Ultimate), and GitLab's Code Quality
// report, which every GitLab tier shows on a merge request; and the
// documentalist's findings on a merge request as a check's annotations,
// which open no conversation (#412). Each places a finding on a file and a
// line, so a finding whose `where` names no file of the repository (a
// commit's subject, a gate, a folder) is left out: it stays in the verdict
// and in --json.
package report

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/verdict"
)

// Item is a finding with the role that made it and how its run ended.
type Item struct {
	Role, Status string
	verdict.Finding
	// A tool's finding a gate counts on a change (ADR-0049): its own SARIF
	// level, and a fingerprint with no line number, stable while it moves.
	ToolLevel, Print string
}

// FromRole lists the findings of one role's run.
func FromRole(role string, r *engine.Result) []Item {
	var out []Item
	for _, f := range r.Findings {
		out = append(out, Item{Role: role, Status: r.Status, Finding: f})
	}
	return out
}

// FromLine lists the findings of every step of a line, and those its
// gates count on a change, each the tool's. A gate whose verdict was read
// from the job that ran it is that job's to report.
func FromLine(r *line.Result) []Item {
	var out []Item
	for _, s := range r.Steps {
		if s.Result != nil {
			out = append(out, FromRole(strings.TrimSuffix(s.Name, " (handoff)"), s.Result)...)
		}
		if s.Gate != nil && s.From == "" {
			for _, f := range s.Gate.Findings {
				out = append(out, Item{Role: s.Name, Status: s.Gate.Status, Finding: f})
			}
		}
		if s.New != nil && s.From == "" {
			for _, f := range s.New.Findings {
				out = append(out, gateItem(s, f))
			}
		}
	}
	return out
}

// gateItem is a tool's finding a gate counts, named by its check and rule.
func gateItem(s line.Step, f gate.Finding) Item {
	where := f.Path
	if f.Line > 0 {
		where = fmt.Sprintf("%s:%d", f.Path, f.Line)
	}
	rule := f.Check
	if f.Rule != "" {
		rule += "/" + f.Rule
	}
	return Item{Role: s.Name, Status: s.Status, Finding: verdict.Finding{Rule: rule, Where: where, Message: f.Message},
		ToolLevel: f.Level, Print: f.Fingerprint}
}

// level is the SARIF level: a finding that blocks is an error, one lowered
// to a warning is a warning, one reported by a run that passed is a note.
func (i Item) level() string {
	switch {
	case i.ToolLevel == "error", i.ToolLevel == "warning", i.ToolLevel == "note":
		return i.ToolLevel
	case i.Level == "question": // a decision put to a person, not a defect (the reviewer, #126)
		return "note"
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
	if i.Print != "" {
		return i.Print
	}
	sum := sha256.Sum256([]byte(i.rule() + "\x00" + i.Where))
	return hex.EncodeToString(sum[:])
}

type placed struct {
	Item
	path string
	line int
}

// place finds the file and line a finding names: a file of the repository,
// at the line its :<line> gives, at the heading its #anchor names, else at
// its first line. A file a range of commits held, and no longer holds, is
// still placed: code scanning shows it on the commit.
func place(repo string, items []Item) []placed {
	return placeIn(func(path string) ([]byte, error) {
		if st, err := os.Stat(filepath.Join(repo, path)); err != nil || !st.Mode().IsRegular() {
			return nil, os.ErrNotExist
		}
		return os.ReadFile(filepath.Join(repo, path))
	}, items)
}

// placeIn is place, the files read by read: a file it cannot read is none.
func placeIn(read func(path string) ([]byte, error), items []Item) []placed {
	var out []placed
	for _, i := range items {
		path, anchor, _ := strings.Cut(i.Where, "#")
		line := 0
		if p, n, ok := strings.Cut(path, ":"); ok {
			if v, err := strconv.Atoi(n); err == nil && v > 0 {
				path, line = p, v
			}
		}
		path = filepath.ToSlash(filepath.Clean(path))
		if i.Where == "" || filepath.IsAbs(path) || strings.HasPrefix(path, "../") {
			continue
		}
		if line == 0 {
			data, err := read(path)
			if err != nil {
				continue
			}
			line = headingLine(data, anchor)
		}
		out = append(out, placed{i, path, line})
	}
	return out
}

// headingLine is the line of the heading whose anchor is given, or 1.
func headingLine(data []byte, anchor string) int {
	if anchor == "" {
		return 1
	}
	s := bufio.NewScanner(bytes.NewReader(data))
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

// CodeQuality writes the findings as a GitLab Code Quality report. A
// tool's finding a gate counts is as severe as the tool says, a blocking
// one of workline's a blocker.
func CodeQuality(repo string, items []Item) ([]byte, error) {
	severity := map[string]string{"error": "blocker", "warning": "minor", "note": "info"}
	toolSeverity := map[string]string{"error": "major", "warning": "minor", "note": "info"}
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
		if p.ToolLevel != "" {
			i.Severity = toolSeverity[p.level()]
		}
		if i.Description == "" {
			i.Description = p.Rule
		}
		i.Location.Path, i.Location.Lines.Begin = p.path, p.line
		out = append(out, i)
	}
	return json.MarshalIndent(out, "", "  ")
}

// postedOnMergeRequest are the roles whose findings a merge request gets
// from workline itself: the documentalist's as a check's annotations, the
// job applying the run writes them; the reviewer's in its own review.
var postedOnMergeRequest = map[string]bool{"documentalist": true, "reviewer": true}

// ForCodeScanning is what of a run on event goes to code scanning: on a
// merge request, neither the documentalist's findings nor the reviewer's,
// which workline posts there itself, nor the gates'. Code scanning would
// open a conversation for each, whatever its level, that a base requiring
// them resolved holds the merge on (#412), and is paid on a private
// repository (ADR-0049); the committer's stay.
func ForCodeScanning(event string, items []Item) []Item {
	if event != "merge-request" {
		return items
	}
	var out []Item
	for _, i := range items {
		if !postedOnMergeRequest[i.Role] && !strings.HasPrefix(i.Role, "gate:") {
			out = append(out, i)
		}
	}
	return out
}

// Most annotations written for one head, and the most of a message: what
// a merge request's own run hands the applying job is data its code could
// have written.
const (
	maxAnnotations   = 500
	maxAnnotationLen = 4000
)

// Annotations are the findings as a check's annotations, each on the file
// and line it names, the files as read gives them — the head judged, not
// the base checked out —; one that names no file is left out, as from the
// reports. Beyond maxAnnotations, the rest are counted, not written.
func Annotations(read func(path string) ([]byte, error), items []Item) (notes []forge.Annotation, left int) {
	level := map[string]string{"error": "failure", "warning": "warning", "note": "notice"}
	for _, p := range placeIn(read, items) {
		if len(notes) == maxAnnotations {
			left++
			continue
		}
		msg := p.Message
		if msg == "" {
			msg = p.Rule
		}
		notes = append(notes, forge.Annotation{Path: p.path, Line: p.line, Level: level[p.level()], Title: p.rule(), Message: clip(msg)})
	}
	return notes, left
}

// clip cuts a message the run wrote to what an annotation holds.
func clip(msg string) string {
	if len(msg) > maxAnnotationLen {
		return strings.ToValidUTF8(msg[:maxAnnotationLen], "") + "…"
	}
	return msg
}

// Write writes the reports asked for of a run on event; an empty file
// name skips that one. Code scanning gets what ForCodeScanning keeps.
func Write(repo, event string, items []Item, sarifFile, codeQualityFile string) error {
	for _, w := range []struct {
		file  string
		items []Item
		make  func(string, []Item) ([]byte, error)
	}{{sarifFile, ForCodeScanning(event, items), SARIF}, {codeQualityFile, items, CodeQuality}} {
		if w.file == "" {
			continue
		}
		data, err := w.make(repo, w.items)
		if err != nil {
			return err
		}
		if err := os.WriteFile(w.file, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}
