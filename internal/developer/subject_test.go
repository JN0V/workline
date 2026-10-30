package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/role"
)

// roleCommitter is the committer's settings as its role ships them.
func roleCommitter(t *testing.T) committer.Settings {
	t.Helper()
	r, err := role.Load("../../roles", "committer")
	if err != nil {
		t.Fatal(err)
	}
	s, err := committer.SettingsFrom(r.MergedSettings(&role.ProjectConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The developer's subjects pass the committer's own check: the same
// function, so they cannot drift (#736).
func TestSubjectsPassTheCommitterCheck(t *testing.T) {
	s := roleCommitter(t)
	for _, tc := range []struct {
		name, title, kind string
		scope             []string
		tests, code       string
	}{
		{"a bug in a folder", "Login fails on a long name", KindFix, []string{"internal/auth/login.go", "internal/auth/"},
			"test(auth): login fails on a long name", "fix(auth): login fails on a long name"},
		{"a feature at the root", "Multiply two numbers", KindFeature, []string{"lib.sh"},
			"test: multiply two numbers", "feat: multiply two numbers"},
		{"a task, folders apart", "Bump the linter", KindTask, []string{"cmd/x", "internal/y"},
			"test: bump the linter", "chore: bump the linter"},
		{"no kind", "Add a flag", "", nil, "test: add a flag", "feat: add a flag"},
		{"codes and references out", "AC-3: Multiply two numbers (F207) for #12.", KindFix, []string{"lib.sh"},
			"test: multiply two numbers for", "fix: multiply two numbers for"},
		{"an acronym kept", "API returns a 500 on an empty body", KindFix, nil,
			"test: API returns a 500 on an empty body", "fix: API returns a 500 on an empty body"},
		{"a long title, cut on a word", "Developer: its commits pass the project's commit check, the tests commit and the code commit alike", KindFix,
			[]string{"internal/developer/"},
			"test(developer): developer: its commits pass the project's commit check", "fix(developer): developer: its commits pass the project's commit check"},
		{"no title", "", KindFix, nil, "test: the code of #4", "fix: the code of #4"},
	} {
		got := subjects(4, tc.title, tc.kind, tc.scope, s)
		if got.Tests != tc.tests || got.Code != tc.code {
			t.Errorf("%s: subjects = %q, %q; want %q, %q", tc.name, got.Tests, got.Code, tc.tests, tc.code)
		}
		for _, m := range []string{got.Tests, got.Code} {
			if f, err := committer.Check(m+"\n\nWorkline-Role: developer", s); err != nil || len(f) > 0 {
				t.Errorf("%s: %q refused by the committer: %v %v", tc.name, m, f, err)
			}
		}
	}
	if m := reviewSubject(3, 2, s); m != "fix: the review's findings on #3, round 2" {
		t.Errorf("review subject = %q", m)
	} else if f, err := committer.Check(m+"\n\nWorkline-Role: developer", s); err != nil || len(f) > 0 {
		t.Errorf("review subject %q refused: %v %v", m, f, err)
	}
}

// A project's own limit holds, its scope dropped when too long to leave
// room.
func TestSubjectFitsTheProjectsLimit(t *testing.T) {
	s := roleCommitter(t)
	s.SubjectMax = 30
	title := strings.Repeat("word ", 20)
	for _, scope := range []string{"", "a-very-long-folder-name"} {
		got := subject("fix", scope, summaryOf(title, s), s)
		if len([]rune(got)) > 30 {
			t.Errorf("scope %q: %q is over 30", scope, got)
		}
		if f, _ := committer.Check(got, s); len(f) > 0 {
			t.Errorf("scope %q: %q refused: %v", scope, got, f)
		}
	}
}
