package developer

// The pull request (ADR-0054, step 4): the App's, opened by the engine at
// apply. Its branch starts from the tester's Draft, the acceptance tests
// on it; its code and its tests in commits apart; its body says what
// changed, what was left out, what ran and what was not tried; its record
// is hidden in workline's own comment on it, for the next rounds and the
// auditor (ADR-0046).

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/testmark"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/commitmsg"
	"github.com/JN0V/workline/internal/kit/testdef"
	"github.com/JN0V/workline/internal/work"
)

// written are the files its patches touch, its code and its tests apart,
// by the project's one definition of a test file (ADR-0065).
func written(patches []intent.Intention, def testdef.Def) (code, tests []string) {
	for _, p := range patches {
		files, _ := touched(p.Value)
		for _, f := range files {
			if def.Test(f) {
				tests = append(tests, f)
			} else {
				code = append(code, f)
			}
		}
	}
	slices.Sort(code)
	slices.Sort(tests)
	return slices.Compact(code), slices.Compact(tests)
}

// Subjects are its commits' subjects, its tests' and its code's (its title,
// a Draft's too); Note, what its body says of a type a person may change;
// Refused, its agent's summary when it did not pass.
type Subjects struct{ Tests, Code, Note, Refused string }

// shippedTaskNote is what a task typed fix says to a person (#2198).
const shippedTaskNote = "- Typed `fix`: a task that changes shipped code, so that release tools list it and bump its package. " +
	"If it removes or renames a public name, a person marks it a breaking change: `fix!:` in its title, before the merge."

// squashRoom is what a squash adds to the title it lands, GitHub's
// " (#N)" for a number of five digits (ADR-0051), as forge.TitleMax.
var squashRoom = len(" (#12345)")

// subjects are its commits' subjects for the issue, as the project's
// committer checks them: `test` and the issue's kind, the folder its
// Scope shares, a task's type by the code it changes; the summary its
// agent wrote (own, one line, its own `type: ` dropped), else its title,
// each as summaryOf reads it, the first whose subject passes the check
// where a squash lands it, a Draft's prefix too, never cut; else "the
// code of #n".
func subjects(issue int, own, title, kind string, scope, code []string, s commitmsg.Settings, always ...string) Subjects {
	sc, typ, r, own := commitScope(scope), commitType(kind, code, always...), s, strings.TrimSpace(own)
	if r.SubjectMax = forge.TitleMax - len("Draft: "); s.SubjectMax > 0 {
		r.SubjectMax = min(s.SubjectMax-squashRoom, forge.TitleMax) - len("Draft: ")
	}
	sum, mine := fmt.Sprintf("the code of #%d", issue), summaryOf(typeHead.ReplaceAllString(own, ""), s)
	if strings.ContainsAny(own, "\r\n") {
		mine = "" // one line, or none
	}
	for _, c := range []string{mine, summaryOf(title, s)} {
		if found, err := commitmsg.Check(subject(typ, sc, c, r), r); c != "" && err == nil && len(found) == 0 {
			sum = c
			break
		}
	}
	sub := Subjects{Tests: subject("test", sc, sum, r), Code: subject(typ, sc, sum, r)}
	if own != "" && mine != sum {
		sub.Refused = own
	}
	if kind == devmark.KindTask && typ == "fix" && strings.HasPrefix(sub.Code, typ) {
		sub.Note = shippedTaskNote
	}
	return sub
}

// mergeRequest is out/merge-request.yaml: the branch's key, the issue's
// title, its body; ready only when green with no cap reached; the branch
// from the tester's Draft, the tests in a commit apart; the record kept
// on it.
func mergeRequest(a devmark.Act, title string, sub Subjects) map[string]any {
	ready := a.Proof != nil && a.Proof.Result == devmark.Green && len(a.Caps) == 0 && a.AgentFailed == ""
	b := body(a, ready)
	if sub.Note != "" {
		b = strings.Replace(b, "## Changes\n\n", "## Changes\n\n"+sub.Note+"\n", 1)
	}
	mr := map[string]any{"key": fmt.Sprint(a.Issue), "title": title, "body": b, "draft": !ready,
		"record": map[string]any{"sticky": devmark.RecordKey, "body": recordBody(a)}, "message": sub.Code}
	if sub.Code != "" {
		mr["title"] = sub.Code
	}
	if a.From != "" {
		mr["from"] = a.From
	}
	if a.Merged != "" {
		mr["merged"] = a.Merged
	}
	if len(a.Tests) > 0 {
		mr["commits"] = []map[string]any{{"message": sub.Tests, "files": a.Tests}}
	}
	return mr
}

