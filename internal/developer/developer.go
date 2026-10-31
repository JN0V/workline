// Package developer is the developer role's built-in steps (ADR-0054): it
// takes a ready issue to a pull request. The pick: the first ready issue
// nobody claimed. The bounds: its patches held at apply to what the issue
// lets it touch (bounds.go). The proof: the project's tests on its code,
// ready when green, else a Draft (proof.go). The pull request: on the
// tester's Draft, code and tests apart, its record on it (pullrequest.go).
package developer

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing  = 10 // pre: no question for the agent, its verdict final
	exitExternal = 3  // an outside service failed
	pickFile     = "pick.json"
	// ActMark opens the hidden line of the developer's record on an issue
	// it took, what the auditor reads (ADR-0046).
	ActMark = "workline:developer-act"
)

// Settings are the developer's settings, the project's over the role's.
type Settings struct {
	// ClaimLabel is the label that says the developer took an issue: the
	// forge's assignee where it cannot assign workline's App (GitHub).
	ClaimLabel string
	// ClaimHours: a claim with no pull request after them is let go, its
	// label off, why said on the issue (ADR-0059, decision 1).
	ClaimHours int
	// MaxTokens, MaxTurns and MaxSeconds are the engine's ai-max-tokens,
	// ai-max-turns and ai-max-seconds: the run's agent reaching one is a
	// cap reached (ADR-0056). ProofSeconds bound the test command's run.
	MaxTokens    int
	MaxTurns     int
	MaxSeconds   int
	ProofSeconds int
	// ProofCommand is the proof's command on the touched folders, holding
	// `{packages}` (#703); none: the tester's test-command, whole.
	ProofCommand string
	// Rounds are the review rounds it runs on its pull request; past
	// them, its issue blocked (ADR-0059).
	Rounds int
}

// SettingsFrom reads the settings.
func SettingsFrom(m map[string]any) Settings {
	s := Settings{ClaimLabel: "workline:developing", ClaimHours: 24, Rounds: 5}
	if n := number(m["claim-hours"]); n > 0 {
		s.ClaimHours = n
	}
	if n := number(m["review-rounds"]); n > 0 {
		s.Rounds = n
	}
	if v, ok := m["claim-label"].(string); ok && strings.TrimSpace(v) != "" {
		s.ClaimLabel = strings.TrimSpace(v)
	}
	s.MaxTokens, s.ProofSeconds = number(m["ai-max-tokens"]), number(m["proof-seconds"])
	s.MaxTurns, s.MaxSeconds = number(m["ai-max-turns"]), number(m["ai-max-seconds"])
	if v, ok := m["proof-command"].(string); ok {
		s.ProofCommand = strings.TrimSpace(v)
	}
	return s
}

// Passed is a ready issue the pick passed over, and why.
type Passed struct {
	Issue int    `json:"issue"`
	Why   string `json:"why"`
	// next is what the issue waits for when the developer's own pick
	// holds it, said on the issue (ADR-0059, decision 1); none for a
	// person's hold, or an open pull request that says it already.
	next string
}

