// Package reviewer holds the deterministic steps of the reviewer role
// (roles/reviewer, ADR-0020): the rules no judgement is needed for, the
// lenses put to the agent as parts of one question, each finding's quotes
// found again, whether it lies in the change or outside it, the judge's
// answers read, and the verdict. The role's scripts call them through
// `workline builtin reviewer pre|post`.
package reviewer

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

const (
	exitBlock    = 1
	exitExternal = 3
	exitNothing  = 10
)

// Settings are the role's settings, as merged by the engine.
type Settings struct {
	Base            string   `json:"base"`              // where a review on a machine starts from, when no range is given
	Ignore          []string `json:"ignore"`            // files that are not code: a change touching only these asks nobody
	Lenses          []string `json:"lenses"`            // the lenses, in the order a merge request takes them in turn
	LensesPerPush   int      `json:"lenses-per-push"`   // on a merge request: lenses a push gets
	FindingsMax     int      `json:"findings-max"`      // findings on the change reported a run; the rest counted
	IssuesMax       int      `json:"issues-max"`        // issues opened a run for what lies outside the change
	DiffLinesMax    int      `json:"diff-lines-max"`    // lines of the change given to a lens
	CodeLinesMax    int      `json:"code-lines-max"`    // lines of the files changed given to a lens, all together
	CommentBlockMax int      `json:"comment-block-max"` // lines of one comment added
	StoryWords      []string `json:"story-words"`       // what tells a bug's story in a comment
	AIFindings      string   `json:"ai-findings"`       // warn, until measured (#90); block: a verified important finding blocks
	JudgeAtLeast    string   `json:"judge-at-least"`    // the independence a verification needs: context, model, provider
	ForgeWrites     bool     `json:"forge-writes"`      // on a merge request: the summary comment and the issues
	FinderFloor     bool     `json:"finder-floor"`      // ask each lens to look for a number of candidates before it stops
}

// Finding is one defect a lens reported, once its quotes were found again.
type Finding struct {
	Lens     string   `json:"lens"`
	Severity string   `json:"severity"` // important or nit
	Title    string   `json:"title"`
	Why      string   `json:"why"`
	Fix      string   `json:"fix,omitempty"`
	Cause    Quote    `json:"cause"`
	Symptom  *Quote   `json:"symptom,omitempty"`
	Where    string   `json:"where"`          // the cause's file and line
	Related  bool     `json:"related"`        // the cause lies in the change: the author fixes it
	Line     string   `json:"line,omitempty"` // outside the change: the line the cause starts at, as it reads, which keys its issue
	Also     []string `json:"also,omitempty"` // other findings on the same line, merged into this one: lens and title
	Verified string   `json:"verified,omitempty"`
	Level    string   `json:"independence,omitempty"`
}

// Quote is a place a finding quotes.
type Quote struct {
	Path  string `json:"path" yaml:"path"`
	Quote string `json:"quote" yaml:"quote"`
}

// state is what pre found on its first pass, for the passes after the
// lenses and the judge answered.
type state struct {
	Head       string            `json:"head"`
	From       string            `json:"from"` // where the commits not reviewed yet start
	Base       string            `json:"base"`
	Change     Change            `json:"change"` // the whole range's
	Mechanical []verdict.Finding `json:"mechanical"`
	Commits    []string          `json:"commits"`
	Lenses     []string          `json:"lenses"`
	Record     Record            `json:"record"`
	Advisories []verdict.Finding `json:"advisories,omitempty"`
}

// candidates is what the lenses answered, its quotes checked, before the judge.
type candidates struct {
	Related, Outside []Finding
	Logged           []verdict.Finding // dropped, each said
	Failed           []string          // lenses with no answer that reads
	Asked            []string          // the questions put to the judge, by finding
}

// Review is what the run found, as post reads it and out/review.json gives
// it to the author's agent.
type Review struct {
	Status   string            `json:"status"`
	Summary  string            `json:"summary"`
	Findings []verdict.Finding `json:"findings"`
	Change   []Finding         `json:"change"`  // on the change: the author fixes them
	Outside  []Finding         `json:"outside"` // outside it: an issue each
	Lenses   []string          `json:"lenses,omitempty"`
	Complete bool              `json:"complete"` // every lens answered: the commits are recorded as reviewed
	Record   Record            `json:"record"`
	Local    bool              `json:"local"` // the record is kept on this machine
}

