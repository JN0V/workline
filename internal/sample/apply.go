package sample

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// IssueTitle is the tracking issue's: one, kept open, a comment a week,
// and the count (ADR-0046).
const IssueTitle = "workline: the weekly sample of every role's acts"

// formerTitle is the tracking issue's before every role's acts were
// sampled: renamed, never opened again.
const formerTitle = "workline: the weekly sample of the docs vouched for"

// StepZeroLabel marks the tracking issue once a `checked` is found false:
// the documentalist is set back to step 0 (ADR-0014), until a person takes
// the label off.
const StepZeroLabel = "documentalist-step-0"

const issueIntro = `Each week, one in ten of the docs the documentalist vouched for — whose
` + "`checked`" + ` it moved — is read whole against its sources, at the commit
` + "`checked`" + ` names, by a judge standing apart from the model that vouched
(ADR-0014, step 4; ADR-0005). The engine checks every quote the judge gives:
a passage is false only when a source's code, not a comment, says otherwise.

A comment a week, below. One false ` + "`checked`" + ` sets the documentalist back to
step 0 (label ` + "`" + StepZeroLabel + "`" + `, which a person takes off), and a merge
request puts that doc's ` + "`checked`" + ` back, for a person to review. What the
judge says false and cannot prove is for a person.

The other roles' acts are drawn too, one in ten per kind of act (ADR-0046):
the product owner's acts done alone, the findings the reviewer's judge
kept or dropped, the commits the committer rewrote. The auditor re-checks
each against what it was done on, on another model than those that
acted; one said wrong is false only with the material's words, found
again, and three judges settle the dispute. A true finding the judge
dropped, still in the code, is opened as an issue (three a week at most).
A product owner's act you find wrong: undo it on its issue, and the role
proposes that kind there from its next run (ADR-0038); the level its acts
done alone suggest is said, never set (ADR-0033).

Each week's comment numbers its acts. Agree with the week: 👍 on it.
Contest rows: 👎, and a reply naming them (` + "`not 3, 7`" + `). One comment, the count,
is written again each week from every week.

Written by ` + "`workline sample --apply`" + `, with no AI.`

// Apply writes what a sample found to the forge: the week's comment on the
// tracking issue, the step-0 label when a `checked` is false, and a merge
// request putting back each false `checked` still standing.
func Apply(file, repo string, f forge.Forge) *Result {
	res := &Result{}
	data, err := os.ReadFile(file)
	if err == nil {
		err = json.Unmarshal(data, res)
	}
	if err != nil {
		return failed(res, err)
	}
	if res.Status == verdict.Block || res.Week == "" {
		// A sample that did not run is not written as a week with nothing in it.
		return failed(res, fmt.Errorf("%s: this sample did not run (%s); nothing written", file, res.Summary))
	}
	res.Status, res.Applied, res.Refused, res.Calls, res.AgentCalls = verdict.Pass, []string{}, []string{}, nil, 0
	res.Findings = nil
	if f == nil {
		return failed(res, errors.New("no forge to write to; "+forge.Missing))
	}
	falseOnes, standing, moved := []Read{}, []Read{}, []Read{}
	for _, r := range res.Reads {
		if r.Verdict != "false" && r.Verdict != "unearned" {
			continue
		}
		falseOnes = append(falseOnes, r)
		if stillVouched(repo, r) {
			standing = append(standing, r)
		} else {
			moved = append(moved, r)
		}
	}
	mr := 0
	if len(standing) > 0 {
		var written []string
		for _, r := range standing {
			if err := putBack(repo, r); err != nil {
				return failed(res, err)
			}
			written = append(written, r.Doc)
		}
		mr, err = engine.ProposeBranch(f, repo, "documentalist", "sample-"+res.Week,
			"docs: put back the checked the weekly sample found false", resetBody(res, standing),
			engine.OwnTrailer+": documentalist", written)
		if err != nil {
			return failed(res, err)
		}
		res.Applied = append(res.Applied, "merge-request")
	}
	former, err := renameFormer(f)
	if err != nil {
		return failed(res, err)
	}
	if err := openDefects(res, repo, f); err != nil {
		return failed(res, fmt.Errorf("the true findings dropped: %w", err))
	}
	id, err := f.KeepIssue(IssueTitle, issueIntro, true)
	if err != nil {
		return failed(res, err)
	}
	if err := foldActs(res, f, former, id); err != nil {
		return failed(res, fmt.Errorf("the product owner's former issue: %w", err))
	}
	issue := forge.Target{Kind: "issue", ID: id}
	if err := f.Sticky(issue, report(res, falseOnes, moved, mr), forge.Marker("sample:"+res.Week), true); err != nil {
		return failed(res, err)
	}
	res.Applied = append(res.Applied, "comment")
	if len(falseOnes) > 0 {
		if err := f.Label(issue, []string{StepZeroLabel}, nil); err != nil {
			return failed(res, err)
		}
		res.Applied = append(res.Applied, "label")
	}
	// The count, written again from every week, this one's included; a
	// forge that cannot read the weeks back leaves the one there.
	if _, ok := f.(forge.Backlog); ok {
		ws, err := weeks(f, id)
		if err != nil {
			return failed(res, fmt.Errorf("the weeks, read back: %w", err))
		}
		if err := f.Sticky(issue, countComment(ws), forge.Marker(countKey), true); err != nil {
			return failed(res, err)
		}
		res.Applied = append(res.Applied, "count")
	}
	for _, a := range res.Drawn {
		if d := a.Defect; d != nil && strings.HasPrefix(d.Said, "opened") {
			res.Applied = append(res.Applied, "issue")
			res.Findings = append(res.Findings, verdict.Finding{Rule: "defect-opened", Level: "info", Where: fmt.Sprintf("#%d", d.Issue),
				Message: fmt.Sprintf("a true finding the reviewer's judge dropped on %s, still at %s: %s", a.Ref, d.Path, d.Said)})
		}
	}
	res.Summary = fmt.Sprintf("%s written to issue #%d", res.Week, id)
	if mr > 0 {
		res.Summary += fmt.Sprintf(", merge request #%d", mr)
	}
	return res
}

