package inspector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// stickyKey keys the inspector's one comment on a merge request, edited
// on each run, the record hidden in it.
const stickyKey = "triage"

// marker is the comment's, as the engine writes it.
var marker = forge.Marker("sticky=inspector/" + stickyKey)

// Triaged is one finding the agent triaged, kept by its fingerprint, which
// holds no line: while it only moves, it is not asked again (ADR-0013).
// At, the head it was triaged at; Model, who triaged it: what the weekly
// sample draws (ADR-0046).
type Triaged struct {
	Fingerprint   string `json:"fingerprint"`
	Gate          string `json:"gate,omitempty"`
	Check         string `json:"check"`
	Tool          string `json:"tool,omitempty"`
	Rule          string `json:"rule,omitempty"`
	Where         string `json:"where"`
	At            string `json:"at"`
	Model         string `json:"model,omitempty"`
	Matters       string `json:"matters"`
	Fix           string `json:"fix,omitempty"`
	FalsePositive bool   `json:"false-positive,omitempty"`
	Why           string `json:"why,omitempty"`
	Order         int    `json:"order,omitempty"`
}

func (t Triaged) ruleName() string {
	n := t.Rule
	if n == "" {
		n = t.Check
	}
	if t.Tool != "" && t.Tool != n {
		return fmt.Sprintf("%s (%s)", n, t.Tool)
	}
	return n
}

// Record is every finding triaged on the merge request, the oldest first:
// hidden in the comment, a line each; on a machine, in the git directory.
type Record struct {
	Triaged []Triaged `json:"triaged,omitempty"`
}

func (r Record) find(fingerprint string) (Triaged, bool) {
	for _, t := range r.Triaged {
		if t.Fingerprint == fingerprint {
			return t, true
		}
	}
	return Triaged{}, false
}

// with is the record holding t, in place of one of its fingerprint.
func (r Record) with(t Triaged) Record {
	out := Record{}
	for _, o := range r.Triaged {
		if o.Fingerprint != t.Fingerprint {
			out.Triaged = append(out.Triaged, o)
		}
	}
	out.Triaged = append(out.Triaged, t)
	return out
}

// ordered sets each finding's place at the head: the agent's, given;
// else the place it had, after those given; else the gates' order.
func (r Record) ordered(found []Found, given map[string]int) Record {
	type place struct {
		given, had, at int
		fp             string
	}
	var places []place
	for i, f := range found {
		t, ok := r.find(f.Fingerprint)
		if !ok {
			continue
		}
		p := place{given: given[f.Fingerprint], had: t.Order, at: i, fp: f.Fingerprint}
		places = append(places, p)
	}
	key := func(p place) (int, int, int) {
		if p.given > 0 {
			return 0, p.given, p.at
		}
		if p.had > 0 {
			return 1, p.had, p.at
		}
		return 2, 0, p.at
	}
	less := func(a, b place) bool {
		a1, a2, a3 := key(a)
		b1, b2, b3 := key(b)
		if a1 != b1 {
			return a1 < b1
		}
		if a2 != b2 {
			return a2 < b2
		}
		return a3 < b3
	}
	for i := 1; i < len(places); i++ {
		for j := i; j > 0 && less(places[j], places[j-1]); j-- {
			places[j], places[j-1] = places[j-1], places[j]
		}
	}
	rank := map[string]int{}
	for i, p := range places {
		rank[p.fp] = i + 1
	}
	out := Record{}
	for _, t := range r.Triaged {
		if n, ok := rank[t.Fingerprint]; ok {
			t.Order = n
		}
		out.Triaged = append(out.Triaged, t)
	}
	return out
}

var triagedLine = regexp.MustCompile(`<!-- workline:triaged (.*?) -->`)

// recordOpening opens each line of the record: a text holding one that
// does not read holds a record damaged, never none.
const recordOpening = "<!-- workline:triaged"