// Act is the record of a pick, hidden in the comment it leaves on the
// issue: the commit the run read, the issue taken, the ready issues before
// it passed over and why.
type Act struct {
	Act    string   `json:"act"`
	At     string   `json:"at,omitempty"`
	Issue  int      `json:"issue"`
	Passed []Passed `json:"passed,omitempty"`
	// Refused: each file its agent's patches reached past its bounds, and
	// the bound (ADR-0054, step 2).
	Refused []Refusal `json:"refused,omitempty"`
	// Unread: its agent's words that did not read, as they came, for its
	// pull request's body (agentWords); never kept in the record.
	Unread []string `json:"-"`
	// Proof: what the project's tests said on its code; Caps: the bounds
	// on the run it reached (step 3).
	Proof *Proof `json:"proof,omitempty"`
	Caps  []Cap  `json:"caps,omitempty"`
	// Turns, Tokens, Cached and Tools: what its agent used, as it
	// reported them (ADR-0056, steps 4, 5 and 7): its fresh tokens, what
	// ai-max-tokens holds, its cache reads apart, the times it used each
	// tool; 0 or none, not reported.
	Turns  int            `json:"turns,omitempty"`
	Tokens int            `json:"tokens,omitempty"`
	Cached int            `json:"cached,omitempty"`
	Tools  map[string]int `json:"tools,omitempty"`
	// AgentFailed: why its agent's call failed, its time aside; what it
	// edited is proven all the same, its pull request a Draft.
	AgentFailed string `json:"agent-failed,omitempty"`
	// NoCode: its agent wrote no code within its bounds, so no pull
	// request; the claim waits to be let go (ADR-0059, decision 1).
	NoCode bool `json:"no-code,omitempty"`
	// From: the tester's Draft's commit its code starts from, the
	// acceptance tests on it, those for the issue in Acceptance; Code and
	// Tests: the files its code and its tests wrote, in commits apart
	// (step 4).
	From       string   `json:"from,omitempty"`
	Acceptance []string `json:"acceptance,omitempty"`
	Code       []string `json:"code,omitempty"`
	Tests      []string `json:"tests,omitempty"`
	// The review loop (step 6): the rounds run, the reviewer's runs and
	// a person's comments read already, the findings asked before (to
	// tell one still there), the five whys run, a person handed it.
	Round      int      `json:"round,omitempty"`
	ReviewRuns int      `json:"review-runs,omitempty"`
	Heard      []string `json:"heard,omitempty"`
	Seen       []string `json:"seen,omitempty"`
	Rounds     []Round  `json:"rounds,omitempty"`
	Whys       bool     `json:"whys,omitempty"`
	Handed     bool     `json:"handed,omitempty"`
	// Board: the card its claim moved to Doing on the forge's board
	// (ADR-0057); none, no board or no Doing column.
	Board *board.Move `json:"board,omitempty"`
}

