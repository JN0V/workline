package documentalist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// Code examples a fix writes (#121). The removal rule reads the words a
// fix takes out; whether a fenced block it adds or changes compiles, or
// runs right, no word says: an example rewritten on DomoticsCore compiled
// and was wrong at run time, merged as one of "8 places" fixed. So a block
// a fix writes is told apart from the places it fixes: said unverified, for
// the person merging to read; or, where the project names the command
// checking its examples (`examples.check`, as `derive` names its commands),
// run on a checkout with the fixed docs written in: a failure withholds the
// example, its output shown; a pass says the example was checked by it. No
// model reads an example.

// fence is a fenced code block of a doc: its lines, fences included, by the
// doc's own numbering; its language, the info string's first word; its code.
type fence struct {
	from, to int
	lang     string
	code     string
}

// fences lists a doc's fenced code blocks, its frontmatter aside, as scan
// reads them; one left open runs to the end.
func fences(content string) []fence {
	var out []fence
	marker := ""
	var cur fence
	var code []string
	for _, l := range scan(content) {
		t := strings.TrimSpace(l.text)
		switch {
		case marker != "":
			cur.to = l.n
			if strings.HasPrefix(t, marker) {
				cur.code, marker = strings.Join(code, "\n"), ""
				out = append(out, cur)
				continue
			}
			code = append(code, l.text)
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			marker, code = t[:3], nil
			cur = fence{from: l.n, to: l.n}
			if f := strings.Fields(strings.TrimLeft(t, "`~")); len(f) > 0 {
				cur.lang = strings.Trim(f[0], "{}.")
			}
		}
	}
	if marker != "" {
		cur.code = strings.Join(code, "\n")
		out = append(out, cur)
	}
	return out
}

// examplesWritten are the fenced blocks of a doc as a fix leaves it (now)
// that the fix adds or changes: each one whose code no block of the doc as
// it was (old) holds. A block whose only change brings line counts the
// engine found off to its numbers is none: the engine's count checked it
// (a fenced listing's "Clock.h (569 lines)").
func examplesWritten(repo string, files map[string]bool, path, old, now string) []fence {
	was := fences(old)
	left := map[string]int{}
	for _, f := range was {
		left[f.code]++
	}
	var changed []fence
	for _, f := range fences(now) {
		if left[f.code] > 0 {
			left[f.code]--
			continue
		}
		changed = append(changed, f)
	}
	if len(changed) == 0 {
		return nil
	}
	var gone []fence // the blocks of old no block of now holds
	for _, f := range was {
		if left[f.code] > 0 {
			left[f.code]--
			gone = append(gone, f)
		}
	}
	counted := map[int][]countOff{}
	if d, _ := ParseDoc(path, []byte(old)); d != nil && len(gone) > 0 {
		for _, c := range countsOff(repo, files, d, old) {
			counted[c.line] = append(counted[c.line], c)
		}
	}
	var out []fence
	for _, f := range changed {
		if !countsBrought(f, gone, counted) {
			out = append(out, f)
		}
	}
	return out
}

// countsBrought says whether a block is one of the blocks gone with only
// line counts the engine found off brought to its numbers.
func countsBrought(f fence, gone []fence, counted map[int][]countOff) bool {
	now := strings.Split(f.code, "\n")
	for _, g := range gone {
		was := strings.Split(g.code, "\n")
		if len(was) != len(now) {
			continue
		}
		same := true
		for i := range was {
			if was[i] != now[i] && !countBroughtOn(was[i], now[i], counted[g.from+1+i]) {
				same = false
				break
			}
		}
		if same {
			return true
		}
	}
	return false
}

// countBroughtOn says whether a line differs from what it was only by line
// counts the engine found off on it, each brought to its number.
func countBroughtOn(was, now string, cs []countOff) bool {
	if len(cs) == 0 {
		return false
	}
	bare := func(s string) string {
		return strings.Join(strings.Fields(strings.ReplaceAll(numberIn.ReplaceAllString(plain(s), "#"), "~", "")), " ")
	}
	if bare(was) != bare(now) {
		return false
	}
	digits := func(s string) []string {
		var out []string
		for _, n := range numberIn.FindAllString(plain(s), -1) {
			out = append(out, strings.ReplaceAll(n, ",", ""))
		}
		return out
	}
	a, b := digits(was), digits(now)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		brought := false
		for _, c := range cs {
			stated := strings.ReplaceAll(strings.TrimPrefix(c.stated, "~"), ",", "")
			brought = brought || stated == a[i] && strconv.Itoa(c.real) == b[i]
		}
		if !brought {
			return false
		}
	}
	return true
}

// exampleRuns are the runs of a doc's changed lines that write one of its
// examples: a line they add in one, or a line they take out of one.
func exampleRuns(old string, f fileDiff, written []fence) []citedBlock {
	at := newLineOf(old, f)
	var out []citedBlock
	for _, b := range bodyBlocks(old, f) {
		for _, li := range b.at {
			if inFences(written, at[[2]int{b.hunk, li}]) {
				out = append(out, b)
				break
			}
		}
	}
	return out
}

