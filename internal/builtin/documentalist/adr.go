package documentalist

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// adrFolders are the folders the usual tools keep decisions in: adr-tools and
// log4brains (doc/adr, docs/adr), MADR (docs/decisions).
var adrFolders = map[string]bool{"adr": true, "adrs": true, "decisions": true}

var (
	adrFile    = regexp.MustCompile(`^(\d+)-[^/]*\.md$`)
	adrMention = regexp.MustCompile(`\bADR[- ]?0*(\d+)\b`)
	adrToolsID = regexp.MustCompile(`\[(\d+)\. `) // adr-tools: "Superseded by [7. Title](0007-title.md)"
	number     = regexp.MustCompile(`\d+`)
	superseded = regexp.MustCompile(`(?i)\bsuper[sc]eded\b`)
	statusLine = regexp.MustCompile(`^\s*(?:[-*]\s+)?\**Status\**\s*:\s*\**\s*(.*)$`)
)

// adr is a decision record: its number, and, once superseded, what replaced it.
type adr struct {
	path       string
	n          int
	superseded bool
	by         int // 0: the status does not say
}

// decisions finds the decision records among the docs: numbered files in a
// decisions folder.
func decisions(t Tree) map[int]*adr {
	out := map[int]*adr{}
	for p := range t.Docs {
		m := adrFile.FindStringSubmatch(path.Base(p))
		if m == nil || !adrFolders[path.Base(path.Dir(p))] {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		out[n] = &adr{path: p, n: n}
	}
	byPath := map[string]int{}
	for n, a := range out {
		byPath[a.path] = n
	}
	for _, a := range out {
		status, supersedes := decisionStatus(t.Docs[a.path])
		if superseded.MatchString(status) {
			a.superseded, a.by = true, cited(a.path, status, byPath)
		}
		for _, old := range supersedes { // MADR, coherence: the successor says what it replaces
			if o := out[old]; o != nil && o != a {
				o.superseded = true
				if o.by == 0 {
					o.by = a.n
				}
			}
		}
	}
	return out
}

// decisionStatus reads a record's status: the frontmatter's `status`, else a
// "Status:" line, else the first paragraph under a "Status" heading; and the
// records its frontmatter says it supersedes.
func decisionStatus(content string) (status string, supersedes []int) {
	if block, n := header(content); n > 0 {
		var fm struct {
			Status     string `yaml:"status"`
			Supersedes any    `yaml:"supersedes"`
		}
		if yaml.Unmarshal([]byte(block), &fm) == nil {
			status = fm.Status
			var ids []string
			switch v := fm.Supersedes.(type) {
			case string:
				ids = []string{v}
			case int:
				ids = []string{strconv.Itoa(v)}
			case []any:
				for _, x := range v {
					ids = append(ids, fmt.Sprint(x))
				}
			}
			for _, id := range ids {
				if m := number.FindString(id); m != "" {
					n, _ := strconv.Atoi(m)
					supersedes = append(supersedes, n)
				}
			}
		}
	}
	if status != "" {
		return status, supersedes
	}
	lines := scan(content)
	for i, l := range lines {
		if l.code {
			continue
		}
		if m := statusLine.FindStringSubmatch(l.text); m != nil {
			return m[1], supersedes
		}
		if l.heading > 0 && strings.EqualFold(strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l.text), "#")), "status") {
			var para []string
			for _, next := range lines[i+1:] {
				if next.heading > 0 || next.code || strings.TrimSpace(next.text) == "" && len(para) > 0 {
					break
				}
				if t := strings.TrimSpace(next.text); t != "" {
					para = append(para, t)
				}
			}
			return strings.Join(para, " "), supersedes
		}
	}
	return "", supersedes
}

// cited is the first record a status names, by link or by number; 0 if none.
func cited(from, text string, byPath map[string]int) int {
	for _, target := range linkTargets(text) {
		if file, _ := resolve(from, target); byPath[file] != 0 {
			return byPath[file]
		}
	}
	for _, re := range []*regexp.Regexp{adrMention, adrToolsID} {
		if m := re.FindStringSubmatch(text); m != nil {
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	return 0
}

// supersededCited reports a doc citing a superseded decision without naming
// what replaced it (coherence's stale_decision_links): by a link to its file,
// or by its number. Decision records themselves are history, and may.
func supersededCited(t Tree) []Problem {
	records := decisions(t)
	byPath := map[string]int{}
	isRecord := map[string]bool{}
	for n, a := range records {
		byPath[a.path] = n
		isRecord[a.path] = true
	}
	var out []Problem
	for p, content := range t.Docs {
		if isRecord[p] {
			continue
		}
		named := map[int]bool{}
		first := map[int]int{} // the line each record is first cited on
		for _, l := range scan(content) {
			if l.code {
				continue
			}
			text := codeSpan.ReplaceAllString(l.text, "")
			var ns []int
			for _, target := range linkTargets(text) {
				if file, _ := resolve(p, target); byPath[file] != 0 {
					ns = append(ns, byPath[file])
				}
			}
			for _, m := range adrMention.FindAllStringSubmatch(text, -1) {
				n, _ := strconv.Atoi(m[1])
				ns = append(ns, n)
			}
			for _, n := range ns {
				named[n] = true
				if first[n] == 0 {
					first[n] = l.n
				}
			}
		}
		var cites []int
		for n := range first {
			if a := records[n]; a != nil && a.superseded && (a.by == 0 || !named[a.by]) {
				cites = append(cites, n)
			}
		}
		sort.Ints(cites)
		for _, n := range cites {
			a := records[n]
			instead := "its status does not say what replaced it: say it there"
			if a.by != 0 {
				instead = "cite what replaced it"
				if b := records[a.by]; b != nil {
					instead += ", " + b.path
				} else {
					instead += fmt.Sprintf(", ADR-%d", a.by)
				}
				instead += ", or name both where the old one still matters"
			}
			out = append(out, Problem{Rule: "cites-superseded", Where: p, Key: "cites-superseded " + p + " " + a.path,
				Message: fmt.Sprintf("line %d cites %s, which is superseded: %s", first[n], a.path, instead)})
		}
	}
	return out
}
