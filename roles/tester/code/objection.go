package tester

// The objection (ADR-0055, a test that does not fit): the builder comments
// on a test's line, opened by `workline:objection`, naming the test, its
// scenario, the kind and why. A fresh tester run reads the scenario first
// and answers in the thread: kept, with its reason, or rewritten, its own
// commit referring to the thread, still red on the base. The need itself
// goes to the product owner. Three answers a thread; still open after the
// second, the third run loads the shared five whys skill and posts the
// analysis with its answer; past the third, the product owner settles it,
// then the issue is blocked (ADR-0059).

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/commitmsg"
	"github.com/JN0V/workline/internal/kit/reviewmark"
	"github.com/JN0V/workline/internal/kit/testmark"
	"github.com/JN0V/workline/internal/kit/verdict"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/whys"
	"github.com/JN0V/workline/internal/work"
)

// The kinds an objection names, as the ADR words them.
var kinds = []struct{ key, name, word string }{
	{"interface", "wrong interface", "interface"},
	{"over-specified", "over-specified", "over"},
	{"contradicts", "contradicts another scenario", "contradict"},
	{"untestable", "untestable as written", "untestable"},
	{"need", "the need itself", "need"},
	// A test the review finds missing, its cause a line of code: the
	// tester adds it or says why none is needed (#1265). No scenario
	// named: the tester reads them all.
	{"missing", "a test missing", "missing"},
}

// What an answer run does, decided before any agent.
const (
	modeAgent     = "agent"     // the tester's judgement: kept or rewritten
	modeNeed      = "need"      // the product owner's: the issue back to refine
	modeSettle    = "settle"    // past the tester's answers: the product owner settles (ADR-0059)
	modeBlocked   = "blocked"   // the builder objects still, past the product owner: Blocked
	modeMalformed = "malformed" // not an objection the tester can answer: said why
)

// objection is an objection's fields, from its first comment.
type objection struct {
	Test, Scenario, Kind, Why string
}

var fieldLine = regexp.MustCompile(`(?i)^\s*(?:[-*+]\s+)?(\**)\s*(test|scenario|kind|why)\s*\**\s*:\s*(.*)$`)

// parseObjection reads `test:`, `scenario:`, `kind:`, `why:` lines; why
// runs to the end. The kind is one of kinds, by its word.
func parseObjection(body string) (objection, []string) {
	var o objection
	text := strings.TrimSpace(body)
	text = strings.TrimSpace(strings.TrimPrefix(text, testmark.ObjectionWord))
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		m := fieldLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		label, v := m[2], strings.TrimSpace(m[3])
		if m[1] != "" { // a bold label, `**test:** value`: its closing marker
			v = strings.TrimSpace(strings.TrimPrefix(v, m[1]))
		}
		if !strings.EqualFold(label, "why") {
			v = unwrap(v)
		}
		switch strings.ToLower(label) {
		case "test":
			o.Test = v
		case "scenario":
			o.Scenario = v
		case "kind":
			o.Kind = v
		case "why":
			o.Why = strings.TrimSpace(v + "\n" + strings.Join(lines[i+1:], "\n"))
		}
		if strings.EqualFold(label, "why") {
			break
		}
	}
	return o, o.missing()
}

// with is o, each field r gives replacing its own.
func (o objection) with(r objection) objection {
	for _, f := range []struct {
		to *string
		v  string
	}{{&o.Test, r.Test}, {&o.Scenario, r.Scenario}, {&o.Kind, r.Kind}, {&o.Why, r.Why}} {
		if f.v != "" {
			*f.to = f.v
		}
	}
	return o
}

// missing names the fields o lacks.
func (o objection) missing() []string {
	var missing []string
	if o.Test == "" {
		missing = append(missing, "test")
	}
	if o.Scenario == "" && kindOf(o.Kind) != "missing" {
		missing = append(missing, "scenario")
	}
	if kindOf(o.Kind) == "" {
		missing = append(missing, "kind (one of: "+kindNames()+")")
	}
	if o.Why == "" {
		missing = append(missing, "why")
	}
	return missing
}

func kindOf(s string) string {
	s = strings.ToLower(s)
	for _, k := range kinds {
		if strings.Contains(s, k.word) {
			return k.key
		}
	}
	return ""
}

func kindName(key string) string {
	for _, k := range kinds {
		if k.key == key {
			return k.name
		}
	}
	return key
}

func kindNames() string {
	var n []string
	for _, k := range kinds {
		n = append(n, k.name)
	}
	return strings.Join(n, ", ")
}

// answer is what the answer run's pre leaves for post, in in/answer.json.
type answer struct {
	Mode      string        `json:"mode"`
	Why       string        `json:"why,omitempty"` // a malformed objection's reason
	MR        int           `json:"merge-request"`
	Issue     int           `json:"issue"`
	Thread    string        `json:"thread"`
	URL       string        `json:"url"`
	File      string        `json:"file"`
	Kind      string        `json:"kind,omitempty"`
	Scenario  work.Scenario `json:"scenario"`
	Answers   int           `json:"answers"`   // the tester's answers in the thread already
	FiveWhys  bool          `json:"five-whys"` // this answer is the one the five whys are due on
	Decided   string        `json:"decided"`   // a person's decision, past the bound
	Record    record        `json:"record"`
	Waiting   int           `json:"waiting"` // the objections left for the next runs
	Objection objection     `json:"objection"`
	// Line: the routing's states, as pre read them: where the issue goes
	// to settle or to block (ADR-0059).
	Line []routing.State `json:"line,omitempty"`
	// Text: the issue's title and body; a rewrite's missing name counts
	// as red only when it names it (#1111).
	Text string `json:"text,omitempty"`
	// Scenarios, a test missing's: the issue's, the test may be any's;
	// From, the pull request whose review found it, where the answer is
	// said too (#1265).
	Scenarios []work.Scenario `json:"scenarios,omitempty"`
	From      int             `json:"from,omitempty"`
	// Found: the line the review found it on, as its why names it: the
	// developer reads the answer by it. Title: the finding's, after it,
	// which the reviewer reads the answer by, its line moved or not.
	Found string `json:"found,omitempty"`
	Title string `json:"title,omitempty"`
	// Code: the file Found is in, as the pull request From has it,
	// often its branch's alone (#1416); CodeUnread, why it is not.
	Code       string `json:"code,omitempty"`
	CodeUnread string `json:"code-unread,omitempty"`
}

