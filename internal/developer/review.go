package developer

// The review loop (ADR-0054, step 6): the same developer fixes the
// review's findings on its own pull request. "Same" is an identity and a
// written trace, never a live process: each reviewer verdict, or a
// person's comment, starts a fresh run that reads the pull request's
// record, the reviewer's comment and the issue, and nothing else. The
// findings that block and the Important ones are asked; nits are said, not
// asked. Its fix is held to the issue's bounds and proven on the project's
// tests, then pushed on its branch. At round three, a finding still there
// loads the shared five whys skill; past the last round, a person takes
// the pull request. Each round is kept in the record, for the next run and
// the auditor (ADR-0046).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/whys"
	"github.com/JN0V/workline/internal/work"
)

const roundFile = "round.json"

// The reviewer's comments a round reads its findings from (its summary,
// and the comment of its lenses that warn when the review is split), and
// the count of its runs their records hold.
var (
	reviewerMarkers = []string{forge.Marker("sticky=reviewer/summary"), forge.Marker("sticky=reviewer/warn")}
	reviewerRuns    = regexp.MustCompile(`<!-- workline:record reviewed=\S*(?: patches=\S*)? runs=(\d+)`)
	// reviewerAt: the head its summary read. A run its rules hold counts
	// no run, its findings new all the same (#1040).
	reviewerAt = regexp.MustCompile(`<!-- workline:at head=([0-9a-f]+)`)
	// reviewerFindings: the hidden record of its findings, a JSON list of
	// lens, holds, where, severity and text (the reviewer's reviewer.go).
	reviewerFindings = regexp.MustCompile(`<!-- workline:findings (\[.*\]) -->`)
	// reviewerOwed: its summary's record owes every lens, its rules alone
	// ran (the reviewer's record.go writes it).
	reviewerOwed = regexp.MustCompile(`<!-- workline:record reviewed=\S*(?: patches=\S*)? runs=\d+ owed=all`)
)

// Round is one round of the review loop, as the record keeps it.
type Round struct {
	N      int      `json:"n"`
	Asked  []string `json:"asked,omitempty"`  // the findings asked, each as its row reads
	Heard  int      `json:"heard,omitempty"`  // a person's comments read
	Nits   int      `json:"nits,omitempty"`   // nits said, not asked
	Result string   `json:"result,omitempty"` // its proof's result; no-code when it wrote none
	Whys   bool     `json:"whys,omitempty"`   // the five whys ran
	Person bool     `json:"person,omitempty"` // past the last round: its issue blocked (ADR-0059)
	// Bounds, a round with no fix: held, nothing to wait for; blocked, a
	// project check stays red, or its fix was held off a green head:
	// blocked for a person (#938); objected, its objection waits for the
	// tester (#987). Asked, in a record before ADR-0065's change 2: it
	// asked the product owner for files outside the Scope (#921).
	// Result not-pushed: its fix, red on a green head, kept off it.
	Bounds string         `json:"bounds,omitempty"`
	Turns  int            `json:"turns,omitempty"`  // what its agent used, as it reported them
	Tokens int            `json:"tokens,omitempty"` // fresh, what ai-max-tokens holds
	Cached int            `json:"cached,omitempty"` // read from cache, apart
	Tools  map[string]int `json:"tools,omitempty"`
	// Beside: the new test files its fix wrote beside the Scope
	// (ADR-0063, §9).
	Beside []string `json:"beside,omitempty"`
	// Beyond: the files its fix wrote beyond the Scope, each with its
	// agent's reason (ADR-0065, change 2).
	Beyond []Beyond `json:"beyond,omitempty"`
	// Tester: the review's test findings it sent to the tester (#1265).
	Tester []string `json:"tester,omitempty"`
	// Merge: a round that merged the base in (ADR-0067): settled, or
	// failed, nothing pushed.
	Merge string `json:"merge,omitempty"`
}

// Asked is a finding a round asks the agent to fix: a reviewer's row that
// blocks, or warns and is Important, or a person's comment.
type Asked struct {
	Key   string `json:"key"` // lens, file, title: the same finding round after round
	Lens  string `json:"lens,omitempty"`
	Where string `json:"where,omitempty"`
	Text  string `json:"text"`
	Holds string `json:"holds"` // blocks, warns, or a person
	// Detail is the finding whole, from the reviewer's comment on its
	// line: its row shows the title alone.
	Detail string `json:"detail,omitempty"`
	// Check: a project check red, whose log the round reads (#917).
	Check *forge.HeadCheck `json:"check,omitempty"`
	// Waited: asked again from the record's last round (#1022): the
	// developer's still, a test finding too.
	Waited bool `json:"waited,omitempty"`
}

