package reviewer

// The judge reads the code it needs (ADR-0045): pre serves the reads a
// judge of the code asks — a file's lines, a definition, a word's places —
// from git at the head reviewed, within judge-read-rounds and
// judge-read-lines-max, and the engine asks it again. A no's quote found
// again in what it was shown or read drops the finding; otherwise, or
// unsure, the finding is shown not verified (roles/reviewer/docs/judge.md).

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/judge"
	"go.yaml.in/yaml/v3"
)

// grepMax is the most places a search is served: past it, none, and the
// judge is asked to narrow it (docs/research/judge-reads-code.md).
const grepMax = 50

// reading is a judge's question that may read, and what it read so far,
// kept as in/judge/<key>/reads.json.
type reading struct {
	Question string      `json:"question"`
	Material string      `json:"material"` // as first asked, before any read
	Cause    string      `json:"cause"`    // the cause's file: a name is looked for there first, then in its folder
	Rounds   []readRound `json:"rounds,omitempty"`
}

// readRound is one round of reads: what the judge asked, and what it was
// given back.
type readRound struct {
	Asked   []judge.Read `json:"asked"`
	Note    string       `json:"note,omitempty"`    // what it said beside its reads
	Served  string       `json:"served"`            // the text it was given back
	Lines   int          `json:"lines"`             // lines served
	Shown   []seen       `json:"shown,omitempty"`   // where they lie
	Refused []string     `json:"refused,omitempty"` // reads not served, each with why
}