// number is a setting's whole number, 0 when not one.
func number(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

var actData = regexp.MustCompile(`<!-- ` + ActMark + ` (\{.*\}) -->`)

// Line is the hidden line a comment keeps the record in.
func (a Act) Line() string {
	data, _ := json.Marshal(a)
	return fmt.Sprintf("<!-- %s %s -->", ActMark, data)
}

// ReadAct reads a pick's record from a comment of the developer's.
func ReadAct(body string) (Act, bool) {
	m := actData.FindStringSubmatch(body)
	var a Act
	if m == nil || json.Unmarshal([]byte(m[1]), &a) != nil || a.Act == "" {
		return Act{}, false
	}
	return a, true
}

// scopePaths are the paths an issue's Scope names, in backticks: a word
// holding a slash or a dot, its line taken off (work.ScopeWords).
func scopePaths(body string) []string {
	var out []string
	for _, w := range work.ScopeWords(work.ByMeaning(body)[work.Scope].Text) {
		p, ok := scopeEntry(w)
		if ok && strings.ContainsAny(w, "/.") && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// overlap is the first path of a under or over one of b's, "" when none.
func overlap(a, b []string) string {
	for _, p := range a {
		for _, q := range b {
			if p == q || strings.HasPrefix(p, q+"/") || strings.HasPrefix(q, p+"/") {
				return p
			}
		}
	}
	return ""
}

// pick is the issue the developer takes: the first, in the backlog's
// order, offered to whoever builds (ready, waiting on no open issue, no
// parts), assigned to nobody, not claimed already, no open merge request
// closing it, its Scope's paths clear of the issues the developer holds.
// testsFirst, where the project routes the tester: an issue whose
// Validation names scenarios waits for the tester's open Draft of its
// tests (ADR-0055), or the kind of proof it recorded instead (proofs,
// ADR-0059). only, when not 0, is the one issue looked at.
// letGo are the claims this run lets go: held no more, taken at a next
// run. passed are the issues offered it passed over, why each.
func pick(issues []forge.Issue, mrs []forge.MergeRequest, s Settings, only int, testsFirst bool, letGo map[int]bool, proofs map[int]tester.Proof) (id int, passed []Passed) {
	open := map[int]bool{}
	for _, is := range issues {
		open[is.ID] = true
	}
	closer, tested := map[int]int{}, map[int]bool{}
	for _, m := range mrs {
		if n := testerIssue(m.Branch); n != 0 {
			tested[n] = true
		}
		for _, n := range forge.Closes(m.Title, m.Body) {
			if _, ok := closer[n]; !ok {
				closer[n] = m.ID
			}
		}
	}
	held := map[int][]string{} // the issues the developer holds: their Scope's paths
	for _, is := range issues {
		if slices.Contains(is.Labels, s.ClaimLabel) && !letGo[is.ID] {
			held[is.ID] = scopePaths(is.Body)
		}
	}
	backlog.Order(issues)
	for _, is := range issues {
		if only != 0 && is.ID != only || !backlog.Offered(is, open) {
			continue
		}
		why, next := "", ""
		switch {
		case len(is.Assignees) > 0:
			why = "assigned to " + strings.Join(is.Assignees, ", ")
		case letGo[is.ID]:
			why = "its claim was let go in this run: taken again at a next one"
		case slices.Contains(is.Labels, s.ClaimLabel):
			why = "taken already, labelled " + s.ClaimLabel
		case closer[is.ID] != 0:
			why = fmt.Sprintf("the open merge request %d closes it", closer[is.ID])
		case testsFirst && !tested[is.ID] && proofs[is.ID].Kind == "" && len(tester.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) > 0:
			why = "its acceptance tests are not written yet: it waits for the tester's Draft"
			next = "the tester writes them, as a Draft of its own; the developer takes this issue at a run after that Draft is open"
		default:
			mine := scopePaths(is.Body)
			for _, n := range slices.Sorted(maps.Keys(held)) {
				if p := overlap(mine, held[n]); p != "" {
					why = fmt.Sprintf("its %s's `%s` overlaps #%d's, taken", work.Name(work.Scope), p, n)
					next = fmt.Sprintf("#%d merged or let go; the developer looks at this issue again at its next run", n)
					break
				}
			}
		}
		if why == "" {
			return is.ID, passed
		}
		passed = append(passed, Passed{Issue: is.ID, Why: why, next: next})
	}
	return 0, passed
}

// Pre picks the issue: with no agent, it names it and stops, nothing
// claimed; with one, it leaves the claim to the engine — the label and the
// comment holding the record — reads the bounds, and asks the agent for
// the code (ADR-0054, build plan).
func Pre(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	// On a pull request's event, its bounds checked, no agent (ADR-0058).
	if os.Getenv("WORKLINE_EVENT") == "merge-request" {
		_, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
		id, _ := strconv.Atoi(num)
		return boundsPre(runDir, repo, id)
	}
	// On its own pull request, a review round (step 6).
	if kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":"); kind == "merge-request" {
		id, _ := strconv.Atoi(num)
		return reviewPre(runDir, repo, s, id)
	}
	// Its code goes to a pull request of its own, never the checkout's
	// branch: a line that opens none (`workline route` with no
	// --open-merge-request) takes nothing.
	if n, ok := os.LookupEnv("WORKLINE_OPEN_MERGE_REQUESTS"); !ok || n == "" {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no pull request can be opened",
			Findings: []verdict.Finding{{Rule: "no-merge-request", Message: "the developer opens its code as a pull request of its own: route it with --open-merge-request and a forge, or run `workline develop`"}}})
	}
	only := 0
	if kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":"); kind == "issue" {
		only, _ = strconv.Atoi(num)
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the issues from",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the developer takes a ready issue from a forge; " + forge.Missing}}})
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the forge lists no issues",
			Findings: []verdict.Finding{{Rule: "no-backlog", Message: "the developer picks a ready issue from the forge's open issues, which this forge does not list"}}})
	}
	issues, err := b.Issues()
	if err != nil {
		return external(err)
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return external(err)
	}
	line, err := routing.Load(repo)
	if err != nil {
		return fail(err)
	}
	// A claim no pull request followed is let go first, said on its issue;
	// it is not taken again in the same run (ADR-0059, decision 1).
	// Its card leaves Doing with it, to the board's ready column
	// (ADR-0057), read only when a claim is there to let go.
	var card *board.Move
	if slices.ContainsFunc(issues, func(is forge.Issue) bool { return slices.Contains(is.Labels, s.ClaimLabel) }) {
		if cfg, err := role.LoadProjectConfig(repo); err == nil {
			if card, err = board.To(f, "ready", cfg.Board); err != nil {
				fmt.Fprintf(os.Stderr, "workline: the forge's board could not be read, no card moved: %s\n", forge.Said(err))
			}
		}
	}
	said, findings, err := stale(b, issues, mrs, s, time.Now(), card)
	if err != nil {
		return external(err)
	}
	letGo := map[int]bool{}
	for _, in := range said {
		if m, _ := in.Value.(map[string]any); in.Kind == "label" {
			letGo[number(m["issue"])] = true
		}
	}
	// The kinds of proof the tester recorded in place of a Draft
	// (ADR-0059, decision 4): accepted as a Draft is.
	proofs, err := proofsOf(b, issues, mrs, line.Runs("tester"))
	if err != nil {
		return external(err)
	}
	id, passed := pick(issues, mrs, s, only, line.Runs("tester"), letGo, proofs)
	for _, p := range passed {
		findings = append(findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	// What waits on the developer's own pick is said on each issue.
	said = append(said, waits(passed)...)
	if id == 0 {
		summary := "no ready issue waits for the developer"
		if only != 0 {
			summary = fmt.Sprintf("#%d is not one the developer may take", only)
			if len(passed) == 0 {
				findings = append(findings, verdict.Finding{Rule: "issue-not-offered", Level: "info", Where: fmt.Sprintf("#%d", only),
					Message: "not open and " + backlog.LabelReady + ", or it waits on an open issue, or it has parts"})
			}
		}
		return say(runDir, verdict.Verdict{Status: verdict.Pass, Summary: summary, Findings: findings}, said)
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		findings = append(findings, verdict.Finding{Rule: "no-agent", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: "with no agent, the developer names the issue it would take, and claims and opens nothing (principle 5)"})
		return say(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("would take #%d: no agent, nothing claimed", id), Findings: findings}, said)
	}
	at, _ := gitOut(repo, "rev-parse", "HEAD")
	act := Act{Act: "pick", At: at, Issue: id, Passed: passed}
	// The claim moves its card to Doing (ADR-0057); a board out of reach
	// is said, and changes nothing else.
	if cfg, err := role.LoadProjectConfig(repo); err == nil {
		mv, err := board.To(f, "doing", cfg.Board)
		if err != nil {
			fmt.Fprintf(os.Stderr, "workline: the forge's board could not be read, no card moved: %s\n", forge.Said(err))
		}
		act.Board = mv
	}
	// The claim first, what it says on the other issues after it (Post).
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), append(claimOf(act, s), said...)); err != nil {
		return fail(err)
	}
	var is forge.Issue
	for _, i := range issues {
		if i.ID == id {
			is = i
		}
	}
	bounds, err := readBounds(f, b, repo, is)
	if err != nil {
		return external(err)
	}
	bounds.Kind, _ = IssueKind(is.Labels)
	if p, ok := proofs[id]; ok && bounds.Draft == 0 {
		bounds.Proof = &p
	}
	// Its code starts from the tester's Draft: the acceptance tests on it
	// run in the proof (ADR-0054, step 4).
	act.From, act.Acceptance = bounds.Head, bounds.acceptance()
	if err := writeIn(runDir, pickFile, act); err != nil {
		return fail(err)
	}
	on := at
	if bounds.Head != "" {
		on = bounds.Head // its code starts from the tester's Draft
	}
	ts, tsErr := testerSettings(repo) // none set, or unread, is said in the question; the proof says it again
	return ask(runDir, repo, on, bounds, taskText(is, bounds, ts.Command, tsErr, s))
}