// Pre prepares the review, in up to three passes: the rules and the lenses'
// questions; the lenses' answers read and the judge's questions written;
// the judge's answers read and what follows proposed.
func Pre(runDir, repo string) int {
	s, err := settings(runDir)
	if err != nil {
		return fail(err)
	}
	switch {
	case os.Getenv("WORKLINE_JUDGED") == "answered":
		return settle(runDir, repo, s)
	case os.Getenv("WORKLINE_PARTS") == "answered":
		return readLenses(runDir, repo, s)
	}
	return prepare(runDir, repo, s)
}

func prepare(runDir, repo string, s Settings) int {
	rng := input(runDir, "range")
	if rng == "" {
		rng = s.Base + "..HEAD"
	}
	head, err := git(repo, "rev-parse", "--verify", gitrange.Head(rng)+"^{commit}")
	if err != nil {
		return fail(fmt.Errorf("the range %s: %v", rng, err))
	}
	head = strings.TrimSpace(head)
	base := head
	if b := gitrange.Base(rng); b != "" {
		mb, err := git(repo, "merge-base", b, head)
		if err != nil {
			return fail(fmt.Errorf("the range %s: no common commit with %s: %v", rng, b, err))
		}
		base = strings.TrimSpace(mb)
	}
	names, err := git(repo, "diff", "--name-only", "--no-renames", base, head)
	if err != nil {
		return fail(err)
	}
	files := code(s, names)
	if len(files) == 0 {
		return final(runDir, Review{Status: verdict.Pass, Summary: "no code changed (only files the reviewer leaves, by its `ignore`): nobody asked"})
	}
	change, err := readChange(repo, base, head, files)
	if err != nil {
		return fail(err)
	}
	codes, err := committerSettings(repo)
	if err != nil {
		return fail(err)
	}
	mech, err := mechanical(change, s, codes)
	if err != nil {
		return fail(err)
	}
	rec, _, err := loadRecord(repo, s)
	var advisories []verdict.Finding
	if err != nil {
		advisories = append(advisories, verdict.Finding{Rule: "record-unread", Level: "warn",
			Message: "what was reviewed before could not be read (" + err.Error() + "): every commit of the range is reviewed again"})
		rec = Record{}
	}
	out, err := git(repo, "rev-list", base+".."+head)
	if err != nil {
		return fail(err)
	}
	commits := strings.Fields(out) // newest first
	start := base
	for _, c := range commits {
		if slices.Contains(rec.Reviewed, c) {
			start = c
			break
		}
	}
	st := state{Head: head, From: start, Base: base, Change: change, Mechanical: mech, Commits: commits, Record: rec, Advisories: advisories}
	if start == head {
		return final(runDir, review(st, nil, fmt.Sprintf("the %d commits were reviewed already: only the rules ran", len(commits))))
	}
	// The rules first: what they block on is fixed before any agent is
	// asked, and asked again after each push while it is not.
	if status(mech) == verdict.Block {
		return final(runDir, review(st, nil, "the rules found what the author must fix first; the review follows once they pass"))
	}
	// A lens reads what the commits not reviewed yet change, not every file
	// of the merge request; when they change no code, nobody is asked.
	if start != base {
		if files, err = codeFiles(repo, s, start, head); err != nil {
			return fail(err)
		}
		if len(files) == 0 {
			return recordOnly(runDir, s, st)
		}
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "not-reviewed", Level: "warn",
			Message: "no agent: only the rules that need no judgement ran; the change waits for a person's review"})
		return final(runDir, review(st, nil, "no agent: only the rules ran"))
	}
	st.Lenses, err = lenses(runDir, s, rec)
	if err != nil {
		return fail(err)
	}
	if err := ask(runDir, repo, s, st, files); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review-state.json"), st); err != nil {
		return fail(err)
	}
	return 0
}

