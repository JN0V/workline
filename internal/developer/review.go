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
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/whys"
)

const roundFile = "round.json"

// The reviewer's comments a round reads its findings from (its summary,
// and the comment of its lenses that warn when the review is split), and
// the count of its runs their records hold.
var (
	reviewerMarkers = []string{forge.Marker("sticky=reviewer/summary"), forge.Marker("sticky=reviewer/warn")}
	reviewerRuns    = regexp.MustCompile(`<!-- workline:record reviewed=\S*(?: patches=\S*)? runs=(\d+)`)
)

// Round is one round of the review loop, as the record keeps it.
type Round struct {
	N      int            `json:"n"`
	Asked  []string       `json:"asked,omitempty"`  // the findings asked, each as its row reads
	Heard  int            `json:"heard,omitempty"`  // a person's comments read
	Nits   int            `json:"nits,omitempty"`   // nits said, not asked
	Result string         `json:"result,omitempty"` // its proof's result; no-code when it wrote none
	Whys   bool           `json:"whys,omitempty"`   // the five whys ran
	Person bool           `json:"person,omitempty"` // past the last round: its issue blocked (ADR-0059)
	Turns  int            `json:"turns,omitempty"`  // what its agent used, as it reported them
	Tokens int            `json:"tokens,omitempty"` // fresh, what ai-max-tokens holds
	Cached int            `json:"cached,omitempty"` // read from cache, apart
	Tools  map[string]int `json:"tools,omitempty"`
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
}

// round is what the review run's pre leaves for post.
type round struct {
	MR       int      `json:"merge-request"`
	Branch   string   `json:"branch"`
	Record   Act      `json:"record"`
	Asked    []Asked  `json:"asked"`
	Heard    []string `json:"heard,omitempty"` // the ids of a person's comments it reads
	Nits     int      `json:"nits"`
	Runs     int      `json:"review-runs"`
	FiveWhys bool     `json:"five-whys"`
	Person   bool     `json:"person"`       // past the last round: no agent, a person
	At       string   `json:"at,omitempty"` // its branch's head, where its fix starts
}

// reviewRows reads the findings of the reviewer's comment: what blocks,
// a lens's or a rule's, and what warns and is Important; the nits
// counted. Another row that warns asks nothing: left.
func reviewRows(comment string) (asked []Asked, nits int) {
	for _, l := range strings.Split(comment, "\n") {
		cells := tableRow(l)
		if len(cells) != 4 {
			continue
		}
		holds, lens, where, msg := strings.Trim(cells[0], "*"), cells[1], strings.Trim(cells[2], "`"), cells[3]
		switch {
		case strings.HasPrefix(msg, "Nit:"):
			nits++
			continue
		case holds == "blocks": // a lens's, or a rule's: a gate's, said bare
		case holds != "warns" || !strings.HasPrefix(msg, "Important:"):
			continue
		}
		title := strings.TrimSpace(strings.TrimPrefix(msg, "Important:"))
		if t, _, ok := strings.Cut(title, " — "); ok {
			title = t
		}
		file, _, _ := strings.Cut(where, ":")
		asked = append(asked, Asked{Key: lens + "|" + file + "|" + title, Lens: lens, Where: where, Text: msg, Holds: holds})
	}
	return asked, nits
}