// reviewedOnWords finds, in a test missing's why, the pull request whose
// review found it, the line and the finding, as the developer writes them.
var reviewedOnWords = regexp.MustCompile("\\bon pull request (\\d+)\\b(?:, `([^`]+)`(?:: ([^\\n]*))?)?")

// reviewedOn is the pull request a test missing's why names, 0 for none,
// the line the review found it on and the finding's title, "" for none.
func reviewedOn(why string) (n int, found, title string) {
	m := reviewedOnWords.FindStringSubmatch(why)
	if m == nil {
		return 0, "", ""
	}
	n, _ = strconv.Atoi(m[1])
	return n, m[2], reviewmark.FindingTitle(m[3])
}

const answerFile = "answer.json"

// foundCode is the file a test missing's finding is on, as the pull
// request whose review found it has it (#1416): the builder's branch,
// fetched with the checkout, holds it when the tester's Draft does not.
// Cut past code-lines-max, said; why when it cannot be read.
func foundCode(f forge.Forge, repo string, mr int, file string, s Settings) (code, unread string) {
	m, err := f.MergeRequest(mr)
	if err != nil {
		return "", fmt.Sprintf("pull request %d could not be read: %v", mr, err)
	}
	var refs []string
	if m.Branch != "" {
		refs = append(refs, "origin/"+m.Branch, m.Branch)
	}
	if m.Head != "" {
		refs = append(refs, m.Head)
	}
	for _, ref := range refs {
		text, err := git(repo, "show", ref+":"+file)
		if err != nil {
			continue
		}
		limit := s.CodeLinesMax
		if limit <= 0 {
			limit = 1500
		}
		if lines := strings.Split(strings.TrimRight(text, "\n"), "\n"); len(lines) > limit {
			text = strings.Join(lines[:limit], "\n") + fmt.Sprintf("\n… cut: %d lines past code-lines-max (%d)", len(lines)-limit, limit)
		}
		return strings.TrimRight(text, "\n"), ""
	}
	return "", fmt.Sprintf("`%s` is not on pull request %d's branch, %s, as this checkout has it", file, mr, m.Branch)
}