// code is the files of a `git diff --name-only` the reviewer reviews: those
// its `ignore` does not leave.
func code(s Settings, names string) []string {
	var files []string
	for _, f := range strings.Split(strings.TrimSpace(names), "\n") { // a name may hold a space
		if f != "" && !pathglob.Any(s.Ignore, f) {
			files = append(files, f)
		}
	}
	return files
}

// codeFiles is the code the commits from..head change.
func codeFiles(repo string, s Settings, from, head string) ([]string, error) {
	names, err := git(repo, "diff", "--name-only", "--no-renames", from, head)
	if err != nil {
		return nil, err
	}
	return code(s, names), nil
}

// recordOnly settles a run whose new commits change no code: nobody asked,
// the commits recorded as reviewed, and the turn of the lenses left where
// it was, for the next push that changes code.
func recordOnly(runDir string, s Settings, st state) int {
	v := review(st, nil, fmt.Sprintf("no code changed since the last review (only files the reviewer leaves, by its `ignore`): nobody asked, the %d new commits recorded", slices.Index(st.Commits, st.From)))
	v.Complete = true
	v.Record = st.Record.add(st.Commits)
	v.Record.Runs = st.Record.Runs
	var fallback []intent.Intention
	if mergeRequest() != nil && s.ForgeWrites {
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{"sticky": SummaryKey, "body": summaryComment(v, 0)}})
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// lenses are the lenses this run asks: every one on a machine, or when the
// run is told `lenses=all`; on a merge request, the next in turn.
func lenses(runDir string, s Settings, rec Record) ([]string, error) {
	if len(s.Lenses) == 0 {
		return nil, fmt.Errorf("settings: no lenses")
	}
	asked := input(runDir, "lenses")
	switch {
	case asked == "all":
		return s.Lenses, nil
	case asked != "":
		var out []string
		for _, l := range strings.Split(asked, ",") {
			l = strings.TrimSpace(l)
			if !slices.Contains(s.Lenses, l) {
				return nil, fmt.Errorf("lens %q: not one of %s", l, strings.Join(s.Lenses, ", "))
			}
			out = append(out, l)
		}
		return out, nil
	case os.Getenv("WORKLINE_EVENT") != "merge-request":
		return s.Lenses, nil
	}
	n := max(1, min(s.LensesPerPush, len(s.Lenses)))
	var out []string
	for i := range n {
		out = append(out, s.Lenses[(rec.Runs+i)%len(s.Lenses)])
	}
	return out, nil
}