// ask leaves its agent the question, a worktree of the commit on where it
// edits, and the bounds post holds its diff to. The question names the
// code, never pastes it: its agent reads it in its worktree (ADR-0056).
func ask(runDir, repo, on string, b Bounds, task string) int {
	if err := writeIn(runDir, boundsFile, b); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
		return fail(err)
	}
	if err := openWorktree(runDir, repo, on); err != nil {
		return fail(err)
	}
	return 0
}

// takenMeanwhile reads the issue again, just before its claim goes to
// apply: why another run holds it now, "" when free. Best effort: a forge
// out of reach is read as free, and two runs whose agents code at the
// same time can still both claim it (docs/triggers.md).
func takenMeanwhile(repo string, id int, s Settings) string {
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil || f == nil {
		return ""
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return ""
	}
	issues, err := b.Issues()
	if err != nil {
		return ""
	}
	for _, is := range issues {
		if is.ID != id {
			continue
		}
		switch {
		case len(is.Assignees) > 0:
			return "assigned to " + strings.Join(is.Assignees, ", ")
		case slices.Contains(is.Labels, s.ClaimLabel):
			return "labelled " + s.ClaimLabel
		}
		return ""
	}
	return "no longer open"
}

// claimOf is the claim: the label and the comment holding the record.
func claimOf(a Act, s Settings) []intent.Intention {
	return []intent.Intention{
		claimLabel(a, s),
		{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "claim", "body": claimBody(a, s)}},
	}
}