// answerPre finds the first objection waiting for the tester's answer and
// decides who answers it.
func answerPre(runDir, repo string, s Settings) int {
	kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	id, _ := strconv.Atoi(num)
	if kind != "merge-request" || id <= 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no merge request to answer on",
			Findings: []verdict.Finding{{Rule: "no-merge-request", Message: "the tester answers the objections of one merge request: run `workline test answer --merge-request <n>`"}}})
	}
	where := fmt.Sprintf("merge request %d", id)
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the objections from",
			Findings: []verdict.Finding{{Rule: "no-forge", Where: where, Message: "an objection is a comment on a line of the merge request, on a forge; " + forge.Missing}}})
	}
	external := func(err error) int { fmt.Fprintln(os.Stderr, err); return exitExternal }
	mr, err := f.MergeRequest(id)
	if errors.Is(err, forge.ErrUnreachable) {
		return external(err)
	} else if err != nil {
		return fail(err)
	}
	issue := testmark.BranchIssue(mr.Branch)
	if issue == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "not a tester's branch",
			Findings: []verdict.Finding{{Rule: "not-a-testers-branch", Where: where,
				Message: "it comes from " + mr.Branch + ": the tester answers objections to its own tests, on " + testmark.TestsBranch + "<n>"}}})
	}
	th, ok := f.(forge.Threaded)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "this forge lists no conversation on a line",
			Findings: []verdict.Finding{{Rule: "no-threads", Where: where, Message: "the objections are read from the merge request's comments on lines, which this forge does not list"}}})
	}
	// own: who wrote the record, its replies too (a GitHub reply in a
	// thread names no App: its login is learnt from the record).
	r, missing, own, err := readRecordKept(f, id)
	if errors.Is(err, forge.ErrUnreachable) {
		return external(err)
	} else if err != nil {
		return fail(err)
	}
	if missing != nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the tester's tests cannot be told", Findings: []verdict.Finding{*missing}})
	}
	threads, err := th.Threads(id)
	if errors.Is(err, forge.ErrUnreachable) {
		return external(err)
	} else if err != nil {
		return fail(err)
	}
	mine := func(n forge.Note, mark string) bool { return own.Wrote(n) && strings.Contains(n.Body, mark) }
	// read counts the tester's answers in a thread; once the product owner
	// was asked to settle it, a person's decision there, else whether the
	// builder objected again.
	read := func(t forge.Thread) (answers int, decided string, settled, again bool) {
		for _, c := range t.Comments {
			switch {
			case mine(c, testmark.SettleMark), mine(c, testmark.PersonMark):
				settled = true
			case mine(c, testmark.AnswerMark):
				answers++
			case settled && c.Insider && !c.Bot:
				decided = c.Body // a person of the project, on the loop, after it
			case settled && !mine(c, testmark.BlockedMark):
				again = true // the builder still objects, past the product owner
			}
		}
		return answers, decided, settled, again
	}
	var waiting []forge.Thread
	for _, t := range threads {
		if t.Resolved || len(t.Comments) == 0 || (!strings.HasPrefix(strings.TrimSpace(t.Comments[0].Body), testmark.ObjectionWord) && !testmark.PersonsThread(t)) {
			continue
		}
		if last := t.Comments[len(t.Comments)-1]; len(neededFiles(runDir)) > 0 && lacked(last.Body) {
			// Run again given the files it lacked (#1443): its turn again.
		} else if mine(last, testmark.AnswerMark) || mine(last, testmark.UnreadMark) || mine(last, testmark.SettleMark) ||
			mine(last, testmark.PersonMark) || mine(last, testmark.BlockedMark) {
			continue // answered, settled or blocked: the builder's turn, or the product owner's
		}
		waiting = append(waiting, t)
	}
	if len(waiting) == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no objection waits for the tester's answer"})
	}
	t := waiting[0]
	a := answer{MR: id, Issue: issue, Thread: t.ID, URL: t.URL, File: t.Path, Record: r, Waiting: len(waiting) - 1}
	var settled, again bool
	a.Answers, a.Decided, settled, again = read(t)
	exchanges := s.Exchanges
	if exchanges <= 0 {
		exchanges = 3
	}
	// The first comment's fields, then those the builder gave in a reply,
	// the latest kept: an unreadable objection is completed in the thread.
	o, _ := parseObjection(t.Comments[0].Body)
	byPerson := testmark.PersonsThread(t)
	if byPerson {
		// A person's review comment: a test missing, on the line it is on.
		o = objection{Test: t.Path, Kind: "missing", Why: strings.TrimSpace(t.Comments[0].Body)}
	}
	for _, c := range t.Comments[1:] {
		if own.Wrote(c) || byPerson {
			continue
		}
		if r, _ := parseObjection(c.Body); r != (objection{}) {
			o = o.with(r)
		}
	}
	unread := o.missing()
	a.Objection, a.Kind = o, kindOf(o.Kind)
	is, err := f.Issue(issue)
	if errors.Is(err, forge.ErrUnreachable) {
		return external(err)
	} else if err != nil {
		return fail(err)
	}
	a.Text = is.Title + "\n" + is.Body
	scenarios := Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)
	for _, sc := range scenarios {
		if o.Scenario != "" && norm(sc.Name) == norm(o.Scenario) {
			a.Scenario = sc
		}
	}
	switch {
	case a.Decided != "":
		a.Mode = modeAgent
	case settled && again && slices.Contains(is.Labels, backlog.LabelToRefine):
		// The product owner has not settled it yet: a reply meanwhile is
		// no objection past it.
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("the product owner settles the objection: #%d waits in refine", issue),
			Findings: []verdict.Finding{{Rule: "objection-at-the-product-owner", Level: "info", Where: t.URL,
				Message: "the builder replied before the product owner settled it: nothing for the tester till it does"}}})
	case settled && again:
		a.Mode = modeBlocked
	case a.Answers >= exchanges:
		a.Mode = modeSettle
	case len(unread) > 0:
		a.Mode, a.Why = modeMalformed, "it names no "+strings.Join(unread, ", no ")
	case r.Files[t.Path] == "":
		a.Mode, a.Why = modeMalformed, "its line is in `"+t.Path+"`, not one of the tester's tests ("+strings.Join(r.files(), ", ")+")"
	case a.Scenario.Name == "" && a.Kind != "missing":
		var names []string
		for _, sc := range scenarios {
			names = append(names, sc.Name)
		}
		a.Mode, a.Why = modeMalformed, fmt.Sprintf("its scenario %q is none of #%d's (%s)", o.Scenario, issue, strings.Join(names, ", "))
	case a.Kind == "need":
		a.Mode = modeNeed
	default:
		a.Mode = modeAgent
	}
	a.FiveWhys = a.Mode == modeAgent && a.Decided == "" && a.Answers == 2
	if a.Kind == "missing" {
		a.Scenarios = scenarios
		a.From, a.Found, a.Title = reviewedOn(o.Why)
		if a.From > 0 && a.Found != "" {
			a.Code, a.CodeUnread = foundCode(f, repo, a.From, work.LineOff(a.Found), s)
		}
	}
	if line, err := routing.Load(repo); err == nil {
		a.Line = line.States
	}
	if data, err := json.Marshal(a); err != nil {
		return fail(err)
	} else if err := os.WriteFile(filepath.Join(runDir, "in", answerFile), data, 0o644); err != nil {
		return fail(err)
	}
	if a.Mode != modeAgent {
		// No task: the engine's own acts, applied with no agent asked.
		acts, _ := plainAnswer(a, s)
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), acts); err != nil {
			return fail(err)
		}
		return 0
	}
	if a.FiveWhys {
		if err := os.WriteFile(filepath.Join(runDir, "in", agent.SkillsFile), []byte(whys.Skill+"\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	return writeAnswerTask(runDir, repo, a, t, is, s)
}

// writeAnswerTask gives the agent the scenario first, then the objection
// and the thread, the test as it stands, and the Scope's code only for
// an interface claim.
func writeAnswerTask(runDir, repo string, a answer, t forge.Thread, is *forge.Issue, s Settings) int {
	var b strings.Builder
	if a.Kind == "missing" {
		fmt.Fprintf(&b, "# A test the review finds missing, for #%d: %s\n\n", a.Issue, is.Title)
		b.WriteString("## The scenarios, a person's\n\n")
		for _, sc := range a.Scenarios {
			fmt.Fprintf(&b, "- **%s:** %s\n", sc.Name, sc.Text)
		}
		b.WriteString("\n")
	} else {
		fmt.Fprintf(&b, "# An objection to a test of #%d: %s\n\n", a.Issue, is.Title)
		fmt.Fprintf(&b, "## The scenario, a person's\n\n- **%s:** %s\n\n", a.Scenario.Name, a.Scenario.Text)
	}
	fmt.Fprintf(&b, "## The objection: %s\n\n", kindName(a.Kind))
	for i, c := range t.Comments {
		who := "the builder"
		if i == 0 && a.Kind == "missing" && testmark.PersonsThread(t) {
			who = "a person of the project"
		}
		switch {
		case strings.Contains(c.Body, testmark.AnswerMark), strings.Contains(c.Body, testmark.UnreadMark):
			who = "you, the tester"
		case strings.Contains(c.Body, testmark.PersonMark), strings.Contains(c.Body, testmark.SettleMark):
			who = "the engine, asking the product owner to settle it"
		case strings.Contains(c.Body, testmark.BlockedMark):
			who = "the engine, blocking the issue"
		case c.Insider && !c.Bot && a.Decided == c.Body:
			who = "a person of the project, deciding"
		}
		body := strings.TrimSpace(strings.NewReplacer(testmark.AnswerMark, "", testmark.PersonMark, "", testmark.SettleMark, "", testmark.BlockedMark, "", testmark.UnreadMark, "").Replace(c.Body))
		fmt.Fprintf(&b, "### %d. From %s\n\n%s\n\n", i+1, who, body)
	}
	if a.Decided != "" {
		b.WriteString("A person of the project decided, last: do what they decided.\n\n")
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(a.File)))
	if err != nil {
		return fail(err)
	}
	fmt.Fprintf(&b, "## The test, `%s`, as it stands\n\n````\n%s\n````\n", a.File, strings.TrimRight(string(data), "\n"))
	if a.Kind == "missing" {
		fmt.Fprintf(&b, "Your other tests of #%d: %s.\n", a.Issue, strings.Join(otherFiles(a.Record, a.File), ", "))
		switch file := work.LineOff(a.Found); {
		case a.Code != "":
			fmt.Fprintf(&b, "\n## The code the finding is on, `%s`, as pull request %d has it\n\n````\n%s\n````\n", file, a.From, a.Code)
		case a.CodeUnread != "":
			fmt.Fprintf(&b, "\n## The code the finding is on\n\nNot given: %s.\n", a.CodeUnread)
		}
	}
	for _, p := range neededFiles(runDir) {
		if data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p))); err == nil {
			fmt.Fprintf(&b, "\n## The code you said you lacked, `%s`\n\n````\n%s\n````\n", p, strings.TrimRight(string(data), "\n"))
		}
	}
	head := b.String()
	b.Reset()
	if a.Kind == "missing" {
		b.WriteString("\n## Your answer\n\n" +
			"The reviewer found a behaviour of the change with no test, its cause in the code: you are the one who writes the acceptance tests.\n\n" +
			"- **Added**: the test, as a `patch` of an existing file (the file whole with `file` and `content`, or a diff of that file), and a `note`, why. " +
			"It goes in `" + a.File + "`, or in another of your tests of #" + fmt.Sprint(a.Issue) + " listed above, if any, never a new file: a patch of any other file is refused. " +
			"It checks a scenario's Then, and fails on the code before the change.\n" +
			"- **None needed**: a `note`, why no acceptance test is needed (a scenario's test covers it already, naming it; or it is no behaviour a scenario asks).\n" +
			"- **Lacks code**: a `note` opening `needs:`, the files you lack and why, when the code given does not let you write it.\n")
	} else {
		b.WriteString("\n## Your answer\n\n" +
			"- **Kept**: a `note`, the reason the test holds for the scenario.\n" +
			"- **Rewritten**: the same file whole as a `patch` with `file` and `content`, and a `note`, why. " +
			"It still checks the scenario's Then, and fails on the code before the change.\n")
	}
	if covers := declaredAt(a.Record); covers != "" {
		b.WriteString("- **Its tests, as declared**: a rewrite keeps these ids, or gives a `covers` entry, " +
			"`- covers: {scenario: <its name>, tests: [<test id>, …]}`, for each scenario whose tests it renames or removes, " +
			"one entry naming every test that covers it now:\n" + covers)
	}
	if a.FiveWhys {
		b.WriteString("- **The five whys**: two answers did not settle this thread. First run the five whys skill below on it; " +
			"give the analysis as its own `note`, and answer as its root cause says: a rewrite when the test is the cause, kept when it is not.\n")
	}
	tail := b.String()
	text := head + tail
	if a.Kind == "interface" {
		var cuts []string
		var lines int
		text, lines = fitTask(runDir, repo, s, func(s Settings) string {
			var code strings.Builder
			cuts = writeCode(&code, repo, is.Title+"\n"+is.Body, work.ByMeaning(is.Body)[work.Scope].Text, nil, nil, s)
			return head + code.String() + tail
		})
		if err := sayCuts(runDir, s, lines, cuts); err != nil {
			return fail(err)
		}
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(text), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// declaredAt lists each scenario of the record and the tests it is
// declared covered by, a line each; "" for a record with none.
func declaredAt(r record) string {
	var b strings.Builder
	for _, sc := range slices.Sorted(maps.Keys(r.Covers)) {
		if len(r.Covers[sc]) > 0 {
			fmt.Fprintf(&b, "  - **%s:** %s\n", sc, codes(r.Covers[sc]))
		}
	}
	return b.String()
}

// answerPost holds the agent's answer to an objection; another's answer,
// written by pre, it only says.
func answerPost(runDir, repo string, s Settings) int {
	var a answer
	if data, err := os.ReadFile(filepath.Join(runDir, "in", answerFile)); err != nil {
		return fail(err)
	} else if err := json.Unmarshal(data, &a); err != nil {
		return fail(err)
	}
	if a.Mode != modeAgent {
		_, v := plainAnswer(a, s)
		return write(runDir, v)
	}
	code := judgeAnswer(runDir, repo, a, s)
	if code == 1 {
		sayRefusal(runDir, a)
	}
	return code
}

// sayRefusal says, in the objection's thread, that the answer was refused
// and why: the verdict's summary and each finding, `rule: message`; written
// to out/on-block.yaml, applied though the run blocks (#1668).
func sayRefusal(runDir string, a answer) {
	v, err := verdict.Read(filepath.Join(runDir, "out", "verdict.yaml"))
	if err != nil || v.Status != verdict.Block {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**The answer is refused**: %s.", strings.TrimSuffix(v.Summary, "."))
	for _, f := range v.Findings {
		fmt.Fprintf(&b, "\n- `%s`: %s", f.Rule, f.Message)
	}
	b.WriteString("\n\nThe objection goes on: reply in this thread.")
	if err := intent.Write(filepath.Join(runDir, "out", "on-block.yaml"), []intent.Intention{a.reply(b.String())}); err != nil {
		fmt.Fprintln(os.Stderr, "tester:", err)
	}
}

// reply is the tester's reply in the objection's thread.
func (a answer) reply(text string) intent.Intention {
	return intent.Intention{Kind: "comment", Value: map[string]any{"thread": a.Thread, "body": text + "\n\n" + testmark.AnswerMark}}
}

// left says the objections left for the next runs.
func (a answer) left() string {
	if a.Waiting == 0 {
		return ""
	}
	return fmt.Sprintf("; %d more objection%s for the next runs", a.Waiting, plural(a.Waiting))
}

// plainAnswer is an answer no agent gives, the engine's acts, which pre
// leaves in in/fallback.yaml, and its verdict: the form asked again, a
// person asked, the need sent to the product owner.
func plainAnswer(a answer, s Settings) ([]intent.Intention, verdict.Verdict) {
	switch a.Mode {
	case modeMalformed:
		return []intent.Intention{{Kind: "comment", Value: map[string]any{"thread": a.Thread, "body": fmt.Sprintf("**Not answered**: %s. An objection opens with `%s`, then `test:`, `scenario:` (its name in #%d), `kind:` (%s) and `why:`, one a line; reply in this thread with the ones to fix.\n\n%s",
				a.Why, testmark.ObjectionWord, a.Issue, kindNames(), testmark.UnreadMark)}}},
			verdict.Verdict{Status: verdict.Pass, Summary: "an objection the tester cannot read, said in its thread" + a.left(),
				Findings: []verdict.Finding{{Rule: "objection-unreadable", Level: "warn", Where: a.URL, Message: a.Why}}}
	case modeSettle:
		quoted := strings.ReplaceAll(a.Objection.Why, "\n", "\n> ")
		return append([]intent.Intention{
				{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "objection-" + a.Thread, "body": backlog.Said("The tester",
					fmt.Sprintf("asks the product owner to settle an objection to its test, scenario **%s**, merge request %d ([the thread](%s))", a.Scenario.Name, a.MR, a.URL),
					fmt.Sprintf("%d answers, and the builder still objects: \"%s\"", a.Answers, quoted),
					"the product owner amends the scenario, or keeps it and says why; the tester then keeps or adapts its test. Objected to again after that, the issue is blocked")}},
				a.reply(fmt.Sprintf("**The product owner settles it**: %d answers, and the objection still stands. #%d goes back to the product owner, which amends the scenario or keeps it; objected to again after that, #%d is blocked.\n\n%s",
					a.Answers, a.Issue, a.Issue, testmark.SettleMark))}, a.move(a.refineState())...),
			verdict.Verdict{Status: verdict.Pass, Summary: "an objection past the tester's answers: the product owner settles it" + a.left(),
				Findings: []verdict.Finding{{Rule: "objection-to-the-product-owner", Level: "warn", Where: a.URL,
					Message: fmt.Sprintf("%d answers did not settle it: #%d back to the product owner", a.Answers, a.Issue)}}}
	case modeBlocked:
		return append([]intent.Intention{
				{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "objection-" + a.Thread, "body": backlog.Said("The tester",
					fmt.Sprintf("blocks this issue: an objection to its test, scenario **%s**, merge request %d ([the thread](%s)), still stands", a.Scenario.Name, a.MR, a.URL),
					"the builder objects again after the product owner settled it: the roles do not agree",
					"a person on the loop reads the thread, decides in it, and moves the issue back by its label")}},
				a.reply(fmt.Sprintf("**Blocked**: objected to again after the product owner settled it. #%d is blocked, its reason on it; a person of the project decides in this thread.\n\n%s", a.Issue, testmark.BlockedMark))},
				a.move(BlockedState)...),
			verdict.Verdict{Status: verdict.Pass, Summary: "an objection past the product owner: the issue blocked" + a.left(),
				Findings: []verdict.Finding{{Rule: "objection-blocked", Level: "warn", Where: a.URL, Message: fmt.Sprintf("objected to again after the product owner settled it: #%d blocked", a.Issue)}}}
	}
	return []intent.Intention{
			{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "objection-" + a.Thread, "body": fmt.Sprintf(
				"**The builder objects to the need itself**, scenario **%s**, on its test in merge request %d ([the thread](%s)):\n\n> %s\n\n"+
					"Back to refine: the product owner amends the scenario, a person accepts it; the tester then adapts its test.",
				a.Scenario.Name, a.MR, a.URL, strings.ReplaceAll(a.Objection.Why, "\n", "\n> "))}},
			a.move(a.refineState())[0],
			a.reply(fmt.Sprintf("**Sent to the product owner**: the need itself is theirs, not the tester's. #%d is back to refine; the tester adapts this test once the scenario is amended and ready again.", a.Issue))},
		verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("an objection to the need: #%d back to the product owner", a.Issue) + a.left()}
}

