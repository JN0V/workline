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
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/codepick"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
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
	// On the forge (#128): the issue read after the product owner's
	// refine, the digest of its body as read, and the round this is.
	Issue  int    `json:"issue,omitempty"`
	Digest string `json:"digest,omitempty"`
	Round  int    `json:"round,omitempty"`
	// Prior is the record the last review left, carried as it was when
	// this one is not whole.
	Prior *backlog.SpecReview `json:"prior,omitempty"`
	// Mine are the issue's sections the product owner may rewrite to
	// answer a finding, the Scope always; the Verification is the
	// tester's (ADR-0066), the others a person's (ADR-0060).
	Mine []string `json:"mine,omitempty"`
}

// heading names the spec where the lenses and the judge read it.
func (sp Spec) heading() string {
	if sp.Title != "" {
		return fmt.Sprintf("Issue %s: %s", sp.Path, sp.Title)
	}
	return sp.Path
}

// prepareSpec is a spec run's first pass: the spec read, the code it names
// found, and the spec lenses asked. On the forge, the specs that wait.
func prepareSpec(runDir, repo string, s Settings) int {
	head, err := git(repo, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return fail(fmt.Errorf("the code a spec is read against: %v", err))
	}
	head = strings.TrimSpace(head)
	file, issue := input(runDir, "spec"), input(runDir, "issue")
	if (os.Getenv("WORKLINE_EVENT") == gardening || os.Getenv("WORKLINE_EVENT") == issueEvent) && file == "" && issue == "" {
		return prepareSpecs(runDir, repo, s, head)
	}
	sp, cut, err := readSpec(repo, file, issue, s.SpecLinesMax)
	if err != nil {
		return fail(err)
	}
	st := state{Head: head, From: head, Base: head, Spec: &sp}
	if cut > 0 {
		st.Advisories = append(st.Advisories, specCut(sp, cut, s))
	}
	if noAgent() {
		st.Advisories = append(st.Advisories, notReviewed(sp))
		return final(runDir, review(st, nil, "no agent: the spec was not reviewed"))
	}
	asked, err := askSpec(runDir, repo, s, &st)
	if err != nil {
		return fail(err)
	}
	if !asked {
		return final(runDir, review(st, nil, "no spec lens asked can read this spec: not asked: "+strings.Join(st.Skipped, ", ")))
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review-state.json"), st); err != nil {
		return fail(err)
	}
	return 0
}

// prepareSpecs is a spec run's first pass on the forge, after the product
// owner: the issues it refined that wait on a review, up to specs-per-run
// (#433), each read as a spec of its own, its lenses asked in parts of its
// own; the run holds them, with the questions to a person and what it says.
func prepareSpecs(runDir, repo string, s Settings, head string) int {
	q, code, ok := specOnForge(runDir, repo, s)
	if !ok {
		return code
	}
	run := state{Head: head, From: head, Base: head, Advisories: q.said, Stops: q.stops, Stopped: q.stopped, Waits: q.rest}
	var subs []state
	for k := range q.read {
		sp := q.read[k]
		cut := 0
		if lines := strings.Split(strings.TrimRight(strings.ReplaceAll(sp.Text, "\r\n", "\n"), "\n"), "\n"); s.SpecLinesMax > 0 && len(lines) > s.SpecLinesMax {
			sp.Text, cut = strings.Join(lines[:s.SpecLinesMax], "\n"), len(lines)-s.SpecLinesMax
		} else {
			sp.Text = strings.Join(lines, "\n")
		}
		sub := state{Head: head, From: head, Base: head, Spec: &sp}
		if cut > 0 {
			sub.Advisories = append(sub.Advisories, specCut(sp, cut, s))
		}
		if len(q.read) > 1 { // its parts told from the others' (#433)
			sub.Part = fmt.Sprintf("%02d", k+1)
		}
		subs = append(subs, sub)
	}
	if noAgent() {
		var paths []string
		for _, sub := range subs {
			run.Advisories = append(append(run.Advisories, sub.Advisories...), notReviewed(*sub.Spec))
			paths = append(paths, sub.Spec.Path)
		}
		v := review(run, nil, "no agent: the spec was not reviewed")
		if len(subs) > 1 {
			v.Summary = fmt.Sprintf("no agent: the %d specs were not reviewed", len(subs))
		}
		v.Subject, v.Spec = "spec", strings.Join(paths, ", ")
		v.Summary += afterSpecs(run)
		if len(run.Stops) == 0 {
			return final(runDir, v)
		}
		// The questions to a person need no agent: written all the same.
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), run.Stops); err != nil {
			return fail(err)
		}
		if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
			return fail(err)
		}
		return 0
	}
	for i := range subs {
		asked, err := askSpec(runDir, repo, s, &subs[i])
		if err != nil {
			return fail(err)
		}
		if !asked {
			sp := subs[i].Spec
			run.Advisories = append(run.Advisories, verdict.Finding{Rule: "not-reviewed", Where: sp.Path, Level: "warn",
				Message: "no spec lens asked can read this spec: not asked: " + strings.Join(subs[i].Skipped, ", ")})
			continue
		}
		run.Specs = append(run.Specs, subs[i])
	}
	if len(run.Specs) == 0 {
		v := review(run, nil, "no spec lens asked can read the specs that wait")
		v.Subject = "spec"
		return final(runDir, v)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review-state.json"), run); err != nil {
		return fail(err)
	}
	return 0
}

