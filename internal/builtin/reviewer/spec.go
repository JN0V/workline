package reviewer

// The spec subject (ADR-0020, #128): a spec read before it is built, a
// file of the repository or an issue by its number, through lenses of its
// own (`spec-lenses`); each finding's cause quoted from the spec and found
// again there, the important ones judged, the verdict by the same rules as
// for code. It runs on the `spec` event: `workline review --spec` or
// `--issue` on a machine; on the forge, the line that routes it there.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// specEvent is the event a spec is reviewed on: its input names the spec,
// a file (`spec`) or an issue's number (`issue`).
const specEvent = "spec"

// Spec is the spec a run reviews, as the lenses were given it.
type Spec struct {
	Path  string `json:"path"`            // the file, or the issue: `#4`
	Title string `json:"title,omitempty"` // an issue's title
	Text  string `json:"text"`            // the file, or the issue's body: where quotes are found again
}

// heading names the spec where the lenses and the judge read it.
func (sp Spec) heading() string {
	if sp.Title != "" {
		return fmt.Sprintf("Issue %s: %s", sp.Path, sp.Title)
	}
	return sp.Path
}

// prepareSpec is a spec run's first pass: the spec read, the code it names
// found, and the spec lenses asked.
func prepareSpec(runDir, repo string, s Settings) int {
	head, err := git(repo, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return fail(fmt.Errorf("the code a spec is read against: %v", err))
	}
	head = strings.TrimSpace(head)
	sp, cut, err := readSpec(repo, input(runDir, "spec"), input(runDir, "issue"), s.SpecLinesMax)
	if err != nil {
		return fail(err)
	}
	st := state{Head: head, From: head, Base: head, Spec: &sp}
	if cut > 0 {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "spec-cut", Where: sp.Path, Level: "warn",
			Message: fmt.Sprintf("the spec is cut past spec-lines-max (%d): its last %d lines were not read", s.SpecLinesMax, cut)})
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "not-reviewed", Where: sp.Path, Level: "warn",
			Message: "no agent: nobody read it; the spec waits for a person's review"})
		return final(runDir, review(st, nil, "no agent: the spec was not reviewed"))
	}
	code := namedCode(repo, head, sp.Text, s.CodeLinesMax)
	skip := map[string]string{}
	read := map[string]Lens{}
	for _, name := range s.SpecLenses {
		l, err := readLens(repo, name)
		if err != nil {
			return fail(err)
		}
		if l.Subject != "spec" {
			return fail(fmt.Errorf("lens %q reads code: it belongs under lenses, not spec-lenses", name))
		}
		read[name] = l
		if l.Needs == "code" && code == "" {
			skip[name] = "the spec names no code the repository holds"
			st.Skipped = append(st.Skipped, fmt.Sprintf("%s (%s)", name, skip[name]))
		}
	}
	if st.Lenses, err = specLenses(runDir, s, skip); err != nil {
		return fail(err)
	}
	if len(st.Lenses) == 0 {
		return final(runDir, review(st, nil, "no spec lens asked can read this spec: not asked: "+strings.Join(st.Skipped, ", ")))
	}
	st.Together = s.LensesTogether && len(st.Lenses) > 1
	material := specIntro(sp) + fenced(sp.Text)
	if slices.ContainsFunc(st.Lenses, func(l string) bool { return read[l].Needs == "code" }) {
		material += code
	}
	floor := ""
	if s.FinderFloor {
		each := ""
		if st.Together {
			each = " for each lens"
		}
		floor = fmt.Sprintf("\nLook for at least %d candidates%s before you stop; then give only those whose cause you can quote, important or nit as each deserves.\n", candidatesFor(len(material)), each)
	}
	write := func(name, task string) error {
		dir := filepath.Join(runDir, "in", "parts", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "task.md"), []byte(task), 0o644)
	}
	if st.Together {
		var b strings.Builder
		fmt.Fprintf(&b, "# Lenses: %s\n\nEach lens below is a question of its own: ask each in turn, of the whole spec, and name in each finding the lens that found it: `lens:` one of %s.\n",
			strings.Join(st.Lenses, ", "), strings.Join(st.Lenses, ", "))
		for _, lens := range st.Lenses {
			fmt.Fprintf(&b, "\n## Lens: %s\n\n%s\n", lens, strings.TrimSpace(read[lens].Text))
		}
		err = write(partName(st, st.Lenses[0]), b.String()+floor+"\n"+material)
	} else {
		for _, lens := range st.Lenses {
			if err = write(partName(st, lens), fmt.Sprintf("# Lens: %s\n\n%s\n%s\n%s", lens, strings.TrimSpace(read[lens].Text), floor, material)); err != nil {
				break
			}
		}
	}
	if err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review-state.json"), st); err != nil {
		return fail(err)
	}
	return 0
}