// judgeAnswer holds the agent's answer: a reason always; a rewrite of the
// objected test only, whole, naming its scenario, red on the base.
func judgeAnswer(runDir, repo string, a answer, s Settings) int {
	reply, left := a.reply, a.left()
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	if v := enginesOnly(intents); v != nil {
		return write(runDir, *v)
	}
	var patches []intent.Intention
	var notes, analyses []string
	for _, in := range intents {
		switch in.Kind {
		case "patch":
			patches = append(patches, in)
		case "note":
			if text := noteText(in.Value); a.FiveWhys && whys.Is(text) {
				analyses = append(analyses, text)
			} else {
				notes = append(notes, text)
			}
		}
	}
	block := func(rule, msg string) int {
		// Said in the thread too, where the builder looks (#1669): the
		// engine posts on-block.yaml when the run is blocked.
		// One sticky comment, whatever the reason: a second refusal updates
		// it, it adds no reply (#1882).
		said := intent.Intention{Kind: "comment", Value: map[string]any{"sticky": "refused",
			"body": "**The tester's answer is refused** (`" + rule + "`): " + msg + "\n\n" + testmark.AnswerMark}}
		if err := intent.Write(filepath.Join(runDir, "out", "on-block.yaml"), []intent.Intention{said}); err != nil {
			return fail(err)
		}
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the tester's answer is refused",
			Findings: []verdict.Finding{{Rule: rule, Where: a.File, Message: msg}}})
	}
	if len(intents) == 0 {
		if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no agent: the objection waits for the tester's answer" + left,
				Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: a.URL,
					Message: "keeping or rewriting a test against its scenario is judgement: with no agent, nothing is answered; the tester runs with one (`--ai`), or a person answers"}}})
		}
	}
	reason := strings.TrimSpace(strings.Join(notes, "\n\n"))
	if reason == "" {
		return block("no-reason", "the answer gives no reason, a `note`: kept or rewritten, the builder is told why")
	}
	why := ""
	if a.FiveWhys {
		if len(analyses) != 1 {
			return block("five-whys-missing", fmt.Sprintf("two answers did not settle this thread: the answer gives the five whys as one `note` opening with `Five whys`, as the skill says (%d given)", len(analyses)))
		}
		w, err := whys.Parse(analyses[0])
		if err != nil {
			return block("five-whys-unreadable", "the five whys do not follow the skill's form: "+err.Error())
		}
		why = "\n\n" + w.Markdown()
	}
	at, _ := git(repo, "rev-parse", "HEAD")
	missing := a.Kind == "missing"
	act := testmark.Answered{Act: "kept", At: strings.TrimSpace(at), Model: asked(runDir), Scenario: a.Scenario, Test: a.File, Whys: a.FiveWhys, Missing: missing}
	if len(patches) == 0 {
		word := "**Kept**: "
		if missing {
			word = "**None needed**: "
		}
		// A note naming the code it lacks is no answer that the test
		// holds, nor that none is needed (#1416): said as lacking.
		_, _, lacks := needsOf(notes)
		if lacks {
			word, act.Act = "**Lacks code**: ", "lacks-code"
		}
		out := []intent.Intention{reply(word + reason + why + "\n" + act.Line())}
		out = append(out, a.toReview(word+reason)...)
		if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), out); err != nil {
			return fail(err)
		}
		if lacks {
			fs := []verdict.Finding{{Rule: "lacks-code", Level: "warn", Where: a.URL, Message: reason}}
			// The files it names, of the repository: the line gives them (#1443).
			if paths, _, _ := needsOf(notes); len(paths) > 0 {
				var known []string
				for _, p := range paths {
					clean := path.Clean(strings.TrimPrefix(p, "./"))
					if clean == "." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) || slices.Contains(known, clean) {
						continue
					}
					if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(clean))); err == nil {
						known = append(known, clean)
					}
				}
				if len(known) > 0 {
					fs = append(fs, verdict.Finding{Rule: verdict.NeedsCode, Level: "info", Where: a.URL, Message: strings.Join(known, ", ")})
				}
			}
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no test written: it lacks code, said in the thread" + whysSaid(a) + left, Findings: fs})
		}
		if missing {
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no test needed, the reason in the thread" + whysSaid(a) + left})
		}
		return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "the test is kept, the reason in the thread" + whysSaid(a) + left})
	}
	for i, p := range patches {
		files := intent.PatchFiles(p.Value)
		if diff, ok := p.Value.(string); ok && missing && len(files) == 1 && a.Record.Files[files[0]] != "" {
			// A test added is a diff by nature: laid on the file, whole.
			content, err := laidWhole(repo, files[0], diff)
			if err != nil {
				return block("patch-unreadable", "the test added does not apply to `"+files[0]+"`: "+err.Error())
			}
			p = intent.Intention{Kind: "patch", Value: map[string]any{"file": files[0], "content": content}}
			patches[i] = p
		}
		if missing && i == 0 && len(files) == 1 && a.Record.Files[files[0]] != "" {
			a.File = files[0] // a test missing goes in any of its tests, one: the first patch's
			act.Test = a.File
		}
		if missing && i == 0 && len(files) == 1 && a.Record.Files[files[0]] == "" && inTestFolder(a.Record, files[0]) {
			// A new file is new: a file already on the branch, not the
			// tester's, is never overwritten by the whole-content patch.
			if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(files[0]))); err == nil {
				return block("rewrite-other-file", "a test missing goes in a new file: `"+files[0]+"` exists already and is not one of the tester's tests")
			}
			// Tests are one case a file: a test missing is a new file, in a
			// folder holding one of the tester's tests (#1668).
			a.File = files[0]
			act.Test = a.File
		}
		if len(files) != 1 || files[0] != a.File {
			if missing {
				return block("rewrite-other-file", "a test missing goes in one of the tester's tests of the issue, one file: not "+strings.Join(files, ", "))
			}
			return block("rewrite-other-file", "a rewrite changes the objected test alone, `"+a.File+"`: not "+strings.Join(files, ", "))
		}
		if _, ok := p.Value.(map[string]any); !ok {
			return block("rewrite-not-whole", "a rewrite gives the test whole, `file` and `content`, never a diff")
		}
	}
	if line, ok := baseLineChanged(repo, a.Record.Base, a.File, patches[len(patches)-1]); !ok {
		// A file the tests share with the base's: its lines stay (#1259).
		return block("test-line-changed", "the rewrite changes or removes "+line+" of this file as the base holds it: a rewrite changes the tester's own lines alone")
	}
	s.issue = a.Text
	entries, _ := coversOf(intents)
	s.guards, _ = guardsOf(entries) // a guard rewritten, its entry saying so, is green on the base
	proof, newTests, blobs, asked, err := proveJudged(runDir, repo, a.Record.Base, patches[len(patches)-1:], s)
	if err != nil {
		return fail(err)
	}
	if asked { // the engine asks the judge, and runs post again with its answer
		return write(runDir, *proof)
	}
	if proof.Status == verdict.Pass && len(newTests) == 0 {
		proof.Status = verdict.Block
		proof.Findings = append(proof.Findings, verdict.Finding{Rule: "no-new-test", Where: s.Report, Message: "the report names no test of the rewrite: it is not proven red"})
	}
	if proof.Status != verdict.Pass {
		proof.Summary = "the rewrite is not red on the base: " + proof.Summary
		return write(runDir, *proof)
	}
	act.Act, act.Blob = "rewritten", blobs[a.File]
	r := a.Record.rewritten(a.File, blobs[a.File], *proof).retested(repo, a.File, newTests, erroredOf(runDir, newTests), s)
	// Its scenario still has its tests, as declared: never one guessed
	// from a name (ADR-0065, change 4).
	if why := r.recovered(intents, newTests, a.File); why != "" {
		return block("scenario-without-test", why)
	}
	word := "**Rewritten**"
	if missing {
		word = "**Added** in `" + a.File + "`"
	}
	out := []intent.Intention{patches[len(patches)-1],
		reply(word + ", the tester's own commit on this branch, red on the base: " + reason + why + unprovenSaid(*proof) + "\n" + act.Line()),
		{Kind: "comment", Value: r.intention()["comment"]}}
	out = append(out, a.toReview(word+": "+reason)...)
	for _, in := range intents {
		if in.Kind == "note" && (!a.FiveWhys || !whys.Is(noteText(in.Value))) {
			out = append(out, in) // the five whys are in the thread already
		}
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), out); err != nil {
		return fail(err)
	}
	data, err := yaml.Marshal(map[string]any{"push": objectionCommit(a.File, a.URL, missing, committerSettings(repo))})
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	done := "the test is rewritten"
	if missing {
		done = "the test missing is added"
	}
	proof.Summary = done + ", red on the base: " + proof.Summary + whysSaid(a) + left
	return write(runDir, *proof)
}