// noAgent says whether the run has no agent to ask.
func noAgent() bool {
	ai := os.Getenv("WORKLINE_AI")
	return ai == "" || ai == "none"
}

// notReviewed says a spec was read by nobody, no agent being there.
func notReviewed(sp Spec) verdict.Finding {
	return verdict.Finding{Rule: "not-reviewed", Where: sp.Path, Level: "warn",
		Message: "no agent: nobody read it; the spec waits for a person's review"}
}

// specCut says a spec was cut past spec-lines-max.
func specCut(sp Spec, cut int, s Settings) verdict.Finding {
	return verdict.Finding{Rule: "spec-cut", Where: sp.Path, Level: "warn",
		Message: fmt.Sprintf("the spec is cut past spec-lines-max (%d): its last %d lines were not read", s.SpecLinesMax, cut)}
}

// askSpec finds the code a spec names and writes its lenses' parts: one
// holding them all, or one a lens. asked is false when no spec lens can
// read it, each lens left out said in st.Skipped.
func askSpec(runDir, repo string, s Settings, st *state) (asked bool, err error) {
	sp := *st.Spec
	code, cut := namedCode(repo, st.Head, sp.Path, sp.Title+"\n"+sp.Text, s.CodeLinesMax, "code-lines-max")
	if cut != "" {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: verdict.CutRule, Where: sp.Path, Level: "warn",
			Message: fmt.Sprintf("the code the spec names given to the reviewer's agent is cut past code-lines-max (%d lines): %s; raise %s", s.CodeLinesMax, cut, verdict.Setting("reviewer", "code-lines-max"))})
	}
	skip := map[string]string{}
	read := map[string]Lens{}
	for _, name := range s.SpecLenses {
		l, err := readLens(repo, name)
		if err != nil {
			return false, err
		}
		if l.Subject != "spec" {
			return false, fmt.Errorf("lens %q reads code: it belongs under lenses, not spec-lenses", name)
		}
		read[name] = l
		if l.Needs == "code" && code == "" {
			skip[name] = "the spec names no code the repository holds"
			st.Skipped = append(st.Skipped, fmt.Sprintf("%s (%s)", name, skip[name]))
		}
	}
	if st.Lenses, err = specLenses(runDir, s, skip); err != nil {
		return false, err
	}
	if len(st.Lenses) == 0 {
		return false, nil
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
		return true, write(partName(*st, st.Lenses[0]), b.String()+floor+"\n"+material)
	}
	for _, lens := range st.Lenses {
		if err := write(partName(*st, lens), fmt.Sprintf("# Lens: %s\n\n%s\n%s\n%s", lens, strings.TrimSpace(read[lens].Text), floor, material)); err != nil {
			return true, err
		}
	}
	return true, nil
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
	return fmt.Sprintf("\n## What you review: a spec, before it is built\n\nNot a change to code: a spec, what a person or an agent will build from. What you find is its author's to fix: the product owner, or the person who wrote it. Each finding's `cause` quotes the spec as it reads, its `path` %q; a `symptom`, only where the code contradicts the spec, quotes the code, its `path` the file. An open point only a person can settle is no defect: a finding holding `decision`, the question for them, its cause the open point quoted.\n\n## The spec: %s\n\n", sp.Path, sp.heading())
}

// fenced puts a text in a fence longer than any it holds.
func fenced(text string) string {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	return fmt.Sprintf("%s\n%s\n%s\n", fence, text, fence)
}

// namedCode is what a spec names of the repository's code, as it reads at
// head, found by the picker the product owner and the tester share
// (ADR-0063 §8): the lines it names after a file, each definition of a
// name it writes as code whole, a file it names whole when it fits, the
// lines naming a name, up to limit lines all together, the rest said; ""
// when it names none (#128, #425). cut is what limit left out of the
// files it names, for the person (codepick's Cut, #1159).
func namedCode(repo, head, spec, text string, limit int, setting string) (code, cut string) {
	pk, err := codepick.New(repo, head)
	if err != nil {
		return "", ""
	}
	a := codepick.Ask{Text: text, Tests: testsOf(repo), Budget: max(limit, 1), BudgetName: setting}
	if spec != "" && !strings.HasPrefix(spec, "#") {
		a.Skip = []string{spec} // the spec itself: given above
	}
	pick := pk.Pick(a)
	if len(pick.Named()) == 0 && len(pick.Many) == 0 {
		return "", ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n## The code the spec names\n\nAs it reads at %s: the lines it names, each definition of a name it writes as code, a file it names whole when it fits, then the lines naming them, up to %d lines all together (%s); the rest said. Found with no build: a name may be taken for another of the same name.\n", short(head), limit, setting)
	pick.Write(&b, codepick.Style{Numbered: true, Namer: "the spec"})
	return b.String(), pick.Cut()
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
	if named, _ := namedCode(repo, st.Head, st.Spec.Path, st.Spec.Title+"\n"+st.Spec.Text, s.JudgeLinesMax, "judge-lines-max"); named != "" {
		b.WriteString(named)
	} else {
		b.WriteString("\n## The code the spec names\n\nNone the repository holds.\n")
	}
	return b.String()
}