// claimLabel is the claim's label, its card moved to Doing in the same
// call when the forge's board has that column (ADR-0057).
func claimLabel(a Act, s Settings) intent.Intention {
	v := map[string]any{"issue": a.Issue, "add": []string{s.ClaimLabel}}
	switch {
	case a.Board.Labels():
		v["add"] = []string{s.ClaimLabel, a.Board.Column}
		if len(a.Board.Off) > 0 {
			v["remove"] = a.Board.Off
		}
	case a.Board.Writes():
		// A GitHub project's Status, set after the label (ADR-0057).
		v["status"] = map[string]any{"project": a.Board.Project, "option": a.Board.Column}
	}
	return intent.Intention{Kind: "label", Value: v}
}

// taskText is the question: the issue, what bounds the code and the run,
// the frozen tests' paths and the test command, or why it could not be
// read (testErr). No code: its agent reads, edits and runs it in its
// worktree (ADR-0056).
func taskText(is forge.Issue, b Bounds, test string, testErr error, s Settings) string {
	return question(is, b, test, testErr, s) +
		"Its diff is your code. When you are done, your last message is `[]` alone: no label, no comment, the claim is the engine's.\n"
}

// question is what an issue run and a review round ask alike: the issue,
// what bounds the code and the run, the frozen tests, the test command;
// it ends on the answer's first words, the caller's own rules after them.
func question(is forge.Issue, b Bounds, test string, testErr error, s Settings) string {
	var t strings.Builder
	fmt.Fprintf(&t, "# Issue #%d: %s\n\n%s\n\n## What you may change\n\n", is.ID, is.Title, strings.TrimSpace(is.Body))
	if len(b.Scope) == 0 {
		t.WriteString("The issue names no path: every change is reverted. Change nothing.\n")
	}
	for _, p := range b.Scope {
		fmt.Fprintf(&t, "- `%s`\n", p)
	}
	t.WriteString("- the tests beside them: in their folder, or in a test folder, the name holding `test` or `spec`.\n\n" +
		"Read any file you need. A change to any other file is reverted once you are done, whatever the reason: say it in your answer, it belongs in an issue. " +
		"Never, whatever the scope: the files that steer agents (AGENTS.md, CLAUDE.md, a page they link, a prompt), " +
		"a role's folder, `.workline/` (but the objections file below), the CI's workflows, a gate's definition.\n")
	if frozen := b.acceptance(); len(frozen) > 0 {
		t.WriteString("\n## The tester's acceptance tests: frozen\n\nIn your worktree, read-only to you; make them pass. A change to one is reverted.\n\n")
		for _, f := range frozen {
			fmt.Fprintf(&t, "- `%s`\n", f)
		}
		// In a file, not in the closing words: a real agent wrote its
		// objection in prose around YAML, and it was lost (#746).
		t.WriteString("\nOne that does not fit: object to it in the file `" + ObjectionsFile + "` of your worktree, " +
			"YAML only, one item an objection:\n\n" +
			"```yaml\n- test: <file>\n  line: <n>\n  scenario: <its name>\n  kind: <interface | over-specified | contradicts | untestable | need>\n  why: <one or two sentences>\n```\n\n" +
			"The engine reads that file and takes it out of your diff; the tester keeps or rewrites each objection.\n")
	}
	if b.Proof != nil {
		fmt.Fprintf(&t, "\n## Its proof: %s, no new test\n\nThe tester recorded it in place of acceptance tests: %s. Its proof: %s.\n",
			b.Proof.Kind, strings.TrimSuffix(b.Proof.Reason, "."), tester.ProofKinds[b.Proof.Kind])
	}
	t.WriteString("\n## The tests\n\n")
	switch {
	case testErr != nil:
		// Not "none set": a setting it cannot read is not one missing.
		fmt.Fprintf(&t, "The project's test command (the tester's `test-command`) could not be read: %v. "+
			"Find how the project runs its tests and run them; the proof reads the setting again and says why it cannot.\n", testErr)
	case test == "":
		t.WriteString("The project sets no test command (the tester's `test-command`): your code cannot be proven, its pull request will be a Draft.\n")
	default:
		fmt.Fprintf(&t, "Run them as the proof will: `%s`, `{report}` a path of your choosing outside the worktree, where it writes a JUnit report. "+
			"Your run is for you: the engine proves your code again, on a fresh tree.\n", test)
	}
	t.WriteString("\n## Your bounds\n\n")
	if s.MaxTurns > 0 {
		fmt.Fprintf(&t, "- %d turns at most.\n", s.MaxTurns)
	}
	if s.MaxSeconds > 0 {
		fmt.Fprintf(&t, "- %d seconds at most: past them you are stopped, your edits kept as they are.\n", s.MaxSeconds)
	}
	t.WriteString("- No commit, no push, no branch: the engine commits your edits and opens the pull request.\n")
	t.WriteString("\n## Your answer\n\nEdit the files in your worktree, your working directory (`WORKLINE_WORKTREE` for a command), and run the tests. ")
	return t.String()
}