// ask writes one part a lens: the lens, the commits not reviewed yet, and
// the files they change.
func ask(runDir, repo string, s Settings, st state, files []string) error {
	diff, err := git(repo, append([]string{"diff", "-U5", "--no-color", "--no-ext-diff", st.From, st.Head, "--"}, files...)...)
	if err != nil {
		return err
	}
	diff = capLines(diff, s.DiffLinesMax, "the change is cut here: review what is above")
	log, err := git(repo, "log", "--format=- %h %s", st.From+".."+st.Head)
	if err != nil {
		return err
	}
	var code strings.Builder
	left := s.CodeLinesMax
	for _, f := range files {
		text, ok := fileAt(repo, st.Head, f)
		if !ok || left <= 0 {
			continue
		}
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		cut := ""
		if len(lines) > left {
			lines, cut = lines[:left], "\n(cut here)"
		}
		left -= len(lines)
		fmt.Fprintf(&code, "### %s\n\n```\n%s%s\n```\n\n", f, strings.Join(lines, "\n"), cut)
	}
	floor := ""
	if s.FinderFloor { // a floor on candidates, never on what is shown; from the size of what is read
		kb := float64(len(diff)+code.Len()) / 1024
		n := min(int(math.Floor(math.Sqrt(kb)+1)), 10)
		floor = fmt.Sprintf("\nLook for at least %d candidates before you stop; then give only those whose cause you can quote, important or nit as each deserves.\n", n)
	}
	for i, lens := range st.Lenses {
		text, err := lensText(repo, lens)
		if err != nil {
			return err
		}
		task := fmt.Sprintf("# Lens: %s\n\n%s\n%s\n## The commits\n\nThey say what the author meant; they prove nothing: check each claim against the code.\n\n%s\n## The change\n\n```diff\n%s```\n\n## The files it changes, as they read now\n\nRead them whole: a defect anywhere in them is reported, the change's or not.\n\n%s",
			lens, strings.TrimSpace(text), floor, log, diff, code.String())
		dir := filepath.Join(runDir, "in", "parts", fmt.Sprintf("%d-%s", i+1, lens))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte(task), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// lensText is what a lens asks: the project's own (.workline/roles/reviewer/
// lenses/<lens>.md), else the role's.
func lensText(repo, lens string) (string, error) {
	for _, dir := range []string{filepath.Join(repo, ".workline", "roles", roleName()), filepath.Join(os.Getenv("WORKLINE_ROLES_DIR"), roleName())} {
		if data, err := os.ReadFile(filepath.Join(dir, "lenses", lens+".md")); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("lens %q: no lenses/%s.md in the role", lens, lens)
}

// readLenses reads what each lens answered, finds each finding's quotes
// again, tells the findings on the change from those outside it, caps them,
// and puts each important one to the judge.
func readLenses(runDir, repo string, s Settings) int {
	var st state
	if err := readJSON(filepath.Join(runDir, "in", "review-state.json"), &st); err != nil {
		return fail(err)
	}
	var c candidates
	unavailable := 0
	for _, lens := range st.Lenses {
		dir := lensDir(runDir, st, lens)
		if why, err := os.ReadFile(filepath.Join(dir, "unanswered")); err == nil {
			c.Failed = append(c.Failed, lens)
			if strings.HasPrefix(string(why), "unavailable:") {
				unavailable++
			}
			c.Logged = append(c.Logged, verdict.Finding{Rule: "lens-failed", Level: "warn",
				Message: fmt.Sprintf("the %s lens got no answer that reads (%s): what it looks for was not reviewed", lens, strings.TrimSpace(string(why)))})
			continue
		}
		if !exists(filepath.Join(dir, "answer.yaml")) {
			c.Failed = append(c.Failed, lens)
			c.Logged = append(c.Logged, verdict.Finding{Rule: "lens-failed", Level: "warn",
				Message: fmt.Sprintf("the %s lens was not asked: what it looks for was not reviewed", lens)})
			continue
		}
		answers, err := intent.Read(filepath.Join(dir, "answer.yaml"))
		if err != nil {
			c.Failed = append(c.Failed, lens)
			c.Logged = append(c.Logged, verdict.Finding{Rule: "lens-failed", Level: "warn",
				Message: fmt.Sprintf("the %s lens's answer does not read (%v): what it looks for was not reviewed", lens, err)})
			continue
		}
		for _, a := range answers {
			f, why := found(repo, st, lens, a.Value)
			if why != "" {
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-unfounded", Level: "warn",
					Message: fmt.Sprintf("dropped, from the %s lens: %q — %s", lens, f.Title, why)})
				continue
			}
			list := &c.Outside
			if f.Related {
				list = &c.Related
			}
			// Two findings on one line are merged, never one dropped: the
			// important one leads, the other said beside it.
			if dup := slices.IndexFunc(*list, func(o Finding) bool { return o.Where == f.Where }); dup >= 0 {
				o := &(*list)[dup]
				if f.Severity == "important" && o.Severity != "important" {
					f.Also, *o = append(o.Also, o.Lens+": "+o.Title), f
				} else {
					o.Also = append(o.Also, f.Lens+": "+f.Title)
				}
				continue
			}
			*list = append(*list, f)
		}
	}
	if unavailable > 0 && unavailable == len(st.Lenses) {
		v := review(st, nil, "the agent could not be reached: only the rules ran")
		v.Status = verdict.BlockedExternal
		if code := final(runDir, v); code != exitNothing {
			return code
		}
		return exitExternal
	}
	important := func(l []Finding) {
		sort.SliceStable(l, func(i, j int) bool { return l[i].Severity == "important" && l[j].Severity != "important" })
	}
	important(c.Related)
	important(c.Outside)
	if n := len(c.Related) - s.FindingsMax; s.FindingsMax > 0 && n > 0 {
		c.Related = c.Related[:s.FindingsMax]
		c.Logged = append(c.Logged, verdict.Finding{Rule: "findings-capped", Level: "warn",
			Message: fmt.Sprintf("%d more findings on the change were not checked nor shown (findings-max %d): fix these, and review again", n, s.FindingsMax)})
	}
	nits := 0
	var outside []Finding
	for _, f := range c.Outside {
		if f.Severity != "important" {
			nits++
			continue
		}
		outside = append(outside, f)
	}
	if nits > 0 {
		c.Logged = append(c.Logged, verdict.Finding{Rule: "nits-outside", Level: "warn",
			Message: fmt.Sprintf("%d nits outside the change left: only what matters there becomes an issue", nits)})
	}
	if n := len(outside) - s.IssuesMax; n > 0 {
		outside = outside[:max(s.IssuesMax, 0)]
		c.Logged = append(c.Logged, verdict.Finding{Rule: "issues-capped", Level: "warn",
			Message: fmt.Sprintf("%d more findings outside the change were not checked nor opened (issues-max %d)", n, s.IssuesMax)})
	}
	c.Outside = outside
	for i, f := range append(slices.Clone(c.Related), c.Outside...) {
		if f.Severity != "important" {
			continue
		}
		key := fmt.Sprintf("%02d", i+1)
		dir := filepath.Join(runDir, "in", "judge", key)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail(err)
		}
		q := map[string]string{"question": "Is this finding about the code true: does the code quoted, as it reads, fail the way the finding says? Answer no if it is not a defect, if the code shown handles it, or if the finding only guesses.",
			"material": material(repo, st, f)}
		data, _ := json.Marshal(q) // JSON, which YAML reads: code quoted may start a line with a tab
		if err := os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644); err != nil {
			return fail(err)
		}
		c.Asked = append(c.Asked, f.Where+"\x00"+key)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "candidates.json"), c); err != nil {
		return fail(err)
	}
	if len(c.Asked) > 0 {
		return 0 // the judge first
	}
	return settle(runDir, repo, s)
}