// round is what the review run's pre leaves for post.
type round struct {
	MR     int      `json:"merge-request"`
	Branch string   `json:"branch"`
	Record Act      `json:"record"`
	Asked  []Asked  `json:"asked"`
	Heard  []string `json:"heard,omitempty"` // the ids of a person's comments it reads
	// Body: its issue's body as the round reads it (BodyMark), when it
	// changed since the code was written (#1271).
	Body     string `json:"body,omitempty"`
	Nits     int    `json:"nits"`
	Runs     int    `json:"review-runs"`
	ReviewAt string `json:"review-at,omitempty"` // the head the reviewer's summary read
	FiveWhys bool   `json:"five-whys"`
	Person   bool   `json:"person"`          // past the last round: no agent, a person
	Ready    bool   `json:"ready,omitempty"` // its Draft green, nothing asked: marked ready, no agent (#969)
	// Backstop: why its issue's runs reached their backstop, no round run
	// (ADR-0064, §5; backstop.go).
	Backstop string `json:"backstop,omitempty"`
	At       string `json:"at,omitempty"`        // its branch's head, where its fix starts
	GatesAt  string `json:"gates-at,omitempty"`  // the head the gates' findings asked were read at
	ChecksAt string `json:"checks-at,omitempty"` // the head the project's red checks were read at
	// ChecksUnread: why the forge did not say the project's checks.
	ChecksUnread string `json:"checks-unread,omitempty"`
	// Back: the product owner sent the issue back since the record's ask
	// for files outside its Scope (#921): the round resumes, the checks
	// at its head asked again with the new bounds.
	Back bool `json:"back,omitempty"`
	// Taken: the tests the tester rewrote, each its blob, taken into its
	// branch this round (#939); Base, the commit its agent's worktree
	// starts from, they in it; From, the tester's Draft's head.
	Taken map[string]string `json:"taken,omitempty"`
	Base  string            `json:"base,omitempty"`
	From  string            `json:"from,omitempty"`
	// Kept: the tester's answers on its Draft that kept a test, as the
	// round read them (KeptRead): an objection of this round waits for
	// one more (#987).
	Kept     int  `json:"kept,omitempty"`
	KeptRead bool `json:"kept-read,omitempty"`
	// ToTester: the review's test findings it sends to the tester
	// (#1265); TesterOnly, they alone were asked: no agent, no round.
	ToTester   []Asked `json:"to-tester,omitempty"`
	TesterOnly bool    `json:"tester-only,omitempty"`
	Sent       int     `json:"sent,omitempty"` // those the record sent already, not asked again
	// Disputed: those sent that still block, the tester holding no test
	// needed: its issue blocked for a person.
	Disputed []Asked `json:"disputed,omitempty"`
	// RecordUnread: why a reviewer comment's findings were not read: its
	// record missing beside a table, or not parsing (#705). Said, not silent.
	RecordUnread string `json:"record-unread,omitempty"`
	// Merge: the base's commit a merge round merges in (ADR-0067), its
	// files in conflict; Base then the merge with their markers, where
	// its agent's worktree starts.
	Merge     string   `json:"merge,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}

// recordUnread says why a reviewer comment's findings record cannot be
// read, "" when it reads, or when the comment shows no finding either.
func recordUnread(comment string) string {
	m := reviewerFindings.FindStringSubmatch(comment)
	if m == nil {
		if strings.Contains(comment, "|---") {
			return "its comment shows a table but holds no findings record (workline:findings): no finding read from the table's words"
		}
		return ""
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(m[1]), &list); err != nil {
		return "its findings record does not parse: " + err.Error()
	}
	return ""
}

// reviewRows reads the findings of the reviewer's comment from its hidden
// findings record, never from its table's words (#705): what blocks, a
// lens's or a rule's, and what warns and is Important; the nits counted.
// Another that warns asks nothing: left. A comment with no record asks
// nothing.
func reviewRows(comment string) (asked []Asked, nits int) {
	m := reviewerFindings.FindStringSubmatch(comment)
	if m == nil {
		return nil, 0
	}
	var list []struct {
		Lens, Holds, Where, Severity, Text string
	}
	if json.Unmarshal([]byte(m[1]), &list) != nil {
		return nil, 0
	}
	for _, x := range list {
		switch {
		case x.Severity == "Nit":
			nits++
			continue
		case x.Holds == "blocks": // a lens's, or a rule's: a gate's, said bare
		case x.Holds != "warns" || x.Severity != "Important":
			continue
		}
		msg := x.Text
		if x.Severity != "" {
			msg = x.Severity + ": " + msg
		}
		asked = append(asked, reviewFinding(x.Holds, x.Lens, x.Where, msg))
	}
	return asked, nits
}

// reviewFinding is a reviewer's finding asked, its key the same round
// after round: lens, file, title.
func reviewFinding(holds, lens, where, msg string) Asked {
	title := strings.TrimSpace(strings.TrimPrefix(msg, "Important:"))
	if t, _, ok := strings.Cut(title, " — "); ok {
		title = t
	}
	file, _, _ := strings.Cut(where, ":")
	return Asked{Key: lens + "|" + file + "|" + title, Lens: lens, Where: where, Text: msg, Holds: holds}
}

// recordedFinding reads a reviewer's finding as the record's round
// wrote it (texts): "warns tests `a.go:3`: Important: …".
var recordedFinding = regexp.MustCompile("^(blocks|warns) (\\S+) `([^`]*)`: (.+)$")

// waited adds to asked the reviewer's findings the record's last round
// asked and could not fix, when the issue comes back to resume it
// (#1022): the reviewer's comment, read again since on a later head,
// may no longer hold them, and a green head asks no check. Those the
// review asks again are not doubled.
func waited(a Act, asked []Asked) []Asked {
	if len(a.Rounds) == 0 {
		return asked
	}
	for _, t := range a.Rounds[len(a.Rounds)-1].Asked {
		m := recordedFinding.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		f := reviewFinding(m[1], m[2], m[3], m[4])
		f.Waited = true
		if !slices.ContainsFunc(asked, func(x Asked) bool { return x.Key == f.Key }) {
			asked = append(asked, f)
		}
	}
	return asked
}

// lineDetails gives each finding of the reviewer's rows its text whole,
// from the reviewer's comment on its line: the row, when the finding has
// one, shows only its title. Only a comment workline's own account wrote
// is read, on the finding's file, carrying its title; an open one before
// one resolved, the latest first.
func lineDetails(asked []Asked, threads []forge.Thread, wrote func(forge.Note) bool) {
	for i, a := range asked {
		if a.Holds == "a person" || a.Holds == issueHolds || a.Holds == gateHolds || a.Holds == checkHolds || a.Holds == testerHolds || a.Holds == keptHolds {
			continue
		}
		_, title, _ := strings.Cut(strings.TrimPrefix(a.Key, a.Lens+"|"), "|")
		title, _, _ = strings.Cut(title, "; also ") // one row, two lenses: two comments
		file, _, _ := strings.Cut(a.Where, ":")
		head := "**Important: " + title + "**"
		found, open := "", false
		for _, t := range threads {
			if len(t.Comments) == 0 || t.Path != file || open && t.Resolved {
				continue
			}
			c := t.Comments[0]
			if !wrote(c) || !strings.HasPrefix(strings.TrimSpace(c.Body), "**workline reviewer**") || !strings.Contains(c.Body, head) {
				continue
			}
			found, open = c.Body, !t.Resolved
		}
		if found == "" {
			continue
		}
		_, body, _ := strings.Cut(found, head)
		var keep []string
		for _, l := range strings.Split(body, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "<!-- workline:") {
				keep = append(keep, l)
			}
		}
		asked[i].Detail = strings.TrimSpace(strings.Join(keep, "\n"))
	}
}

// developerIssue is the issue of a developer's branch, 0 for another's.
func developerIssue(branch string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(branch, Branch))
	if err != nil || !strings.HasPrefix(branch, Branch) {
		return 0
	}
	return n
}

// reviewPre reads the trace of its pull request and decides the round:
// nothing new, a round for the agent, or a person past the last.
func reviewPre(runDir, repo string, s Settings, id int) int {
	where := fmt.Sprintf("merge request %d", id)
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the review from",
			Findings: []verdict.Finding{{Rule: "no-forge", Where: where, Message: "the developer reads the review on its pull request, on a forge; " + forge.Missing}}})
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the forge lists no comments",
			Findings: []verdict.Finding{{Rule: "no-backlog", Where: where, Message: "the review and its record are comments, which this forge does not list"}}})
	}
	mr, err := f.MergeRequest(id)
	if err != nil {
		return external(err)
	}
	issue := developerIssue(mr.Branch)
	if issue == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "not a developer's pull request",
			Findings: []verdict.Finding{{Rule: "not-a-developers-branch", Where: where,
				Message: "it comes from " + mr.Branch + ": the developer fixes the review of its own pull requests, on " + Branch + "<n>"}}})
	}
	target := forge.Target{Kind: "merge-request", ID: id}
	kept, err := forge.ReadKept(f, target)
	if err != nil {
		return external(err)
	}
	// The trace, never a memory: its record on the pull request.
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	if !ok || !read {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "its record on the pull request cannot be read",
			Findings: []verdict.Finding{{Rule: "record-unread", Where: where,
				Message: "a round reads what the last ones did from the developer's record on its pull request, workline's own comment: none it can read, so no round is counted; a person takes it"}}})
	}
	// The gates read at its branch's head, checked out; a head git
	// cannot read fails the round below.
	head, _ := gitOut(repo, "rev-parse", "HEAD")
	r, newReview, err := readAsked(f, b, kept, repo, a, id, head)
	if errors.Is(err, errOwnAccount) {
		return fail(err)
	}
	if err != nil {
		return external(err)
	}
	r.Branch = mr.Branch
	if a.Handed {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "a person has its pull request: the developer runs no more rounds",
			Findings: []verdict.Finding{{Rule: "review-to-a-person", Level: "info", Where: where, Message: fmt.Sprintf("past round %d, a person of the project takes it from here", a.Round)}}})
	}
	// Its merge refused for a conflict with its base (ADR-0067): the
	// round merges the base in, and asks nothing else.
	if base, due := conflictDue(kept, a, repo, head); due {
		return conflictPre(runDir, repo, s, f, b, mr, a, head, base)
	}
	// The review's test findings on code are the tester's (#1265): its
	// issue and bounds read first, when the review asks one.
	var is *forge.Issue
	var bounds Bounds
	if slices.ContainsFunc(r.Asked, testFinding) {
		if is, bounds, err = roundBounds(f, b, repo, a, issue); err != nil {
			return external(err)
		}
		r.Asked, r.ToTester = splitTests(r.Asked, bounds)
	}
	if len(r.Asked) == 0 && len(r.ToTester) == 0 && len(r.Disputed) > 0 {
		// Blocked on test findings the tester holds needless: no round
		// settles it, a person does (#1265).
		if err := writeIn(runDir, roundFile, r); err != nil {
			return fail(err)
		}
		return writeIntents(runDir, "fallback.yaml", disputedOff(r, repo))
	}
	if len(r.Asked) == 0 && len(r.ToTester) == 0 {
		var kept string
		if mr.Draft {
			// Its Draft made green: marked ready, no agent asked (#969).
			var ready bool
			if ready, kept = draftReady(f, a, head); ready {
				r.Ready = true
				if err := writeIn(runDir, roundFile, r); err != nil {
					return fail(err)
				}
				return writeIntents(runDir, "fallback.yaml", readied(head))
			}
		}
		v := verdict.Verdict{Status: verdict.Pass, Summary: "nothing the review asks: no round"}
		if kept != "" {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "draft-kept", Level: "info", Where: where, Message: "a Draft kept: " + kept})
		}
		if !newReview {
			v.Summary = "nothing new since its last round: it waits for the reviewer's next verdict, the gates' next report, or the project's checks on a new head"
		}
		if r.ChecksUnread != "" {
			v.Findings = append(v.Findings, checksUnread(where, r.ChecksUnread))
		}
		if r.RecordUnread != "" && newReview {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "findings-record-unread", Level: "warn", Where: where,
				Message: "the reviewer's findings were not read, none asked: " + r.RecordUnread})
		}
		if r.Sent > 0 && newReview {
			v.Findings = append(v.Findings, testsSent(where, r.Sent))
		}
		if r.Nits > 0 && newReview {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "nits-not-asked", Level: "info", Where: where,
				Message: fmt.Sprintf("%d nit%s shown by the reviewer, not asked: a nit never holds the merge (ADR-0051)", r.Nits, plural(r.Nits))})
		}
		return final(runDir, v)
	}
	if Backstop != nil {
		// Its issue's runs at their backstop: no round more (ADR-0064, §5).
		said, err := Backstop(f, repo, issue)
		if err != nil {
			return external(err)
		}
		if said != "" {
			r.Backstop = said
			if err := writeIn(runDir, roundFile, r); err != nil {
				return fail(err)
			}
			return writeIntents(runDir, "fallback.yaml", stopped(r, repo, issue))
		}
	}
	if len(r.Asked) == 0 {
		// Test findings alone: the tester's, no agent, no round counted;
		// the rounds never run out on them (#1265).
		r.TesterOnly = true
		if err := writeIn(runDir, roundFile, r); err != nil {
			return fail(err)
		}
		return writeIntents(runDir, "fallback.yaml", toTesterOnly(r, bounds, repo))
	}
	if a.Round >= s.Rounds {
		r.Person = true
		if err := writeIn(runDir, roundFile, r); err != nil {
			return fail(err)
		}
		return writeIntents(runDir, "fallback.yaml", handOff(r, bounds, repo))
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("no agent: %d finding%s wait for a run with one, no round counted", len(r.Asked), plural(len(r.Asked))),
			Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where, Message: "fixing a finding is judgement on code: with no agent, nothing is fixed (principle 5); waiting: " + strings.Join(texts(r.Asked), "; ")}}})
	}
	n := a.Round + 1
	r.FiveWhys = n >= 3 && !a.Whys && slices.ContainsFunc(r.Asked, func(x Asked) bool { return slices.Contains(a.Seen, x.Key) })
	if is == nil {
		if is, bounds, err = roundBounds(f, b, repo, a, issue); err != nil {
			return external(err)
		}
	}
	if !r.KeptRead && bounds.Draft != 0 {
		// The tester's answers so far: an objection of this round waits
		// for one more (#987).
		tests, err := testerKept(f, issue)
		if errors.Is(err, errOwnAccount) {
			return fail(err)
		}
		if err != nil {
			return external(err)
		}
		r.Kept, r.KeptRead = len(tests), true
	}
	// A rewrite is taken as the tester's record names it, none other.
	if r.Asked, r.Taken = takeRewrites(r.Asked, bounds); len(r.Asked) == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "nothing the review asks: no round",
			Findings: []verdict.Finding{{Rule: "rewrite-not-recorded", Level: "warn", Where: where,
				Message: "the tester's branch holds an acceptance test otherwise than this branch, and its record on its Draft does not name it: not taken, the record is the frozen test"}}})
	}
	// Its fix starts from its branch's head, checked out: a head git
	// cannot read fails the round, never counted for nothing.
	if r.At, err = gitOut(repo, "rev-parse", "HEAD"); err != nil {
		return fail(fmt.Errorf("the round's code: git cannot read its branch's head in %s: %w", repo, err))
	}
	on := r.At
	if len(r.Taken) > 0 {
		if r.Base, err = withRewrites(repo, r.At, r.Taken); err != nil {
			return fail(fmt.Errorf("the tester's rewritten tests, laid on its branch's head: %w", err))
		}
		on, r.From = r.Base, bounds.Head
	}
	checkLogs(f, r.Asked)
	if err := writeIn(runDir, roundFile, r); err != nil {
		return fail(err)
	}
	if r.FiveWhys {
		if err := os.WriteFile(filepath.Join(runDir, "in", agent.SkillsFile), []byte(whys.Skill+"\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	ts, tsErr := testerSettings(repo)
	return ask(runDir, repo, on, bounds, roundTask(r, *is, bounds, s, ts.Command, tsErr))
}

var errOwnAccount = errors.New("workline's own account")

// roundBounds reads a round's issue and its bounds: its frozen tests
// those of its branch too, whatever the tester's Draft, and the test
// files an earlier round wrote beside the Scope, still beside it.
func roundBounds(f forge.Forge, b forge.Backlog, repo string, a Act, issue int) (*forge.Issue, Bounds, error) {
	is, err := f.Issue(issue)
	if err != nil {
		return nil, Bounds{}, err
	}
	bounds, err := readBounds(f, b, repo, *is)
	if err != nil {
		return nil, Bounds{}, err
	}
	for _, t := range a.Acceptance {
		bounds.Frozen[t] = issue
	}
	bounds.Beside = a.Beside
	return is, bounds, nil
}

// testsSent says the review's test findings the tester has already.
func testsSent(where string, n int) verdict.Finding {
	return verdict.Finding{Rule: "tests-with-the-tester", Level: "info", Where: where,
		Message: fmt.Sprintf("%d test finding%s of the review sent to the tester already: not asked again (#1265)", n, plural(n))}
}

// issueHolds is who asks a round its issue's new body: the issue.
const issueHolds = "the issue"

// issueRewritten is the mark of a's issue's body when it changed since the
// record's (#1271): the person's change is the developer's to hold its
// code to, as a person's comment on its pull request is. "" when it did
// not, or a record from before Body has none to compare.
func issueRewritten(f forge.Forge, a Act) (string, error) {
	if a.Body == "" || a.Issue == 0 {
		return "", nil
	}
	is, err := f.Issue(a.Issue)
	if err != nil {
		return "", err
	}
	if BodyChanged(a.Body, is.Body) {
		return BodyMark(is.Body), nil
	}
	return "", nil
}

// ErrRecordUnread: the developer's record on its merge request, workline's
// own comment, was not read; on GitHub with an App, WORKLINE_APP_ID names
// the account its comments are trusted from.
var ErrRecordUnread = errors.New("the developer's record on it was not read as workline's own comment (with a GitHub App, WORKLINE_APP_ID names it)")

// readAsked reads what a round on merge request id asks, no agent: the
// reviewer's findings when it reviewed since the last round, the gates'
// new findings when they reported on head since it (head "": not
// known), a person's comments not heard yet. fresh: the reviewer or the
// gates said something new.
func readAsked(f forge.Forge, b forge.Backlog, kept forge.Kept, repo string, a Act, id int, head string) (r round, fresh bool, err error) {
	r = round{MR: id, Record: a}
	back, err := backFromAsk(f, a)
	if err != nil {
		return r, false, err
	}
	// The tester's answer to the round's objection (#987): a test kept
	// asks again what the round held, at the same head. A rewrite is
	// asked below, as any.
	var keptNow []Asked
	if a.Widen != nil && a.Widen.Objected && !back {
		tests, err := testerKept(f, a.Issue)
		if err != nil {
			return r, false, err
		}
		r.Kept, r.KeptRead = len(tests), true
		if len(tests) > a.Widen.Kept {
			for _, t := range tests[a.Widen.Kept:] {
				keptNow = append(keptNow, Asked{Key: "kept|" + t, Where: t, Holds: keptHolds, Text: "kept on its Draft"})
			}
		}
	}
	if (back || len(keptNow) > 0) && a.Widen.Review {
		a.ReviewRuns, a.ReviewAt = 0, "" // the findings it held off a green head, asked again
	}
	for _, m := range reviewerMarkers {
		c, ok := kept.Last(m)
		if !ok {
			continue
		}
		asked, nits := reviewRows(c)
		r.Asked, r.Nits = append(r.Asked, asked...), r.Nits+nits
		if why := recordUnread(c); why != "" && r.RecordUnread == "" {
			r.RecordUnread = why
		}
		if m := reviewerRuns.FindStringSubmatch(c); m != nil {
			n, _ := strconv.Atoi(m[1])
			r.Runs += n
		}
		if at := reviewerAt.FindStringSubmatch(c); at != nil && r.ReviewAt == "" {
			r.ReviewAt = at[1]
		}
	}
	// A review is new when the reviewer ran its lenses since, or read
	// another head: its rules alone, holding the lenses, count no run
	// (#1040). A record from before review-at counts runs alone.
	newReview := r.Runs > a.ReviewRuns ||
		r.ReviewAt != "" && r.ReviewAt != a.ReviewAt && (a.ReviewAt != "" || a.ReviewRuns == 0)
	r.Disputed = disputed(r.Asked, a, kept)
	if !newReview { // the review read already
		r.Asked = nil
	}
	// A test finding sent to the tester is never asked again (#1265).
	r.Asked = slices.DeleteFunc(r.Asked, func(x Asked) bool {
		if slices.Contains(a.Tester, x.Key) {
			r.Sent++
			return true
		}
		return false
	})
	if back {
		r.Asked = waited(a, r.Asked)
	}
	if th, ok := f.(forge.Threaded); ok && len(r.Asked) > 0 {
		threads, err := th.Threads(id)
		if err != nil {
			return r, false, err
		}
		lineDetails(r.Asked, threads, kept.Wrote)
	}
	at, gates := gatesAsked(kept, a, head)
	r.GatesAt, r.Asked = at, append(r.Asked, gates...)
	if r.Back = back; back || legacyNoCode(a) || len(keptNow) > 0 {
		a.ChecksAt = "" // asked again at its head: new bounds, a test kept, or a round that could not ask for them
	}
	r.Asked = append(r.Asked, keptNow...)
	checksAt, checks, unread := checksAsked(f, a, head)
	r.ChecksAt, r.ChecksUnread, r.Asked = checksAt, unread, append(r.Asked, checks...)
	rewrites, err := rewritesAsked(f, repo, a, head)
	if err != nil {
		return r, false, err
	}
	r.Asked = append(r.Asked, rewrites...)
	target := forge.Target{Kind: "merge-request", ID: id}
	notes, err := b.Notes(target)
	if err != nil && !errors.Is(err, forge.ErrAuthorsUnknown) {
		return r, false, err
	}
	own, err := forge.OwnAccount(f)
	if err != nil {
		return r, false, fmt.Errorf("%w: %w", errOwnAccount, err)
	}
	for _, n := range notes {
		if !n.Insider || n.Bot || own.Wrote(n) || slices.Contains(a.Heard, n.ID) || strings.Contains(n.Body, "<!-- workline:") {
			continue
		}
		r.Heard = append(r.Heard, n.ID)
		who := "a person"
		if n.Author != "" {
			who = "@" + n.Author
		}
		r.Asked = append(r.Asked, Asked{Key: "person|" + n.ID, Where: who, Text: strings.TrimSpace(n.Body), Holds: "a person"})
	}
	if r.Body, err = issueRewritten(f, a); err != nil {
		return r, false, err
	}
	if r.Body != "" {
		r.Asked = append(r.Asked, Asked{Key: "issue|" + r.Body, Where: fmt.Sprintf("#%d", a.Issue), Holds: issueHolds,
			Text: "its body changed since its code was written"})
	}
	return r, newReview || at != "" || checksAt != "" || len(rewrites) > 0 || len(keptNow) > 0, nil
}

// RoundDue says whether a review round is due on the developer's merge
// request mr, no agent: the reconciler's fallback for an event missed
// (ADR-0059). Due when the round would ask something new; not, on
// another's branch, with no record of the developer's that reads, or
// handed to a person. Its branch's head is read in repo, as fetched from
// origin; not found, the round itself compares the gates' head.
func RoundDue(f forge.Forge, repo string, mr forge.MergeRequest) (bool, error) {
	if developerIssue(mr.Branch) == 0 {
		return false, nil
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return false, nil
	}
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: mr.ID})
	if err != nil {
		return false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	a, read := ReadAct(body)
	if !ok || !read {
		return false, ErrRecordUnread
	}
	if a.Handed {
		return false, nil
	}
	if _, due := conflictDue(kept, a, repo, branchHead(f, repo, mr.Branch)); due {
		return true, nil // its merge refused for a conflict: a merge round (ADR-0067)
	}
	head, _ := gitOut(repo, "rev-parse", "--verify", "-q", "refs/remotes/origin/"+mr.Branch)
	r, _, err := readAsked(f, b, kept, repo, a, mr.ID, head)
	if err == nil && len(r.Asked) == 0 && r.ChecksUnread != "" {
		err = fmt.Errorf("its project's checks: %s", r.ChecksUnread)
	}
	if err == nil && len(r.Asked) == 0 && len(r.Disputed) > 0 {
		return true, nil // blocked on test findings the tester holds needless: its round blocks the issue
	}
	if err == nil && len(r.Asked) == 0 && mr.Draft {
		ready, _ := draftReady(f, a, head) // its Draft made green: its round marks it ready (#969)
		return ready, nil
	}
	return len(r.Asked) > 0, err
}

// roundTask is the round's question: the trace of the rounds before, the
// findings to fix, then what an issue run asks (the issue, its bounds,
// the test command), and the round's answer. No code: its agent reads its
// branch in its worktree (ADR-0056, step 6).
func roundTask(r round, is forge.Issue, b Bounds, s Settings, test string, testErr error) string {
	var t strings.Builder
	n := r.Record.Round + 1
	fmt.Fprintf(&t, "# Review round %d of %d on your pull request %d, for issue #%d: %s\n\n", n, s.Rounds, r.MR, is.ID, is.Title)
	t.WriteString("You wrote this pull request's code; you remember none of it. What you did is below, from its record; your worktree holds its branch.\n\n")
	if len(r.Record.Rounds) > 0 {
		t.WriteString("## The rounds before, from its record\n\n")
		for _, x := range r.Record.Rounds {
			fmt.Fprintf(&t, "- Round %d: %d asked, the tests %s", x.N, len(x.Asked), x.Result)
			if x.Whys {
				t.WriteString(", the five whys run")
			}
			t.WriteString(".\n")
			for _, a := range x.Asked {
				fmt.Fprintf(&t, "  - %s\n", a)
			}
		}
		t.WriteString("\n")
	}
	t.WriteString("## What to fix\n\n")
	for _, a := range r.Asked {
		seen := ""
		if slices.Contains(r.Record.Seen, a.Key) {
			seen = " — **still there** after a round"
		}
		if a.Holds == "a person" {
			fmt.Fprintf(&t, "- A person of the project, %s%s:\n\n  > %s\n\n", a.Where, seen, strings.ReplaceAll(a.Text, "\n", "\n  > "))
			continue
		}
		if a.Holds == issueHolds {
			fmt.Fprintf(&t, "- The issue, %s: %s. Its body as it reads now is below: hold your code to it.\n", a.Where, a.Text)
			continue
		}
		if a.Holds == checkHolds {
			fmt.Fprintf(&t, "- %s%s. The end of its failed steps' log:\n\n  ~~~~\n  %s\n  ~~~~\n\n", checkLine(a), seen, strings.ReplaceAll(a.Detail, "\n", "\n  "))
			continue
		}
		if a.Holds == testerHolds {
			fmt.Fprintf(&t, "- %s. Your worktree holds it as rewritten: make it pass; it stays read-only to you.\n", rewriteLine(a))
			continue
		}
		if a.Holds == keptHolds {
			fmt.Fprintf(&t, "- %s. Make it pass; or, if you cannot, say why in a note: the issue goes to a person, no second objection waited for at this head.\n", keptLine(a))
			continue
		}
		if a.Holds == gateHolds {
			fmt.Fprintf(&t, "- %s%s%s\n", gateLine(a), seen, frozenNote(a, b))
			if a.Detail != "" {
				fmt.Fprintf(&t, "\n  > The inspector: %s\n\n", strings.ReplaceAll(a.Detail, "\n", "\n  > "))
			}
			continue
		}
		fmt.Fprintf(&t, "- %s, the %s lens, `%s`%s: %s%s\n", a.Holds, a.Lens, a.Where, seen, a.Text, frozenNote(a, b))
		if a.Detail != "" {
			fmt.Fprintf(&t, "\n  > %s\n\n", strings.ReplaceAll(a.Detail, "\n", "\n  > "))
		}
	}
	if len(r.ToTester) > 0 {
		t.WriteString("\n## Sent to the tester, not yours\n\nThe review's test findings on your code: the tester adds each acceptance test, or says why none is needed. Write no test for them, change nothing for them.\n\n")
		for _, l := range testerLines(r.ToTester) {
			fmt.Fprintf(&t, "- %s\n", l)
		}
	}
	if r.Nits > 0 {
		fmt.Fprintf(&t, "\nThe reviewer showed %d nit%s too: not asked, leave them.\n", r.Nits, plural(r.Nits))
	}
	t.WriteString("\n" + question(is, b, test, testErr, s))
	t.WriteString("A fix that lies in a file beyond the issue's " + work.Name(work.Scope) + ": make it there, and say why in `" + BeyondFile + "`, as above. " +
		"Never a fix of the symptom within the " + work.Name(work.Scope) + " in its place.\n\n")
	t.WriteString("A finding whose fix would turn a frozen test red: never undo the code a frozen test needs for it. " +
		"Object to the test, as above; say so in the finding's note. " +
		"On a green head, a fix the proof finds red on a frozen test is not pushed.\n\n")
	t.WriteString("Its diff is your fix: the cause, not the symptom. Then answer a `note` a finding, one line, what you did; " +
		"a finding you hold wrong: say why in its note, change nothing for it, the reviewer decides a finding is fixed. An objection, if any, as above.\n")
	if r.FiveWhys {
		t.WriteString("\n**The five whys**: a finding is still there at this round. First run the five whys skill below on it; give the analysis as its own `note`, and fix the root cause it names.\n")
	}
	return t.String()
}

// frozenNote tells a finding on a line of a frozen test whose it is to fix:
// the tester's, by an objection, never a dead end the developer waits on
// (#1040). "" for any other finding.
func frozenNote(a Asked, b Bounds) string {
	file, _, _ := strings.Cut(a.Where, ":")
	if _, frozen := b.Frozen[file]; !frozen || file == "" {
		return ""
	}
	if b.Draft == 0 {
		return ". It lies in a frozen test, read-only to you, and the tester's Draft is closed: change nothing for it, say so in its note"
	}
	return fmt.Sprintf(". It lies in a frozen test, read-only to you: object to that test at that line in `%s`, as below, the finding as its why, so that the tester rewrites it; change nothing for it", ObjectionsFile)
}

// handOff ends the rounds, past the last one: the reviewer and the
// developer do not agree. Its issue goes to the blocked state, the reason
// on it and on the pull request (ADR-0059, decision 6); no person is
// asked: one on the loop reads the board.
func handOff(r round, b Bounds, repo string) []intent.Intention {
	a := r.Record
	a.Handed = true
	a.Rounds = append(a.Rounds, Round{N: a.Round + 1, Asked: texts(r.Asked), Person: true, Tester: testerLines(r.ToTester)})
	var tests []intent.Intention
	if len(r.ToTester) > 0 { // the tester's all the same: no disagreement on them
		sendTests(&a, r.ToTester)
		tests = append(tests, testRequests(repo, b, r.MR, r.ToTester))
	}
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	if r.ReviewAt != "" {
		a.ReviewAt = r.ReviewAt
	}
	if r.GatesAt != "" {
		a.GatesAt = r.GatesAt
	}
	if r.ChecksAt != "" {
		a.ChecksAt = r.ChecksAt
	}
	add, remove := blockedMove(repo)
	why := fmt.Sprintf("%d review rounds on merge request %d, and the review still asks %d finding%s: the reviewer and the developer do not agree",
		a.Round, r.MR, len(r.Asked), plural(len(r.Asked)))
	return append(tests,
		intent.Intention{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": add, "remove": remove, "card": "blocked"}},
		sayOn(a.Issue, backlog.Said(who, "blocks this issue", why,
			"a person on the loop reads the pull request, fixes the findings or closes it, and moves the issue back by its label")),
		intent.Intention{Kind: "comment", Value: map[string]any{"sticky": "round", "body": fmt.Sprintf(
			"**Blocked**: %d rounds, and the review still asks %d finding%s. #%d is blocked, its reason on it: a person of the project fixes them, or closes this pull request.\n\n%s",
			a.Round, len(r.Asked), plural(len(r.Asked)), a.Issue, askedList(r.Asked))}},
		intent.Intention{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}},
	)
}

func texts(l []Asked) []string {
	out := make([]string, len(l))
	for i, a := range l {
		switch a.Holds {
		case "a person":
			out[i] = "a person, " + a.Where
		case issueHolds:
			out[i] = fmt.Sprintf("the issue, %s: %s", a.Where, a.Text)
		case gateHolds:
			out[i] = gateLine(a)
		case checkHolds:
			out[i] = checkLine(a)
		case testerHolds:
			out[i] = rewriteLine(a)
		case keptHolds:
			out[i] = keptLine(a)
		case conflictHolds:
			out[i] = "merge " + a.Where + " in: it " + a.Text
		default:
			out[i] = fmt.Sprintf("%s %s `%s`: %s", a.Holds, a.Lens, a.Where, a.Text)
		}
	}
	return out
}

func askedList(l []Asked) string {
	var b strings.Builder
	for _, t := range texts(l) {
		fmt.Fprintf(&b, "- %s\n", t)
	}
	return b.String()
}

// reviewPost holds the round's answer: its patches to the bounds, the
// five whys when due, its proof on the branch's head; the fix pushed on
// its branch, the round kept in its record.
func reviewPost(runDir, repo string, s Settings) int {
	var r round
	if err := readIn(runDir, roundFile, &r); err != nil {
		return fail(err)
	}
	where := fmt.Sprintf("merge request %d", r.MR)
	if r.Ready {
		return readyPost(runDir, r)
	}
	if r.Merge != "" && r.Backstop == "" {
		return mergePost(runDir, repo, s, r)
	}
	if r.Backstop != "" {
		return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("at its backstop: its issue #%d blocked, no round run", r.Record.Issue),
			Findings: []verdict.Finding{{Rule: "backstop-crossed", Level: "warn", Where: where,
				Message: fmt.Sprintf("%s: no review round runs; #%d blocked, this pull request kept open", r.Backstop, r.Record.Issue)}}})
	}
	if r.TesterOnly {
		return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("%d test finding%s of the review sent to the tester: no round counted", len(r.ToTester), plural(len(r.ToTester))),
			Findings: []verdict.Finding{{Rule: "tests-to-the-tester", Level: "info", Where: where,
				Message: fmt.Sprintf("the review asks test findings alone, the tester's: on its Draft, #%d waits for its tests (#1265)", r.Record.Issue)}}})
	}
	if len(r.Disputed) > 0 && len(r.Asked) == 0 && len(r.ToTester) == 0 {
		return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("the review blocks on test findings the tester holds needless: its issue #%d blocked", r.Record.Issue),
			Findings: []verdict.Finding{{Rule: "tests-disputed", Level: "warn", Where: where,
				Message: fmt.Sprintf("%d test finding%s still block, the tester answering no test is needed: #%d blocked for a person, its reason on it", len(r.Disputed), plural(len(r.Disputed)), r.Record.Issue)}}})
	}
	if r.Person {
		return writeVerdict(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("past round %d: its issue #%d blocked", r.Record.Round, r.Record.Issue),
			Findings: []verdict.Finding{{Rule: "review-blocked", Level: "warn", Where: where,
				Message: fmt.Sprintf("%d rounds did not settle the review: #%d blocked, its reason on it", r.Record.Round, r.Record.Issue)}}})
	}
	var b Bounds
	if err := readIn(runDir, boundsFile, &b); err != nil {
		return fail(err)
	}
	proposed, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	var notes, analyses []string
	var rest []intent.Intention
	for _, in := range proposed {
		if in.Kind != "note" {
			rest = append(rest, in)
			continue
		}
		if text := noteText(in.Value); r.FiveWhys && whys.Is(text) {
			analyses = append(analyses, text)
		} else if text != "" {
			notes = append(notes, text)
		}
	}
	analysis := ""
	if r.FiveWhys {
		if len(analyses) != 1 {
			return refuseAnswer(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the round's answer is refused",
				Findings: []verdict.Finding{{Rule: "five-whys-missing", Where: where, Message: fmt.Sprintf("a finding is still there at round three: the answer gives the five whys as one `note` opening with `Five whys`, as the skill says (%d given)", len(analyses))}}})
		}
		w, err := whys.Parse(analyses[0])
		if err != nil {
			return refuseAnswer(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the round's answer is refused",
				Findings: []verdict.Finding{{Rule: "five-whys-unreadable", Where: where, Message: "the five whys do not follow the skill's form: " + err.Error()}}})
		}
		analysis = w.Markdown()
	}
	// Its worktree's diff from its branch's head, held as an issue run's.
	base := r.At
	if r.Base != "" {
		base = r.Base // the tester's rewrites in it: its agent's diff is its own
	}
	kept, refused, said, unread, why, err := held(runDir, repo, base, b, nil, rest)
	if err != nil {
		return fail(err)
	}
	takes, err := rewritePatches(repo, r.Taken)
	if err != nil {
		return fail(err)
	}
	for _, u := range unread {
		analysis += "\n" + u // as it came, in a fence of its own: not a note's one line
	}
	a := r.Record
	n := a.Round + 1
	v := verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("round %d of %d on %s", n, s.Rounds, where),
		Findings: append([]verdict.Finding{rightsFinding(where)}, refusalFindings(where, refused, said)...)}
	if len(unread) > 0 {
		v.Findings = append(v.Findings, unreadFinding(where, unread, true))
	}
	if r.ChecksUnread != "" {
		v.Findings = append(v.Findings, checksUnread(where, r.ChecksUnread))
	}
	var patches, others []intent.Intention
	for _, in := range kept {
		if in.Kind == "patch" && unchanged(repo, in.Value) {
			continue // nothing to commit: a fix that changes nothing is none
		}
		if in.Kind == "patch" {
			patches = append(patches, in)
		} else {
			others = append(others, in) // an objection on the tester's Draft
		}
	}
	this := Round{N: n, Asked: texts(r.Asked), Heard: len(r.Heard), Nits: r.Nits, Whys: r.FiveWhys, Tester: testerLines(r.ToTester)}
	if len(r.ToTester) > 0 {
		sendTests(&a, r.ToTester)
		others = append(others, testRequests(repo, b, r.MR, r.ToTester))
		v.Findings = append(v.Findings, verdict.Finding{Rule: "tests-to-the-tester", Level: "info", Where: where,
			Message: fmt.Sprintf("%d test finding%s of the review sent to the tester, on its Draft !%d (#1265)", len(r.ToTester), plural(len(r.ToTester)), b.Draft)})
	}
	a.Refused = append(a.Refused, refused...)
	if r.Back {
		a.Widen.Back = true
	}
	var ask []intent.Intention
	objected := objectedOn(others)
	// Objected already at this head, the tester's answer read: no second
	// wait at the same head (#987).
	answered := a.Widen != nil && a.Widen.Objected && sameCommit(a.Widen.At, r.At)
	// A green head (#987): its last proof green, no red check nor rewrite
	// asked. A fix of the review that turns its frozen tests red is not
	// pushed: the green code is never undone for a finding.
	green := a.Proof != nil && a.Proof.Result == Green && len(redChecks(r.Asked)) == 0 && len(takes) == 0
	var heldRed []string
	if len(patches) > 0 || len(takes) > 0 {
		// The tester's rewrites first, as its record names them, then
		// its agent's code: proven together, pushed together.
		p, u, caps, found := proven(runDir, repo, r.At, where, slices.Concat(takes, patches), union(a.Acceptance, union(a.Code, a.Tests)), again(a.Proof), s)
		this.Result, this.Turns, this.Tokens, this.Cached, this.Tools = p.Result, u.Turns, u.Tokens, u.Cached, u.Tools
		// Only a frozen test red holds it: a red of its own tests is the
		// next round's to fix, pushed as before.
		if frozen := frozenRed(repo, r.At, a.Acceptance, p.Red); green && p.Result == Red && len(frozen) > 0 {
			heldRed = frozen
			this.Result, patches = "not-pushed", nil
			msg := fmt.Sprintf("its fix turns the tests red on a green head (%s): not pushed, the green head kept", strings.Join(heldRed, ", "))
			v.Findings = append(v.Findings, verdict.Finding{Rule: "green-kept", Level: "warn", Where: where, Message: msg})
			v.Summary += "; its fix turns the tests red on a green head: not pushed"
		} else {
			a.Proof, a.Caps = &p, caps
			code, tests := written(patches, b.tests())
			a.Code, a.Tests = union(a.Code, code), union(a.Tests, tests)
			if this.Beside = besideKept(repo, b, patches); len(this.Beside) > 0 {
				a.Beside = union(a.Beside, this.Beside)
			}
			if this.Beyond = beyondKept(repo, b, patches, why); len(this.Beyond) > 0 {
				a.Beyond = unionBeyond(a.Beyond, this.Beyond)
				v.Findings = append(v.Findings, beyondFinding(where, this.Beyond))
			}
			if len(takes) > 0 && r.From != "" {
				a.From = r.From // the Draft's head the frozen tests are read at now
			}
			for _, f := range found {
				f.Message = strings.ReplaceAll(f.Message, "; its pull request opens as a Draft", "; pushed, the forge's checks and the reviewer read it next")
				v.Findings = append(v.Findings, f)
			}
			v.Summary += ", its fix pushed, the tests " + p.Result
			cs, unread := committerSettings(repo, where)
			v.Findings = append(v.Findings, unread...)
			push := reviewSubject(a.Issue, n, cs)
			if len(this.Beyond) > 0 {
				// In its commit, where the reviewer's intent lens reads
				// it with the commits not reviewed yet.
				push += "\n\n" + BeyondHeading + ":\n\n" + strings.TrimRight(beyondLines(this.Beyond), "\n")
			}
			if err := writeMergeRequest(runDir, map[string]any{"push": push}); err != nil {
				return fail(err)
			}
		}
	}
	if len(patches) == 0 && len(takes) == 0 {
		this.Bounds = "held"
		if this.Result == "" {
			this.Result = "no-code"
			v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
				Message: "its agent wrote no fix that changes the branch: the round is counted, nothing pushed"})
		}
		if objected && !answered && b.Draft != 0 {
			// Its objection is on the tester's Draft (#987): the issue
			// waits for its answer, not for a person. A rewrite is the
			// next round; a test kept asks it again at this head.
			a.Widen, this.Bounds = &Widen{At: r.At, Scope: scopeKey(b.Scope), Objected: true, Draft: b.Draft, Kept: r.Kept, Review: heldRed != nil}, "objected"
			msg := fmt.Sprintf("its objection to the frozen tests is on the tester's Draft !%d: #%d waits for its answer", b.Draft, a.Issue)
			v.Findings = append(v.Findings, verdict.Finding{Rule: "objected", Level: "info", Where: where, Message: msg})
			v.Summary += "; " + msg
		} else if red := redChecks(r.Asked); len(red) > 0 {
			// A red check it holds wrong, or cannot fix, and no file to
			// ask for (#938): the check is not asked again at this head,
			// so nothing would move. Blocked for a person, its reason
			// written; one who sends it back resumes the round.
			ask = holdCheck(repo, a, r, red, notes, others)
			a.Widen, this.Bounds = &Widen{At: r.At, Scope: scopeKey(b.Scope), Held: true}, "blocked"
			msg := fmt.Sprintf("no fix for the red check %s: #%d blocked for a person", strings.Join(red, ", "), a.Issue)
			v.Findings = append(v.Findings, verdict.Finding{Rule: "check-held", Level: "warn", Where: where, Message: msg})
			v.Summary += "; " + msg
		} else if heldRed != nil {
			// Its fix held off a green head, no objection waited for nor
			// file asked (#987): nothing else would move. Blocked for a
			// person; one who sends it back has the findings asked again.
			ask = holdGreen(repo, a, r, heldRed, notes, others)
			a.Widen, this.Bounds = &Widen{At: r.At, Scope: scopeKey(b.Scope), Held: true, Review: true}, "blocked"
			msg := fmt.Sprintf("no objection left for the tester at this head: #%d blocked for a person", a.Issue)
			v.Findings = append(v.Findings, verdict.Finding{Rule: "green-held", Level: "warn", Where: where, Message: msg})
			v.Summary += "; " + msg
		} else if objected && answered {
			// Objected again at this head, the tester's answer read
			// (#987): no second wait, nothing else would move.
			why := fmt.Sprintf("its round on pull request %d objected again at this head, after the tester's answer, and pushed no fix: no second wait for the tester at the same head", r.MR)
			next := "a person on the loop reads the tester's answer and the developer's note below, then settles the test or the finding; and moves the issue back to ready: the next round asks the findings again at the same head"
			ask = holdFor(repo, a, r, why, next, notes, others)
			a.Widen, this.Bounds = &Widen{At: r.At, Scope: scopeKey(b.Scope), Held: true, Review: true}, "blocked"
			msg := fmt.Sprintf("objected again at this head after the tester's answer: #%d blocked for a person", a.Issue)
			v.Findings = append(v.Findings, verdict.Finding{Rule: "objection-held", Level: "warn", Where: where, Message: msg})
			v.Summary += "; " + msg
		}
	}
	switch {
	case a.Widen == nil || !a.Widen.Objected || this.Bounds == "objected":
	case len(patches) > 0 || len(takes) > 0:
		a.Widen.Objected = false // a new head pushed: the wait ends with the old one
	case r.KeptRead:
		a.Widen.Kept = r.Kept // the tester's answers read: one more is news
	}
	if r.FiveWhys {
		v.Summary += ", the five whys with it"
	}
	a.Round, a.Whys = n, a.Whys || r.FiveWhys
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	if r.ReviewAt != "" {
		a.ReviewAt = r.ReviewAt
	}
	if r.GatesAt != "" {
		a.GatesAt = r.GatesAt
	}
	if r.ChecksAt != "" {
		a.ChecksAt = r.ChecksAt
	}
	a.Heard = append(a.Heard, r.Heard...)
	if r.Body != "" {
		a.Body = r.Body // read: the same body is not asked again
	}
	for _, x := range r.Asked {
		if x.Holds != "a person" && !slices.Contains(a.Seen, x.Key) {
			a.Seen = append(a.Seen, x.Key)
		}
	}
	a.Rounds = append(a.Rounds, this)
	out := append(slices.Concat(takes, patches), intent.Intention{Kind: "comment", Value: map[string]any{"sticky": "round", "body": roundBody(this, s, notes, analysis, a.Widen)}},
		intent.Intention{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}})
	out = append(out, others...)
	if len(this.Beside) > 0 {
		out = append(out, besideSaid(a))
	}
	switch {
	case len(ask) > 0:
		out = append(out, ask...)
	case r.Back:
		// Back from the product owner, its bounds wider: taken again.
		out = append(out, moveTo(repo, "doing", a.Issue, []string{s.ClaimLabel}, []string{backlog.LabelReady}))
	}
	if err := writeIntents(runDir, "", out); err != 0 {
		return err
	}
	return writeVerdict(runDir, v)
}

