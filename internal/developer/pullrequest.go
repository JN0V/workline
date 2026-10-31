package developer

// The pull request (ADR-0054, step 4): the App's, opened by the engine at
// apply. Its branch starts from the tester's Draft, the acceptance tests
// on it; its code and its tests in commits apart; its body says what
// changed, what was left out, what ran and what was not tried; its record
// is hidden in workline's own comment on it, for the next rounds and the
// auditor (ADR-0046).

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/work"
)

// RecordKey is the sticky key of the record on its pull request.
const RecordKey = "code"

// Branch opens its branches' names: workline/developer/<issue>.
const Branch = "workline/developer/"

// ReadRecord reads the record of its code on its merge request id, from
// workline's own comment; ok is false when there is none it can read.
func ReadRecord(f forge.Forge, id int) (Act, bool, error) {
	kept, err := forge.ReadKept(f, forge.Target{Kind: "merge-request", ID: id})
	if err != nil {
		return Act{}, false, err
	}
	body, ok := kept.Last(forge.Marker("sticky=developer/" + RecordKey))
	if !ok {
		return Act{}, false, nil
	}
	a, ok := ReadAct(body)
	return a, ok && a.Act == "code", nil
}

// ProofLines are a proof's lines, as its pull request's body says them.
func ProofLines(p Proof, caps []Cap) string { return proofLines(p, caps) }

// isTest says whether a file is a test: "test" or "spec" a word of its
// name (test_x.py, x_test.go, x.spec.ts, XTest.java), or in a test folder.
func isTest(p string) bool {
	if testName(p) {
		return true
	}
	for _, part := range strings.Split(path.Dir(p), "/") {
		if part == "test" || part == "tests" || part == "spec" || part == "__tests__" {
			return true
		}
	}
	return false
}

// testName says whether a file's name is a test's: "test" or "spec" a
// word of it. A hidden file's never is: .env.test is a setting (#745).
func testName(p string) bool {
	base := path.Base(p)
	if strings.HasPrefix(base, ".") {
		return false
	}
	stem := strings.TrimSuffix(base, path.Ext(base))
	for _, w := range strings.FieldsFunc(stem, func(c rune) bool { return c == '_' || c == '.' || c == '-' }) {
		switch lw := strings.ToLower(w); {
		case lw == "test" || lw == "tests" || lw == "spec" || lw == "conftest":
			return true
		case strings.HasSuffix(w, "Test") || strings.HasSuffix(w, "Tests") || strings.HasSuffix(w, "Spec"):
			return true // XTest.java, XSpec.scala
		case len(w) > 4 && strings.HasPrefix(w, "Test") && unicode.IsUpper(rune(w[4])):
			return true // TestX.java, not Testimony.go
		}
	}
	return false
}