// found reads one finding a lens answered, and finds its quotes again: its
// cause in the file at the head of the range, or among the lines the change
// removed; its symptom, when it gives one, in its file. Related: its cause
// lies on a line the change added or removed. why says why it is dropped.
func found(repo string, st state, lens string, v any) (Finding, string) {
	var raw struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Why      string `json:"why"`
		Fix      string `json:"fix"`
		Cause    *Quote `json:"cause"`
		Symptom  *Quote `json:"symptom"`
	}
	// Through JSON: YAML would not read back a quote starting with a tab.
	data, _ := json.Marshal(v)
	if err := json.Unmarshal(data, &raw); err != nil {
		return Finding{Title: fmt.Sprint(v)}, "it does not read: " + err.Error()
	}
	f := Finding{Lens: lens, Severity: strings.ToLower(strings.TrimSpace(raw.Severity)), Title: strings.TrimSpace(raw.Title), Why: strings.TrimSpace(raw.Why), Fix: strings.TrimSpace(raw.Fix), Symptom: raw.Symptom}
	if f.Severity != "important" {
		f.Severity = "nit"
	}
	if raw.Cause == nil || strings.TrimSpace(raw.Cause.Quote) == "" || strings.TrimSpace(raw.Cause.Path) == "" {
		return f, "it quotes no cause"
	}
	f.Cause = Quote{Path: clean(raw.Cause.Path), Quote: raw.Cause.Quote}
	text, ok := fileAt(repo, st.Head, f.Cause.Path)
	var places []Place
	if ok {
		places = locate(strings.Split(text, "\n"), f.Cause.Quote)
	}
	for _, p := range places {
		for n := p.From; n <= p.To; n++ {
			if st.Change.addedAt(f.Cause.Path, n) {
				f.Where, f.Related = fmt.Sprintf("%s:%d", f.Cause.Path, p.From), true
				break
			}
		}
		if f.Related {
			break
		}
	}
	if f.Where == "" && len(places) > 0 {
		f.Where = fmt.Sprintf("%s:%d", f.Cause.Path, places[0].From)
		f.Line = norm(strings.Split(text, "\n")[places[0].From-1])
	}
	if f.Where == "" {
		var removed []string
		for _, l := range st.Change.Removed[f.Cause.Path] {
			removed = append(removed, l.Text)
		}
		if at := locate(removed, f.Cause.Quote); len(at) > 0 {
			f.Where, f.Related = fmt.Sprintf("%s:%d", f.Cause.Path, st.Change.Removed[f.Cause.Path][at[0].From-1].At), true
		}
	}
	if f.Where == "" {
		return f, fmt.Sprintf("its cause is not found in %s, as it reads at %s nor among the lines the change removed", f.Cause.Path, short(st.Head))
	}
	if f.Symptom != nil {
		f.Symptom.Path = clean(f.Symptom.Path)
		text, ok := fileAt(repo, st.Head, f.Symptom.Path)
		if !ok || len(locate(strings.Split(text, "\n"), f.Symptom.Quote)) == 0 {
			return f, fmt.Sprintf("its symptom is not found in %s as it reads", f.Symptom.Path)
		}
	}
	return f, ""
}