func failed(res *Result, err error) *Result {
	res.Status, res.Summary = verdict.Block, err.Error()
	if errors.Is(err, forge.ErrUnreachable) {
		res.Status = verdict.BlockedExternal
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
	return res
}

// stillVouched: the doc, as it is now, still carries the `checked` read,
// in this repository alone.
func stillVouched(repo string, r Read) bool {
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(r.Doc)))
	if err != nil || len(r.Checked) != 1 || r.Checked[""] == "" {
		return false
	}
	d, _ := documentalist.ParseDoc(r.Doc, data)
	return d != nil && len(d.Checked) == 1 && d.Checked[""] == r.Checked[""]
}

var (
	checkedLine = regexp.MustCompile(`^checked:\s*\S.*$`)
	judgedLine  = regexp.MustCompile(`^judged:\s*\S.*$`)
)

// putBack sets the doc's `checked` back to what it was before it was
// vouched for, and `judged` to the commit it was vouched at: the doc shows
// as suspect again, for a person (as 0669684 undid the `checked` no task
// earned, ADR-0014 step 0).
func putBack(repo string, r Read) error {
	p := filepath.Join(repo, filepath.FromSlash(r.Doc))
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	_, n := documentalist.Header(string(data))
	lines := strings.Split(string(data), "\n")
	var head []string
	judged := false
	for _, l := range lines[:n] {
		switch {
		case checkedLine.MatchString(l):
			if prev := r.Previous[""]; prev != "" {
				head = append(head, "checked: "+prev)
			}
			head = append(head, "judged: "+r.Checked[""])
			judged = true
		case judgedLine.MatchString(l):
			// replaced beside `checked`
		default:
			head = append(head, l)
		}
	}
	if !judged {
		return fmt.Errorf("%s: no `checked` line in its header", r.Doc)
	}
	return os.WriteFile(p, []byte(strings.Join(append(head, lines[n:]...), "\n")), 0o644)
}

// resetBody is the merge request's description.
func resetBody(res *Result, docs []Read) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The weekly sample (%s) read these docs against their sources at the commit their `checked` names, and found that `checked` false. It is put back to what it was before, `judged` at the commit vouched for: the doc shows as suspect again, for a person.\n\n", res.Week)
	for _, r := range docs {
		b.WriteString(docLines(r))
	}
	b.WriteString("\nMerge it, or close it if the doc is right. ADR-0014, step 4; the tracking issue: \"" + IssueTitle + "\".\n")
	return b.String()
}