// acceptance are the tests the tester froze for the issue, sorted.
func (b Bounds) acceptance() []string {
	var out []string
	for f, n := range b.Frozen {
		if n == b.Issue {
			out = append(out, f)
		}
	}
	slices.Sort(out)
	return out
}

// claimBody is the comment the claim leaves on the issue, its record hidden.
func claimBody(a Act, s Settings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer** takes this issue: the first ready one nobody took, in the backlog's order. "+
		"`%s` says so; a person who takes it back removes the label, or assigns themselves.\n", s.ClaimLabel)
	if a.Board != nil {
		fmt.Fprintf(&b, "\n%s\n", a.Board.Told())
	}
	if len(a.Passed) > 0 {
		b.WriteString("\nPassed over before it:\n\n")
		for _, p := range a.Passed {
			fmt.Fprintf(&b, "- #%d: %s\n", p.Issue, p.Why)
		}
	}
	if len(a.Refused) > 0 {
		b.WriteString("\nRefused at apply, left out of its code:\n\n")
		for _, r := range a.Refused {
			fmt.Fprintf(&b, "- `%s`: %s\n", r.File, why(r))
		}
	}
	if a.Proof != nil {
		b.WriteString("\nIts code on the project's tests:\n\n" + proofLines(*a.Proof, a.Caps))
	}
	if a.NoCode {
		fmt.Fprintf(&b, "\n- **Why no pull request**: its agent wrote no code within its bounds.\n"+
			"- **Next**: the claim is let go %d hours after it was taken, said here; then the developer may take it again, or a person assigns themselves.\n", s.ClaimHours)
	}
	if len(a.Unread) > 0 && a.Proof == nil {
		// No pull request to post them on: here, never lost (#746).
		b.WriteString("\nIts agent's words, unread: a person reads them, an objection to a frozen test may be among them.\n\n")
		for _, u := range a.Unread {
			b.WriteString(u + "\n")
		}
	}
	fmt.Fprintf(&b, "\n%s\n", a.Line())
	return b.String()
}