// material is what the judge reads of a finding: the finding, and the code
// around its cause, and its symptom.
func material(repo string, st state, f Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The finding\n\n%s: %s\n\nWhy: %s\n", f.Severity, f.Title, f.Why)
	around := func(q Quote, at int) {
		text, _ := fileAt(repo, st.Head, q.Path)
		lines := strings.Split(text, "\n")
		from, to := max(at-15, 1), min(at+15, len(lines))
		fmt.Fprintf(&b, "\n## %s, lines %d to %d\n\n```\n", q.Path, from, to)
		for n := from; n <= to; n++ {
			fmt.Fprintf(&b, "%d  %s\n", n, lines[n-1])
		}
		b.WriteString("```\n")
	}
	_, line, _ := strings.Cut(f.Where, ":")
	at := 1
	fmt.Sscan(line, &at)
	fmt.Fprintf(&b, "\nIts cause, quoted:\n\n```\n%s\n```\n", strings.TrimSpace(f.Cause.Quote))
	around(f.Cause, at)
	if f.Symptom != nil {
		text, _ := fileAt(repo, st.Head, f.Symptom.Path)
		if p := locate(strings.Split(text, "\n"), f.Symptom.Quote); len(p) > 0 {
			around(*f.Symptom, p[0].From)
		}
	}
	if diff, err := git(repo, "diff", "-U3", "--no-color", "--no-ext-diff", st.Base, st.Head, "--", f.Cause.Path); err == nil && diff != "" {
		fmt.Fprintf(&b, "\n## What the change did to %s\n\n```diff\n%s```\n", f.Cause.Path, capLines(diff, 200, "cut"))
	}
	return b.String()
}

