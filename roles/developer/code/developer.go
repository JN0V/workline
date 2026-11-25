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
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/testmark"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/clock"
	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/testdef"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/work"
)

// The role joins the engine by name (internal/contract).
func init() {
	contract.Join("developer", contract.Role{Pre: Pre, Post: Post, Code: &contract.Code{
		ProveMerge: ProveMerge, Guarded: Guarded,
		RoundDue: RoundDue, Held: Held, Overlapping: Overlapping, Changed: Changed, RepoFiles: RepoFiles,
		ClaimHours: func(m map[string]any) int { return SettingsFrom(m).ClaimHours },
	}})
}

const (
	exitNothing  = 10 // pre: no question for the agent, its verdict final
	exitExternal = 3  // an outside service failed
	pickFile     = "pick.json"
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
	// FlakyRuns: the runs a flaky-n-of-n proof makes when the tester's
	// record names none (ADR-0059, decision 4).
	FlakyRuns int
	// Rounds are the review rounds it runs on its pull request; past
	// them, its issue blocked (ADR-0059).
	Rounds int
	// Level is its autonomy level, LevelFrom where it comes from
	// (ADR-0060, §3): what its levels lay is Rounds; its task says both.
	Level, LevelFrom string
	// rolesDir: where the tester's role is read for the proof, "" the
	// one the engine gives a role's run (ProveMerge's, from follow).
	rolesDir string
}