// readRecord reads the record a text holds; none when it holds no line.
func readRecord(text string) (Record, error) {
	var r Record
	lines := triagedLine.FindAllStringSubmatch(text, -1)
	for _, m := range lines {
		var t Triaged
		if err := json.Unmarshal([]byte(m[1]), &t); err != nil || t.Fingerprint == "" {
			return Record{}, fmt.Errorf("a line of the record does not read: %.80s", m[0])
		}
		r = r.with(t)
	}
	if len(lines) != strings.Count(text, recordOpening) {
		return Record{}, errors.New("a line of the record is cut")
	}
	return r, nil
}

// lines is the record as the comment hides it: a line each, the oldest
// gone first past room, those of the findings at the head kept; past it
// still, the oldest of those too, counted: they are asked again.
func (r Record) lines(keep map[string]bool, room int) (string, int) {
	all := make([]string, len(r.Triaged))
	size := 0
	for i, t := range r.Triaged {
		all[i] = "\n<!-- workline:triaged " + hidden(t) + " -->"
		size += len(all[i])
	}
	gone, left := map[int]bool{}, 0
	for _, theirs := range []bool{false, true} {
		for i := 0; i < len(all) && size > room; i++ {
			if !gone[i] && keep[r.Triaged[i].Fingerprint] == theirs {
				gone[i] = true
				size -= len(all[i])
				if theirs {
					left++
				}
			}
		}
	}
	var b strings.Builder
	for i, l := range all {
		if !gone[i] {
			b.WriteString(l)
		}
	}
	return b.String(), left
}

// hidden is a value as JSON on one line, safe in a comment hidden in
// Markdown: a dash written as its escape, and `<`, `>` escaped by the
// JSON library, so nothing the agent wrote ends the comment hiding it.
func hidden(v any) string {
	data, _ := json.Marshal(v)
	return strings.ReplaceAll(string(data), "-", `\u002d`)
}

// commentMax is the most characters the comment holds: GitHub takes
// 65,536. shownMax, the most triages it lists; the run's summary holds
// them all.
const (
	commentMax = 60000
	shownMax   = 25
)

// comment is the inspector's one comment on the merge request: the
// triage of the findings at the head, in the order to fix them, and the
// record hidden below.
func comment(run Run, shown []Triaged) string {
	var b strings.Builder
	n := len(run.Found)
	fps := 0
	for _, t := range shown {
		if t.FalsePositive {
			fps++
		}
	}
	switch {
	case n == 0:
		fmt.Fprintf(&b, "**workline inspector** — no new finding of the gates at %s.\n\n", span(short(run.Head)))
	default:
		fmt.Fprintf(&b, "**workline inspector** — the gates' %d new %s at %s: %d triaged, %d likely %s.\n\n", n, plural(n, "finding", "findings"),
			span(short(run.Head)), len(shown), fps, plural(fps, "a false positive", "false positives"))
	}
	b.WriteString("It reads apart from the verdict and never changes it: a finding that blocks still blocks. The gates show each finding beside its line; here, why it matters and the order to fix them.\n\n")
	for i, t := range shown {
		if i == shownMax {
			fmt.Fprintf(&b, "\n%d more triaged, not listed here: the run's summary lists them all.\n", len(shown)-shownMax)
			break
		}
		fmt.Fprintf(&b, "%d. **%s** at %s: ", i+1, span(t.ruleName()), span(t.Where))
		if t.FalsePositive {
			b.WriteString("**likely a false positive**: " + sentence(inert(t.Why)) + " ")
		}
		b.WriteString(sentence(inert(t.Matters)))
		if t.Fix != "" {
			b.WriteString(" Fix: " + sentence(inert(t.Fix)))
		}
		b.WriteString("\n")
	}
	if len(shown) > 0 {
		b.WriteString("\n")
	}
	for _, s := range run.Said {
		b.WriteString("- " + s + "\n")
	}
	keep := map[string]bool{}
	for _, f := range run.Found {
		keep[f.Fingerprint] = true
	}
	const note = "- %d triages not kept in this comment, past its size: asked again at the next push.\n"
	body := strings.TrimRight(b.String(), "\n") + "\n"
	record, left := run.Record.lines(keep, commentMax-len(body)-len(note)-10)
	if left > 0 {
		body += fmt.Sprintf(note, left)
	}
	return body + record
}