// written are the files its patches touch, its code and its tests apart.
func written(patches []intent.Intention) (code, tests []string) {
	for _, p := range patches {
		files, _ := touched(p.Value)
		for _, f := range files {
			if isTest(f) {
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

// Subjects are its commits' subjects, its tests' and its code's, and its
// pull request's title, ready and as a Draft: the code's subject, room
// left for what a squash adds.
type Subjects struct{ Tests, Code, Title, DraftTitle string }

// squashRoom is what a squash adds to the title it lands, GitHub's
// " (#N)" for a number of five digits (ADR-0051), as forge.TitleMax.
var squashRoom = len(" (#12345)")

// subjects are its commits' subjects for the issue, as the project's
// committer checks them: `test` and the issue's kind, the folder its
// Scope shares, its title. Its pull request's title is the code's
// subject, cut so that a squash lands it within the limit, and the
// engine keeps it: within forge.TitleMax, GitLab's "Draft: " too.
func subjects(issue int, title, kind string, scope []string, s committer.Settings) Subjects {
	sum := summaryOf(title, s)
	if sum == "" {
		sum = fmt.Sprintf("the code of #%d", issue)
	}
	sc, typ := commitScope(scope), commitType(kind)
	room := func(less int) committer.Settings {
		r := s
		r.SubjectMax = forge.TitleMax - less
		if s.SubjectMax > 0 {
			r.SubjectMax = min(s.SubjectMax-squashRoom, forge.TitleMax) - less
		}
		return r
	}
	return Subjects{Tests: subject("test", sc, sum, s), Code: subject(typ, sc, sum, s),
		Title: subject(typ, sc, sum, room(0)), DraftTitle: subject(typ, sc, sum, room(len("Draft: ")))}
}

// mergeRequest is out/merge-request.yaml: the branch's key, the issue's
// title, its body; ready only when green with no cap reached; the branch
// from the tester's Draft, the tests in a commit apart; the record kept
// on it.
func mergeRequest(a Act, title string, sub Subjects) map[string]any {
	ready := a.Proof != nil && a.Proof.Result == Green && len(a.Caps) == 0 && a.AgentFailed == ""
	mr := map[string]any{"key": fmt.Sprint(a.Issue), "title": title, "body": body(a, ready), "draft": !ready,
		"record": map[string]any{"sticky": RecordKey, "body": recordBody(a)}, "message": sub.Code}
	switch {
	case ready && sub.Title != "":
		mr["title"] = sub.Title
	case !ready && sub.DraftTitle != "":
		mr["title"] = sub.DraftTitle
	}
	if a.From != "" {
		mr["from"] = a.From
	}
	if len(a.Tests) > 0 {
		mr["commits"] = []map[string]any{{"message": sub.Tests, "files": a.Tests}}
	}
	return mr
}

// body is its pull request's body: what closes, what changed, what was
// left out, what ran, what was tried and not.
func body(a Act, ready bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Closes #%d\n\n", a.Issue)
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
	if len(a.Acceptance) > 0 {
		fmt.Fprintf(&b, "- The tester's acceptance tests, read-only to the developer, from `%s%d`: %s.\n", tester.TestsBranch, a.Issue, files(a.Acceptance))
	}
	if len(a.Refused) > 0 {
		b.WriteString("\n## Left out\n\nRefused at apply, out of its bounds: an issue for a person, not this change.\n\n")
		for _, r := range a.Refused {
			fmt.Fprintf(&b, "- `%s`: %s\n", r.File, why(r))
		}
	}
	if len(a.Unread) > 0 {
		b.WriteString("\n## Its agent's words, unread\n\nA person reads them: an objection to a frozen test may be among them.\n\n")
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
		b.WriteString(proofLines(*a.Proof, a.Caps))
	}
	b.WriteString("\n## Tried\n\n- Its code on the project's tests, on a fresh checkout, before this opened: above.\n" +
		"- Not tried: the commands the issue names to verify it, nor a person's use of it.\n")
	return b.String()
}

// recordBody is the comment holding the record on its pull request: what
// a fresh run reads of the last, and the auditor of its code.
func recordBody(a Act) string {
	r := a
	r.Act, r.Passed = "code", nil
	var b strings.Builder
	b.WriteString("**The developer**'s record of this pull request, for its next runs and the auditor.\n\n")
	for _, x := range r.Rounds {
		fmt.Fprintf(&b, "- Review round %d: %d asked, %s", x.N, len(x.Asked), roundResult(x))
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

// RoundLines are its review rounds as the auditor reads them: each, what
// was asked and how it ended; "" for none (ADR-0046).
func RoundLines(a Act) string {
	var b strings.Builder
	for _, x := range a.Rounds {
		fmt.Fprintf(&b, "- Round %d: %s", x.N, roundResult(x))
		if x.Whys {
			b.WriteString(", the five whys run")
		}
		if x.Nits > 0 {
			fmt.Fprintf(&b, ", %d nit%s not asked", x.Nits, plural(x.Nits))
		}
		b.WriteString(".\n")
		for _, t := range x.Asked {
			fmt.Fprintf(&b, "  - asked: %s\n", t)
		}
	}
	return b.String()
}

// roundResult says how a review round ended.
func roundResult(x Round) string {
	switch {
	case x.Person:
		return "its issue blocked"
	case x.Result == "no-code":
		return "no fix"
	}
	return "its fix pushed, the tests " + x.Result
}