// SettingsFrom reads the settings.
func SettingsFrom(m map[string]any) Settings {
	s := Settings{ClaimLabel: "workline:developing", ClaimHours: 24, Rounds: 5, FlakyRuns: 10}
	if n := number(m["claim-hours"]); n > 0 {
		s.ClaimHours = n
	}
	if n := number(m["flaky-runs"]); n > 0 {
		s.FlakyRuns = n
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
	s.Level, s.LevelFrom = role.LevelOf(m, "autonomy")
	return s
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

// scopePaths are the paths a Scope names: in backticks, a word with a
// slash or a dot, its line off (work.ScopeWords); in plain words, a path
// the repository holds (#901). files are the repository's, nil unread:
// the words in backticks taken as written then. Else a word with a slash
// is kept when work.NewPath says so, or a pattern or a file it names is, or
// its first folder is the repository's: `dorny/paths-filter` is not.
func scopePaths(body string, files []string) []string {
	text := work.ByMeaning(body)[work.Scope].Text
	var out []string
	add := func(p string) {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	for _, w := range work.ScopeWords(text) {
		p, ok := scopeEntry(w)
		if !ok || !strings.ContainsAny(w, "/.") {
			continue
		}
		if files == nil || !strings.Contains(p, "/") || work.NewPath(w) ||
			len(work.Files(p, files)) > 0 || holds(files, strings.SplitN(p, "/", 2)[0]) {
			add(p)
		}
	}
	if files == nil {
		return out
	}
	for _, m := range backlog.PathLike.FindAllString(work.Plain(text), -1) {
		if p, ok := scopeEntry(work.LineOff(backlog.RepoPath(m))); ok && holds(files, p) {
			add(p)
		}
	}
	return out
}

// holds says whether files hold p, a file or a folder.
func holds(files []string, p string) bool {
	return slices.ContainsFunc(files, func(f string) bool { return f == p || strings.HasPrefix(f, p+"/") })
}

// repoFiles are the files the repository's index holds, nil when they
// cannot be read.
func repoFiles(repo string) []string {
	out, err := gitOut(repo, "ls-files", "-z")
	if err != nil {
		return nil
	}
	files := []string{}
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files
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

// Held are the issues the developer holds with code, each the files it
// holds: claimed (claimLabel), and an open merge request closes it
// (#952). A claim whose agent wrote no code holds no file: nothing of
// its can conflict, and the hours it waits out are a pause before it is
// taken again, not a hold on the issues beside it. letGo: the claims a
// run lets go; files: the repository's, the Scopes read against them.
// The files it holds are its merge requests' changed files, as changed
// reads them: facts, not its Scope's prediction (ADR-0065, change 2,
// #1256). One changed cannot read holds its prediction, as before: its
// Scope's paths and those its pull request lists beyond the Scope.
// Only a claimed issue's merge requests are read: an unclaimed one's
// branch is never fetched, a wait each pick would feel.
func Held(issues []forge.Issue, mrs []forge.MergeRequest, claimLabel string, letGo map[int]bool, files []string, changed contract.Diffs) map[int][]string {
	closing := map[int][]forge.MergeRequest{}
	for _, m := range mrs {
		for _, n := range forge.ClosesOn(os.Getenv("WORKLINE_FORGE"), m.Title, m.Body) {
			closing[n] = append(closing[n], m)
		}
	}
	held := map[int][]string{}
	for _, is := range issues {
		ms := closing[is.ID]
		if is.Closed || !slices.Contains(is.Labels, claimLabel) || len(ms) == 0 || letGo[is.ID] {
			continue // its merge requests' branches not fetched: a claim alone is read
		}
		var diff, beyond []string
		unread := false
		for _, m := range ms {
			beyond = union(beyond, beyondListed(m.Body))
			var d []string
			ok := false
			if changed != nil {
				d, ok = changed(m)
			}
			diff, unread = union(diff, d), unread || !ok
		}
		held[is.ID] = diff
		if unread {
			held[is.ID] = union(diff, union(scopeHeld(is.Body, files), beyond))
		}
	}
	return held
}

// scopeHeld are a Scope's paths as they hold other issues back: none when
// the Scope says no code was given when it was written, as Scopes said
// before the `Guessed:` mark (#998): such paths were never read.
func scopeHeld(body string, files []string) []string {
	if work.CodeUnread(work.ByMeaning(body)[work.Scope].Text) {
		return nil
	}
	return inRepo(scopePaths(body, files), files)
}

// beyondListed are the files a pull request's body lists under its
// Beyond the Scope heading, as body writes them.
func beyondListed(body string) []string {
	var out []string
	in := false
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "## "):
			in = l == "## "+BeyondHeading
		case in && strings.HasPrefix(l, "- `"):
			if f, _, ok := strings.Cut(l[3:], "`"); ok && f != "" && !slices.Contains(out, f) {
				out = append(out, f)
			}
		}
	}
	return out
}

// Overlapping is the first issue of held, by number, whose files (Held)
// overlap is's Scope's paths, and the path; 0 when none, or is itself. A Scope
// its writer marked as a guess (work.Guessed) waits on nothing: its paths
// were named before their code was read (#970).
func Overlapping(is forge.Issue, held map[int][]string, files []string) (int, string) {
	if work.Guessed(work.ByMeaning(is.Body)[work.Scope].Text) {
		return 0, ""
	}
	mine := scopeHeld(is.Body, files)
	for _, n := range slices.Sorted(maps.Keys(held)) {
		if p := overlap(mine, held[n]); p != "" && n != is.ID {
			return n, p
		}
	}
	return 0, ""
}

// kindOf is the kind issue n's labels give it, "" when none or not among
// issues.
func kindOf(issues []forge.Issue, n int) string {
	for _, is := range issues {
		if is.ID == n {
			k, _ := devmark.IssueKind(is.Labels)
			return k
		}
	}
	return ""
}

// RepoFiles are the files the repository's index holds, nil when they
// cannot be read: what a Scope's paths are read against.
func RepoFiles(repo string) []string { return repoFiles(repo) }

// pick is the issue the developer takes: the first, in the backlog's
// order, offered to whoever builds (ready, waiting on no open issue, no
// parts), assigned to nobody, not claimed already, no open merge request
// closing it, its Scope's paths clear of the issues the developer holds
// with code (Held).
// testsFirst, where the project routes the tester: an issue whose
// Validation names scenarios waits for the tester's open Draft of its
// tests (ADR-0055), or the kind of proof it recorded instead (proofs,
// ADR-0059). only, when not 0, is the one issue looked at. files are the
// repository's, its Scopes' paths read against them (scopePaths);
// changed reads the files held merge requests change (Held).
// rank orders priorities: 1 first, none last.
func rank(is forge.Issue) int {
	if p := backlog.Priority(is); p != 0 {
		return p
	}
	return math.MaxInt
}

// outranked is held less the issues of a lower priority than is's: a more
// urgent issue is not held by them; they keep their work and rebase. The
// same priority still waits.
func outranked(is forge.Issue, issues []forge.Issue, held map[int][]string) map[int][]string {
	byID := map[int]forge.Issue{}
	for _, o := range issues {
		byID[o.ID] = o
	}
	out := map[int][]string{}
	for n, files := range held {
		if o, ok := byID[n]; ok && rank(o) > rank(is) {
			continue
		}
		out[n] = files
	}
	return out
}

// letGo are the claims this run lets go: held no more, taken at a next
// run. passed are the issues offered it passed over, why each.
func pick(issues []forge.Issue, mrs []forge.MergeRequest, s Settings, only int, testsFirst bool, letGo map[int]bool, proofs map[int]backlog.Proof, files []string, changed contract.Diffs) (id int, passed []devmark.Passed) {
	open := map[int]bool{}
	for _, is := range issues {
		open[is.ID] = true
	}
	closer, tested := map[int]int{}, map[int]bool{}
	for _, m := range mrs {
		if n := testerIssue(m.Branch); n != 0 {
			tested[n] = true
		}
		for _, n := range forge.ClosesOn(os.Getenv("WORKLINE_FORGE"), m.Title, m.Body) {
			if _, ok := closer[n]; !ok {
				closer[n] = m.ID
			}
		}
	}
	held := Held(issues, mrs, s.ClaimLabel, letGo, files, changed)
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
		case testsFirst && !tested[is.ID] && proofs[is.ID].Kind == "" && len(work.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) > 0:
			why = "its acceptance tests are not written yet: it waits for the tester's Draft"
			next = "the tester writes them, as a Draft of its own; the developer takes this issue at a run after that Draft is open"
		default:
			if n, p := Overlapping(is, outranked(is, issues, held), files); n != 0 {
				why = devmark.OverlapWhy(n, p)
				next = fmt.Sprintf("#%d merged or let go; the developer looks at this issue again at its next run", n)
				if k := kindOf(issues, n); k != devmark.KindFix && closer[n] != 0 {
					// Only a fix merges alone (ADR-0060): the wait ends on
					// a person's yes, said where they look (#998).
					said := "no label says its kind"
					if k != "" {
						said = "a " + k
					}
					next = fmt.Sprintf("#%d's pull request #%d merges on a person's yes (#%[1]d: %[3]s), then the developer looks at this issue again at its next run; or #%[1]d is let go", n, closer[n], said)
				}
			}
		}
		if why == "" {
			return is.ID, passed
		}
		passed = append(passed, devmark.Passed{Issue: is.ID, Why: why, Next: next})
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
	said, findings, err := stale(b, issues, mrs, s, clock.Now(), card)
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
	proofs, err := proofsOf(b, issues, mrs, line.Runs("tester"), only)
	if err != nil {
		return external(err)
	}
	// An issue carrying the product owner's untestable label is held like a
	// parked one, ready or not: the line builds nothing it cannot test.
	untestable, err := untestableLabel(repo)
	if err != nil {
		return fail(err)
	}
	issues = backlog.Hold(issues, untestable)
	id, passed := pick(issues, mrs, s, only, line.Runs("tester"), letGo, proofs, repoFiles(repo), Changed(repo, forge.KeepsBranches(f)))
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
	act := devmark.Act{Act: "pick", At: at, Issue: id, Passed: passed}
	if i := slices.IndexFunc(issues, func(is forge.Issue) bool { return is.ID == id }); i >= 0 {
		act.Body = devmark.BodyMark(issues[i].Body)
	}
	if act.Unopened, err = unopenedBefore(b, id); err != nil {
		return external(err)
	}
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
	bounds.Kind, _ = devmark.IssueKind(is.Labels)
	if p, ok := proofs[id]; ok && bounds.Draft == 0 {
		bounds.Proof = &p
	}
	// Its code starts from the tester's Draft: the acceptance tests on it
	// run in the proof (ADR-0054, step 4). A Draft behind the base gets
	// the base merged in first, in this clone only: its agent's diff then
	// applies on the base's paths (#1941).
	from, conflicts, err := draftOnBase(repo, bounds.Head, at)
	if err != nil {
		return fail(err)
	}
	if len(conflicts) > 0 {
		return draftBehind(runDir, repo, id, bounds.Head, at, conflicts, said)
	}
	act.From, act.Acceptance, act.RedUnproven = bounds.Head, bounds.acceptance(), bounds.RedUnproven
	if from != bounds.Head {
		act.Merged = at
	}
	if err := writeIn(runDir, pickFile, act); err != nil {
		return fail(err)
	}
	on := at
	if from != "" {
		on = from // its code starts from the tester's Draft, the base in it
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
func claimOf(a devmark.Act, s Settings) []intent.Intention {
	return []intent.Intention{
		claimLabel(a, s),
		{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "claim", "body": claimBody(a, s)}},
	}
}

// claimLabel is the claim's label, a move from Ready to Doing: the ready
// label off in the same call, one state at a time (ADR-0059, §2;
// #1081); its card moved to Doing too when the forge's board has that
// column (ADR-0057).
func claimLabel(a devmark.Act, s Settings) intent.Intention {
	v := map[string]any{"issue": a.Issue, "add": []string{s.ClaimLabel}, "remove": []string{backlog.LabelReady}}
	switch {
	case a.Board.Labels():
		v["add"] = []string{s.ClaimLabel, a.Board.Column}
		v["remove"] = append([]string{backlog.LabelReady}, a.Board.Off...)
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

// testsBeside tells the agent the tests that are beside the Scope, by
// the project's one definition of a test file (ADR-0065): its patterns
// and where a test's code lives, never a language's rule.
func testsBeside(d testdef.Def) string {
	var homes []string
	for _, h := range d.Homes {
		homes = append(homes, fmt.Sprintf("`%s` → `%s`", h.Tests, h.Code))
	}
	where := "a test's code lives in its own folder"
	if len(homes) > 0 {
		where = "a test's code lives where its folder maps (" + strings.Join(homes, ", ") + "), else in its own folder"
	}
	return fmt.Sprintf("- the test files beside them: one in their folder, or a new one whose code lives beside them. The test files: `%s`; %s. A file matching none is code.\n",
		strings.Join(d.Patterns(), "`, `"), where)
}

// question is what an issue run and a review round ask alike: the issue,
// what bounds the code and the run, the frozen tests, the test command;
// it ends on the answer's first words, the caller's own rules after them.
func question(is forge.Issue, b Bounds, test string, testErr error, s Settings) string {
	var t strings.Builder
	fmt.Fprintf(&t, "# Issue #%d: %s\n\n%s\n\n## What you may change\n\n", is.ID, is.Title, strings.TrimSpace(is.Body))
	if len(b.Scope) == 0 {
		fmt.Fprintf(&t, "The issue's %s names no path.\n", scopeName)
	} else {
		fmt.Fprintf(&t, "The issue's %s, the product owner's hint of the files the change needs:\n\n", scopeName)
	}
	for _, p := range b.Scope {
		fmt.Fprintf(&t, "- `%s`\n", p)
	}
	t.WriteString(testsBeside(b.tests()))
	t.WriteString("\nRead any file you need, and change any file the change needs: the " + scopeName + " is a hint, not a fence. " +
		"For each file you change beyond it, give one line of why in the file `" + BeyondFile + "` of your worktree, YAML, `path/to/file: why the change needs it`; " +
		"the engine reads it and takes it out of your diff, and the pull request lists those files for the reviewer, who judges each. " +
		"Stay on the issue: a fix it does not ask, found on the way, belongs in another issue, said in your answer. " +
		"Never, whatever the change needs: the files the project's CODEOWNERS lists, agent files (AGENTS.md, CLAUDE.md…) and, with no CODEOWNERS, the pages they link, " +
		"`.workline/` (but the files named here), the CI's workflows, a gate's definition: such a change is reverted.\n")
	if frozen := b.acceptance(); len(frozen) > 0 {
		t.WriteString("\n## The tester's acceptance tests: frozen\n\nIn your worktree, read-only to you; make them pass. A change to one is reverted.\n\n")
		for _, f := range frozen {
			fmt.Fprintf(&t, "- `%s`\n", f)
		}
		if len(b.Covers) > 0 {
			t.WriteString("\nWhich test covers which scenario, as the tester declared it:\n\n")
			for _, sc := range b.Scenarios {
				if ids := b.Covers[sc]; len(ids) > 0 {
					fmt.Fprintf(&t, "- %s: `%s`\n", sc, strings.Join(ids, "`, `"))
				}
			}
		}
		// In a file, not in the closing words: a real agent wrote its
		// objection in prose around YAML, and it was lost (#746).
		t.WriteString("\nOne that does not fit: object to it in the file `" + ObjectionsFile + "` of your worktree, " +
			"YAML only, one item an objection:\n\n" +
			"```yaml\n- test: <file>\n  line: <n>\n  scenario: " + scenarioHint(b.Scenarios) + "\n  kind: <interface | over-specified | contradicts | untestable | need>\n  why: <one or two sentences>\n```\n\n" +
			"The scenario is the issue's, as its " + work.Name(work.Validation) + " names it, or the id of a test the tester declared for it; never a test's function name. " +
			"The engine reads that file and takes it out of your diff; the tester keeps or rewrites each objection.\n\n" +
			"A test that cannot pass within your bounds (it calls a command against its documented rule, for instance without a flag its no-terminal rule requires) is such an objection, of kind `contradicts`: object to it in that file, never in a text note, which reaches no tester.\n")
	}
	if b.Proof != nil && b.Proof.Kind == backlog.KindNone {
		fmt.Fprintf(&t, "\n## No acceptance test applies\n\nThe tester found that no acceptance test applies to this issue, and handed it over: %s. There is no frozen test; build the issue from its text, and its project's own tests prove the rest.\n",
			strings.TrimSuffix(b.Proof.Reason, "."))
	} else if b.Proof != nil {
		fmt.Fprintf(&t, "\n## Its proof: %s, no new test\n\nThe tester recorded it in place of acceptance tests: %s. Its proof: %s.\n",
			b.Proof.Kind, strings.TrimSuffix(b.Proof.Reason, "."), backlog.ProofKinds[b.Proof.Kind])
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
	if s.Level != "" {
		fmt.Fprintf(&t, "- Autonomy: %s (%s): %d review rounds disputing a blocking finding on your pull request, then its issue is blocked for a person.\n", s.Level, s.LevelFrom, s.Rounds)
	}
	t.WriteString("\n## Your answer\n\nEdit the files in your worktree, your working directory (`WORKLINE_WORKTREE` for a command), and run the tests. ")
	return t.String()
}

// scenarioHint is the objection's scenario in its format: one of the
// issue's, by name (#913), each whole: a name may hold a comma.
func scenarioHint(scenarios []string) string {
	if len(scenarios) == 0 {
		return "<its name in the issue's " + work.Name(work.Validation) + ">"
	}
	return "<one of: " + strings.Join(scenarios, " | ") + ">"
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
func claimBody(a devmark.Act, s Settings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer** takes this issue: the first ready one nobody took, in the backlog's order. "+
		"`%s` says so; a person who takes it over assigns themselves; one who gives it back to the line swaps it for `%s`: "+
		"the label taken off alone leaves the issue in no state, off the line.\n", s.ClaimLabel, backlog.LabelReady)
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
		b.WriteString("\nIts code on the project's tests:\n\n" + devmark.ProofLines(*a.Proof, a.Caps))
	}
	if a.NoCode && a.Widen == nil {
		fmt.Fprintf(&b, "\n- **Why no pull request**: its agent wrote no code.\n"+
			"- **Next**: %d hours after it was taken, its body unchanged, the line blocks it for a person, never running the developer on the same input again; "+
			"a change to its body before lets the claim go at once, and the developer may take it again. "+
			"Meanwhile it holds no paths: an issue whose %s overlaps this one's may be taken.\n", s.ClaimHours, work.Name(work.Scope))
	}
	if len(a.Unread) > 0 && a.Proof == nil {
		// No pull request to post them on: here, never lost (#746).
		b.WriteString("\nIts agent's words, unread: a person reads them, an objection to a frozen test may be among them.\n\n")
		for _, u := range a.Unread {
			b.WriteString(u + "\n")
		}
	}
	if len(a.Beside) > 0 {
		fmt.Fprintf(&b, "\nNew test files beside its %s: `%s`; the product owner adds them to the %[1]s at its next run.\n\n%[3]s\n",
			work.Name(work.Scope), strings.Join(a.Beside, "`, `"), backlog.ScopeKeptMark(a.Beside))
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
	var a devmark.Act
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
	if a.Merged != "" {
		// The base merged into it, as pre made it: the same commit.
		if on, _, err = draftOnBase(repo, a.From, a.Merged); err != nil || on == "" {
			return fail(fmt.Errorf("the base %s merged into the tester's Draft %s again: %v", devmark.ShortSHA(a.Merged), devmark.ShortSHA(a.From), err))
		}
	}
	kept, refused, said, unread, why, err := held(runDir, repo, on, b, claim, proposed)
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
	// Its agent wrote nothing on a tester's Draft a person may have made
	// green themselves, with a change only they may make (#1434): proven
	// on the Draft's head as it is, green opens its pull request from it.
	var draft *drafted
	if len(patches) == 0 && a.From != "" {
		p, u, caps, found := proven(runDir, repo, on, where, nil, a.Acceptance, flakyOf(b.Proof, s), s)
		if p.Result == devmark.Green && len(caps) == 0 && u.failed == "" {
			draft = &drafted{p, u, caps, found}
		}
	}
	if len(patches) == 0 && draft == nil && onlyProtected(refused) {
		// Its only changes were refused as a person's to make: nothing
		// to wait 24 hours for, the issue goes to a person at once (#1418).
		v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
			Message: "its agent's only changes were refused as protected: nothing to prove, no pull request; blocked for a person at once, said on the issue"})
		v.Summary += "; blocked for a person"
		return writeBlockedProtected(runDir, repo, a, s, told, refused, v)
	}
	if len(patches) == 0 && draft == nil {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
			Message: fmt.Sprintf("its agent wrote no code: nothing to prove, no pull request; blocked for a person after %d hours on the same body, said on the issue", s.ClaimHours)})
		a.NoCode = true
	} else {
		a.Code, a.Tests = written(patches, b.tests())
		a.Beside, a.Beyond = besideKept(repo, b, patches), beyondKept(repo, b, patches, why)
		if len(a.Beyond) > 0 {
			v.Findings = append(v.Findings, beyondFinding(where, a.Beyond))
		}
		var p devmark.Proof
		var u Used
		var caps []devmark.Cap
		var found []verdict.Finding
		if draft != nil {
			p, u, caps, found = draft.p, draft.u, draft.caps, draft.found
		} else {
			p, u, caps, found = proven(runDir, repo, on, where, patches, a.Acceptance, flakyOf(b.Proof, s), s)
		}
		if p.Result == devmark.Red && len(p.Red) == 0 && len(caps) == 0 && !objects(kept) {
			// Its code does not build, or breaks the run, no test red, and
			// no objection to a frozen test beside it, the tester's: the
			// agent's to mend, asked again once with the error, as the
			// tester's test that does not build (#1083); still so, a
			// machine error the line runs again (ADR-0063, I3). Nothing
			// claimed nor opened.
			v.Status, v.Again = verdict.Block, true
			v.Summary = where + ": its code does not build"
			v.Findings = append(v.Findings, verdict.Finding{Rule: "build-broken", Where: where,
				Message: "its code does not build, or breaks the test run: " + p.Why})
			if code := writeIntents(runDir, "", told); code != 0 {
				return code
			}
			if code := writeVerdict(runDir, v); code != 0 {
				return code
			}
			return 1 // blocked: the exit its verdict says (role contract)
		}
		a.Proof, a.Caps = &p, caps
		a.NoTest = b.Proof != nil && b.Proof.Kind == backlog.KindNone
		a.Unproven = unprovenLines(b.Unproven)
		a.Turns, a.Tokens, a.Cached, a.Tools, a.AgentFailed = u.Turns, u.Tokens, u.Cached, u.Tools, u.failed
		v.Findings = append(v.Findings, found...)
		title := issueTitle(runDir, a.Issue)
		cs, unread := committerSettings(repo, where)
		v.Findings = append(v.Findings, unread...)
		mr := mergeRequest(a, title, subjects(a.Issue, title, b.Kind, b.Scope, a.Code, cs))
		mr["unopened"] = unopened(a, s, repo)
		if note := noteOf(kept[len(claim):], a.Issue); note != "" {
			mr["body"] = withNote(fmt.Sprint(mr["body"]), note)
		}
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

// withNote puts the agent's one note, the text posted on the issue, in the
// pull request's `## Left out`, opening that section when it has none.
func withNote(body, note string) string {
	note = devmark.InertText(note)
	const head = "\n## Left out\n\n"
	if i := strings.Index(body, head); i >= 0 {
		i += len(head)
		return body[:i] + note + "\n\n" + body[i:]
	}
	verif := "\n## " + work.Name(work.Verification) + "\n\n"
	if i := strings.Index(body, verif); i >= 0 {
		return body[:i] + head + note + "\n" + body[i:]
	}
	return body + head + note + "\n"
}

// held is its agent's code held to the bounds: what it proposed, the
// objections it wrote in its worktree and its worktree's diff against
// base, the claim pre made first; the files refused, the worktree's
// before the bounds', and the rest said; its words that did not read, as
// they came (agentWords).
func held(runDir, repo, base string, b Bounds, claim, proposed []intent.Intention) (kept []intent.Intention, refused []devmark.Refusal, said, unread []string, beyond map[string]string, err error) {
	objections, beyond, unread, err := agentWords(runDir)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	// What its agent edited in the worktree is code too: its diff, held
	// to the same bounds; the engine removes the worktree (ADR-0056).
	fromTree, treeRefused, err := worktreeCode(runDir, base)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	kept, refused, said = bounded(repo, b, claim, append(append(proposed, objections...), fromTree...))
	return kept, append(treeRefused, refused...), said, unread, beyond, nil
}

// beyondFinding says, in the verdict, the files a change wrote beyond its
// issue's Scope (ADR-0065, change 2): applied, listed for the reviewer.
func beyondFinding(where string, l []devmark.Beyond) verdict.Finding {
	files := make([]string, len(l))
	for i, x := range l {
		files[i] = x.File
	}
	return verdict.Finding{Rule: "beyond-scope", Level: "info", Where: where,
		Message: fmt.Sprintf("%d file%s beyond the issue's %s, applied and listed on the pull request for the reviewer's intent lens: %s",
			len(l), devmark.Plural(len(l)), scopeName, strings.Join(files, ", "))}
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
func refusalFindings(where string, refused []devmark.Refusal, said []string) []verdict.Finding {
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
func proven(runDir, repo, on, where string, patches []intent.Intention, also []string, n *Flaky, s Settings) (devmark.Proof, Used, []devmark.Cap, []verdict.Finding) {
	p, over := Prove(repo, on, patches, also, n, s)
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
	return p, u, caps, append(found, ProofFindings(where, p, caps)...)
}

// drafted is the proof of a tester's Draft its agent wrote nothing on,
// green as it is (#1434).
type drafted struct {
	p     devmark.Proof
	u     Used
	caps  []devmark.Cap
	found []verdict.Finding
}

// objects says whether what its agent proposed, held to the bounds,
// holds an objection to a frozen test: the tester's to answer, so its
// code that does not build beside it is no machine error (ADR-0055).
func objects(kept []intent.Intention) bool {
	return slices.ContainsFunc(kept, func(i intent.Intention) bool {
		m, _ := i.Value.(map[string]any)
		return i.Kind == "comment" && m["sticky"] == "objection"
	})
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
	s := SettingsFrom(m)
	if s.Level != "" && !slices.Contains(role.AutonomyLevels, s.Level) {
		// A level someone believes in and nothing applies (principle 12).
		return s, fmt.Errorf("autonomy: %q is not one of %s", s.Level, strings.Join(role.AutonomyLevels, ", "))
	}
	return s, nil
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
// Draft open. Read only where the project routes the tester; with only,
// the issue a run was started for, its own alone: each ready issue's
// comments would spend the token's API budget at every step (#804).
func proofsOf(b forge.Backlog, issues []forge.Issue, mrs []forge.MergeRequest, testsFirst bool, only int) (map[int]backlog.Proof, error) {
	out := map[int]backlog.Proof{}
	if !testsFirst {
		return out, nil
	}
	tested := map[int]bool{}
	for _, m := range mrs {
		tested[testerIssue(m.Branch)] = true
	}
	for _, is := range issues {
		if only != 0 && is.ID != only || !slices.Contains(is.Labels, backlog.LabelReady) || tested[is.ID] || len(work.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) == 0 {
			continue
		}
		comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return nil, err
		}
		if p, ok := testmark.ReadProof(comments); ok {
			out[is.ID] = p
		} else if line.PersonReady(comments) {
			// A person put it back in ready after the tester refused it:
			// their decision, no test of the tester's to wait for.
			out[is.ID] = backlog.Proof{Kind: backlog.KindNone, Reason: "a person decided it is ready without a tester's test"}
		}
	}
	return out, nil
}

// onlyProtected says whether every refusal is a file only a person
// changes (a gate's or the CI's, a steering file), and there is one.
func onlyProtected(refused []devmark.Refusal) bool {
	if len(refused) == 0 {
		return false
	}
	for _, r := range refused {
		if r.Rule != "gate-file" && r.Rule != "steering-file" {
			return false
		}
	}
	return true
}

// writeBlockedProtected claims the issue and blocks it for a person in
// the same run, the refused files and why said on it (#1418).
func writeBlockedProtected(runDir, repo string, a devmark.Act, s Settings, told []intent.Intention, refused []devmark.Refusal, v verdict.Verdict) int {
	add, remove := []string{backlog.LabelBlocked}, []string{backlog.LabelReady}
	if line, err := routing.Load(repo); err == nil {
		if ad, rm, err := line.Move("blocked"); err == nil {
			add, remove = ad, rm
		}
	}
	if !slices.Contains(remove, s.ClaimLabel) {
		remove = append(remove, s.ClaimLabel)
	}
	var b strings.Builder
	for _, r := range refused {
		fmt.Fprintf(&b, "`%s`: %s; ", r.File, why(r))
	}
	say := backlog.Said(who, "blocks this issue",
		"its agent's only changes were refused at apply: "+strings.TrimSuffix(b.String(), "; "),
		"a person makes the change, or decides it, then moves the issue back by its label")
	out := slices.Concat(claimOf(a, s), told, []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": add, "remove": remove, "card": "blocked"}},
		{Kind: "comment", Value: map[string]any{"issue": a.Issue, "body": say}},
	})
	if code := writeIntents(runDir, "", out); code != 0 {
		return code
	}
	return writeVerdict(runDir, v)
}

// untestableLabel is the product owner's `untestable-label` in the project's
// settings; "" when unset. A config that cannot be read is an error, not an
// unset label: the hold must not fail open.
func untestableLabel(repo string) (string, error) {
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return "", err
	}
	l, _ := cfg.Roles["product-owner"].Settings["untestable-label"].(string)
	return strings.TrimSpace(l), nil
}
