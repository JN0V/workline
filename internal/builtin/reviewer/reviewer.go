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
	"path"
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
	Base            string   `json:"base"`                // where a review on a machine starts from, when no range is given
	Ignore          []string `json:"ignore"`              // files that are not code: a change touching only these asks nobody
	Lenses          []string `json:"lenses"`              // the lenses, in the order a merge request takes them in turn
	LensesPerPush   int      `json:"lenses-per-push"`     // on a merge request: lenses a push gets
	FindingsMax     int      `json:"findings-max"`        // findings on the change reported a run; the rest counted
	IssuesMax       int      `json:"issues-max"`          // issues opened a run for what lies outside the change
	DiffLinesMax    int      `json:"diff-lines-max"`      // lines of the change given to a lens
	CodeLinesMax    int      `json:"code-lines-max"`      // lines of the files changed given to a lens, all together
	CommentBlockMax int      `json:"comment-block-max"`   // lines of one comment added
	StoryWords      []string `json:"story-words"`         // what tells a bug's story in a comment
	AIFindings      Gate     `json:"ai-findings"`         // warn, until measured (#90); block: a verified important finding blocks; or one of the two by lens
	JudgeAtLeast    string   `json:"judge-at-least"`      // the independence a verification needs: context, model, provider
	ForgeWrites     bool     `json:"forge-writes"`        // on a merge request: the summary comment and the issues
	FinderFloor     bool     `json:"finder-floor"`        // ask each lens to look for a number of candidates before it stops
	Tests           []string `json:"tests"`               // the test files: what a judge reading tests is shown of them
	TestsLinesMax   int      `json:"tests-lines-max"`     // lines of those tests a judge is shown, all together
	JudgeLinesMax   int      `json:"judge-lines-max"`     // lines of code a judge is shown: the cause's function, then those it reaches; 0: 31 lines around the cause
	LensesTogether  bool     `json:"lenses-together"`     // the lenses of a run asked in one call, the change given once
	AIMaxTokens     int      `json:"ai-max-tokens"`       // what a run may spend, all calls together; the engine stops asking
	IssueLinesMax   int      `json:"issue-lines-max"`     // lines of the issues the change closes the lenses are given, all together
	TestimonyLines  int      `json:"testimony-lines-max"` // lines of what the author says (commit messages, the merge request) the lenses are given
}

// Finding is one defect a lens reported, once its quotes were found again.
type Finding struct {
	Lens     string    `json:"lens"`
	Severity string    `json:"severity"` // important or nit
	Title    string    `json:"title"`
	Why      string    `json:"why"`
	Fix      string    `json:"fix,omitempty"`
	Claim    string    `json:"claim,omitempty"` // the author's words it contradicts, found again in what they said (#126)
	Cause    Quote     `json:"cause"`
	Symptom  *Quote    `json:"symptom,omitempty"`
	Where    string    `json:"where"`          // the cause's file and line
	Related  bool      `json:"related"`        // the cause lies in the change: the author fixes it
	Line     string    `json:"line,omitempty"` // outside the change: the line the cause starts at, as it reads, which keys its issue
	Also     []Finding `json:"also,omitempty"` // other findings on the same line, grouped under this one, each judged by its own lens
	Verified string    `json:"verified,omitempty"`
	Level    string    `json:"independence,omitempty"`
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
	Together   bool              `json:"together,omitempty"` // the lenses asked in one call (lenses-together)
	Record     Record            `json:"record"`
	Advisories []verdict.Finding `json:"advisories,omitempty"`
	Issues     []ClosedIssue     `json:"issues,omitempty"`    // the issues the change says it closes, as read (#126)
	Testimony  string            `json:"testimony,omitempty"` // what the author says of it, as the lenses were given it
	Skipped    []string          `json:"skipped,omitempty"`   // lenses not asked, each with why
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
	Local    bool              `json:"local"`           // the record is kept on this machine
	Calls    []Spent           `json:"calls,omitempty"` // what each call to an agent used
}

