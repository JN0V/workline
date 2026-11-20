package tester

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/kit/commitmsg"
)

// The tester's commit on an objection keeps its subject within the
// committer's limit whatever the path, the full path in the body (#1941).
func TestObjectionCommitSubjectWithinLimit(t *testing.T) {
	long := "tests/conformance/cases/tester/no-test-applies-hands-over.yaml"
	longer := "tests/" + strings.Repeat("deep/", 10) + strings.Repeat("x", 80) + ".yaml"
	for _, c := range []struct {
		file    string
		missing bool
		want    string
	}{
		{"tests/test_sub.sh", false, "test: rewrite test_sub.sh on an objection"},
		{"calc/calc_test.go", true, "test: add a test the review asks to calc_test.go"},
		{long, false, "test: rewrite no-test-applies-hands-over.yaml on an objection"},
		{long, true, "test: add a test the review asks to no-test-applies-hands-over.yaml"},
		{longer, false, "test: rewrite a test on an objection"},
		{longer, true, "test: add a test the review asks for"},
	} {
		msg := objectionCommit(c.file, "https://example.test/pr/1#c2", c.missing, commitmsg.Settings{SubjectMax: subjectMax, Types: []string{"test"}})
		subject, body, _ := strings.Cut(msg, "\n\n")
		if subject != c.want {
			t.Errorf("%s (missing %v): subject %q, want %q", c.file, c.missing, subject, c.want)
		}
		if n := utf8.RuneCountInString(subject); n > subjectMax {
			t.Errorf("%s: subject is %d characters, over %d", c.file, n, subjectMax)
		}
		if f, err := commitmsg.Check(msg, commitmsg.Settings{SubjectMax: subjectMax, Types: []string{"test"}}); err != nil || len(f) > 0 {
			t.Errorf("%s: the committer refuses %q: %v %v", c.file, msg, f, err)
		}
		if !strings.Contains(body, c.file) || !strings.HasSuffix(body, "Refs https://example.test/pr/1#c2") {
			t.Errorf("%s: body %q lacks the full path or the reference", c.file, body)
		}
	}
}
