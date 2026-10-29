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

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
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
	// MaxTokens is the engine's ai-max-tokens: the run's calls spending it
	// is a cap reached. ProofSeconds bound the test command's run.
	MaxTokens    int
	ProofSeconds int
	// Rounds are the review rounds it runs on its pull request; past
	// them, a person, by the label Person (step 6).
	Rounds int
	Person string
}

// SettingsFrom reads the settings.
func SettingsFrom(m map[string]any) Settings {
	s := Settings{ClaimLabel: "workline:developing", Rounds: 5, Person: "workline:human"}
	if n := number(m["review-rounds"]); n > 0 {
		s.Rounds = n
	}
	if v, ok := m["person"].(string); ok && strings.TrimSpace(v) != "" {
		s.Person = strings.TrimSpace(v)
	}
	if v, ok := m["claim-label"].(string); ok && strings.TrimSpace(v) != "" {
		s.ClaimLabel = strings.TrimSpace(v)
	}
	s.MaxTokens, s.ProofSeconds = number(m["ai-max-tokens"]), number(m["proof-seconds"])
	return s
}

// Passed is a ready issue the pick passed over, and why.
type Passed struct {
	Issue int    `json:"issue"`
	Why   string `json:"why"`
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
	// Proof: what the project's tests said on its code; Caps: the bounds
	// on the run it reached (step 3).
	Proof *Proof `json:"proof,omitempty"`
	Caps  []Cap  `json:"caps,omitempty"`
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

// pathWord is a word in backticks: a path, when it holds a slash or a dot.
var pathWord = regexp.MustCompile("`([^`\\s]+)`")

// scopePaths are the paths an issue's Scope names, in backticks.
func scopePaths(body string) []string {
	var out []string
	for _, m := range pathWord.FindAllStringSubmatch(work.ByMeaning(body)[work.Scope].Text, -1) {
		p, ok := scopeEntry(m[1])
		if ok && strings.ContainsAny(m[1], "/.") && !slices.Contains(out, p) {
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
// tests (ADR-0055). only, when not 0, is the one issue looked at.
// passed are the issues offered it passed over, why each.
func pick(issues []forge.Issue, mrs []forge.MergeRequest, s Settings, only int, testsFirst bool) (id int, passed []Passed) {
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
		if slices.Contains(is.Labels, s.ClaimLabel) {
			held[is.ID] = scopePaths(is.Body)
		}
	}
	backlog.Order(issues)
	for _, is := range issues {
		if only != 0 && is.ID != only || !backlog.Offered(is, open) {
			continue
		}
		why := ""
		switch {
		case len(is.Assignees) > 0:
			why = "assigned to " + strings.Join(is.Assignees, ", ")
		case slices.Contains(is.Labels, s.ClaimLabel):
			why = "taken already, labelled " + s.ClaimLabel
		case closer[is.ID] != 0:
			why = fmt.Sprintf("the open merge request %d closes it", closer[is.ID])
		case testsFirst && !tested[is.ID] && len(tester.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)) > 0:
			why = "its acceptance tests are not written yet: it waits for the tester's Draft"
		default:
			mine := scopePaths(is.Body)
			for _, n := range slices.Sorted(maps.Keys(held)) {
				if p := overlap(mine, held[n]); p != "" {
					why = fmt.Sprintf("its %s's `%s` overlaps #%d's, taken", work.Name(work.Scope), p, n)
					break
				}
			}
		}
		if why == "" {
			return is.ID, passed
		}
		passed = append(passed, Passed{Issue: is.ID, Why: why})
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
	id, passed := pick(issues, mrs, s, only, line.Runs("tester"))
	var findings []verdict.Finding
	for _, p := range passed {
		findings = append(findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	if id == 0 {
		summary := "no ready issue waits for the developer"
		if only != 0 {
			summary = fmt.Sprintf("#%d is not one the developer may take", only)
			if len(passed) == 0 {
				findings = append(findings, verdict.Finding{Rule: "issue-not-offered", Level: "info", Where: fmt.Sprintf("#%d", only),
					Message: "not open and " + backlog.LabelReady + ", or it waits on an open issue, or it has parts"})
			}
		}
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: summary, Findings: findings})
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		findings = append(findings, verdict.Finding{Rule: "no-agent", Level: "info", Where: fmt.Sprintf("#%d", id),
			Message: "with no agent, the developer names the issue it would take, and claims and opens nothing (principle 5)"})
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("would take #%d: no agent, nothing claimed", id), Findings: findings})
	}
	at, _ := gitOut(repo, "rev-parse", "HEAD")
	act := Act{Act: "pick", At: at, Issue: id, Passed: passed}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), claimOf(act, s)); err != nil {
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
	// Its code starts from the tester's Draft: the acceptance tests on it
	// run in the proof (ADR-0054, step 4).
	act.From, act.Acceptance = bounds.Head, bounds.acceptance()
	data, _ := json.Marshal(act)
	if err := os.WriteFile(filepath.Join(runDir, "in", pickFile), data, 0o644); err != nil {
		return fail(err)
	}
	if data, err = json.Marshal(bounds); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", boundsFile), data, 0o644); err != nil {
		return fail(err)
	}
	on := at
	if bounds.Head != "" {
		on = bounds.Head // its code starts from the tester's Draft
	}
	code, err := scopeCode(repo, on, bounds)
	if err != nil {
		return fail(err)
	}
	task := taskText(is, bounds, frozenTexts(repo, bounds), code)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
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
		{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": []string{s.ClaimLabel}}},
		{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "claim", "body": claimBody(a, s)}},
	}
}