// Spent is what one call of the review used: a lens's (or the lenses',
// asked together), or a judge's on one finding.
type Spent struct {
	For       string `json:"for"`             // the part (1-lenses, 2-edge-cases) or the judge's question (judge/03)
	Lenses    string `json:"lenses"`          // the lenses it was for
	Where     string `json:"where,omitempty"` // a judge's: the finding's place
	Model     string `json:"model,omitempty"`
	TokensIn  int    `json:"tokens-in"`
	TokensOut int    `json:"tokens-out"`
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
		return held(runDir, s, review(st, nil, fmt.Sprintf("the %d commits were reviewed already: only the rules ran", len(commits))))
	}
	// The rules first: what they block on is fixed before any agent is
	// asked, and asked again after each push while it is not.
	if status(mech) == verdict.Block {
		return held(runDir, s, review(st, nil, "the rules found what the author must fix first; the review follows once they pass"))
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
	// What the change is for, and what its author says of it (#126): the
	// merge request's text and the commit messages name the issues it
	// closes; those are read from the forge.
	mr, err := mergeRequestText(repo)
	if err != nil {
		st.Advisories = append(st.Advisories, verdict.Finding{Rule: "merge-request-unread", Level: "warn",
			Message: "the merge request's title and body could not be read (" + err.Error() + "): the lenses are given the commit messages only"})
	}
	msgs, err := git(repo, "log", "--format=%B", base+".."+head)
	if err != nil {
		return fail(err)
	}
	var ids []int
	for _, name := range s.Lenses {
		if l, err := readLens(repo, name); err == nil && l.Needs == "issue" { // one failing to read stops the run below
			ids = closes(mr, msgs)
			break
		}
	}
	var unread []verdict.Finding
	st.Issues, unread = readIssues(repo, ids, s.IssueLinesMax)
	st.Advisories = append(st.Advisories, unread...)
	if st.Testimony, err = testimony(repo, st, mr, s.TestimonyLines); err != nil {
		return fail(err)
	}
	skip, err := unasked(repo, s.Lenses, st.Issues, ids)
	if err != nil {
		return fail(err)
	}
	for _, l := range s.Lenses {
		if why, ok := skip[l]; ok {
			st.Skipped = append(st.Skipped, fmt.Sprintf("%s (%s)", l, why))
		}
	}
	st.Lenses, err = lenses(runDir, s, rec, skip)
	if err != nil {
		return fail(err)
	}
	if len(st.Lenses) == 0 {
		return final(runDir, review(st, nil, "no lens asked can review this change: not asked: "+strings.Join(st.Skipped, ", ")))
	}
	st.Together = s.LensesTogether && len(st.Lenses) > 1
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
// run is told `lenses=all`; on a merge request, the next in turn. A lens
// in skip, which has nothing to read in this change, is never asked.
func lenses(runDir string, s Settings, rec Record, skip map[string]string) ([]string, error) {
	if len(s.Lenses) == 0 {
		return nil, fmt.Errorf("settings: no lenses")
	}
	var usable []string
	for _, l := range s.Lenses {
		if _, ok := skip[l]; !ok {
			usable = append(usable, l)
		}
	}
	asked := input(runDir, "lenses")
	switch {
	case asked == "all":
		return usable, nil
	case asked != "":
		var out []string
		for _, l := range strings.Split(asked, ",") {
			l = strings.TrimSpace(l)
			if !slices.Contains(s.Lenses, l) {
				return nil, fmt.Errorf("lens %q: not one of %s", l, strings.Join(s.Lenses, ", "))
			}
			if slices.Contains(usable, l) {
				out = append(out, l)
			}
		}
		return out, nil
	case os.Getenv("WORKLINE_EVENT") != "merge-request":
		return usable, nil
	case len(usable) == 0:
		return nil, nil
	}
	// In turn over every lens, one with nothing to read passed over: the
	// same run count names the same place in the turn whatever this
	// change closes.
	n := max(1, min(s.LensesPerPush, len(usable)))
	var out []string
	for i := 0; len(out) < n && i < len(s.Lenses); i++ { // a lens named twice counts once
		if l := s.Lenses[(rec.Runs+i)%len(s.Lenses)]; slices.Contains(usable, l) && !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return out, nil
}

// unasked are the lenses this change gives nothing to read, each with why:
// one needing the issue the change closes, when it closes none it could
// read (#126).
func unasked(repo string, names []string, issues []ClosedIssue, ids []int) (map[string]string, error) {
	skip := map[string]string{}
	for _, name := range names {
		l, err := readLens(repo, name)
		if err != nil {
			return nil, err
		}
		if l.Needs == "issue" && len(issues) == 0 {
			skip[name] = "the change closes no issue"
			if len(ids) > 0 {
				skip[name] = "no issue it closes could be read"
			}
		}
	}
	return skip, nil
}

// ask writes one part a lens: the lens, the commits not reviewed yet, and
// the files they change.
func ask(runDir, repo string, s Settings, st state, files []string) error {
	// Each file once (#147): whole, the change marked in it, while the
	// files fit in code-lines-max; the others by the change's hunks. A file
	// given whole and again in the diff was a third of a lens's call.
	var code strings.Builder
	var hunked []string
	left := s.CodeLinesMax
	for _, f := range files {
		body, ok := marked(repo, st.From, st.Head, f)
		n := strings.Count(body, "\n")
		if !ok || n > left {
			hunked = append(hunked, f)
			continue
		}
		left -= n
		fmt.Fprintf(&code, "### %s\n\n```diff\n%s```\n\n", f, body)
	}
	diff := "(none: every file it changes is given whole below)\n"
	if len(hunked) > 0 {
		var err error
		if diff, err = git(repo, append([]string{"diff", "-U5", "--no-color", "--no-ext-diff", st.From, st.Head, "--"}, hunked...)...); err != nil {
			return err
		}
		diff = capLines(diff, s.DiffLinesMax, "the change is cut here: review what is above")
	}
	if code.Len() == 0 {
		code.WriteString("(none: each is too long to give whole; its change is above)\n")
	}
	floor := ""
	if s.FinderFloor { // a floor on candidates, never on what is shown; from the size of what is read
		kb := float64(len(diff)+code.Len()) / 1024
		n := min(int(math.Floor(math.Sqrt(kb)+1)), 10)
		each := ""
		if st.Together {
			each = " for each lens"
		}
		floor = fmt.Sprintf("\nLook for at least %d candidates%s before you stop; then give only those whose cause you can quote, important or nit as each deserves.\n", n, each)
	}
	read := map[string]Lens{}
	for _, lens := range st.Lenses {
		l, err := readLens(repo, lens)
		if err != nil {
			return err
		}
		read[lens] = l
	}
	// The issues closed only when a lens asked reads them: on a merge
	// request, a push asking another lens does not pay for them.
	purpose := ""
	if slices.ContainsFunc(st.Lenses, func(l string) bool { return read[l].Needs == "issue" }) {
		purpose = whatFor(st.Issues)
	}
	material := fmt.Sprintf("%s\n%s## The change, in the files too long to give whole\n\n```diff\n%s```\n\n## The files it changes, whole, as they read now\n\nThe change is marked in them: `+` a line it added, `-` a line it removed, a space a line it kept. Read them whole: a defect anywhere in them is reported, the change's or not.\n\n%s",
		st.Testimony, purpose, diff, code.String())
	write := func(name, task string) error {
		dir := filepath.Join(runDir, "in", "parts", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "task.md"), []byte(task), 0o644)
	}
	if st.Together {
		// One call, the change given once: the lenses' questions differ,
		// what they read does not (#147).
		var b strings.Builder
		fmt.Fprintf(&b, "# Lenses: %s\n\nEach lens below is a question of its own: ask each in turn, of the whole change, and name in each finding the lens that found it: `lens:` one of %s.\n",
			strings.Join(st.Lenses, ", "), strings.Join(st.Lenses, ", "))
		for _, lens := range st.Lenses {
			fmt.Fprintf(&b, "\n## Lens: %s\n\n%s\n", lens, strings.TrimSpace(read[lens].Text))
		}
		return write(partName(st, st.Lenses[0]), b.String()+floor+"\n"+material)
	}
	for _, lens := range st.Lenses {
		if err := write(partName(st, lens), fmt.Sprintf("# Lens: %s\n\n%s\n%s\n%s", lens, strings.TrimSpace(read[lens].Text), floor, material)); err != nil {
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

// judgeQuestion is what the judge of an important finding is asked when its
// lens names no question of its own: whether the code fails.
const judgeQuestion = "Is this finding about the code true: does the code quoted, as it reads, fail the way the finding says? Answer no if it is not a defect, if the code shown handles it, or if the finding only guesses."

// Lens is a lens's file: what it asks, and, in its front matter, what the
// judge of its important findings is asked and shown besides the finding,
// the code around it and the change (#223).
type Lens struct {
	Text  string `yaml:"-"`
	Needs string `yaml:"needs"` // issue: asked only of a change closing an issue, given it (#126)
	Cites string `yaml:"cites"` // claim: each finding quotes a claim of the author, found again in what they said, or is dropped
	Judge struct {
		Question string `yaml:"question"` // empty: judgeQuestion
		Reads    string `yaml:"reads"`    // tests: the tests that touch the cause's file; issue: the issues the change closes
	} `yaml:"judge"`
}

// readLens reads a lens's file, its front matter apart from its text.
func readLens(repo, name string) (Lens, error) {
	var l Lens
	text, err := lensText(repo, name)
	if err != nil {
		return l, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n") // a checkout with CRLF endings
	l.Text = text
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		head, body, closed := strings.Cut("\n"+rest+"\n", "\n---\n") // empty, or closed on the last line
		if !closed {
			return l, fmt.Errorf("lens %q: its front matter is not closed by a line `---`", name)
		}
		if err := yaml.Unmarshal([]byte(head), &l); err != nil {
			return l, fmt.Errorf("lens %q: its front matter: %w", name, err)
		}
		l.Text = body
	}
	if r := l.Judge.Reads; r != "" && r != "tests" && r != "issue" {
		return l, fmt.Errorf("lens %q: judge.reads %q: only `tests` and `issue` are known", name, r)
	}
	if l.Needs != "" && l.Needs != "issue" {
		return l, fmt.Errorf("lens %q: needs %q: only `issue` is known", name, l.Needs)
	}
	if l.Cites != "" && l.Cites != "claim" {
		return l, fmt.Errorf("lens %q: cites %q: only `claim` is known", name, l.Cites)
	}
	return l, nil
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
	read := map[string]Lens{}
	lensOf := func(name string) (Lens, error) {
		l, ok := read[name]
		if !ok {
			var err error
			if l, err = readLens(repo, name); err != nil {
				return l, err
			}
			read[name] = l
		}
		return l, nil
	}
	unavailable := 0
	var asked [][]string // the lenses of each part: one a part, or all in one
	if st.Together {
		asked = append(asked, st.Lenses)
	} else {
		for _, lens := range st.Lenses {
			asked = append(asked, []string{lens})
		}
	}
	for _, ls := range asked {
		dir := lensDir(runDir, st, ls[0])
		which := fmt.Sprintf("the %s lens", ls[0])
		if len(ls) > 1 {
			which = fmt.Sprintf("the lenses %s, asked together,", strings.Join(ls, ", "))
		}
		failed := func(message string) {
			c.Failed = append(c.Failed, ls...)
			c.Logged = append(c.Logged, verdict.Finding{Rule: "lens-failed", Level: "warn", Message: message})
		}
		if why, err := os.ReadFile(filepath.Join(dir, "unanswered")); err == nil {
			if strings.HasPrefix(string(why), "unavailable:") {
				unavailable += len(ls)
			}
			failed(fmt.Sprintf("%s got no answer that reads (%s): what it looks for was not reviewed", which, strings.TrimSpace(string(why))))
			continue
		}
		if !exists(filepath.Join(dir, "answer.yaml")) {
			failed(fmt.Sprintf("%s was not asked: what it looks for was not reviewed", which))
			continue
		}
		answers, err := intent.Read(filepath.Join(dir, "answer.yaml"))
		if err != nil {
			failed(fmt.Sprintf("%s: its answer does not read (%v): what it looks for was not reviewed", which, err))
			continue
		}
		for _, a := range answers {
			lens := ls[0]
			if len(ls) > 1 {
				// Asked together, a finding names its lens. One naming none
				// is read as the first lens's, said: dropping it lost a whole
				// call's findings (tried.md, #147). One naming a lens not
				// asked is dropped, said.
				named, title := lensNamed(a.Value)
				switch {
				case named == "":
					c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-lens-unnamed", Level: "warn",
						Message: fmt.Sprintf("%q names no lens: read as the %s lens's, and judged by its question", title, ls[0])})
				case !slices.Contains(ls, named):
					c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-unfounded", Level: "warn",
						Message: fmt.Sprintf("dropped: %q names the lens %q, not one asked (%s)", title, named, strings.Join(ls, ", "))})
					continue
				default:
					lens = named
				}
			}
			l, err := lensOf(lens)
			if err != nil {
				return fail(err)
			}
			f, why := found(repo, st, lens, l.Cites == "claim", a.Value)
			if why != "" {
				c.Logged = append(c.Logged, verdict.Finding{Rule: "finding-unfounded", Level: "warn",
					Message: fmt.Sprintf("dropped, from the %s lens: %q — %s", lens, f.Title, why)})
				continue
			}
			list := &c.Outside
			if f.Related {
				list = &c.Related
			}
			// Two findings on one line are grouped, never one dropped: each
			// is judged by its own lens, the group only shows them together
			// (#229); the one ahead leads, the others beside it.
			if dup := slices.IndexFunc(*list, func(o Finding) bool { return o.Where == f.Where }); dup >= 0 {
				(*list)[dup] = group(s, append(members((*list)[dup]), f))
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
	var judged []Finding
	for _, g := range append(slices.Clone(c.Related), c.Outside...) {
		judged = append(judged, members(g)...)
	}
	n := 0
	for _, f := range judged {
		if f.Severity != "important" {
			continue
		}
		l, err := lensOf(f.Lens)
		if err != nil {
			return fail(err)
		}
		question := l.Judge.Question
		if question == "" {
			question = judgeQuestion
		}
		m := material(repo, st, s, f)
		switch {
		case l.Judge.Reads == "tests":
			m += testsTouching(repo, st.Head, f.Cause.Path, s.Tests, s.TestsLinesMax)
		case l.Judge.Reads == "issue" && issueOf(st, f.Cause.Path) == nil: // one quoting an issue is shown it already
			m += "\n" + whatFor(st.Issues)
		}
		n++
		key := fmt.Sprintf("%02d", n)
		dir := filepath.Join(runDir, "in", "judge", key)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail(err)
		}
		q := map[string]string{"question": strings.Join(strings.Fields(question), " "), "material": m}
		data, _ := json.Marshal(q) // JSON, which YAML reads: code quoted may start a line with a tab
		if err := os.WriteFile(filepath.Join(dir, "question.yaml"), data, 0o644); err != nil {
			return fail(err)
		}
		c.Asked = append(c.Asked, JudgeKey(f)+"\x00"+key)
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
// lies on a line the change added or removed, or on a kept line beside a
// removal. why says why it is dropped.
func found(repo string, st state, lens string, cites bool, v any) (Finding, string) {
	var raw struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Why      string `json:"why"`
		Fix      string `json:"fix"`
		Claim    string `json:"claim"`
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
	// A lens checking the author's claims quotes the claim: found again in
	// what the author said, or the finding is dropped (#126).
	if cites {
		f.Claim = strings.TrimSpace(raw.Claim)
		if f.Claim == "" {
			return f, "it quotes no claim of the author"
		}
		if len(locate(saidByAuthor(st.Testimony), f.Claim)) == 0 {
			return f, "its claim is not found in what the author said"
		}
	}
	f.Cause = Quote{Path: clean(raw.Cause.Path), Quote: raw.Cause.Quote}
	// A cause quoted from the issue the change closes: what it asks and the
	// change does not do, the author's to do (#126).
	if strings.HasPrefix(f.Cause.Path, "#") {
		is := issueOf(st, f.Cause.Path)
		if is == nil {
			return f, fmt.Sprintf("its cause names %s, not an issue the change closes", f.Cause.Path)
		}
		if len(locate(strings.Split(is.Text, "\n"), f.Cause.Quote)) == 0 {
			return f, fmt.Sprintf("its cause is not found in %s, as the lenses were given it", f.Cause.Path)
		}
		f.Where, f.Related, f.Symptom = f.Cause.Path, true, nil
		return f, ""
	}
	text, ok := fileAt(repo, st.Head, f.Cause.Path)
	var places []Place
	if ok {
		places = locate(strings.Split(text, "\n"), f.Cause.Quote)
	}
	// The change's: a line it added, then a kept line beside a removal, then
	// a line it removed, even one also kept elsewhere (#224).
	for _, in := range []func(string, int) bool{st.Change.addedAt, st.Change.exposedAt} {
		for _, p := range places {
			for n := p.From; n <= p.To && f.Where == ""; n++ {
				if in(f.Cause.Path, n) {
					f.Where, f.Related = fmt.Sprintf("%s:%d", f.Cause.Path, p.From), true
				}
			}
		}
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
	if f.Where == "" && len(places) > 0 {
		f.Where = fmt.Sprintf("%s:%d", f.Cause.Path, places[0].From)
		f.Line = norm(strings.Split(text, "\n")[places[0].From-1])
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

// material is what the judge reads of a finding: the finding, the code it
// stands on (reach), and what the change did near its cause.
func material(repo string, st state, s Settings, f Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The finding\n\n%s: %s\n\nWhy: %s\n", f.Severity, f.Title, f.Why)
	_, line, _ := strings.Cut(f.Where, ":")
	at := 1
	fmt.Sscan(line, &at)
	fmt.Fprintf(&b, "\nIts cause, quoted:\n\n```\n%s\n```\n", strings.TrimSpace(f.Cause.Quote))
	if f.Claim != "" {
		fmt.Fprintf(&b, "\nThe author's claim it contradicts, quoted from what they said:\n\n> %s\n", strings.ReplaceAll(f.Claim, "\n", "\n> "))
	}
	// A cause quoted from an issue: the issue, and the change it is
	// compared with: the whole range, base..head, commits reviewed before
	// included, so a part one of them did is not missing; up to code-lines-max.
	if is := issueOf(st, f.Cause.Path); is != nil {
		b.WriteString("\n" + whatFor([]ClosedIssue{*is}))
		if diff, err := git(repo, "diff", "-U1", "--no-color", "--no-ext-diff", st.Base, st.Head, "--"); err == nil {
			fmt.Fprintf(&b, "\n## The whole change: every commit of the merge request, those reviewed before too\n\n```diff\n%s```\n", capLines(diff, s.CodeLinesMax, "the change is cut here, past code-lines-max"))
		}
		return b.String()
	}
	symptomAt := 0
	if f.Symptom != nil {
		text, _ := fileAt(repo, st.Head, f.Symptom.Path)
		if p := locate(strings.Split(text, "\n"), f.Symptom.Quote); len(p) > 0 {
			symptomAt = p[0].From
		}
	}
	b.WriteString(reach(repo, st.Head, s.Tests, f, at, symptomAt, s.JudgeLinesMax))
	if diff, err := git(repo, "diff", "-U3", "--no-color", "--no-ext-diff", st.Base, st.Head, "--", f.Cause.Path); err == nil && diff != "" {
		near, far := hunksNear(diff, at, nearCause)
		fmt.Fprintf(&b, "\n## What the change did to %s, near the cause\n\n", f.Cause.Path)
		if near != "" {
			fmt.Fprintf(&b, "```diff\n%s```\n", capLines(near, 200, "cut"))
		} else {
			fmt.Fprintf(&b, "Nothing within %d lines of it.\n", nearCause)
		}
		if far > 0 {
			fmt.Fprintf(&b, "\n(%d %s further from the cause not shown.)\n", far, plural(far, "hunk", "hunks"))
		}
	}
	return b.String()
}

// nearCause is how far from a finding's cause, in lines, a hunk of the
// change is shown to its judge: what the change did there, not the whole
// file's diff at every call (#147).
const nearCause = 40

// hunksNear keeps of one file's diff its header and the hunks whose lines,
// at the head, come within `within` lines of line at; far counts the others.
func hunksNear(diff string, at, within int) (near string, far int) {
	var head strings.Builder
	var hunks []string
	for _, l := range strings.SplitAfter(diff, "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			hunks = append(hunks, l)
		case len(hunks) == 0:
			head.WriteString(l)
		default:
			hunks[len(hunks)-1] += l
		}
	}
	var kept strings.Builder
	for _, h := range hunks {
		from, n := 0, 1 // "@@ -a,b +from,n @@"
		if _, plus, ok := strings.Cut(h, " +"); ok {
			r, _, _ := strings.Cut(plus, " ")
			first, count, has := strings.Cut(r, ",")
			fmt.Sscan(first, &from)
			if has {
				fmt.Sscan(count, &n)
			}
		}
		if at+within < from || at-within > from+max(n, 1)-1 {
			far++
			continue
		}
		kept.WriteString(h)
	}
	if kept.Len() == 0 {
		return "", far
	}
	return head.String() + kept.String(), far
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// testsTouching is what a judge reading tests is shown besides: the test
// files at the head (the `tests` setting) that touch the cause's file — in
// its folder, or naming it as a word — whole, its folder's first, up to limit
// lines all together; those left out are named.
func testsTouching(repo, head, file string, patterns []string, limit int) string {
	var b strings.Builder
	dir, stem := path.Dir(file), strings.TrimSuffix(path.Base(file), path.Ext(file))
	fmt.Fprintf(&b, "\n## The tests that touch %s\n\nThe test files in %s/ and those naming `%s`, as they read at %s.\n", file, dir, stem, short(head))
	if len(patterns) == 0 {
		b.WriteString("\nNone shown: no `tests` setting says which files are tests.\n")
		return b.String()
	}
	var near, far []string
	if out, err := git(repo, "ls-tree", "-r", "--name-only", head, "--", dir+"/"); err == nil {
		for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
			if f != "" && path.Dir(f) == dir && pathglob.Any(patterns, f) {
				near = append(near, f)
			}
		}
	}
	if out, err := git(repo, "grep", "-l", "-w", "-F", "-e", stem, head, "--"); err == nil { // none: git grep fails
		for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
			f = strings.TrimPrefix(f, head+":")
			if f != "" && path.Dir(f) != dir && pathglob.Any(patterns, f) {
				far = append(far, f)
			}
		}
	}
	files := append(near, far...)
	if len(files) == 0 {
		b.WriteString("\nNo test file touches it.\n")
		return b.String()
	}
	left := limit
	var cut []string
	for _, f := range files {
		text, ok := fileAt(repo, head, f)
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if !ok || len(lines) > left {
			cut = append(cut, f)
			continue
		}
		left -= len(lines)
		fmt.Fprintf(&b, "\n### %s\n\n```\n%s\n```\n", f, strings.Join(lines, "\n"))
	}
	if len(cut) > 0 {
		fmt.Fprintf(&b, "\nNot shown, past %d lines: %s. A test in them may exercise it.\n", limit, strings.Join(cut, ", "))
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
	keys := map[string]string{}
	for _, a := range c.Asked {
		i := strings.LastIndex(a, "\x00")
		keys[a[:i]] = a[i+1:]
	}
	floor := s.JudgeAtLeast
	if floor == "" {
		floor = "context"
	}
	unjudged := 0 // findings the judge was not asked of, the run's tokens spent
	verify := func(l []Finding) []Finding {
		var kept []Finding
		for _, f := range l {
			key, ok := keys[JudgeKey(f)]
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
			if strings.Contains(a.Error, tokensSpent) {
				unjudged++
			}
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
	// Each finding of a group is judged apart; what its judge keeps is
	// grouped again, the one ahead leading (#229).
	regroup := func(l []Finding) []Finding {
		var kept []Finding
		for _, g := range l {
			if ms := verify(members(g)); len(ms) > 0 {
				kept = append(kept, group(s, ms))
			}
		}
		return kept
	}
	related, outside := regroup(c.Related), regroup(c.Outside)
	v := review(st, nil, "")
	v.Lenses, v.Change, v.Complete = st.Lenses, related, len(c.Failed) == 0
	var fallback []intent.Intention
	onForge := mergeRequest() != nil && s.ForgeWrites
	for _, f := range related {
		level := "warn"
		if s.AIFindings.Blocks(f.Lens) && f.Severity == "important" && strings.HasPrefix(f.Verified, "verified") {
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
	// A finding left unjudged for want of tokens leaves the review not
	// whole: its commits are reviewed again, the judge asked then (#147).
	if unjudged > 0 {
		v.Complete = false
		c.Logged = append(c.Logged, verdict.Finding{Rule: "review-not-whole", Level: "warn",
			Message: fmt.Sprintf("%d %s not judged, the run having spent its ai-max-tokens (%d): the commits are not recorded as reviewed; review again, with fewer commits or a larger ai-max-tokens", unjudged, plural(unjudged, "finding", "findings"), s.AIMaxTokens)})
	}
	v.Findings = append(v.Findings, c.Logged...)
	if v.Complete {
		v.Record = st.Record.add(st.Commits)
	}
	v.Calls = spent(runDir, st, c)
	v.Status = status(v.Findings)
	v.Summary = summary(v, len(fallback), len(c.Failed), unjudged)
	if len(st.Skipped) > 0 {
		v.Summary += "; not asked: " + strings.Join(st.Skipped, ", ")
	}
	v.Summary += tokensLine(v.Calls, s.AIMaxTokens)
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
// once ai-findings says block for its lens.
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
	if f.Claim != "" {
		m += fmt.Sprintf(" (the author: %q)", f.Claim)
	}
	if len(f.Also) > 0 {
		var also []string
		for _, o := range f.Also {
			a := o.Lens + ": " + o.Title
			if o.Claim != "" {
				a += fmt.Sprintf(" (the author: %q)", o.Claim)
			}
			if o.Verified != "" {
				a += " (" + o.Verified + ")"
			}
			also = append(also, a)
		}
		m += " (also found on this line — " + strings.Join(also, "; ") + ")"
	}
	if f.Verified != "" {
		m += " (" + f.Verified + ")"
	}
	return m
}

// members are the findings a group holds: the one leading, then the others.
func members(g Finding) []Finding {
	lead := g
	lead.Also = nil
	return append([]Finding{lead}, g.Also...)
}

// group puts findings on one line together: the one ahead leads, the others
// beside it in the order they came.
func group(s Settings, ms []Finding) Finding {
	lead := 0
	for i := range ms {
		if ahead(s, ms[i], ms[lead]) {
			lead = i
		}
	}
	g := ms[lead]
	g.Also = slices.Delete(slices.Clone(ms), lead, lead+1)
	return g
}

// ahead tells whether a finding leads another on their line: a verified
// one first, then an important one from a blocking lens (ai-findings,
// #222), then an important one; of two alike, the first found.
func ahead(s Settings, a, b Finding) bool {
	rank := func(f Finding) int {
		r := 0
		if f.Severity == "important" {
			r++
			if s.AIFindings.Blocks(f.Lens) {
				r++
			}
		}
		if strings.HasPrefix(f.Verified, "verified") {
			r += 4
		}
		return r
	}
	return rank(a) > rank(b)
}

// JudgeKey keys a finding's question to the judge: its line, lens,
// severity, title and cause; two findings alike in all read one answer.
func JudgeKey(f Finding) string {
	return strings.Join([]string{f.Where, f.Lens, f.Severity, f.Title, f.Cause.Quote}, "\x00")
}

func fixLine(f Finding) string {
	if f.Fix == "" {
		return ""
	}
	return "What would fix it: " + f.Fix
}

func summary(v Review, issues, failed, unjudged int) string {
	if v.Status == verdict.BlockedExternal {
		return v.Summary
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("%d findings on the change", len(v.Change)))
	if issues > 0 {
		parts = append(parts, fmt.Sprintf("%d issues proposed for what lies outside it", issues))
	}
	s := strings.Join(parts, ", ") + "; lenses: " + strings.Join(v.Lenses, ", ")
	switch {
	case v.Complete:
	case failed > 0:
		s += "; not every lens answered: not reviewed whole"
	case unjudged > 0:
		s += "; not every finding judged, the tokens spent: not reviewed whole"
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
		// What holds the merge request first, said so: the author reads it
		// before the warnings (#226).
		rows := slices.Clone(v.Findings)
		blocks := func(f verdict.Finding) bool { return status([]verdict.Finding{f}) == verdict.Block }
		sort.SliceStable(rows, func(i, j int) bool { return blocks(rows[i]) && !blocks(rows[j]) })
		b.WriteString("| | | Where | Finding |\n|---|---|---|---|\n")
		for _, f := range rows {
			where := f.Where
			if where != "" {
				where = "`" + where + "`"
			}
			holds := "warns"
			if blocks(f) {
				holds = "**blocks**"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", holds, f.Rule, where, strings.ReplaceAll(f.Message, "|", "\\|"))
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

// held settles a run the rules alone decide: blocked on a merge request it
// may write to, the summary comment still says why, the record left as it
// was, no lens having run (#226); post then blocks. Otherwise, final.
func held(runDir string, s Settings, v Review) int {
	if v.Status != verdict.Block || mergeRequest() == nil || !s.ForgeWrites {
		return final(runDir, v)
	}
	fallback := []intent.Intention{{Kind: "comment", Value: map[string]any{"sticky": SummaryKey, "body": summaryComment(v, 0)}}}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "review.json"), v); err != nil {
		return fail(err)
	}
	return 0
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
	return filepath.Join(runDir, "in", "parts", partName(st, lens))
}

// partName is the part a lens is asked in: its own, or, the lenses asked
// together, the one part holding them all.
func partName(st state, lens string) string {
	if st.Together {
		return "1-lenses"
	}
	return fmt.Sprintf("%d-%s", slices.Index(st.Lenses, lens)+1, lens)
}

// lensNamed is the lens a finding names, written as the lenses are named,
// and its title, to say which was dropped.
func lensNamed(v any) (lens, title string) {
	m, _ := v.(map[string]any)
	lens, _ = m["lens"].(string)
	title, _ = m["title"].(string)
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(lens)), " ", "-"), title
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
	s := Settings{Base: "main", FindingsMax: 10, IssuesMax: 3, DiffLinesMax: 1500, CodeLinesMax: 600, TestsLinesMax: 300, JudgeLinesMax: 200, LensesPerPush: 1, IssueLinesMax: 80, TestimonyLines: 80}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("settings: %w", err)
	}
	for lens := range s.AIFindings.ByLens {
		if !slices.Contains(s.Lenses, lens) {
			return s, fmt.Errorf("settings: ai-findings names %q, not one of the lenses %v", lens, s.Lenses)
		}
	}
	return s, nil
}

// Gate is what a verified important finding does: warn or block, one value
// for every lens, or a map by lens where a lens not named warns (#222).
type Gate struct {
	All    string            // the one value, when no map is given
	ByLens map[string]string // by lens, when a map is given
}

// Blocks says whether a verified important finding of the lens blocks.
func (g Gate) Blocks(lens string) bool {
	if g.ByLens != nil {
		return g.ByLens[lens] == "block"
	}
	return g.All == "block"
}

// UnmarshalJSON reads warn, block, or a map of the two by lens; anything
// else is refused, never read as warn (principle 12).
func (g *Gate) UnmarshalJSON(data []byte) error {
	check := func(v string) error {
		if v != "warn" && v != "block" {
			return fmt.Errorf("ai-findings: %q is neither warn nor block", v)
		}
		return nil
	}
	if string(data) == "null" {
		*g = Gate{}
		return nil
	}
	var one string
	if json.Unmarshal(data, &one) == nil {
		*g = Gate{All: one}
		return check(one)
	}
	var by map[string]string
	if err := json.Unmarshal(data, &by); err != nil {
		return fmt.Errorf("ai-findings: warn, block, or a map of the two by lens: %w", err)
	}
	for lens, v := range by {
		if err := check(v); err != nil {
			return fmt.Errorf("%w, for the %s lens", err, lens)
		}
	}
	if by == nil {
		by = map[string]string{}
	}
	*g = Gate{ByLens: by}
	return nil
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