// subjectMax is the committer's subject limit as it ships: a subject
// past it is refused, and the developer, which only adds commits, cannot
// clear a refusal on the tester's (#1941).
const subjectMax = 72

// committerSettings are the committer's settings for the project: its
// subject limit and allowed types. Unread, the role's own as it ships
// them, else the limit alone.
func committerSettings(repo string) commitmsg.Settings {
	r, err := role.Load(os.Getenv("WORKLINE_ROLES_DIR"), "committer")
	if err != nil {
		return commitmsg.Settings{SubjectMax: subjectMax}
	}
	own, _ := commitmsg.SettingsFrom(r.MergedSettings(&role.ProjectConfig{}))
	if own.SubjectMax <= 0 {
		own.SubjectMax = subjectMax
	}
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return own
	}
	s, err := commitmsg.SettingsFrom(r.MergedSettings(cfg))
	if err != nil {
		return own
	}
	if s.SubjectMax <= 0 {
		s.SubjectMax = subjectMax
	}
	return s
}

// objectionCommit is the message of the tester's commit on an
// objection: the file's base name in the subject, the subject within
// the committer's limit whatever the path, of a type it allows, the full
// path in the body.
func objectionCommit(file, url string, missing bool, cs commitmsg.Settings) string {
	typ := "test"
	if len(cs.Types) > 0 && !slices.Contains(cs.Types, typ) {
		typ = cs.Types[0]
	}
	head := typ + ": "
	s, short := head+"rewrite "+path.Base(file)+" on an objection", head+"rewrite a test on an objection"
	if missing {
		s, short = head+"add a test the review asks to "+path.Base(file), head+"add a test the review asks for"
	}
	limit := cs.SubjectMax
	if limit <= 0 {
		limit = subjectMax
	}
	if utf8.RuneCountInString(s) > limit {
		s = short
	}
	if utf8.RuneCountInString(s) > limit {
		var cut string
		for _, w := range strings.Fields(s) {
			next := strings.TrimSpace(cut + " " + w)
			if utf8.RuneCountInString(next) > limit {
				break
			}
			cut = next
		}
		if cut == "" || utf8.RuneCountInString(cut) <= utf8.RuneCountInString(head) {
			cut = string([]rune(s)[:limit])
		}
		s = cut
	}
	return s + "\n\nFile: " + file + "\n\nRefs " + url
}