// readSpec reads the spec a run is given: a file of the repository, as it
// reads in the working tree, or an issue's body from the forge. Past limit
// lines it is cut; cut says how many were left.
func readSpec(repo, file, issue string, limit int) (sp Spec, cut int, err error) {
	switch {
	case file != "" && issue != "":
		return sp, 0, fmt.Errorf("a spec is a file (spec) or an issue (issue), not both")
	case file != "":
		p := clean(file)
		if filepath.IsAbs(p) || p == ".." || strings.HasPrefix(p, "../") {
			return sp, 0, fmt.Errorf("the spec %s: not a file of the repository", file)
		}
		data, err := os.ReadFile(filepath.Join(repo, p))
		if err != nil {
			return sp, 0, fmt.Errorf("the spec: %v", err)
		}
		sp = Spec{Path: p, Text: string(data)}
	case issue != "":
		n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(issue), "#"))
		if err != nil || n <= 0 {
			return sp, 0, fmt.Errorf("the issue %q: not an issue's number", issue)
		}
		f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
		if err != nil {
			return sp, 0, fmt.Errorf("the issue #%d: the forge: %v", n, err)
		}
		if f == nil {
			return sp, 0, fmt.Errorf("the issue #%d: no forge to read it from (--forge, or the project's `forge` setting)", n)
		}
		is, err := f.Issue(n)
		if err != nil {
			return sp, 0, fmt.Errorf("the issue #%d: %v", n, err)
		}
		sp = Spec{Path: fmt.Sprintf("#%d", n), Title: strings.TrimSpace(is.Title), Text: is.Body}
	default:
		return sp, 0, fmt.Errorf("the %s event reviews a spec: give its file (spec) or its issue's number (issue)", specEvent)
	}
	sp.Text = strings.TrimRight(strings.ReplaceAll(sp.Text, "\r\n", "\n"), "\n")
	if strings.TrimSpace(sp.Text) == "" {
		return sp, 0, fmt.Errorf("the spec %s is empty: nothing to review", sp.Path)
	}
	if lines := strings.Split(sp.Text, "\n"); limit > 0 && len(lines) > limit {
		sp.Text, cut = strings.Join(lines[:limit], "\n"), len(lines)-limit
	}
	return sp, cut, nil
}

// specLenses are the spec lenses a run asks: every one, or those it is
// told (`lenses`), a lens with nothing to read (skip) left out.
func specLenses(runDir string, s Settings, skip map[string]string) ([]string, error) {
	if len(s.SpecLenses) == 0 {
		return nil, fmt.Errorf("settings: no spec-lenses")
	}
	want := s.SpecLenses
	if asked := input(runDir, "lenses"); asked != "" && asked != "all" {
		want = nil
		for _, l := range strings.Split(asked, ",") {
			l = strings.TrimSpace(l)
			if !slices.Contains(s.SpecLenses, l) {
				return nil, fmt.Errorf("lens %q: not one of the spec lenses, %s", l, strings.Join(s.SpecLenses, ", "))
			}
			want = append(want, l)
		}
	}
	var out []string
	for _, l := range want {
		if _, ok := skip[l]; !ok && !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return out, nil
}

// specIntro tells the lenses what they read: a spec, not a change, whose
// author fixes what they find.
func specIntro(sp Spec) string {
	return fmt.Sprintf("\n## What you review: a spec, before it is built\n\nNot a change to code: a spec, what a person or an agent will build from. What you find is its author's to fix: the product owner, or the person who wrote it. Each finding's `cause` quotes the spec as it reads, its `path` %q; a `symptom`, only where the code contradicts the spec, quotes the code, its `path` the file. An open point only a person can settle is a `decision`, not a finding.\n\n## The spec: %s\n\n", sp.Path, sp.heading())
}

// fenced puts a text in a fence longer than any it holds.
func fenced(text string) string {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	return fmt.Sprintf("%s\n%s\n%s\n", fence, text, fence)
}

// pathWord is a word of a spec that may name a file: letters, digits and
// `._-/`, the punctuation closing a sentence left.
var pathWord = regexp.MustCompile(`[\w.\-/]+`)

// namedCode is what a spec names of the repository's code, as it reads at
// head: the files it names, whole, then the functions it names that those
// do not hold, whole, up to limit lines all together, the rest named; ""
// when it names none (#128). Found as a judge's code is (#127), with no
// build.
func namedCode(repo, head, text string, limit int) string {
	out, err := git(repo, "ls-tree", "-r", "--name-only", head)
	if err != nil {
		return ""
	}
	tracked := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		tracked[f] = true
	}
	var files []string
	for _, w := range pathWord.FindAllString(text, -1) {
		w = strings.TrimPrefix(strings.TrimRight(w, ".-"), "./")
		if tracked[w] && languageOf(w) != nil && !slices.Contains(files, w) {
			files = append(files, w)
		}
	}
	r := &reacher{repo: repo, head: head, files: map[string][]string{}}
	names, weak := namedIn(text)
	names = append(names, weak...)
	found := r.defs(names, nil)
	var b strings.Builder
	left := limit
	var cut, many []string
	for _, f := range files {
		lines := r.lines(f)
		if len(lines) > left {
			cut = append(cut, f)
			continue
		}
		left -= len(lines)
		fmt.Fprintf(&b, "\n### %s\n\n%s", f, fenced(strings.Join(lines, "\n")))
	}
	shown := 0
	for _, n := range names {
		ds := found[n]
		ds = slices.DeleteFunc(slices.Clone(ds), func(d fn) bool { return slices.Contains(files, d.file) })
		switch {
		case len(ds) == 0:
			continue
		case len(ds) > 2:
			many = append(many, fmt.Sprintf("`%s` (%d definitions)", n, len(ds)))
			continue
		}
		for _, d := range ds {
			if d.lines() > left {
				cut = append(cut, fmt.Sprintf("`%s` (%s:%d-%d)", d.name, d.file, d.from, d.to))
				continue
			}
			left -= d.lines()
			shown++
			lines := r.lines(d.file)
			fmt.Fprintf(&b, "\n### %s, lines %d to %d: `%s`\n\n%s", d.file, d.from, d.to, d.name, fenced(strings.Join(lines[d.from-1:min(d.to, len(lines))], "\n")))
		}
	}
	if b.Len() == 0 && len(cut) == 0 && len(many) == 0 {
		return ""
	}
	if len(cut) > 0 {
		fmt.Fprintf(&b, "\nNot shown, past %d lines: %s.\n", limit, strings.Join(cut, ", "))
	}
	if len(many) > 0 {
		fmt.Fprintf(&b, "\nNot shown, defined in too many places to tell which: %s.\n", strings.Join(many, ", "))
	}
	return fmt.Sprintf("\n## The code the spec names\n\nAs it reads at %s: the files the spec names, whole, then the functions it names, up to %d lines all together; the rest named. Found with no build: a name may be taken for another of the same name.\n%s", short(head), limit, b.String())
}