// roundBody is the round's comment on its pull request: what was asked,
// what the developer did, its proof, the five whys.
func roundBody(x Round, s Settings, notes []string, analysis string, w *Widen) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer**, review round %d of %d: ", x.N, s.Rounds)
	if x.Result == "not-pushed" {
		b.WriteString("its fix turns the frozen tests red on a green head, not pushed: the green head kept. ")
	}
	switch {
	case x.Bounds == "objected":
		fmt.Fprintf(&b, "Nothing pushed; its objection is on the tester's Draft !%d, on its lines: its issue waits for the tester's answer. A rewrite is taken by the next round; a test kept asks it again here.\n\n", w.Draft)
	case x.Bounds == "blocked" && x.Result == "not-pushed":
		b.WriteString("No objection left for the tester at this head: its issue is blocked for a person, its reason on it; moved back to ready, the next round asks the findings again here.\n\n")
	case x.Bounds == "blocked" && w.Review:
		b.WriteString("no fix, and it objected again at this head after the tester's answer: its issue is blocked for a person, its reason on it; moved back to ready, the next round asks the findings again here.\n\n")
	case x.Result == "not-pushed" && x.Bounds == "held":
		b.WriteString("\n\n")
	case x.Bounds == "blocked":
		b.WriteString("no fix: the red check stays, its issue is blocked for a person, its reason on it; moved back to ready, the next round runs here.\n\n")
	case x.Result == "no-code":
		b.WriteString("no fix.\n\n")
	default:
		fmt.Fprintf(&b, "its fix pushed, the project's tests %s.\n\n", x.Result)
	}
	if len(x.Beside) > 0 {
		fmt.Fprintf(&b, "New test files beside the issue's %s: `%s`; the product owner adds them to the %[1]s at its next run.\n\n", scopeName, strings.Join(x.Beside, "`, `"))
	}
	if len(x.Beyond) > 0 {
		fmt.Fprintf(&b, "%s, the issue's %s a hint; the reviewer's intent lens judges each:\n\n%s\n", BeyondHeading, scopeName, beyondLines(x.Beyond))
	}
	b.WriteString("Asked:\n\n")
	for _, t := range x.Asked {
		fmt.Fprintf(&b, "- %s\n", t)
	}
	if len(x.Tester) > 0 {
		b.WriteString("\nSent to the tester, its to add or to hold needless:\n\n")
		for _, t := range x.Tester {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	if x.Nits > 0 {
		fmt.Fprintf(&b, "\n%d nit%s shown, not asked.\n", x.Nits, plural(x.Nits))
	}
	if len(notes) > 0 {
		b.WriteString("\nWhat it did:\n\n")
		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", strings.ReplaceAll(strings.TrimSpace(n), "\n", " "))
		}
	}
	if analysis != "" {
		b.WriteString("\n" + analysis)
	}
	b.WriteString("\nThe reviewer decides a finding is fixed.\n")
	return b.String()
}