// taskText is the question: the issue, and what bounds the code.
func taskText(is forge.Issue, b Bounds, texts map[string]string, code string) string {
	var t strings.Builder
	fmt.Fprintf(&t, "# Issue #%d: %s\n\n%s\n\n## What you may touch\n\n", is.ID, is.Title, strings.TrimSpace(is.Body))
	if len(b.Scope) == 0 {
		t.WriteString("The issue names no path: every patch is refused. Propose none.\n")
	}
	for _, p := range b.Scope {
		fmt.Fprintf(&t, "- `%s`\n", p)
	}
	t.WriteString("- the tests beside them: in their folder, or in a test folder, the name holding `test` or `spec`.\n\n" +
		"Anything else is refused at apply, whatever the reason: say it in your answer, it belongs in an issue. " +
		"Never, whatever the scope: the files that steer agents (AGENTS.md, CLAUDE.md, a page they link, a prompt), " +
		"a role's folder, `.workline/`, the CI's workflows, a gate's definition.\n")
	if frozen := b.acceptance(); len(frozen) > 0 {
		t.WriteString("\n## The tester's acceptance tests: frozen\n\nRead-only to you; make them pass. A patch touching one is refused. " +
			"Your code goes on the tester's branch, where they are.\n\n")
		for _, f := range frozen {
			if text, ok := texts[f]; ok {
				fmt.Fprintf(&t, "### `%s`\n\n```\n%s\n```\n\n", f, strings.TrimRight(text, "\n"))
			} else {
				fmt.Fprintf(&t, "- `%s`\n", f)
			}
		}
		// The answer's generic `comment` is a text: a real agent wrote its
		// objection so, and it was refused (measured, docs/status.md).
		t.WriteString("\nOne that does not fit: object to it, a `comment` whose value is an objection, " +
			"never a text (a text comment is refused):\n\n" +
			"```yaml\n- comment:\n    objection: {test: <file>, line: <n>, scenario: <its name>, kind: <interface | over-specified | contradicts | untestable | need>, why: <one or two sentences>}\n```\n\n" +
			"The tester keeps or rewrites it.\n")
	}
	t.WriteString(code)
	t.WriteString("\n## Your answer\n\nPatches only, and objections if any. No label, no other comment: the claim is the engine's.\n")
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

// frozenTexts are the acceptance tests as the tester's Draft holds them,
// for the agent to read: its checkout is the default branch's.
func frozenTexts(repo string, b Bounds) map[string]string {
	texts := map[string]string{}
	if b.Head == "" {
		return texts
	}
	for _, f := range b.acceptance() {
		if text, err := gitOut(repo, "show", b.Head+":"+f); err == nil {
			texts[f] = text
		}
	}
	return texts
}

// codeRoom is the most of the Scope's code a question shows, in bytes:
// about 25k tokens, under the role's context budget with the rest.
const codeRoom = 100_000

// scopeCode is the code the agent may touch, as it reads on the commit its
// code starts from: the agent has no tools, so a file it is not shown, it
// cannot patch (measured, docs/status.md). A Scope's folder shows its
// files; a path not there yet is said; past codeRoom, files are named.
// A git that cannot read the commit is an error, never read as a path not
// there yet: the agent would write over a file it was not shown.
func scopeCode(repo, at string, b Bounds) (string, error) {
	if at == "" || len(b.Scope) == 0 {
		return "", nil
	}
	var t strings.Builder
	t.WriteString("\n## The code in scope, as it reads\n\n")
	room := codeRoom
	for _, p := range b.Scope {
		// A path absent from the commit lists nothing, with no error.
		list, err := gitOut(repo, "ls-tree", "-r", "--name-only", at, "--", p)
		if err != nil {
			return "", fmt.Errorf("the code in scope: git cannot list %s at %s: %w", p, at, err)
		}
		if list == "" {
			fmt.Fprintf(&t, "- `%s`: not there yet; a patch creates it.\n", p)
			continue
		}
		for _, f := range strings.Split(list, "\n") {
			if _, frozen := b.Frozen[f]; frozen {
				continue // shown with the acceptance tests
			}
			text, err := gitOut(repo, "show", at+":"+f)
			switch {
			case err != nil:
				fmt.Fprintf(&t, "- `%s`: not shown, git cannot read it.\n", f)
			case len(text) > room:
				fmt.Fprintf(&t, "- `%s`: not shown, over the question's room.\n", f)
			default:
				room -= len(text)
				fmt.Fprintf(&t, "### `%s`\n\n```\n%s\n```\n\n", f, text)
			}
		}
	}
	return t.String(), nil
}

// claimBody is the comment the claim leaves on the issue, its record hidden.
func claimBody(a Act, s Settings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer** takes this issue: the first ready one nobody took, in the backlog's order. "+
		"`%s` says so; a person who takes it back removes the label, or assigns themselves.\n", s.ClaimLabel)
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
	fmt.Fprintf(&b, "\n%s\n", a.Line())
	return b.String()
}