// report is the week's comment on the tracking issue.
func report(res *Result, falseOnes, moved []Read, mr int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", res.Week)
	rs := rows(res)
	b.WriteString(synthesis(res, rs))
	hidden := "\n" + hiddenRows(rs) + "\n"
	if res.Vouched == 0 {
		fmt.Fprintf(&b, "No doc vouched for by the documentalist in the commits %s: nothing read.\n", res.Window)
		return b.String() + actsSection(res) + hidden
	}
	fmt.Fprintf(&b, "%s vouched for by the documentalist in the commits %s (`%.7s..%.7s`); %d read, one in ten.\n\n", plural(res.Vouched, "doc"), res.Window, res.From, res.To, len(res.Reads))
	b.WriteString("| # | Doc | Vouched for in | `checked` | Verdict | Judge, independence | Panel |\n|---|---|---|---|---|---|---|\n")
	for i, r := range res.Reads {
		judgeWords := "—"
		if r.Judge != "" {
			judgeWords = fmt.Sprintf("`%s`, %s", r.Judge, r.Independence)
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | **%s** | %s | %s |\n", i+1, r.Doc, r.Commit, checkedWords(r.Checked), r.Verdict, judgeWords, cell(r.Panel.Words()))
	}
	b.WriteString("\n")
	for _, r := range res.Reads {
		if r.Verdict != "true" {
			b.WriteString(docLines(r))
		}
	}
	if len(falseOnes) > 0 {
		b.WriteString("\n**The documentalist is set back to step 0** (ADR-0014): a false `checked` was found. Label `" + StepZeroLabel + "`; a person takes it off once step 0 is passed again.\n")
		if mr > 0 {
			fmt.Fprintf(&b, "Merge request #%d puts the `checked` back, for a person to review.\n", mr)
		}
		for _, r := range moved {
			fmt.Fprintf(&b, "%s no longer carries that `checked`, or names another repository's: for a person to put back.\n", r.Doc)
		}
	}
	return b.String() + actsSection(res) + hidden
}

// synthesis opens the week's comment, for the maintainer: the acts read
// per role, how many stand, each false one in a line with the panel's
// vote, and how to answer.
func synthesis(res *Result, rs []Row) string {
	if len(rs) == 0 {
		return "Nothing drawn this week.\n\n"
	}
	var b strings.Builder
	read, stand := 0, 0
	type tally struct{ read, stand, falseOnes, other int }
	per := map[string]*tally{}
	for _, r := range rs {
		t := per[r.Role]
		if t == nil {
			t = &tally{}
			per[r.Role] = t
		}
		if r.Verdict == "not-read" {
			t.other++
			continue
		}
		read++
		t.read++
		switch r.Verdict {
		case "true":
			stand++
			t.stand++
		case "false":
			t.falseOnes++
		default:
			t.other++
		}
	}
	fmt.Fprintf(&b, "**%s read of %d drawn: %d stand.**\n\n", plural(read, "act"), len(rs), stand)
	for _, role := range roleOrder {
		if t := per[role]; t != nil {
			fmt.Fprintf(&b, "- %s: %d read, %d stand, %d false, %d for a person\n", roleWords(role), t.read, t.stand, t.falseOnes, t.other)
		}
	}
	var falseLines []string
	for i, r := range res.Reads {
		if r.Verdict == "false" {
			falseLines = append(falseLines, fmt.Sprintf("- %d. %s, its `checked` false — %s", i+1, r.Doc, panelLine(r.Panel)))
		}
	}
	for i, a := range res.Drawn {
		if a.Verdict == "false" {
			l := fmt.Sprintf("- %d. %s — %s", len(res.Reads)+i+1, oneLine(actWords(Act{Role: a.Role, Kind: a.Kind, Ref: a.Ref, Where: a.Where, Day: a.Day, By: a.By, In: a.In}), 160), panelLine(a.Panel))
			if a.Defect != nil && a.Defect.Said != "" {
				l += "; " + a.Defect.Said
			}
			falseLines = append(falseLines, l)
		}
	}
	if len(falseLines) > 0 {
		b.WriteString("\nFalse, each with the panel's vote:\n\n" + strings.Join(falseLines, "\n") + "\n")
	}
	b.WriteString("\nAgree with the week: 👍 on this comment. Contest rows: 👎, and a reply naming them (`not 3, 7`).\n\n")
	return b.String()
}

// panelLine says how a false verdict was settled.
func panelLine(p *Panel) string {
	if p == nil {
		return "no panel: the auditor's reading, for a person"
	}
	return "the panel " + p.Words()
}

// renameFormer gives the tracking issue its title widened to every role's
// acts, when it still has its former one: one issue, never a second. It
// says the product owner's own tracking issues still open, to fold in
// (ADR-0046).
func renameFormer(f forge.Forge) ([]int, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, nil
	}
	open, err := b.Issues()
	if err != nil {
		return nil, err
	}
	former, kept := 0, false
	var acts []int
	for _, is := range open {
		switch is.Title {
		case IssueTitle:
			kept = true
		case formerTitle:
			if former == 0 {
				former = is.ID
			}
		case formerActsTitle:
			acts = append(acts, is.ID)
		}
	}
	if former == 0 || kept {
		return acts, nil
	}
	return acts, b.SetTitle(former, IssueTitle)
}