// laidWhole is file as diff leaves it, laid on HEAD on an index of its
// own: the checkout is not touched.
func laidWhole(repo, file, diff string) (string, error) {
	tmp, err := os.MkdirTemp("", "workline-laid-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	run := func(stdin string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env, cmd.Stdin = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index")), strings.NewReader(stdin)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
		}
		return string(out), nil
	}
	if _, err := run("", "read-tree", "HEAD"); err != nil {
		return "", err
	}
	if _, err := run(intent.NormalizeDiff(diff), "apply", "--cached", "--recount", "-"); err != nil {
		return "", err
	}
	return run("", "show", ":"+file)
}

// toReview is a test missing's answer said on the pull request whose
// review found it (#1265), where the review and the developer read it;
// none for another objection, or a why naming no pull request.
func (a answer) toReview(text string) []intent.Intention {
	if a.Kind != "missing" || a.From == 0 {
		return nil
	}
	return []intent.Intention{{Kind: "comment", Value: map[string]any{"merge-request": a.From, "sticky": testmark.TesterAnswerKey + a.Thread,
		"body": reviewmark.TesterAnswer(a.Found, a.Title, a.URL, text)}}}
}

// otherFiles are the record's tests but file, sorted, each in code.
func otherFiles(r record, file string) []string {
	var out []string
	for _, f := range r.files() {
		if f != file {
			out = append(out, "`"+f+"`")
		}
	}
	if len(out) == 0 {
		return []string{"none"}
	}
	return out
}

