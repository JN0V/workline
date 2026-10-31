package tester

// The objection (ADR-0055, a test that does not fit): the builder comments
// on a test's line, opened by `workline:objection`, naming the test, its
// scenario, the kind and why. A fresh tester run reads the scenario first
// and answers in the thread: kept, with its reason, or rewritten, its own
// commit referring to the thread, still red on the base. The need itself
// goes to the product owner. Three answers a thread; still open after the
// second, the third run loads the shared five whys skill and posts the
// analysis with its answer; past the third, a person decides.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/whys"
	"github.com/JN0V/workline/internal/work"
)

// ObjectionWord opens an objection's first comment.
const ObjectionWord = "workline:objection"

// The tester's own comments in a thread carry answerMark, a person's turn
// personMark, an objection it could not read unreadMark (the builder's
// turn, not an answer counted): workline's own account may write for
// other roles too.
var (
	answerMark = forge.Marker("tester-answer")
	personMark = forge.Marker("tester-person-decides") // before ADR-0059: read, never written
	// settleMark: past the tester's answers, the product owner settles
	// (ADR-0059, decision 6); blockedMark: past it, the issue Blocked.
	settleMark  = forge.Marker("tester-product-owner-settles")
	blockedMark = forge.Marker("tester-objection-blocked")
	unreadMark  = forge.Marker("tester-unreadable")
)