// body is its pull request's body: what closes, what changed, what was
// left out, what ran, what was tried and not.
func body(a devmark.Act, ready bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Closes #%d\n\n", a.Issue)
	if left := personsRefusals(a.Refused); len(left) > 0 {
		// On top, where the reviewer's intent lens and a person read
		// first: what only a person may change, left out of this change.
		fmt.Fprintf(&b, "## %s\n\nRefused at apply: only a person changes these files. Merged as it is, #%d may close with them still to do.\n\n", LeftHeading, a.Issue)
		for _, r := range left {
			fmt.Fprintf(&b, "- `%s`: %s\n", r.File, why(r))
		}
		b.WriteString("\n")
	}
	if ready {
		fmt.Fprintf(&b, "The developer's code for #%d, proven on the project's tests.\n\n", a.Issue)
	} else {
		fmt.Fprintf(&b, "The developer's code for #%d, **a Draft**: not proven green, a cap reached, or its agent failed. "+
			"A person takes it from here, or removes the claim on the issue.\n\n", a.Issue)
	}
	b.WriteString("## Changes\n\n")
	files := func(l []string) string {
		q := make([]string, len(l))
		for i, f := range l {
			q[i] = "`" + f + "`"
		}
		return strings.Join(q, ", ")
	}
	if len(a.Code) > 0 {
		fmt.Fprintf(&b, "- Code: %s.\n", files(a.Code))
	}
	if len(a.Tests) > 0 {
		fmt.Fprintf(&b, "- Tests, in a commit apart: %s.\n", files(a.Tests))
	}
	if len(a.Beside) > 0 {
		fmt.Fprintf(&b, "- New test files beside the issue's %s: %s; the product owner adds them to the %[1]s at its next run.\n", work.Name(work.Scope), files(a.Beside))
	}
	if a.NoTest {
		b.WriteString("- The tester found that no acceptance test applied to this issue: no test is frozen for it.\n")
	}
	if len(a.Acceptance) > 0 {
		fmt.Fprintf(&b, "- The tester's acceptance tests, read-only to the developer, from `%s%d`: %s.\n", testmark.TestsBranch, a.Issue, files(a.Acceptance))
	}
	if a.RedUnproven != "" {
		fmt.Fprintf(&b, "- **Red unproven**: their red proof on the base was not settled by the tester's judge: %s.\n", a.RedUnproven)
	}
	if len(a.Beyond) > 0 {
		fmt.Fprintf(&b, "\n## %s\n\nFiles changed beyond the issue's %s, the product owner's hint, each with its agent's reason: the reviewer's intent lens judges each.\n\n%s",
			BeyondHeading, work.Name(work.Scope), devmark.BeyondLines(a.Beyond))
	}
	if len(a.Unproven) > 0 {
		fmt.Fprintf(&b, "\n## %s\n\n%s\n\n%s\n", UnprovenHeading, unprovenLead, strings.Join(a.Unproven, "\n"))
	}
	if len(a.Refused) > 0 {
		b.WriteString("\n## Left out\n\nRefused at apply, out of its bounds: an issue for a person, not this change.\n\n")
		for _, r := range a.Refused {
			fmt.Fprintf(&b, "- `%s`: %s\n", r.File, why(r))
		}
	}
	if len(a.Unread) > 0 {
		b.WriteString("\n## Its agent's words, unread\n\nA person reads them: a part not done, or an objection to a frozen test, may be among them.\n\n")
		for _, u := range a.Unread {
			b.WriteString(u + "\n")
		}
	}
	fmt.Fprintf(&b, "\n## %s\n\n", work.Name(work.Verification))
	fmt.Fprintf(&b, "- %s\n", rightsLine())
	if a.AgentFailed != "" {
		fmt.Fprintf(&b, "- Its agent failed: %s. What it edited was proven all the same.\n", a.AgentFailed)
	}
	if a.Proof != nil {
		b.WriteString(devmark.ProofLines(*a.Proof, a.Caps))
	}
	b.WriteString("\n## Tried\n\n- Its code on the project's tests, on a fresh checkout, before this opened: above.\n" +
		"- Not tried: the commands the issue names to verify it, nor a person's use of it.\n")
	return b.String()
}

// LeftHeading heads what its change left for a person, on top of its
// pull request.
const LeftHeading = "Left for a person"

// personsRefusals are the refusals of a file only a person changes — a
// gate's, the CI's, one that steers agents, one not text: work the issue
// may need, left. A frozen test, a path out of the worktree, are not.
func personsRefusals(refused []devmark.Refusal) []devmark.Refusal {
	var out []devmark.Refusal
	for _, r := range refused {
		switch r.Rule {
		case "gate-file", "steering-file", "not-text":
			out = append(out, r)
		}
	}
	return out
}

// recordBody is the comment holding the record on its pull request: what
// a fresh run reads of the last, and the auditor of its code.
func recordBody(a devmark.Act) string {
	r := a
	r.Act, r.Passed = "code", nil
	var b strings.Builder
	b.WriteString("**The developer**'s record of this pull request, for its next runs and the auditor.\n\n")
	for _, x := range r.Rounds {
		fmt.Fprintf(&b, "- Review round %d: %d asked, %s", x.N, len(x.Asked), devmark.RoundResult(x))
		if x.Whys {
			b.WriteString(", the five whys run")
		}
		b.WriteString(".\n")
	}
	if len(r.Rounds) > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%s\n", r.Line())
	return b.String()
}
