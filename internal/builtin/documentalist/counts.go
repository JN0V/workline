package documentalist

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A line count off (ADR-0014, step 2): a count of lines a doc gives for one
// of its source files, off from the file as it is. No agent counts: the
// engine does, and only lines — a count of tests, fields or anything else is
// never guessed here; a project derives those (`derive`).

// countOff is one count a doc states wrong.
type countOff struct {
	line   int    // the doc's line, 1-based
	named  string // the file as the doc names it
	file   string // the source file it is
	stated string // the count as written, "~283" or "524"
	approx string // the word making it a rough count, as written: "about", "~"
	real   int
}

func (c countOff) message() string {
	return fmt.Sprintf("`%s` has %d lines, not %s (line %d, %s)", c.named, c.real, c.stated, c.line, c.file)
}

var (
	// countIn is a count of lines in prose or a listing: "524 lines",
	// "(~283 lines)", "about 1,008 LOC". The singular ("an 800-line limit")
	// is an adjective, never a count.
	countIn = regexp.MustCompile(`(?i)(~\s*|\b(?:about|approximately|approx\.|around|roughly|nearly)\s+)?\b(\d{1,3}(?:,\d{3})+|\d+)\s+(?:lines|LOC)\b`)
	// fileNamed is a file name as a doc writes it: `Clock.h`, include/Clock.h.
	fileNamed = regexp.MustCompile(`[A-Za-z0-9_][A-Za-z0-9_./-]*\.[A-Za-z][A-Za-z0-9]{0,7}\b`)
	// limitWord makes a number a limit or a bound, not a count.
	limitWord = regexp.MustCompile(`(?i)[<>≤≥]|\b(?:under|over|below|above|limit|limits|max|maximum|min|minimum|target|less than|more than|fewer than|up to|at most|at least|exceeds?|exceeding)\b`)
	// limitAfter, right after a count, makes it a limit: "800 lines max".
	limitAfter = regexp.MustCompile(`(?i)^\W*(?:max|maximum|limit|hard limit|at most|or (?:less|fewer)|per file)\b`)
	// sentenceEnd between a file and a count: they belong to two sentences.
	sentenceEnd = regexp.MustCompile(`[.!?;]\s+[A-Z(]`)
	// countCell is a table cell holding a count alone: "524", "~283", "1,008".
	countCell = regexp.MustCompile(`^(~\s*)?(\d{1,3}(?:,\d{3})+|\d+)$`)
	// linesHeader is the heading of a table's column of line counts.
	linesHeader = regexp.MustCompile(`(?i)^(?:lines|loc|line count|lines of code|# ?lines)$`)
)

// countReach is how far, in characters, a count may stand after the file it
// belongs to on the same line: "`MQTT_impl.h` is the largest file at
// approximately 557 lines" is 45; a listing's comment column, about 60.
const countReach = 80

// isHistory says whether a doc records what was true when written — a
// changelog, a decision record, a tried or research record — where an old
// count or version is the record, not a mistake.
func isHistory(p string) bool {
	base := strings.ToLower(path.Base(p))
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch stem {
	case "changelog", "changes", "history", "news", "release-notes", "releasenotes", "tried":
		return true
	}
	for _, dir := range strings.Split(path.Dir(p), "/") {
		if adrFolders[dir] || dir == "research" {
			return true
		}
	}
	return false
}