// seen is lines of a file a judge was shown or read.
type seen struct {
	Path string `json:"path"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// readAnswer is a judge's answer as the engine wrote it (answer.yaml).
type readAnswer struct {
	Yes                      *bool
	Unsure                   bool
	Why, Model, Level, Error string
	Judge, Author            string
	Reads                    []judge.Read
	Quote                    *judge.Quote
}

func readingOf(dir string) (reading, bool) {
	var r reading
	return r, readJSON(filepath.Join(dir, "reads.json"), &r) == nil
}

// readRoundsMax bounds judge-read-rounds: the engine asks a judge at most
// ten times a run.
const readRoundsMax = 8

// left is what the judge may still read: rounds, and lines.
func (r reading) left(s Settings) (rounds, lines int) {
	used := 0
	for _, rd := range r.Rounds {
		used += rd.Lines
	}
	return min(s.JudgeReadRounds, readRoundsMax) - len(r.Rounds), s.JudgeReadMax - used
}

// said is what the judge read, said beside its answer.
func (r reading) said() string {
	var asked []string
	for _, rd := range r.Rounds {
		for _, a := range rd.Asked {
			asked = append(asked, a.String())
		}
	}
	if len(asked) == 0 {
		return ""
	}
	return ", having read " + strings.Join(asked, ", ")
}

// readsSaid is each read, as said.
func readsSaid(reads []judge.Read) []string {
	out := make([]string, len(reads))
	for i, r := range reads {
		out[i] = r.String()
	}
	return out
}

// write keeps the reading and asks its question: the question, the
// material first asked, what the judge read each round, and what it may
// still read.
func (r reading) write(dir string, s Settings, head string) error {
	if err := writeJSON(filepath.Join(dir, "reads.json"), r); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(r.Material)
	for i, rd := range r.Rounds {
		fmt.Fprintf(&b, "\n## What you read, round %d\n\nAs the repository reads at %s.\n%s", i+1, short(head), rd.Served)
	}
	// How to answer a question that lets the judge read is said here, in
	// its question alone: the judge's facets, which every judge shares,
	// ask one note.
	rounds, lines := r.left(s)
	if rounds > 0 && lines > 0 {
		fmt.Fprintf(&b, "\n## Reading more\n\nThis question lets you read before you answer, in place of the one note asked otherwise. You may ask to read the repository as it reads at %s: %d more %s of reads, %d more lines. To read, answer with reads instead of a note, each saying why: a file's lines (`path`, `lines`), where a name is defined (`name`), or the lines naming a word (`grep`, at most %d places). You are then asked again, with what you read.\n", short(head), rounds, plural(rounds, "round", "rounds"), lines, grepMax)
	} else {
		b.WriteString("\n## No read is left\n\nYou read what you could (judge-read-rounds, judge-read-lines-max): answer now.\n")
	}
	b.WriteString("\nYour answer: yes; no, with the lines that refute the finding quoted beside the note, as they read, with their path; or unsure, when you still cannot tell. A no whose quote is not found again in what you were shown or read leaves the finding not verified:\n\n" +
		"- note: \"no: the loop adds every value before the division.\"\n- quote: {path: \"calc/calc.go\", text: \"for _, v := range values {\"}\n\n" +
		"- note: \"unsure: whether a caller passes an empty list is in no code read.\"\n")
	data, _ := json.Marshal(map[string]any{"question": r.Question, "material": b.String(), "reads": true}) // JSON, which YAML reads
	return os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644)
}

// readsAsked serves the reads the judges asked, each within what it may
// still read, and asks each of them again; true when one is. A judge
// asking past its rounds or lines is not served: settle takes it as
// unsure.
func readsAsked(runDir, repo string, s Settings) (bool, error) {
	var st state
	if err := readJSON(filepath.Join(runDir, "in", "review-state.json"), &st); err != nil {
		return false, err
	}
	files, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", "*", "reads.json"))
	again := false
	for _, f := range files {
		dir := filepath.Dir(f)
		r, ok := readingOf(dir)
		a, err := answerIn(dir)
		if !ok || err != nil || a.Error != "" || len(a.Reads) == 0 {
			continue
		}
		rounds, lines := r.left(s)
		if rounds <= 0 || lines <= 0 {
			continue
		}
		rd := serve(repo, st.Head, r.Cause, a.Reads, lines, s.JudgeReadMax)
		rd.Note = a.Why
		r.Rounds = append(r.Rounds, rd)
		// The answer that asked is kept beside, by its round.
		if err := os.Rename(filepath.Join(dir, "answer.yaml"), filepath.Join(dir, fmt.Sprintf("answer-%d.yaml", len(r.Rounds)))); err != nil {
			return false, err
		}
		if err := r.write(dir, s, st.Head); err != nil {
			return false, err
		}
		again = true
	}
	return again, nil
}

func answerIn(dir string) (readAnswer, error) {
	var a readAnswer
	data, err := os.ReadFile(filepath.Join(dir, "answer.yaml"))
	if err == nil {
		err = yaml.Unmarshal(data, &a)
	}
	return a, err
}

// serve reads what a judge asked at head, from git's objects, at most
// left lines all together (of linesMax, judge-read-lines-max): each read
// whole while it fits, cut past it, the rest named; a read refused is
// said, with why.
func serve(repo, head, cause string, reads []judge.Read, left, linesMax int) readRound {
	rd := readRound{Asked: reads}
	var b strings.Builder
	refuse := func(what, why string) {
		rd.Refused = append(rd.Refused, what+": "+why)
		fmt.Fprintf(&b, "\nNot read: %s: %s.\n", what, why)
	}
	// show writes lines from..to of a file, as far as lines are left.
	show := func(p string, lines []string, from, to int, title string) {
		if left <= 0 {
			refuse(fmt.Sprintf("%s:%d-%d", p, from, to), fmt.Sprintf("past judge-read-lines-max (%d)", linesMax))
			return
		}
		end := min(to, from+left-1)
		fmt.Fprintf(&b, "\n### %s, lines %d to %d: %s\n\n```\n", p, from, end, title)
		for n := from; n <= end; n++ {
			fmt.Fprintf(&b, "%d  %s\n", n, lines[n-1])
		}
		b.WriteString("```\n")
		rd.Shown = append(rd.Shown, seen{p, from, end})
		rd.Lines += end - from + 1
		left -= end - from + 1
		if end < to {
			fmt.Fprintf(&b, "\n%s: lines %d to %d not read: past judge-read-lines-max (%d).\n", p, end+1, to, linesMax)
		}
	}
	rc := &reacher{repo: repo, head: head, files: map[string][]string{}}
	for _, r := range reads {
		switch {
		case r.Path != "":
			p, why := inRepo(r.Path)
			if why != "" {
				refuse(r.Path, why)
				continue
			}
			if kind, err := git(repo, "cat-file", "-t", head+":"+p); err != nil || strings.TrimSpace(kind) != "blob" {
				refuse(p, "not a file the repository tracks at "+short(head))
				continue
			}
			lines := rc.lines(p)
			from, to, ok := lineRange(r.Lines, len(lines))
			switch {
			case !ok:
				refuse(r.String(), "its lines do not read: give them as 40-80")
			case from > len(lines):
				refuse(r.String(), fmt.Sprintf("past its end (%d lines)", len(lines)))
			default:
				show(p, lines, from, to, "read")
			}
		case r.Name != "":
			name := strings.TrimSuffix(strings.Trim(r.Name, "` "), "()")
			name = name[strings.LastIndex(name, ".")+1:] // a method, Type.Name
			defs, many := pick(rc.defs([]string{name}, nil)[name], cause)
			switch {
			case many:
				refuse(r.String(), "defined in too many places to tell which: read it by its path and lines")
			case len(defs) == 0:
				refuse(r.String(), "no definition found at "+short(head))
			}
			for _, d := range defs {
				show(d.file, rc.lines(d.file), d.from, d.to, "`"+d.name+"`, read")
			}
		case r.Grep != "":
			places, why := grepWord(repo, head, r.Grep)
			if why != "" {
				refuse(r.String(), why)
				continue
			}
			if left <= 0 {
				refuse(r.String(), fmt.Sprintf("past judge-read-lines-max (%d)", linesMax))
				continue
			}
			var served []string
			for _, h := range places[:min(len(places), left)] {
				served = append(served, fmt.Sprintf("%s:%d: %s", h.Path, h.From, h.text))
				rd.Shown = append(rd.Shown, h.seen)
			}
			rd.Lines += len(served)
			left -= len(served)
			fmt.Fprintf(&b, "\n### grep `%s`: %d %s\n\n```\n%s\n```\n", r.Grep, len(places), plural(len(places), "place", "places"), strings.Join(served, "\n"))
			if n := len(places) - len(served); n > 0 {
				fmt.Fprintf(&b, "\n%d more %s not read: past judge-read-lines-max (%d).\n", n, plural(n, "place", "places"), linesMax)
			}
		default:
			refuse(r.String(), "a read names a path, a name or a word to grep")
		}
	}
	rd.Served = b.String()
	return rd
}