// Post holds the agent's proposals to the bounds, with no agent: the
// claim, the patches within them and the objections kept, the rest
// refused and said, in the verdict and in the claim's record.
func Post(runDir, repo string) int {
	if _, err := os.Stat(filepath.Join(runDir, "in", roundFile)); err == nil {
		s, err := settingsIn(runDir)
		if err != nil {
			return fail(err)
		}
		return reviewPost(runDir, repo, s)
	}
	var a Act
	data, err := os.ReadFile(filepath.Join(runDir, "in", pickFile))
	if err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return fail(err)
	}
	var b Bounds
	if data, err = os.ReadFile(filepath.Join(runDir, "in", boundsFile)); err != nil {
		return fail(err)
	}
	if err := json.Unmarshal(data, &b); err != nil {
		return fail(err)
	}
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	// Another run may have claimed the issue while its agent coded: read
	// again, it backs off, nothing claimed nor opened (docs/triggers.md).
	if why := takenMeanwhile(repo, a.Issue, s); why != "" {
		if err := os.Remove(filepath.Join(runDir, "out", "intentions.yaml")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fail(err)
		}
		where := fmt.Sprintf("#%d", a.Issue)
		v := verdict.Verdict{Status: verdict.Pass, Summary: where + " taken meanwhile: nothing claimed, its code left",
			Findings: []verdict.Finding{{Rule: "taken-meanwhile", Level: "warn", Where: where, Message: why + " while its agent coded: another run took it first"}}}
		if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
			return fail(err)
		}
		return 0
	}
	proposed, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	claim, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return fail(err)
	}
	kept, refused, said := bounded(repo, b, claim, proposed)
	a.Refused = refused
	where := fmt.Sprintf("#%d", a.Issue)
	v := verdict.Verdict{Status: verdict.Pass, Summary: "took " + where}
	for _, p := range a.Passed {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "issue-passed", Level: "info", Where: fmt.Sprintf("#%d", p.Issue), Message: p.Why})
	}
	for _, r := range refused {
		v.Findings = append(v.Findings, verdict.Finding{Rule: r.Rule, Level: "warn", Where: r.File, Message: "refused at apply, its patch left out: " + why(r)})
	}
	for _, w := range said {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "not-the-developers", Level: "warn", Where: where, Message: "refused at apply: " + w})
	}
	if n := len(refused) + len(said); n > 0 {
		v.Summary += fmt.Sprintf("; %d of its agent's proposals refused at apply", n)
	}
	var patches []intent.Intention
	for _, in := range kept[len(claim):] {
		if in.Kind == "patch" {
			patches = append(patches, in)
		}
	}
	if len(patches) == 0 {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
			Message: "its agent wrote no code within its bounds: nothing to prove, no pull request; the claim stays until a person removes it"})
	} else {
		a.Code, a.Tests = written(patches)
		on := a.At
		if a.From != "" {
			on = a.From // the tester's Draft, its acceptance tests on it
		}
		p, over := prove(repo, on, patches, s)
		a.Proof = &p
		if over != nil {
			a.Caps = append(a.Caps, *over)
		}
		if c := tokensCap(runDir, s.MaxTokens); c != nil {
			a.Caps = append(a.Caps, *c)
		}
		v.Findings = append(v.Findings, proofFindings(where, p, a.Caps)...)
		mr := mergeRequest(a, issueTitle(runDir, a.Issue))
		if mr["draft"] == true {
			v.Summary += "; a Draft: " + p.Result
			if len(a.Caps) > 0 {
				v.Summary += ", a cap reached"
			}
		} else {
			v.Summary += "; proven green, its pull request ready"
		}
		data, err := yaml.Marshal(mr)
		if err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
			return fail(err)
		}
	}
	// The claim carries the record, the refusals in it (ADR-0046).
	kept = append(claimOf(a, s), kept[len(claim):]...)
	if err := os.Remove(filepath.Join(runDir, "out", "intentions.yaml")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fail(err)
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), kept); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
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