// rewritten is the record once file is rewritten to blob, proven by
// proof: a rewrite red unproven says why (ADR-0065); one proven keeps
// what an earlier proof said, the other tests' proof not run again.
func (r record) rewritten(file, blob string, proof verdict.Verdict) record {
	files := map[string]string{}
	for f, b := range r.Files {
		files[f] = b
	}
	files[file] = blob
	r.Files = files
	if why := unprovenOf(proof); why != "" {
		r.RedUnproven = why
	}
	return r
}

// retested is the record once file is rewritten, its new tests added: the
// file's old tests come off its Tests, those named file:name and those a
// run of the base with the file as frozen (HEAD's) reports. That run
// failing, the record keeps no test ids, said by the green proof: never
// a test required that the rewrite renamed.
func (r record) retested(repo, file string, added, errored []string, s Settings) record {
	if len(r.Tests) == 0 {
		return r
	}
	head, err := git(repo, "rev-parse", "--verify", "HEAD")
	if err != nil {
		r.Tests, r.Errored = nil, nil
		return r
	}
	old, f := run(repo, r.Base, []string{file}, strings.TrimSpace(head), s, "on the base with "+file+" as frozen")
	if f != nil && f.Rule != "report-missing" { // no report: a build broken, its tests named file:name
		r.Tests, r.Errored = nil, nil
		return r
	}
	return r.withTests(file, old.results, added, errored)
}