// inRepo is a path a judge asked, as the repository names it; why, when
// it lies outside.
func inRepo(p string) (string, string) {
	p = strings.TrimPrefix(strings.TrimSpace(filepath.ToSlash(p)), "./")
	c := path.Clean(p)
	if strings.HasPrefix(p, "/") || c == ".." || strings.HasPrefix(c, "../") {
		return "", "outside the repository"
	}
	return c, ""
}

// lineRange reads "40-80", "40", or none (the whole file), within n lines.
func lineRange(s string, n int) (from, to int, ok bool) {
	if s == "" {
		return 1, max(n, 1), true
	}
	a, b, isRange := strings.Cut(s, "-")
	from, err := strconv.Atoi(a)
	if err != nil || from < 1 {
		return 0, 0, false
	}
	to = from
	if isRange {
		if to, err = strconv.Atoi(b); err != nil || to < from {
			return 0, 0, false
		}
	}
	return from, min(to, n), true
}

// grepHit is a line naming a word.
type grepHit struct {
	seen
	text string
}

// grepWord is every line naming a word, as a word, at head; why, when
// none, or past grepMax.
func grepWord(repo, head, word string) ([]grepHit, string) {
	if strings.ContainsAny(word, "\n\r") || strings.TrimSpace(word) == "" {
		return nil, "a word to grep is one line"
	}
	out, err := git(repo, "grep", "-n", "-I", "-w", "-F", "-e", word, head, "--")
	if err != nil {
		return nil, "written nowhere at " + short(head) // git grep fails when it finds none
	}
	var hits []grepHit
	for _, h := range strings.Split(strings.TrimSpace(out), "\n") {
		if file, n, text, ok := hit(h, head); ok {
			hits = append(hits, grepHit{seen{file, n, n}, strings.TrimSpace(text)})
		}
	}
	if len(hits) > grepMax {
		return nil, fmt.Sprintf("written in %d places, past %d: narrow it, by a path's lines or a name", len(hits), grepMax)
	}
	return hits, ""
}