// settle reads the judge's answers and proposes what follows: the findings
// on the change in the verdict, an issue for each verified one outside it,
// and the summary on the merge request.
func settle(runDir, repo string, s Settings) int {
	var st state
	var c candidates
	if err := readJSON(filepath.Join(runDir, "in", "review-state.json"), &st); err != nil {
		return fail(err)
	}
	if err := readJSON(filepath.Join(runDir, "in", "candidates.json"), &c); err != nil {
		return fail(err)
	}
	asked := map[string]string{}
	for _, a := range c.Asked {
		where, key, _ := strings.Cut(a, "\x00")
		asked[where] = key
	}
	floor := s.JudgeAtLeast
	if floor == "" {
		floor = "context"
	}
	verify := func(l []Finding) []Finding {
		var kept []Finding
		for _, f := range l {
			key, ok := asked[f.Where]
			if !ok {
				kept = append(kept, f) // a nit: reported as found
				continue
			}
			var a struct {
				Yes                      *bool
				Why, Model, Level, Error string
				Judge, Author            string
			}
			data, err := os.ReadFile(filepath.Join(runDir, "in", "judge", key, "answer.yaml"))
			if err == nil {
				err = yaml.Unmarshal(data, &a)
			}
			who := fmt.Sprintf("independence: %s (%s → %s)", a.Level, a.Author, a.Model)
			switch {
			case err != nil || a.Yes == nil:
				why := a.Error
				if why == "" {
					why = "the judge was not asked"
				}
				f.Verified = "not verified: " + why
			case !*a.Yes:
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-judged-no", Level: "warn",
					Message: fmt.Sprintf("dropped, from the %s lens: %q at %s — the judge: %s (%s)", f.Lens, f.Title, f.Where, a.Why, who)})
				continue
			case judge.Levels[a.Level] < judge.Levels[floor]:
				f.Verified, f.Level = fmt.Sprintf("not verified: the judge stood at %s, below %s (%s)", a.Level, floor, a.Why), a.Level
			default:
				f.Verified, f.Level = "verified, "+who, a.Level
			}
			kept = append(kept, f)
		}
		return kept
	}
	related, outside := verify(c.Related), verify(c.Outside)
	v := review(st, nil, "")
	v.Lenses, v.Change, v.Complete = st.Lenses, related, len(c.Failed) == 0
	var fallback []intent.Intention
	onForge := mergeRequest() != nil && s.ForgeWrites
	for _, f := range related {
		level := "warn"
		if s.AIFindings == "block" && f.Severity == "important" && strings.HasPrefix(f.Verified, "verified") {
			level = ""
		}
		v.Findings = append(v.Findings, verdict.Finding{Rule: f.Lens, Where: f.Where, Level: level, Message: message(f)})
	}
	for _, f := range outside {
		if !strings.HasPrefix(f.Verified, "verified") {
			c.Logged = append(c.Logged, verdict.Finding{Rule: "outside-unverified", Level: "warn",
				Message: fmt.Sprintf("outside the change, at %s, not opened as an issue (%s): %s", f.Where, f.Verified, f.Title)})
			continue
		}
		v.Outside = append(v.Outside, f)
		if s.ForgeWrites && os.Getenv("WORKLINE_FORGE") != "" {
			fallback = append(fallback, intent.Intention{Kind: "issue", Value: map[string]any{
				"title": f.Title, "at": map[string]any{"path": f.Cause.Path, "text": f.Line},
				"body": fmt.Sprintf("%s\n\nAt `%s`:\n\n```\n%s\n```\n\n%s\n\nFound outside the change it reviewed (%s), so not the author's to fix there: the reviewer's %s lens, %s.",
					f.Why, f.Where, strings.TrimSpace(f.Cause.Quote), fixLine(f), short(st.Head), f.Lens, f.Verified)}})
		} else {
			c.Logged = append(c.Logged, verdict.Finding{Rule: "outside-the-change", Level: "warn",
				Message: fmt.Sprintf("at %s, outside the change, for an issue (none opened: no forge to write to): %s — %s", f.Where, f.Title, f.Why)})
		}
	}
	v.Findings = append(v.Findings, c.Logged...)
	if v.Complete {
		v.Record = st.Record.add(st.Commits)
	}
	v.Status = status(v.Findings)
	v.Summary = summary(v, len(fallback))
	if onForge {
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: map[string]any{"sticky": SummaryKey, "body": summaryComment(v, len(fallback))}})
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
}

// Post writes the verdict from what pre settled, and keeps the record on
// this machine once every lens answered.
func Post(runDir, repo string) int {
	var v Review
	if err := readJSON(filepath.Join(runDir, "in", "review.json"), &v); err != nil {
		return fail(err)
	}
	if v.Complete {
		if err := saveLocal(repo, v.Record); err != nil {
			return fail(err)
		}
	}
	if err := writeJSON(filepath.Join(runDir, "out", "review.json"), v); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: v.Status, Summary: v.Summary, Findings: v.Findings, Final: true}); err != nil {
		return fail(err)
	}
	if v.Status == verdict.Block {
		return exitBlock
	}
	return 0
}

// review starts a review from the rules' findings.
func review(st state, extra []verdict.Finding, summary string) Review {
	v := Review{Findings: append(append(slices.Clone(st.Mechanical), extra...), st.Advisories...), Record: st.Record}
	v.Status = status(v.Findings)
	v.Summary = summary
	return v
}

// status blocks on a finding that blocks: a rule's, or a verified finding
// once ai-findings says block.
func status(fs []verdict.Finding) string {
	for _, f := range fs {
		if f.Level == "" || f.Level == "block" {
			return verdict.Block
		}
	}
	return verdict.Pass
}

