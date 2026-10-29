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

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/work"
)

// RecordKey is the sticky key of the record on its pull request.
const RecordKey = "code"

// isTest says whether a file is a test: its name holding "test" or
// "spec", or in a test folder.
func isTest(p string) bool {
	name := strings.ToLower(path.Base(p))
	if strings.Contains(name, "test") || strings.Contains(name, "spec") {
		return true
	}
	for _, part := range strings.Split(path.Dir(p), "/") {
		if part == "test" || part == "tests" || part == "spec" || part == "__tests__" {
			return true
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

// testsMessage is the message of the commit holding its tests.
func testsMessage(issue int) string {
	return fmt.Sprintf("test: the tests beside the code of #%d", issue)
}

// mergeRequest is out/merge-request.yaml: the branch's key, the issue's
// title, its body; ready only when green with no cap reached; the branch
// from the tester's Draft, the tests in a commit apart; the record kept
// on it.
func mergeRequest(a Act, title string) map[string]any {
	ready := a.Proof != nil && a.Proof.Result == Green && len(a.Caps) == 0
	mr := map[string]any{"key": fmt.Sprint(a.Issue), "title": title, "body": body(a, ready), "draft": !ready,
		"record": map[string]any{"sticky": RecordKey, "body": recordBody(a)}}
	if a.From != "" {
		mr["from"] = a.From
	}
	if len(a.Tests) > 0 {
		mr["commits"] = []map[string]any{{"message": testsMessage(a.Issue), "files": a.Tests}}
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
		fmt.Fprintf(&b, "The developer's code for #%d, **a Draft**: not proven green, or a cap reached. "+
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
	fmt.Fprintf(&b, "\n## %s\n\n", work.Name(work.Verification))
	if a.Proof != nil {
		b.WriteString(proofLines(*a.Proof, a.Caps))
	}
	b.WriteString("\n## Tried\n\n- Its code on the project's tests, in a checkout apart, before this opened: above.\n" +
		"- Not tried: the commands the issue names to verify it, nor a person's use of it.\n")
	return b.String()
}

// recordBody is the comment holding the record on its pull request: what
// a fresh run reads of the last, and the auditor of its code.
func recordBody(a Act) string {
	r := a
	r.Act, r.Passed = "code", nil
	return fmt.Sprintf("**The developer**'s record of this pull request, for its next runs and the auditor.\n\n%s\n", r.Line())
}