// lineDetails gives each finding of the reviewer's rows its text whole,
// from the reviewer's comment on its line: the row, when the finding has
// one, shows only its title. Only a comment workline's own account wrote
// is read, on the finding's file, carrying its title; an open one before
// one resolved, the latest first.
func lineDetails(asked []Asked, threads []forge.Thread, wrote func(forge.Note) bool) {
	for i, a := range asked {
		if a.Holds == "a person" {
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

// tableRow is a Markdown table's row, its cells; nil for any other line,
// its header and its rule.
func tableRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") || strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| | |") {
		return nil
	}
	var cells []string
	var cell strings.Builder
	for i := 1; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteByte(line[i])
		}
	}
	return cells
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
	r := round{MR: id, Branch: mr.Branch, Record: a}
	for _, m := range reviewerMarkers {
		c, ok := kept.Last(m)
		if !ok {
			continue
		}
		asked, nits := reviewRows(c)
		r.Asked, r.Nits = append(r.Asked, asked...), r.Nits+nits
		if m := reviewerRuns.FindStringSubmatch(c); m != nil {
			n, _ := strconv.Atoi(m[1])
			r.Runs += n
		}
	}
	if th, ok := f.(forge.Threaded); ok && len(r.Asked) > 0 {
		threads, err := th.Threads(id)
		if err != nil {
			return external(err)
		}
		lineDetails(r.Asked, threads, kept.Wrote)
	}
	notes, err := b.Notes(target)
	if err != nil && !errors.Is(err, forge.ErrAuthorsUnknown) {
		return external(err)
	}
	own, err := forge.OwnAccount(f)
	if err != nil {
		return fail(err)
	}
	for _, n := range notes {
		if !n.Insider || n.Bot || own.Wrote(n) || slices.Contains(a.Heard, n.ID) || strings.Contains(n.Body, "<!-- workline:") {
			continue
		}
		r.Heard = append(r.Heard, n.ID)
		r.Asked = append(r.Asked, Asked{Key: "person|" + n.ID, Where: "@" + n.Author, Text: strings.TrimSpace(n.Body), Holds: "a person"})
	}
	newReview := r.Runs > a.ReviewRuns
	if !newReview { // the review read already: only a person's comment is new
		r.Asked = slices.DeleteFunc(r.Asked, func(x Asked) bool { return x.Holds != "a person" })
	}
	if a.Handed {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "a person has its pull request: the developer runs no more rounds",
			Findings: []verdict.Finding{{Rule: "review-to-a-person", Level: "info", Where: where, Message: fmt.Sprintf("past round %d, a person of the project takes it from here", a.Round)}}})
	}
	if len(r.Asked) == 0 {
		v := verdict.Verdict{Status: verdict.Pass, Summary: "nothing the review asks: no round"}
		if !newReview {
			v.Summary = "nothing new since its last round: it waits for the reviewer's next verdict"
		}
		if r.Nits > 0 && newReview {
			v.Findings = append(v.Findings, verdict.Finding{Rule: "nits-not-asked", Level: "info", Where: where,
				Message: fmt.Sprintf("%d nit%s shown by the reviewer, not asked: a nit never holds the merge (ADR-0051)", r.Nits, plural(r.Nits))})
		}
		return final(runDir, v)
	}
	if a.Round >= s.Rounds {
		r.Person = true
		if err := writeIn(runDir, roundFile, r); err != nil {
			return fail(err)
		}
		return writeIntents(runDir, "fallback.yaml", handOff(r, repo))
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("no agent: %d finding%s wait for a run with one, no round counted", len(r.Asked), plural(len(r.Asked))),
			Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where, Message: "fixing a finding is judgement on code: with no agent, nothing is fixed (principle 5)"}}})
	}
	n := a.Round + 1
	r.FiveWhys = n >= 3 && !a.Whys && slices.ContainsFunc(r.Asked, func(x Asked) bool { return slices.Contains(a.Seen, x.Key) })
	is, err := f.Issue(issue)
	if err != nil {
		return external(err)
	}
	bounds, err := readBounds(f, b, repo, *is)
	if err != nil {
		return external(err)
	}
	for _, t := range a.Acceptance { // frozen, on its branch, whatever the tester's Draft
		bounds.Frozen[t] = issue
	}
	// Its fix starts from its branch's head, checked out: a head git
	// cannot read fails the round, never counted for nothing.
	if r.At, err = gitOut(repo, "rev-parse", "HEAD"); err != nil {
		return fail(fmt.Errorf("the round's code: git cannot read its branch's head in %s: %w", repo, err))
	}
	if err := writeIn(runDir, roundFile, r); err != nil {
		return fail(err)
	}
	if r.FiveWhys {
		if err := os.WriteFile(filepath.Join(runDir, "in", agent.SkillsFile), []byte(whys.Skill+"\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	ts, tsErr := testerSettings(repo)
	return ask(runDir, repo, r.At, bounds, roundTask(r, *is, bounds, s, ts.Command, tsErr))
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
		fmt.Fprintf(&t, "- %s, the %s lens, `%s`%s: %s\n", a.Holds, a.Lens, a.Where, seen, a.Text)
		if a.Detail != "" {
			fmt.Fprintf(&t, "\n  > %s\n\n", strings.ReplaceAll(a.Detail, "\n", "\n  > "))
		}
	}
	if r.Nits > 0 {
		fmt.Fprintf(&t, "\nThe reviewer showed %d nit%s too: not asked, leave them.\n", r.Nits, plural(r.Nits))
	}
	t.WriteString("\n" + question(is, b, test, testErr, s))
	t.WriteString("Its diff is your fix: the cause, not the symptom. Then answer a `note` a finding, one line, what you did; " +
		"a finding you hold wrong: say why in its note, change nothing for it, the reviewer decides a finding is fixed. An objection, if any, as above.\n")
	if r.FiveWhys {
		t.WriteString("\n**The five whys**: a finding is still there at this round. First run the five whys skill below on it; give the analysis as its own `note`, and fix the root cause it names.\n")
	}
	return t.String()
}

// handOff ends the rounds, past the last one: the reviewer and the
// developer do not agree. Its issue goes to the blocked state, the reason
// on it and on the pull request (ADR-0059, decision 6); no person is
// asked: one on the loop reads the board.
func handOff(r round, repo string) []intent.Intention {
	a := r.Record
	a.Handed = true
	a.Rounds = append(a.Rounds, Round{N: a.Round + 1, Asked: texts(r.Asked), Person: true})
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	add, remove := []string{backlog.LabelBlocked}, []string{backlog.LabelReady}
	if line, err := routing.Load(repo); err == nil {
		if ad, rm, err := line.Move("blocked"); err == nil {
			add, remove = ad, rm
		}
	}
	why := fmt.Sprintf("%d review rounds on merge request %d, and the review still asks %d finding%s: the reviewer and the developer do not agree",
		a.Round, r.MR, len(r.Asked), plural(len(r.Asked)))
	return []intent.Intention{
		{Kind: "label", Value: map[string]any{"issue": a.Issue, "add": add, "remove": remove}},
		sayOn(a.Issue, backlog.Said(who, "blocks this issue", why,
			"a person on the loop reads the pull request, fixes the findings or closes it, and moves the issue back by its label")),
		{Kind: "comment", Value: map[string]any{"sticky": "round", "body": fmt.Sprintf(
			"**Blocked**: %d rounds, and the review still asks %d finding%s. #%d is blocked, its reason on it: a person of the project fixes them, or closes this pull request.\n\n%s",
			a.Round, len(r.Asked), plural(len(r.Asked)), a.Issue, askedList(r.Asked))}},
		{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}},
	}
}