// newLineOf numbers, by hunk and line, where each changed line of a diff
// stands in the doc it makes: an added line, its own; a removed one, the
// line after the place it leaves.
func newLineOf(old string, f fileDiff) map[[2]int]int {
	out := map[[2]int]int{}
	n, next := 0, 0 // lines of the new doc written, of the old one read
	for hi, h := range f.hunks {
		start := h.oldStart - 1
		if h.oldCount == 0 {
			start = h.oldStart // inserted after that line
		}
		n += start - next
		next = start
		for li, l := range h.lines {
			switch l[0] {
			case ' ':
				n++
				next++
			case '-':
				out[[2]int{hi, li}] = n + 1
				next++
			case '+':
				n++
				out[[2]int{hi, li}] = n
			}
		}
	}
	return out
}

// inFences says whether a line of a doc is in one of these blocks.
func inFences(fs []fence, line int) bool {
	for _, f := range fs {
		if line >= f.from && line <= f.to {
			return true
		}
	}
	return false
}

// exampleRun is what the project's command said of a doc's examples.
type exampleRun struct {
	ok  bool
	out string
}

// checkExamples runs the project's command checking examples on a checkout
// of HEAD with the fixed docs written in it, the docs named its arguments
// ("$@"): whether it passed, and, when not, what it said.
func checkExamples(repo, command string, docs map[string]string, named []string) exampleRun {
	tmp, err := os.MkdirTemp("", "workline-examples-")
	if err != nil {
		return exampleRun{out: err.Error()}
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, "tree")
	if _, err := git(repo, "worktree", "add", "-q", "--detach", dir, "HEAD"); err != nil {
		return exampleRun{out: "no checkout to run it in: " + err.Error()}
	}
	defer git(repo, "worktree", "remove", "--force", dir)
	for p, content := range docs {
		file := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return exampleRun{out: err.Error()}
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return exampleRun{out: err.Error()}
		}
	}
	cmd := exec.Command("sh", append([]string{"-c", command, "sh"}, named...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return exampleRun{ok: true}
	}
	return exampleRun{out: outputSaid(string(out), err)}
}

// outputSaid is a command's output on one line, its last lines kept: what
// a finding and a merge request's list can show.
func outputSaid(out string, err error) string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return err.Error()
	}
	if len(lines) > 10 {
		lines = append([]string{"…"}, lines[len(lines)-10:]...)
	}
	return strings.Join(lines, " / ")
}

// linesOf says where blocks stand: "10-12", "10-12, 20-25".
func linesOf(fs []fence, lang bool) string {
	var out []string
	for _, f := range fs {
		s := fmt.Sprintf("%d-%d", f.from, f.to)
		if lang && f.lang != "" {
			s += ", " + f.lang
		}
		out = append(out, s)
	}
	if lang {
		return strings.Join(out, "; ")
	}
	return strings.Join(out, ", ")
}

// nExamples says n code examples, in words.
func nExamples(n int) string {
	if n == 1 {
		return "1 code example"
	}
	return fmt.Sprintf("%d code examples", n)
}

// exampleFindings says, of each doc the accepted fixes wrote code examples
// in, whether the project's command checked them: never a plain fix.
func (j *judgedFixes) exampleFindings() []verdict.Finding {
	if j == nil {
		return nil
	}
	var out []verdict.Finding
	for _, p := range sortedKeysOf(j.examples) {
		w := j.examples[p]
		said := fmt.Sprintf("%s (lines %s)", nExamples(len(w)), linesOf(w, true))
		if j.checked[p] {
			out = append(out, verdict.Finding{Rule: "example-checked", Where: p,
				Message: fmt.Sprintf("%s, checked by `%s`", said, j.command)})
			continue
		}
		it := "it"
		if len(w) > 1 {
			it = "them"
		}
		msg := fmt.Sprintf("%s, unverified: no check ran on %s; read %s before merging", said, it, it)
		if j.command == "" {
			msg += "; a command checking the project's examples can be named in the documentalist's `examples.check`"
		}
		out = append(out, verdict.Finding{Rule: "example-unverified", Where: p, Level: "warn", Message: msg})
	}
	return out
}

// examplesOutcome says, for a merge request's list of docs, the code
// examples a fix wrote in a doc: unverified, or checked by the command.
func (j *judgedFixes) examplesOutcome(p string) string {
	if j == nil || len(j.examples[p]) == 0 {
		return ""
	}
	w := j.examples[p]
	the, it := "the code example", "it"
	if len(w) > 1 {
		the, it = fmt.Sprintf("the %d code examples", len(w)), "them"
	}
	if j.checked[p] {
		return fmt.Sprintf("; %s checked by `%s` (lines %s)", the, j.command, linesOf(w, false))
	}
	return fmt.Sprintf("; %s unverified (lines %s): read %s before merging", the, linesOf(w, false), it)
}

// inExample says whether a hunk of a zero-context diff of a doc's body
// changes only lines of the examples a fix wrote in it, and blank lines.
func (j *judgedFixes) inExample(path string, from, count int) bool {
	w := j.examples[path]
	if len(w) == 0 {
		return false
	}
	_, head := header(j.after[path])
	if count == 0 { // lines taken out after line from
		return inFences(w, from+head) && inFences(w, from+head+1)
	}
	lines := strings.Split(body(j.after[path]), "\n")
	for n := from; n < from+count; n++ {
		if !inFences(w, n+head) && n <= len(lines) && strings.TrimSpace(lines[n-1]) != "" {
			return false
		}
	}
	return true
}

func sortedKeysOf(m map[string][]fence) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