// inSpec finds a spec finding's quotes again: its cause in the spec, as
// the lenses were given it; its symptom, when it gives one, in the code at
// head. Every finding on a spec is its author's to fix.
func inSpec(repo string, st state, f Finding) (Finding, string) {
	sp := st.Spec
	if f.Cause.Path != sp.Path {
		return f, fmt.Sprintf("its cause names %s, not the spec reviewed, %s", f.Cause.Path, sp.Path)
	}
	places := locate(strings.Split(sp.Text, "\n"), f.Cause.Quote)
	if len(places) == 0 {
		return f, fmt.Sprintf("its cause is not found in %s, as the lenses were given it", sp.Path)
	}
	f.Where, f.Related = fmt.Sprintf("%s:%d", sp.Path, places[0].From), true
	if f.Symptom != nil {
		f.Symptom.Path = clean(f.Symptom.Path)
		text, ok := fileAt(repo, st.Head, f.Symptom.Path)
		if !ok || len(locate(strings.Split(text, "\n"), f.Symptom.Quote)) == 0 {
			return f, fmt.Sprintf("its symptom is not found in %s as it reads at %s", f.Symptom.Path, short(st.Head))
		}
	}
	return f, ""
}

// specMaterial is what the judge of a spec's finding reads: the finding,
// the spec whole, and, for a lens whose judge reads code, the code it
// stands on: the function its symptom lies in and those it reaches (#127),
// else the code the spec names, up to judge-lines-max lines.
func specMaterial(repo string, st state, s Settings, f Finding, code bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The finding\n\n%s: %s\n\nWhy: %s\n", f.Severity, f.Title, f.Why)
	fmt.Fprintf(&b, "\nIts cause, quoted from the spec (%s):\n\n```\n%s\n```\n", f.Where, strings.TrimSpace(f.Cause.Quote))
	if f.Symptom != nil {
		fmt.Fprintf(&b, "\nThe code it says contradicts the spec, quoted from %s:\n\n```\n%s\n```\n", f.Symptom.Path, strings.TrimSpace(f.Symptom.Quote))
	}
	fmt.Fprintf(&b, "\n## The spec: %s\n\n%s", st.Spec.heading(), fenced(st.Spec.Text))
	if !code {
		return b.String()
	}
	if f.Symptom != nil {
		text, _ := fileAt(repo, st.Head, f.Symptom.Path)
		at := 1
		if p := locate(strings.Split(text, "\n"), f.Symptom.Quote); len(p) > 0 {
			at = p[0].From
		}
		on := Finding{Title: f.Title, Why: f.Why, Fix: f.Fix, Cause: *f.Symptom}
		b.WriteString(reach(repo, st.Head, s.Tests, on, at, 0, s.JudgeLinesMax))
		return b.String()
	}
	if named := namedCode(repo, st.Head, st.Spec.Text, s.JudgeLinesMax); named != "" {
		b.WriteString(named)
	} else {
		b.WriteString("\n## The code the spec names\n\nNone the repository holds.\n")
	}
	return b.String()
}