func texts(l []Asked) []string {
	out := make([]string, len(l))
	for i, a := range l {
		if a.Holds == "a person" {
			out[i] = "a person, " + a.Where
		} else {
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
	kept, refused, said, unread, err := held(runDir, repo, r.At, b, nil, rest)
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
	this := Round{N: n, Asked: texts(r.Asked), Heard: len(r.Heard), Nits: r.Nits, Whys: r.FiveWhys}
	a.Refused = append(a.Refused, refused...)
	if len(patches) == 0 {
		this.Result = "no-code"
		v.Findings = append(v.Findings, verdict.Finding{Rule: "no-code", Level: "warn", Where: where,
			Message: "its agent wrote no fix within its bounds that changes the branch: the round is counted, nothing pushed"})
	} else {
		p, u, caps, found := proven(runDir, repo, r.At, where, patches, union(a.Acceptance, union(a.Code, a.Tests)), again(a.Proof), s)
		this.Result, this.Turns, this.Tokens, this.Cached, this.Tools = p.Result, u.Turns, u.Tokens, u.Cached, u.Tools
		a.Proof, a.Caps = &p, caps
		code, tests := written(patches)
		a.Code, a.Tests = union(a.Code, code), union(a.Tests, tests)
		for _, f := range found {
			f.Message = strings.ReplaceAll(f.Message, "; its pull request opens as a Draft", "; pushed, the forge's checks and the reviewer read it next")
			v.Findings = append(v.Findings, f)
		}
		v.Summary += ", its fix pushed, the tests " + p.Result
		cs, unread := committerSettings(repo, where)
		v.Findings = append(v.Findings, unread...)
		if err := writeMergeRequest(runDir, map[string]any{"push": reviewSubject(a.Issue, n, cs)}); err != nil {
			return fail(err)
		}
	}
	if r.FiveWhys {
		v.Summary += ", the five whys with it"
	}
	a.Round, a.Whys = n, a.Whys || r.FiveWhys
	a.ReviewRuns = max(a.ReviewRuns, r.Runs)
	a.Heard = append(a.Heard, r.Heard...)
	for _, x := range r.Asked {
		if x.Holds != "a person" && !slices.Contains(a.Seen, x.Key) {
			a.Seen = append(a.Seen, x.Key)
		}
	}
	a.Rounds = append(a.Rounds, this)
	out := append(patches, intent.Intention{Kind: "comment", Value: map[string]any{"sticky": "round", "body": roundBody(this, s, notes, analysis)}},
		intent.Intention{Kind: "comment", Value: map[string]any{"sticky": RecordKey, "body": recordBody(a)}})
	out = append(out, others...)
	if err := writeIntents(runDir, "", out); err != 0 {
		return err
	}
	return writeVerdict(runDir, v)
}

// roundBody is the round's comment on its pull request: what was asked,
// what the developer did, its proof, the five whys.
func roundBody(x Round, s Settings, notes []string, analysis string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer**, review round %d of %d: ", x.N, s.Rounds)
	switch x.Result {
	case "no-code":
		b.WriteString("no fix within its bounds.\n\n")
	default:
		fmt.Fprintf(&b, "its fix pushed, the project's tests %s.\n\n", x.Result)
	}
	b.WriteString("Asked:\n\n")
	for _, t := range x.Asked {
		fmt.Fprintf(&b, "- %s\n", t)
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