func message(f Finding) string {
	sev := "Important"
	if f.Severity != "important" {
		sev = "Nit"
	}
	m := fmt.Sprintf("%s: %s — %s", sev, f.Title, f.Why)
	if f.Fix != "" {
		m += " Fix: " + f.Fix
	}
	if f.Symptom != nil {
		m += fmt.Sprintf(" (shows at %s)", f.Symptom.Path)
	}
	if len(f.Also) > 0 {
		m += " (also found on this line — " + strings.Join(f.Also, "; ") + ")"
	}
	if f.Verified != "" {
		m += " (" + f.Verified + ")"
	}
	return m
}

func fixLine(f Finding) string {
	if f.Fix == "" {
		return ""
	}
	return "What would fix it: " + f.Fix
}

func summary(v Review, issues int) string {
	if v.Status == verdict.BlockedExternal {
		return v.Summary
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("%d findings on the change", len(v.Change)))
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d issues proposed for what lies outside it", issues))
	}
	s := strings.Join(parts, ", ") + "; lenses: " + strings.Join(v.Lenses, ", ")
	if !v.Complete {
		s += "; not every lens answered: not reviewed whole"
	}
	if v.Status == verdict.Block {
		s += "; the author fixes what blocks"
	}
	return s
}

// summaryComment is the one summary the merge request gets, edited on each run,
// the record hidden in it.
func summaryComment(v Review, issues int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**workline reviewer** — %s.\n\n", v.Summary)
	if len(v.Findings) > 0 {
		b.WriteString("| | Where | Finding |\n|---|---|---|\n")
		for _, f := range v.Findings {
			where := f.Where
			if where != "" {
				where = "`" + where + "`"
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", f.Rule, where, strings.ReplaceAll(f.Message, "|", "\\|"))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("No finding on the commits reviewed.\n\n")
	}
	if issues > 0 {
		fmt.Fprintf(&b, "What lies outside this change goes to %d issues, labelled `needs-triage`: not the author's to fix here.\n\n", issues)
	}
	b.WriteString("The reviewer never approves and never changes the code: the author fixes, a person merges.\n\n")
	b.WriteString(v.Record.String())
	return b.String()
}

// final writes a verdict pre settles alone, with nothing to ask: no agent,
// nothing new, no code.
func final(runDir string, v Review) int {
	if v.Status == "" {
		v.Status = verdict.Pass
	}
	if err := writeJSON(filepath.Join(runDir, "out", "review.json"), v); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: v.Status, Summary: v.Summary, Findings: v.Findings}); err != nil {
		return fail(err)
	}
	return exitNothing
}

func lensDir(runDir string, st state, lens string) string {
	return filepath.Join(runDir, "in", "parts", fmt.Sprintf("%d-%s", slices.Index(st.Lenses, lens)+1, lens))
}

// committerSettings are the committer's, as the project sets them: a code
// it refuses in a subject is refused in a comment, and one it allows,
// allowed.
func committerSettings(repo string) (committer.Settings, error) {
	var s committer.Settings
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer")
	if err != nil {
		return s, fmt.Errorf("the committer's internal codes: %w", err)
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return s, err
	}
	data, _ := json.Marshal(r.MergedSettings(cfg))
	return s, json.Unmarshal(data, &s)
}

func settings(runDir string) (Settings, error) {
	s := Settings{Base: "main", FindingsMax: 10, IssuesMax: 3, DiffLinesMax: 1500, CodeLinesMax: 1200, LensesPerPush: 1}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("settings: %w", err)
	}
	return s, nil
}

func roleName() string {
	if r := os.Getenv("WORKLINE_ROLE"); r != "" {
		return r
	}
	return "reviewer"
}

func input(runDir, name string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
	return strings.TrimSpace(string(data))
}

func capLines(s string, max int, why string) string {
	lines := strings.SplitAfter(s, "\n")
	if max <= 0 || len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "") + "\n(" + why + ")\n"
}

func clean(p string) string {
	p = strings.TrimSpace(p)
	for _, pre := range []string{"./", "a/", "b/"} {
		p = strings.TrimPrefix(p, pre)
	}
	return filepath.ToSlash(filepath.Clean(p))
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "reviewer:", err)
	return 99
}
