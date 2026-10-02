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

// IssueTitle is the tracking issue's: one, kept open, a comment a week.
const IssueTitle = "workline: the weekly sample of the docs vouched for"

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
		return failed(res, errors.New("a forge is needed (--forge github, gitlab)"))
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
	id, err := f.KeepIssue(IssueTitle, issueIntro, true)
	if err != nil {
		return failed(res, err)
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
	if res.Vouched == 0 {
		fmt.Fprintf(&b, "No doc vouched for by the documentalist in the commits %s: nothing read.\n", res.Window)
		return b.String()
	}
	fmt.Fprintf(&b, "%s vouched for by the documentalist in the commits %s (`%.7s..%.7s`); %d read, one in ten.\n\n", plural(res.Vouched, "doc"), res.Window, res.From, res.To, len(res.Reads))
	b.WriteString("| Doc | Vouched for in | `checked` | Verdict | Judge, independence |\n|---|---|---|---|---|\n")
	for _, r := range res.Reads {
		judgeWords := "—"
		if r.Judge != "" {
			judgeWords = fmt.Sprintf("`%s`, %s", r.Judge, r.Independence)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | **%s** | %s |\n", r.Doc, r.Commit, checkedWords(r.Checked), r.Verdict, judgeWords)
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
	return b.String()
}

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