// sourceFiles lists the tracked files a doc's sources cover, in this
// repository: a file named, or every file under a folder named.
func sourceFiles(d *Doc, files map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, src := range d.Sources {
		name, p, _ := splitSource(src)
		if name != "" {
			continue
		}
		p = strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/")
		for f := range files {
			if (f == p || p == "." || strings.HasPrefix(f, p+"/")) && !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	sort.Strings(out)
	return out
}

// sourceNamed finds the one source file a name means, by path suffix; "" when
// none or several do.
func sourceNamed(named string, files []string) string {
	named = strings.TrimPrefix(named, "./")
	found := ""
	for _, f := range files {
		if f == named || strings.HasSuffix(f, "/"+named) {
			if found != "" {
				return ""
			}
			found = f
		}
	}
	return found
}

// derivedLines are the doc's lines inside a derived block, by number: the
// code's own facts, regenerated, never a count to check.
func derivedLines(content string) map[int]bool {
	out := map[int]bool{}
	code := codeRanges(content)
	for _, loc := range derived.FindAllStringIndex(content, -1) {
		if inRanges(code, loc[0]) {
			continue
		}
		from := strings.Count(content[:loc[0]], "\n") + 1
		to := strings.Count(content[:loc[1]], "\n") + 1
		for n := from; n <= to; n++ {
			out[n] = true
		}
	}
	return out
}

// countsOff lists the line counts a doc states wrong for its source files,
// counted in repo as it is now. A doc with no sources, a history doc and a
// derived block say nothing checked here.
func countsOff(repo string, files map[string]bool, d *Doc, content string) []countOff {
	if d == nil || isHistory(d.Path) {
		return nil
	}
	sources := sourceFiles(d, files)
	if len(sources) == 0 {
		return nil
	}
	lineCounts := map[string]int{}
	real := func(f string) (int, bool) {
		if n, ok := lineCounts[f]; ok {
			return n, n >= 0
		}
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(f)))
		if err != nil {
			lineCounts[f] = -1
			return 0, false
		}
		lineCounts[f] = lineCount(string(data))
		return lineCounts[f], true
	}
	var out []countOff
	check := func(n int, named, file, approx, number string) {
		got, ok := real(file)
		if !ok {
			return
		}
		stated, err := strconv.Atoi(strings.ReplaceAll(number, ",", ""))
		if err != nil {
			return
		}
		diff := stated - got
		if diff < 0 {
			diff = -diff
		}
		if approx != "" && diff*10 <= got || approx == "" && diff <= 1 {
			return
		}
		s := number
		if approx != "" {
			s = "~" + number
		}
		out = append(out, countOff{line: n, named: named, file: file, stated: s, approx: approx, real: got})
	}
	skip := derivedLines(content)
	lines := scan(content)
	linesCol := -1 // the column of line counts in the table being read
	for i, l := range lines {
		if skip[l.n] {
			continue
		}
		text := l.text
		// A table: its header names the column of counts, each row a file.
		if !l.code && strings.HasPrefix(strings.TrimSpace(text), "|") {
			cells := tableCells(text)
			if i+1 < len(lines) && isTableRule(lines[i+1].text) {
				linesCol = -1
				for c, h := range cells {
					if linesHeader.MatchString(strings.Trim(h, "*` ")) {
						linesCol = c
					}
				}
				continue
			}
			if linesCol >= 0 && linesCol < len(cells) && !isTableRule(text) {
				if m := countCell.FindStringSubmatch(strings.Trim(cells[linesCol], "*` ")); m != nil {
					file, named := "", ""
					for c, cell := range cells {
						if c == linesCol {
							continue
						}
						for _, name := range fileNamed.FindAllString(cell, -1) {
							if f := sourceNamed(name, sources); f != "" && f != file {
								if file != "" {
									file = "-" // two files on the row: whose count is it
								} else {
									file, named = f, name
								}
							}
						}
					}
					if file != "" && file != "-" {
						check(l.n, named, file, strings.TrimSpace(m[1]), m[2])
					}
					continue
				}
			}
		} else {
			linesCol = -1
		}
		// Prose or a listing: a count belongs to the file named last before
		// it on the line, close by, in the same sentence, with no limit word
		// between them or right after.
		names := fileNamed.FindAllStringIndex(text, -1)
		for _, m := range countIn.FindAllStringSubmatchIndex(text, -1) {
			at := -1
			for k, nm := range names {
				if nm[1] <= m[0] {
					at = k
				}
			}
			if at < 0 {
				continue
			}
			named := text[names[at][0]:names[at][1]]
			between := text[names[at][1]:m[0]]
			if len(between) > countReach || sentenceEnd.MatchString(between) || limitWord.MatchString(between) || limitAfter.MatchString(text[m[1]:]) {
				continue
			}
			file := sourceNamed(named, sources)
			if file == "" {
				continue
			}
			approx := ""
			if m[2] >= 0 {
				approx = text[m[2]:m[3]]
			}
			check(l.n, named, file, strings.TrimSpace(approx), text[m[4]:m[5]])
		}
	}
	return out
}

// tableCells splits a table row into its cells, the outer bars dropped.
func tableCells(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	cells := strings.Split(row, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// isTableRule says whether a line is the rule under a table's header.
func isTableRule(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "|") && strings.Trim(s, "|-: ") == "" && strings.Contains(s, "-")
}

// CountsOff reports, for every doc declaring its sources, each line count it
// states wrong for one of them (ADR-0014, step 2). While one stands, the
// doc's `checked` cannot move.
func CountsOff(repo string, t Tree, docs []*Doc) []Problem {
	var out []Problem
	for _, d := range docs {
		for _, c := range countsOff(repo, t.Files, d, t.Docs[d.Path]) {
			out = append(out, Problem{Rule: "count-off", Where: d.Path,
				Key:     fmt.Sprintf("count-off %s %s %s", d.Path, c.file, c.stated),
				Message: c.message()})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Where < out[j].Where })
	return out
}