// actsSection is what the week's comment says of every role's acts: how
// many of each kind the window held and how many were drawn, the other
// roles' acts drawn, for a person, and what could not be drawn (ADR-0046).
func actsSection(res *Result) string {
	var b strings.Builder
	b.WriteString("\n#### Every role's acts\n\n")
	b.WriteString("One in ten of a role's acts of each kind, at least one when the kind has any, each re-checked by the auditor against what it was done on (ADR-0046). False only with the material's words, found again; the rest that does not stand is for a person.\n\n")
	if len(res.Counts) == 0 {
		b.WriteString("No act of any role in the window.\n")
	} else {
		b.WriteString("| Role | Kind of act | In the window | Drawn |\n|---|---|---|---|\n")
		for _, c := range res.Counts {
			fmt.Fprintf(&b, "| %s | %s | %d | %d |\n", strings.TrimPrefix(roleWords(c.Role), "the "), kindWords(c), c.Acts, c.Drawn)
		}
	}
	if len(res.Drawn) > 0 {
		b.WriteString("\n| # | Act | Role, kind | Done | What its record says | Verdict | Judge, independence | Panel |\n|---|---|---|---|---|---|---|---|\n")
		for i, a := range res.Drawn {
			act := a.Ref
			if a.In != "" {
				act += " (" + a.In + ")"
			}
			if a.Where != "" {
				act += " `" + a.Where + "`"
			}
			done := a.Day
			switch {
			case a.Role == productOwner && a.By != "":
				done += ", at " + a.By
			case a.Role == reviewRole && a.Commit != "":
				done = fmt.Sprintf("at %.7s", a.Commit)
			case a.Role == reviewRole:
				done = "its last review"
			}
			line := oneLine(a.Line, 300)
			if a.Role != productOwner && a.By != "" && !strings.Contains(line, a.By) {
				line += " (" + a.By + ")"
			}
			if a.Undone != "" {
				line += " — **undone**: " + a.Undone
			}
			judgeWords := "—"
			if a.Judge != "" {
				judgeWords = fmt.Sprintf("`%s`, %s", a.Judge, a.Independence)
			}
			fmt.Fprintf(&b, "| %d | %s | %s, %s | %s | %s | **%s** | %s | %s |\n", len(res.Reads)+i+1, cell(act), strings.TrimPrefix(roleWords(a.Role), "the "), cell(kindWords(Count{Role: a.Role, Kind: a.Kind})), cell(done), cell(line), a.Verdict, cell(judgeWords), cell(a.Panel.Words()))
		}
		b.WriteString("\n")
		for _, a := range res.Drawn {
			b.WriteString(actLines(a))
		}
	}
	if len(res.Older) > 0 {
		w := olderWords(res.Older)
		fmt.Fprintf(&b, "\n%s%s.\n", strings.ToUpper(w[:1]), w[1:])
	}
	if res.Autonomy != nil {
		b.WriteString(autonomyLines(res.Autonomy))
	}
	if len(res.NotDrawn) > 0 {
		b.WriteString("\nNot drawn:\n\n")
		for _, why := range res.NotDrawn {
			fmt.Fprintf(&b, "- %s\n", why)
		}
	}
	return b.String()
}

// actLines say what was found of an act that does not stand, for a person.
func actLines(a Act) string {
	var b strings.Builder
	name := a.Ref
	if a.Where != "" {
		name += " `" + a.Where + "`"
	}
	switch a.Verdict {
	case "false":
		fmt.Fprintf(&b, "- **%s**, false:\n", name)
		for _, p := range a.Proofs {
			fmt.Fprintf(&b, "  - %s\n", p.String())
		}
		fmt.Fprintf(&b, "  - %s\n", panelLine(a.Panel))
		if a.Defect != nil && a.Defect.Said != "" {
			fmt.Fprintf(&b, "  - the defect: %s\n", a.Defect.Said)
		}
	case "unproven":
		fmt.Fprintf(&b, "- **%s**: said wrong, not proven, for a person:\n", name)
		for _, p := range a.Unproven {
			fmt.Fprintf(&b, "  - %s\n", p.Why)
		}
	case "not-read":
		fmt.Fprintf(&b, "- **%s**, not read: %s\n", name, a.Why)
	}
	return b.String()
}

// cell is a text as a table's cell holds it: `|` escaped.
func cell(t string) string { return strings.ReplaceAll(t, "|", `\|`) }

// docLines say what was found in one doc, for a person.
func docLines(r Read) string {
	var b strings.Builder
	switch r.Verdict {
	case "false":
		fmt.Fprintf(&b, "- **%s**, false:\n", r.Doc)
		for _, p := range r.Proofs {
			fmt.Fprintf(&b, "  - %s\n", p.String())
		}
	case "unproven":
		fmt.Fprintf(&b, "- **%s**: said false, not proven, for a person:\n", r.Doc)
		for _, p := range r.Unproven {
			fmt.Fprintf(&b, "  - %q — %s\n", p.Passage, p.Why)
		}
	case "unearned":
		fmt.Fprintf(&b, "- **%s**, unearned: %s\n", r.Doc, r.Why)
	case "not-read":
		fmt.Fprintf(&b, "- **%s**, not read: %s\n", r.Doc, r.Why)
	}
	return b.String()
}