// quoted finds a no's quote again: at head, in the file it names (or the
// issue the change closes), within what the judge was shown or read.
// where: its place; why: why it is not found.
func (r reading) quoted(repo string, st state, q *judge.Quote) (where, why string) {
	if q == nil || norm(q.Text) == "" {
		return "", "it quotes no line"
	}
	p := strings.TrimPrefix(strings.TrimSpace(q.Path), "./")
	if strings.HasPrefix(p, "#") {
		is := issueOf(st, p)
		if is == nil {
			return "", fmt.Sprintf("its quote names %s, not an issue the change closes", p)
		}
		if len(locate(strings.Split(is.Text, "\n"), q.Text)) == 0 {
			return "", fmt.Sprintf("its quote is not found in %s", p)
		}
		return p, ""
	}
	text, ok := fileAt(repo, st.Head, p)
	if !ok {
		return "", fmt.Sprintf("its quote names %s, not a file at %s", p, short(st.Head))
	}
	places := locate(strings.Split(text, "\n"), q.Text)
	if len(places) == 0 {
		return "", fmt.Sprintf("its quote is not found in %s at %s", p, short(st.Head))
	}
	spans := shownIn(r.Material)
	for _, rd := range r.Rounds {
		spans = append(spans, rd.Shown...)
	}
	for _, pl := range places {
		if covered(spans, p, pl) {
			return fmt.Sprintf("%s:%d", p, pl.From), ""
		}
	}
	return "", fmt.Sprintf("its quote, at %s:%d, is not in what it was shown or read", p, places[0].From)
}

var (
	shownLines = regexp.MustCompile(`^### (.+?), lines (\d+) to (\d+)(?::|$)`)
	shownWhole = regexp.MustCompile(`^### (\S+)$`)
	diffFile   = regexp.MustCompile(`^\+\+\+ b/(.+)$`)
	diffHunk   = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)
)

// shownIn is where the code a judge was first shown lies: the functions
// and lines the material gives (### path, lines a to b), the test files
// given whole (### path), and the hunks of the change it holds.
func shownIn(material string) []seen {
	var spans []seen
	file := ""
	for _, l := range strings.Split(material, "\n") {
		switch {
		case shownLines.MatchString(l):
			m := shownLines.FindStringSubmatch(l)
			from, _ := strconv.Atoi(m[2])
			to, _ := strconv.Atoi(m[3])
			spans = append(spans, seen{m[1], from, to})
		case shownWhole.MatchString(l):
			spans = append(spans, seen{shownWhole.FindStringSubmatch(l)[1], 1, math.MaxInt})
		case diffFile.MatchString(l):
			file = diffFile.FindStringSubmatch(l)[1]
		case file != "" && diffHunk.MatchString(l):
			m := diffHunk.FindStringSubmatch(l)
			from, _ := strconv.Atoi(m[1])
			n := 1
			if m[2] != "" {
				n, _ = strconv.Atoi(m[2])
			}
			spans = append(spans, seen{file, from, from + max(n, 1) - 1})
		}
	}
	return spans
}

// covered says whether every line of a place lies in a lines of its file.
func covered(spans []seen, p string, pl Place) bool {
	for n := pl.From; n <= pl.To; n++ {
		in := false
		for _, s := range spans {
			if s.Path == p && n >= s.From && n <= s.To {
				in = true
				break
			}
		}
		if !in {
			return false
		}
	}
	return true
}