// unchanged says a patch writes what the branch's head holds already:
// given whole, {file, content}, its file's content; given as a diff, the
// tree it leaves, laid as the engine lays it on an index of its own (a
// diff removing and adding the same lines). A diff git cannot lay is not
// said unchanged: the apply refuses it.
func unchanged(repo string, v any) bool {
	if diff, ok := v.(string); ok {
		return diffChangesNothing(repo, diff)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	file, _ := m["file"].(string)
	content, ok := m["content"].(string)
	if file == "" || !ok {
		return false
	}
	out, err := exec.Command("git", "-C", repo, "show", "HEAD:"+file).Output()
	return err == nil && string(out) == content
}

func diffChangesNothing(repo, diff string) bool {
	tmp, err := os.MkdirTemp("", "workline-unchanged-")
	if err != nil {
		return false
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
	run := func(stdin string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env, cmd.Stdin = env, strings.NewReader(stdin)
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	}
	if _, err := run("", "read-tree", "HEAD"); err != nil {
		return false
	}
	if _, err := run(intent.NormalizeDiff(diff), "apply", "--cached", "--recount", "--unidiff-zero", "-"); err != nil {
		return false
	}
	tree, err := run("", "write-tree")
	head, herr := run("", "rev-parse", "HEAD^{tree}")
	return err == nil && herr == nil && tree == head
}

// noteText is a note's text, given bare or as {text}.
func noteText(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]any:
		s, _ := t["text"].(string)
		return strings.TrimSpace(s)
	}
	return ""
}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, x := range b {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	slices.Sort(out)
	return out
}

// writeIntents writes in/<name> (pre's fallback) or, name "", the run's
// out/intentions.yaml, replacing the agent's.
func writeIntents(runDir, name string, l []intent.Intention) int {
	p := filepath.Join(runDir, "in", name)
	if name == "" {
		p = filepath.Join(runDir, "out", "intentions.yaml")
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fail(err)
		}
	}
	if err := intent.Write(p, l); err != nil {
		return fail(err)
	}
	return 0
}

// refuseAnswer blocks a round's answer, nothing of it kept: on a block,
// the engine still applies the role's comments (on-block), meant for the
// bounds' record, never for what an agent proposed.
func refuseAnswer(runDir string, v verdict.Verdict) int {
	if code := writeIntents(runDir, "", nil); code != 0 {
		return code
	}
	return writeVerdict(runDir, v)
}

func writeVerdict(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return 0
}