// The kinds an objection names, as the ADR words them.
var kinds = []struct{ key, name, word string }{
	{"interface", "wrong interface", "interface"},
	{"over-specified", "over-specified", "over"},
	{"contradicts", "contradicts another scenario", "contradict"},
	{"untestable", "untestable as written", "untestable"},
	{"need", "the need itself", "need"},
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
	text = strings.TrimSpace(strings.TrimPrefix(text, ObjectionWord))
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
	if o.Scenario == "" {
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
	Mode      string    `json:"mode"`
	Why       string    `json:"why,omitempty"` // a malformed objection's reason
	MR        int       `json:"merge-request"`
	Issue     int       `json:"issue"`
	Thread    string    `json:"thread"`
	URL       string    `json:"url"`
	File      string    `json:"file"`
	Kind      string    `json:"kind,omitempty"`
	Scenario  Scenario  `json:"scenario"`
	Answers   int       `json:"answers"`   // the tester's answers in the thread already
	FiveWhys  bool      `json:"five-whys"` // this answer is the one the five whys are due on
	Decided   string    `json:"decided"`   // a person's decision, past the bound
	Record    record    `json:"record"`
	Waiting   int       `json:"waiting"` // the objections left for the next runs
	Objection objection `json:"objection"`
	// Line: the routing's states, as pre read them: where the issue goes
	// to settle or to block (ADR-0059).
	Line []routing.State `json:"line,omitempty"`
}

const answerFile = "answer.json"

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
	issue := issueOf(mr.Branch)
	if issue == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "not a tester's branch",
			Findings: []verdict.Finding{{Rule: "not-a-testers-branch", Where: where,
				Message: "it comes from " + mr.Branch + ": the tester answers objections to its own tests, on " + branchPrefix + "<n>"}}})
	}
	th, ok := f.(forge.Threaded)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "this forge lists no conversation on a line",
			Findings: []verdict.Finding{{Rule: "no-threads", Where: where, Message: "the objections are read from the merge request's comments on lines, which this forge does not list"}}})
	}
	r, missing, err := readRecord(f, id)
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
	own, err := forge.OwnAccount(f)
	if err != nil {
		return fail(err)
	}
	mine := func(n forge.Note, mark string) bool { return own.Wrote(n) && strings.Contains(n.Body, mark) }
	// read counts the tester's answers in a thread; once the product owner
	// was asked to settle it, a person's decision there, else whether the
	// builder objected again.
	read := func(t forge.Thread) (answers int, decided string, settled, again bool) {
		for _, c := range t.Comments {
			switch {
			case mine(c, settleMark), mine(c, personMark):
				settled = true
			case mine(c, answerMark):
				answers++
			case settled && c.Insider && !c.Bot:
				decided = c.Body // a person of the project, on the loop, after it
			case settled && !mine(c, blockedMark):
				again = true // the builder still objects, past the product owner
			}
		}
		return answers, decided, settled, again
	}
	var waiting []forge.Thread
	for _, t := range threads {
		if t.Resolved || len(t.Comments) == 0 || !strings.HasPrefix(strings.TrimSpace(t.Comments[0].Body), ObjectionWord) {
			continue
		}
		if last := t.Comments[len(t.Comments)-1]; mine(last, answerMark) || mine(last, unreadMark) || mine(last, settleMark) ||
			mine(last, personMark) || mine(last, blockedMark) {
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
	for _, c := range t.Comments[1:] {
		if own.Wrote(c) {
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
	scenarios := Scenarios(work.ByMeaning(is.Body)[work.Validation].Text)
	for _, sc := range scenarios {
		if o.Scenario != "" && norm(sc.Name) == norm(o.Scenario) {
			a.Scenario = sc
		}
	}
	switch {
	case a.Decided != "":
		a.Mode = modeAgent
	case settled && again:
		a.Mode = modeBlocked
	case a.Answers >= exchanges:
		a.Mode = modeSettle
	case len(unread) > 0:
		a.Mode, a.Why = modeMalformed, "it names no "+strings.Join(unread, ", no ")
	case r.Files[t.Path] == "":
		a.Mode, a.Why = modeMalformed, "its line is in `"+t.Path+"`, not one of the tester's tests ("+strings.Join(r.files(), ", ")+")"
	case a.Scenario.Name == "":
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
	fmt.Fprintf(&b, "# An objection to a test of #%d: %s\n\n", a.Issue, is.Title)
	fmt.Fprintf(&b, "## The scenario, a person's\n\n- **%s:** %s\n\n", a.Scenario.Name, a.Scenario.Text)
	fmt.Fprintf(&b, "## The objection: %s\n\n", kindName(a.Kind))
	for i, c := range t.Comments {
		who := "the builder"
		switch {
		case strings.Contains(c.Body, answerMark), strings.Contains(c.Body, unreadMark):
			who = "you, the tester"
		case strings.Contains(c.Body, personMark), strings.Contains(c.Body, settleMark):
			who = "the engine, asking the product owner to settle it"
		case strings.Contains(c.Body, blockedMark):
			who = "the engine, blocking the issue"
		case c.Insider && !c.Bot && a.Decided == c.Body:
			who = "a person of the project, deciding"
		}
		body := strings.TrimSpace(strings.NewReplacer(answerMark, "", personMark, "", settleMark, "", blockedMark, "", unreadMark, "").Replace(c.Body))
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
	if a.Kind == "interface" {
		writeCode(&b, repo, work.ByMeaning(is.Body)[work.Scope].Text, s)
	}
	b.WriteString("\n## Your answer\n\n" +
		"- **Kept**: a `note`, the reason the test holds for the scenario.\n" +
		"- **Rewritten**: the same file whole as a `patch` with `file` and `content`, and a `note`, why. " +
		"It still checks the scenario's Then, names the scenario, and fails on the code before the change.\n")
	if a.FiveWhys {
		b.WriteString("- **The five whys**: two answers did not settle this thread. First run the five whys skill below on it; " +
			"give the analysis as its own `note`, and answer as its root cause says: a rewrite when the test is the cause, kept when it is not.\n")
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(b.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
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
	return judgeAnswer(runDir, repo, a, s)
}

// reply is the tester's reply in the objection's thread.
func (a answer) reply(text string) intent.Intention {
	return intent.Intention{Kind: "comment", Value: map[string]any{"thread": a.Thread, "body": text + "\n\n" + answerMark}}
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
				a.Why, ObjectionWord, a.Issue, kindNames(), unreadMark)}}},
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
					a.Answers, a.Issue, a.Issue, settleMark))}, a.move(a.refineState())...),
			verdict.Verdict{Status: verdict.Pass, Summary: "an objection past the tester's answers: the product owner settles it" + a.left(),
				Findings: []verdict.Finding{{Rule: "objection-to-the-product-owner", Level: "warn", Where: a.URL,
					Message: fmt.Sprintf("%d answers did not settle it: #%d back to the product owner", a.Answers, a.Issue)}}}
	case modeBlocked:
		return append([]intent.Intention{
				{Kind: "comment", Value: map[string]any{"issue": a.Issue, "sticky": "objection-" + a.Thread, "body": backlog.Said("The tester",
					fmt.Sprintf("blocks this issue: an objection to its test, scenario **%s**, merge request %d ([the thread](%s)), still stands", a.Scenario.Name, a.MR, a.URL),
					"the builder objects again after the product owner settled it: the roles do not agree",
					"a person on the loop reads the thread, decides in it, and moves the issue back by its label")}},
				a.reply(fmt.Sprintf("**Blocked**: objected to again after the product owner settled it. #%d is blocked, its reason on it; a person of the project decides in this thread.\n\n%s", a.Issue, blockedMark))},
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
	act := Answered{Act: "kept", At: strings.TrimSpace(at), Model: asked(runDir), Scenario: a.Scenario, Test: a.File, Whys: a.FiveWhys}
	if len(patches) == 0 {
		out := []intent.Intention{reply("**Kept**: " + reason + why + "\n" + act.line())}
		if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), out); err != nil {
			return fail(err)
		}
		return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "the test is kept, the reason in the thread" + whysSaid(a) + left})
	}
	var content string
	for _, p := range patches {
		files := intent.PatchFiles(p.Value)
		if len(files) != 1 || files[0] != a.File {
			return block("rewrite-other-file", "a rewrite changes the objected test alone, `"+a.File+"`: not "+strings.Join(files, ", "))
		}
		m, ok := p.Value.(map[string]any)
		if !ok {
			return block("rewrite-not-whole", "a rewrite gives the test whole, `file` and `content`, never a diff")
		}
		content, _ = m["content"].(string)
	}
	if !names(content, a.Scenario.Name) {
		return block("scenario-without-test", "the rewritten test no longer names its scenario, "+a.Scenario.Name)
	}
	proof, newTests, blobs, err := proveOnBase(repo, a.Record.Base, patches[len(patches)-1:], s)
	if err != nil {
		return fail(err)
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
	r := a.Record
	files := map[string]string{}
	for f, b := range r.Files {
		files[f] = b
	}
	files[a.File] = blobs[a.File]
	r.Files = files
	out := []intent.Intention{patches[len(patches)-1],
		reply("**Rewritten**, the tester's own commit on this branch, red on the base: " + reason + why + "\n" + act.line()),
		{Kind: "comment", Value: r.intention()["comment"]}}
	for _, in := range intents {
		if in.Kind == "note" && (!a.FiveWhys || !whys.Is(noteText(in.Value))) {
			out = append(out, in) // the five whys are in the thread already
		}
	}
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), out); err != nil {
		return fail(err)
	}
	data, err := yaml.Marshal(map[string]any{"push": fmt.Sprintf("test: rewrite %s on an objection\n\nRefs %s", a.File, a.URL)})
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	proof.Summary = "the test is rewritten, red on the base: " + proof.Summary + whysSaid(a) + left
	return write(runDir, *proof)
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