// Post holds the agent's proposals to the bounds, with no agent: the
// claim, the patches within them and the objections kept, the rest
// refused and said, in the verdict and in the claim's record.
func Post(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "in", prBoundsFile)); err == nil {
		return boundsPost(runDir)
	}
	if _, err := os.Stat(filepath.Join(runDir, "in", roundFile)); err == nil {
		return reviewPost(runDir, repo, s)
	}
	if hasSaid(runDir) {
		return saidPost(runDir)
	}
	var a Act
	var b Bounds
	if err := readIn(runDir, pickFile, &a); err != nil {
		return fail(err)
	}
	if err := readIn(runDir, boundsFile, &b); err != nil {
		return fail(err)
	}
	claim, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return fail(err)
	}
	// What pre said on the other issues, after the claim: written whatever
	// becomes of the claim.
	told := claim[min(len(claimOf(a, s)), len(claim)):]
	where := fmt.Sprintf("#%d", a.Issue)
	// Another run may have claimed the issue while its agent coded: read
	// again, it backs off, nothing claimed nor opened (docs/triggers.md).
	if why := takenMeanwhile(repo, a.Issue, s); why != "" {
		if code := writeIntents(runDir, "", told); code != 0 {
			return code
		}
		return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass, Summary: where + " taken meanwhile: nothing claimed, its code left",
			Findings: []verdict.Finding{{Rule: "taken-meanwhile", Level: "warn", Where: where, Message: why + " while its agent coded: another run took it first"}}})
	}
	proposed, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	on := a.At
	if a.From != "" {
		on = a.From // the tester's Draft, its acceptance tests on it
	}
	kept, refused, said, unread, err := held(runDir, repo, on, b, claim, proposed)
	if err != nil {
		return fail(err)
	}
	a.Refused, a.Unread = refused, unread
	v := verdict.Verdict{Status: verdict.Pass, Summary: "took " + where, Findings: []verdict.Finding{rightsFinding(where)}}
	for _, p := range a.Passed {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	v.Findings = append(v.Findings, a.Board.Findings()...)
	v.Findings = append(v.Findings, refusalFindings(where, refused, said)...)
	if n := len(refused) + len(said); n > 0 {
		v.Summary += fmt.Sprintf("; %d of its agent's proposals refused at apply", n)
	}
	var patches []intent.Intention
	for _, in := range kept[len(claim):] {
		if in.Kind == "patch" {
			patches = append(patches, in)
		}
	}
	if len(unread) > 0 {
		v.Findings = append(v.Findings, unreadFinding(where, unread, len(patches) > 0))
	}
	if len(patches) == 0 {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
			Message: fmt.Sprintf("its agent wrote no code within its bounds: nothing to prove, no pull request; the claim is let go after %d hours, said on the issue", s.ClaimHours)})
		a.NoCode = true
	} else {
		a.Code, a.Tests = written(patches)
		p, u, caps, found := proven(runDir, repo, on, where, patches, a.Acceptance, s)
		a.Proof, a.Caps = &p, caps
		a.Turns, a.Tokens, a.Cached, a.Tools, a.AgentFailed = u.Turns, u.Tokens, u.Cached, u.Tools, u.failed
		v.Findings = append(v.Findings, found...)
		title := issueTitle(runDir, a.Issue)
		cs, unread := committerSettings(repo, where)
		v.Findings = append(v.Findings, unread...)
		mr := mergeRequest(a, title, subjects(a.Issue, title, b.Kind, b.Scope, cs))
		if mr["draft"] == true {
			v.Summary += "; a Draft: " + p.Result
			if len(a.Caps) > 0 {
				v.Summary += ", a cap reached"
			}
			if u.failed != "" {
				v.Summary += ", its agent failed"
			}
		} else {
			v.Summary += "; proven green, its pull request ready"
		}
		if err := writeMergeRequest(runDir, mr); err != nil {
			return fail(err)
		}
	}
	// The claim carries the record, the refusals in it (ADR-0046).
	// What pre said on the other issues follows it.
	if code := writeIntents(runDir, "", slices.Concat(claimOf(a, s), told, kept[len(claim):])); code != 0 {
		return code
	}
	return writeVerdict(runDir, v)
}

// held is its agent's code held to the bounds: what it proposed, the
// objections it wrote in its worktree and its worktree's diff against
// base, the claim pre made first; the files refused, the worktree's
// before the bounds', and the rest said; its words that did not read, as
// they came (agentWords).
func held(runDir, repo, base string, b Bounds, claim, proposed []intent.Intention) (kept []intent.Intention, refused []Refusal, said, unread []string, err error) {
	objections, unread, err := agentWords(runDir)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// What its agent edited in the worktree is code too: its diff, held
	// to the same bounds; the engine removes the worktree (ADR-0056).
	fromTree, treeRefused, err := worktreeCode(runDir, base)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	kept, refused, said = bounded(repo, b, claim, append(append(proposed, objections...), fromTree...))
	return kept, append(treeRefused, refused...), said, unread, nil
}