// task is the one part put to the agent: the findings to triage, each
// with the head's code around it, and those triaged before, for their
// place in the order only.
func task(repo string, s Settings, run Run, asked []Found) string {
	var b strings.Builder
	b.WriteString("# The gates' new findings to triage\n\n")
	fmt.Fprintf(&b, "The project's own tools, its gates, report these findings on the change, each new since the merge base. "+
		"Their verdict decides whether the change may merge: you never change it. At the head, %s.\n\n", short(run.Head))
	b.WriteString("For each finding to triage, answer one `triage`: why it matters in this project, in its own words, from the code shown; what to change; whether it looks like a false positive, and why; and its place in the order to fix every finding listed here, 1 the first.\n\n")
	for _, f := range asked {
		fmt.Fprintf(&b, "## %s — `%s`, %s", f.ID, oneLine(f.name()), f.Where())
		if f.Level != "" {
			fmt.Fprintf(&b, " (%s)", f.Level)
		}
		b.WriteString("\n\n")
		if f.Message != "" {
			fmt.Fprintf(&b, "%s\n\n", oneLine(f.Message))
		}
		b.WriteString(around(repo, run.Head, f, s.LinesAround))
		b.WriteString("\n")
	}
	var before []Found
	for _, f := range run.Found {
		if strings.HasPrefix(f.ID, "r") {
			before = append(before, f)
		}
	}
	if len(before) > 0 {
		b.WriteString("## Triaged before, still new at the head\n\nGive each its place in the order only: `{finding: r1, order: 3}`.\n\n")
		for _, f := range before {
			t, _ := run.Record.find(f.Fingerprint)
			fp := ""
			if t.FalsePositive {
				fp = " (likely a false positive)"
			}
			fmt.Fprintf(&b, "- %s — `%s`, %s: %s%s\n", f.ID, oneLine(f.name()), f.Where(), t.Matters, fp)
		}
	}
	return b.String()
}

// around is the head's code around a finding's line, numbered, the line
// marked; a file the head does not hold is said.
func around(repo, head string, f Found, n int) string {
	if f.Path == "" || f.Line <= 0 {
		return "(no line named)\n"
	}
	out, err := git(repo, "show", head+":"+f.Path)
	if err != nil {
		return fmt.Sprintf("(%s is not at the head)\n", f.Path)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	from, to := max(f.Line-n, 1), min(f.Line+n, len(lines))
	if f.Line > len(lines) {
		return fmt.Sprintf("(%s holds %d lines at the head)\n", f.Path, len(lines))
	}
	var code strings.Builder
	for i := from; i <= to; i++ {
		mark := " "
		if i == f.Line {
			mark = ">"
		}
		fmt.Fprintf(&code, "%s%5d  %s\n", mark, i, lines[i-1])
	}
	fence := "```"
	for strings.Contains(code.String(), fence) {
		fence += "`"
	}
	return fence + "\n" + code.String() + fence + "\n"
}

// loadRecord reads the record: from the merge request's comment when the
// run is on one and may write there, else from the machine's. It says
// where it read it, whether it is the machine's, and why it could not.
// Only a comment a bot or a person of the project wrote is read: on a
// public merge request anyone may comment, and the record is shown again
// under workline's name.
func loadRecord(repo string, s Settings) (Record, string, bool, error) {
	if t := mergeRequest(); t != nil && s.ForgeWrites {
		where := fmt.Sprintf("in its comment on merge request %d", t.ID)
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil {
			return Record{}, where, false, fmt.Errorf("the forge: %v", err)
		}
		if f == nil {
			return Record{}, where, false, errors.New("no forge to read the merge request's comments from")
		}
		b, ok := f.(forge.Backlog)
		if !ok {
			return Record{}, where, false, errors.New("this forge cannot list a merge request's comments")
		}
		notes, err := b.Notes(*t)
		if err != nil && !errors.Is(err, forge.ErrAuthorsUnknown) {
			return Record{}, where, false, err
		}
		if text, ok := recordComment(notes); ok {
			r, err := readRecord(text)
			return r, where, false, err
		}
		return Record{}, where, false, nil
	}
	file := localRecord(repo)
	where := "in " + file
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return Record{}, where, true, nil
	}
	if err != nil {
		return Record{}, where, true, err
	}
	r, err := readRecord(string(data))
	return r, where, true, err
}