// withTests is r, file's old tests (those old reports, or named file:name)
// replaced by added, errored those of added the red run wrote with no
// class, failed.
func (r record) withTests(file string, old map[string]result, added, errored []string) record {
	var tests []string
	for _, t := range r.Tests {
		if _, ok := old[t]; !ok && !strings.HasPrefix(t, file+":") {
			tests = append(tests, t)
		}
	}
	r.Tests = slices.Compact(slices.Sorted(slices.Values(append(tests, added...))))
	kept := slices.DeleteFunc(append(slices.Clone(r.Errored), errored...), func(t string) bool { return !slices.Contains(r.Tests, t) })
	r.Errored = slices.Compact(slices.Sorted(slices.Values(kept)))
	return r
}

// unprovenSaid is the reply's line on a rewrite the judge did not settle.
func unprovenSaid(proof verdict.Verdict) string {
	if why := unprovenOf(proof); why != "" {
		return "\n\n**Red unproven**: " + why
	}
	return ""
}

// whysSaid says the five whys were posted with the answer.
func whysSaid(a answer) string {
	if !a.FiveWhys {
		return ""
	}
	return ", the five whys with it"
}

// unwrap strips one pair of markdown markers wrapping the whole value,
// bold, italic or code; never a marker that only ends the text, nor the
// outer two of several spans.
func unwrap(v string) string {
	for _, m := range []string{"**", "*", "`"} {
		if len(v) > 2*len(m) && strings.HasPrefix(v, m) && strings.HasSuffix(v, m) {
			if in := v[len(m) : len(v)-len(m)]; !strings.Contains(in, m) {
				return strings.TrimSpace(in)
			}
			return v
		}
	}
	return v
}

// BlockedState is the state an issue the roles do not agree on goes to
// (ADR-0059, decision 6).
const BlockedState = "blocked"

// refineState is the product owner's state in the routing's table, as
// pre read it: refine, shipped.
func (a answer) refineState() string {
	for _, st := range a.Line {
		if st.Role == "product-owner" {
			return st.Name
		}
	}
	return "refine"
}

// move is the label moving the objection's issue to the state named, its
// other states' labels off (the routing's table); the shipped labels when
// the table is not read.
func (a answer) move(to string) []intent.Intention {
	add, remove := []string{backlog.LabelToRefine}, []string{backlog.LabelReady, backlog.LabelNeedsTests, backlog.LabelBlocked}
	if to == BlockedState {
		add, remove = []string{backlog.LabelBlocked}, []string{backlog.LabelReady, backlog.LabelNeedsTests, backlog.LabelToRefine, "workline:developing"}
	}
	if ad, rm, err := (&routing.Config{States: a.Line}).Move(to); err == nil {
		add, remove = ad, rm
	}
	return []intent.Intention{{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": add, "remove": remove}}}
}

// lacked says whether a comment is the tester's answer that it lacks code.
func lacked(body string) bool {
	a, ok := testmark.ReadAnswered(body)
	return ok && a.Act == "lacks-code" && testmark.IsAnswered(body)
}

// inTestFolder says whether file sits in a folder holding one of the
// record's tests.
func inTestFolder(r record, file string) bool {
	dir := path.Dir(file)
	for f := range r.Files {
		if path.Dir(f) == dir {
			return true
		}
	}
	return false
}