// unreadFinding says its agent's words that did not read, and where a
// person reads them as they came.
func unreadFinding(where string, unread []string, posted bool) verdict.Finding {
	at := "posted as they came in its claim on the issue: no pull request to post them on"
	if posted {
		at = "posted as they came on its pull request"
	}
	return verdict.Finding{Rule: "agent-words-unread", Level: "warn", Where: where,
		Message: fmt.Sprintf("%d of its agent's answers beside its code could not be read (its closing words, its objections file): nothing of them applied, %s, so that an objection written there is not lost; its code is held as any", len(unread), at)}
}

// rights is with whose rights its agent ran, said to whoever runs it:
// no refusal, the forge's guards decide what merges (ADR-0058,
// principle 6). In CI, the job's: what its token may do is not checked
// here.
func rights() string {
	if forge.InCI() != "" {
		return "the agent runs with the CI job's rights; the forge's guards decide what merges"
	}
	return "the agent runs with your account's rights; the forge's guards decide what merges"
}

// rightsLine says it in the pull request's description, to its reviewer.
func rightsLine() string {
	who := "the account that ran it"
	if forge.InCI() != "" {
		who = "the CI job"
	}
	return "Its agent ran with the rights of " + who + "; the forge's guards decide what merges."
}

// rightsFinding says it once in the run's output.
func rightsFinding(where string) verdict.Finding {
	return verdict.Finding{Rule: "agent-rights", Level: "info", Where: where, Message: rights()}
}

// refusalFindings say in the verdict what the bounds refused.
func refusalFindings(where string, refused []Refusal, said []string) []verdict.Finding {
	var out []verdict.Finding
	for _, r := range refused {
		out = append(out, verdict.Finding{Rule: r.Rule, Level: "warn", Where: r.File, Message: "refused at apply, its patch left out: " + why(r)})
	}
	for _, w := range said {
		out = append(out, verdict.Finding{Rule: "not-the-developers", Level: "warn", Where: where, Message: "refused at apply: " + w})
	}
	return out
}

// proven proves the patches on the commit on, on a fresh tree, and reads
// what its agent used: the proof, the caps the run reached, and what the
// verdict says of them. Also: the files whose folders run beside the
// patches' (the acceptance tests).
func proven(runDir, repo, on, where string, patches []intent.Intention, also []string, s Settings) (Proof, Used, []Cap, []verdict.Finding) {
	p, over := prove(repo, on, patches, also, s)
	u := used(runDir)
	caps := agentCaps(u, s)
	if over != nil {
		caps = append(caps, *over)
	}
	var found []verdict.Finding
	if u.failed != "" {
		found = append(found, agentFailed(where, u.failed))
	}
	found = append(found, unreported(where, u, s)...)
	return p, u, caps, append(found, proofFindings(where, p, caps)...)
}

func writeMergeRequest(runDir string, mr map[string]any) error {
	data, err := yaml.Marshal(mr)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644)
}

// readIn reads a run's in/<name>, JSON, into v.
func readIn(runDir, name string, v any) error {
	data, err := os.ReadFile(filepath.Join(runDir, "in", name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// writeIn writes v, JSON, to a run's in/<name>.
func writeIn(runDir, name string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runDir, "in", name), data, 0o644)
}

func settingsIn(runDir string) (Settings, error) {
	m := map[string]any{}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return Settings{}, err
	}
	return SettingsFrom(m), nil
}

func final(runDir string, v verdict.Verdict) int {
	v.Final = true
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return exitNothing
}

// external says a forge that did not answer, never read as no issue.
func external(err error) int {
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	return fail(err)
}

func gitOut(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "developer:", err)
	return 1
}

// proofsOf reads the kind of proof the tester recorded on each ready
// issue whose tests would be waited for: named scenarios, no tester's
// Draft open. Read only where the project routes the tester.
func proofsOf(b forge.Backlog, issues []forge.Issue, mrs []forge.MergeRequest, testsFirst bool) (map[int]tester.Proof, error) {
	out := map[int]tester.Proof{}
	if !testsFirst {
		return out, nil
	}
	tested := map[int]bool{}
	for _, m := range mrs {
		tested[testerIssue(m.Branch)] = true
	}
	for _, is := range issues {
		if !slices.Contains(is.Labels, backlog.LabelReady) || tested[is.ID] || len(tester.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) == 0 {
			continue
		}
		comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, err
		}
		if p, ok := tester.ReadProof(comments); ok {
			out[is.ID] = p
		}
	}
	return out, nil
}