// recordComment is the comment of a merge request's the record is read
// from: the last carrying the marker, the one the engine edits, a bot's
// or a person's of the project.
func recordComment(notes []forge.Note) (string, bool) {
	for i := len(notes) - 1; i >= 0; i-- {
		if n := notes[i]; strings.Contains(n.Body, marker) && (n.Bot || n.Insider) {
			return n.Body, true
		}
	}
	return "", false
}

// MergeComment is body, the comment a run writes, its record merged with
// the one the comment holds now, among notes, when a run applied since
// this one read read (#580): what that run triaged is kept, this run's
// triage of a finding both triaged, the others' gone first past the
// comment's size. No comment, or a record that does not read, leaves
// body as it is.
func MergeComment(body, read string, notes []forge.Note) string {
	found, ok := recordComment(notes)
	if !ok {
		return body
	}
	ours, err1 := readRecord(body)
	now, err2 := readRecord(found)
	was, err3 := readRecord(read)
	if err1 != nil || err2 != nil || err3 != nil {
		return body
	}
	var since []Triaged // triaged by a run applied since this one read
	for _, t := range now.Triaged {
		if w, ok := was.find(t.Fingerprint); ok && w == t {
			continue
		}
		if _, ok := ours.find(t.Fingerprint); !ok {
			since = append(since, t)
		}
	}
	if len(since) == 0 {
		return body
	}
	head := body
	if at := strings.Index(body, recordOpening); at >= 0 {
		head = strings.TrimSuffix(body[:at], "\n")
	}
	keep := map[string]bool{}
	for _, t := range ours.Triaged {
		keep[t.Fingerprint] = true
	}
	// This run's own lines fit this room already, its note counting those
	// it left out: past it, only triages of the run applied since go, the
	// oldest first and uncounted, as comment drops those of findings not
	// at the head; none of ours goes, so the count is 0.
	merged := Record{Triaged: append(since, ours.Triaged...)}
	record, _ := merged.lines(keep, commentMax-len(head)-10)
	return head + record
}

// localRecord is the record's file on this machine, in the git directory.
func localRecord(repo string) string {
	dir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return filepath.Join(repo, ".git", "workline", "inspector-record")
	}
	return filepath.Join(strings.TrimSpace(dir), "workline", "inspector-record")
}

// saveLocal keeps the record on this machine.
func saveLocal(repo string, r Record) error {
	file := localRecord(repo)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	text, _ := r.lines(nil, 1<<30)
	return os.WriteFile(file, []byte(strings.TrimPrefix(text, "\n")+"\n"), 0o644)
}

// mergeRequest is the merge request the run is on, when a forge and a
// target are given and the target is one.
func mergeRequest() *forge.Target {
	if os.Getenv("WORKLINE_FORGE") == "" {
		return nil
	}
	kind, id, ok := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	n, err := strconv.Atoi(id)
	if !ok || err != nil || kind != "merge-request" {
		return nil
	}
	return &forge.Target{Kind: kind, ID: n}
}

// span is text in a code span, its backticks taken out.
func span(s string) string { return "`" + strings.ReplaceAll(oneLine(s), "`", "'") + "`" }

// inert is the agent's text as the comment shows it: one line, its
// mentions and HTML defused.
func inert(s string) string {
	s = strings.ReplaceAll(oneLine(s), "@", "@\u200b")
	return strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
}

// sentence is a text ending as a sentence does.
func sentence(s string) string {
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "…") {
		return s
	}
	return s + "."
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
