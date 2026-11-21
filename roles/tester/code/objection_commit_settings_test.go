package tester

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JN0V/workline/internal/kit/commitmsg"
)

// Given a project with subject-max: 50, when the tester commits an
// objection rewrite of test_mul_two_by_three.sh, then the subject is at
// most 50 characters, and the committer accepts it.
func TestObjectionCommitSubjectFollowsTheProjectsSubjectMax(t *testing.T) {
	s := commitmsg.Settings{SubjectMax: 50, Types: []string{"test", "feat", "fix"}}
	file := "tests/test_mul_two_by_three.sh"
	for _, missing := range []bool{false, true} {
		msg := objectionCommit(file, "https://example.test/pr/1#c2", missing, s)
		subject, _, _ := strings.Cut(msg, "\n\n")
		if n := utf8.RuneCountInString(subject); n > 50 {
			t.Errorf("missing %v: subject %q is %d characters, over 50", missing, subject, n)
		}
		if f, err := commitmsg.Check(msg, s); err != nil || len(f) > 0 {
			t.Errorf("missing %v: the committer refuses %q: %v %v", missing, msg, f, err)
		}
	}
}

// Given a project whose allowed commit types exclude test, when the
// tester commits an objection rewrite, then the subject uses an allowed
// type.
func TestObjectionCommitSubjectUsesAnAllowedType(t *testing.T) {
	types := []string{"feat", "fix", "chore"}
	s := commitmsg.Settings{SubjectMax: 72, Types: types}
	for _, missing := range []bool{false, true} {
		msg := objectionCommit("tests/test_sub.sh", "https://example.test/pr/1#c2", missing, s)
		subject, _, _ := strings.Cut(msg, "\n\n")
		typ, _, _ := strings.Cut(subject, ": ")
		typ, _, _ = strings.Cut(typ, "(")
		if !slices.Contains(types, typ) {
			t.Errorf("missing %v: subject %q has type %q, want one of %v", missing, subject, typ, types)
		}
		if f, err := commitmsg.Check(msg, s); err != nil || len(f) > 0 {
			t.Errorf("missing %v: the committer refuses %q: %v %v", missing, msg, f, err)
		}
	}
}
