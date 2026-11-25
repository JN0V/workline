package developer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/kit/commitmsg"
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/role"
)

// roleCommitter is the committer's settings as its role ships them.
func roleCommitter(t *testing.T) commitmsg.Settings {
	t.Helper()
	r, err := role.Load("../..", "committer")
	if err != nil {
		t.Fatal(err)
	}
	s, err := commitmsg.SettingsFrom(r.MergedSettings(&role.ProjectConfig{}))
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
		{"a bug in a folder", "Login fails on a long name", devmark.KindFix, []string{"internal/auth/login.go", "internal/auth/"},
			"test(auth): login fails on a long name", "fix(auth): login fails on a long name"},
		{"a feature at the root", "Multiply two numbers", devmark.KindFeature, []string{"lib.sh"},
			"test: multiply two numbers", "feat: multiply two numbers"},
		{"a task, folders apart", "Bump the linter", devmark.KindTask, []string{"cmd/x", "internal/y"},
			"test: bump the linter", "chore: bump the linter"},
		{"no kind", "Add a flag", "", nil, "test: add a flag", "feat: add a flag"},
		{"codes and references out", "AC-3: Multiply two numbers (F207) for #12.", devmark.KindFix, []string{"lib.sh"},
			"test: multiply two numbers for", "fix: multiply two numbers for"},
		{"its agent's own", "", devmark.KindFix, nil, "test: multiply two whole numbers", "fix: multiply two whole numbers"},
		{"an acronym kept", "API returns a 500 on an empty body", devmark.KindFix, nil,
			"test: API returns a 500 on an empty body", "fix: API returns a 500 on an empty body"},
		{"a long title, never cut", "Developer: its commits pass the project's commit check, the tests commit and the code commit alike", devmark.KindFix,
			[]string{"internal/developer/"}, "test(developer): the code of #4", "fix(developer): the code of #4"},
		{"no title", "", devmark.KindFix, nil, "test: the code of #4", "fix: the code of #4"},
	} {
		own := ""
		if tc.name == "its agent's own" {
			own = "multiply two whole numbers\n"
		}
		got := subjects(4, own, tc.title, tc.kind, tc.scope, nil, s)
		if got.Tests != tc.tests || got.Code != tc.code {
			t.Errorf("%s: subjects = %q, %q; want %q, %q", tc.name, got.Tests, got.Code, tc.tests, tc.code)
		}
		for _, m := range []string{got.Tests, got.Code} {
			if f, err := commitmsg.Check(m+"\n\nWorkline-Role: developer", s); err != nil || len(f) > 0 {
				t.Errorf("%s: %q refused by the committer: %v %v", tc.name, m, f, err)
			}
		}
		// Its pull request's title, squashed: the forge's " (#N)" after
		// it, a number of five digits, as the committer's squash rule
		// reads it; GitLab's "Draft: " on top for a Draft.
		for _, title := range []string{got.Code, got.DraftTitle} {
			if !strings.HasPrefix(got.Code, strings.TrimRight(title, " ")) || !strings.HasPrefix(title, strings.SplitN(got.Code, ": ", 2)[0]+": ") {
				t.Errorf("%s: title %q is not the code's subject %q", tc.name, title, got.Code)
			}
			if f, err := commitmsg.Check(title+" (#12345)", s); err != nil || len(f) > 0 {
				t.Errorf("%s: the squash of %q refused: %v %v", tc.name, title, f, err)
			}
		}
		if n := len([]rune("Draft: " + got.DraftTitle)); n > forge.TitleMax {
			t.Errorf("%s: Draft title %q is %d characters, over %d", tc.name, got.DraftTitle, n, forge.TitleMax)
		}
	}
	if m := reviewSubject(3, 2, s); m != "fix: the review's findings on #3, round 2" {
		t.Errorf("review subject = %q", m)
	} else if f, err := commitmsg.Check(m+"\n\nWorkline-Role: developer", s); err != nil || len(f) > 0 {
		t.Errorf("review subject %q refused: %v %v", m, f, err)
	}
}

// A project's own limit holds, its scope dropped when too long to leave
// room; a summary past it never cut.
func TestSubjectFitsTheProjectsLimit(t *testing.T) {
	s := roleCommitter(t)
	s.SubjectMax = 39 // 30 where a squash lands it
	title := strings.Repeat("word ", 20)
	for _, scope := range [][]string{nil, {"a-very-long-folder-name/x.go"}} {
		got := subjects(4, "", title, devmark.KindFix, scope, nil, s).Code
		if got != "fix: the code of #4" {
			t.Errorf("scope %q: %q, want the code of #4, whole", scope, got)
		}
		if len([]rune(got)) > 30 {
			t.Errorf("scope %q: %q is over 30", scope, got)
		}
		if f, _ := commitmsg.Check(got, s); len(f) > 0 {
			t.Errorf("scope %q: %q refused: %v", scope, got, f)
		}
	}
}

// A project config that cannot be read is said, never passed over in
// silence: the role's own rules then, its limit kept.
func TestCommitterSettingsUnreadSaid(t *testing.T) {
	t.Setenv("WORKLINE_ROLES_DIR", "../..")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("roles: [not, a, map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, found := committerSettings(repo, "#4")
	if len(found) != 1 || found[0].Rule != "commit-rules-unread" {
		t.Fatalf("findings = %v, want commit-rules-unread", found)
	}
	if s.SubjectMax != 72 || len(s.Types) == 0 {
		t.Errorf("fallback = %d, %v; want the role's own", s.SubjectMax, s.Types)
	}
	if _, found := committerSettings(t.TempDir(), "#4"); len(found) != 0 {
		t.Errorf("no config: findings = %v, want none", found)
	}
}

// A task is typed by what its code changes: shipped code a fix, so
// that release tools list it and bump its package; docs, CI, a tool's
// dotfile, a chore (#2198).
func TestTaskTypedByItsCode(t *testing.T) {
	for _, tc := range []struct {
		code []string
		want string
	}{
		{[]string{"lib/DomoticsCore-WebUI/include/WebUI.h"}, "fix"},
		{[]string{"docs/mqtt.md", "src/mqtt.cpp"}, "fix"},
		{[]string{"docs/mqtt.md", "README.md", "CHANGELOG.md"}, "chore"},
		{[]string{".github/workflows/ci.yml", ".golangci.yml", "LICENSE"}, "chore"},
		{[]string{"CMakeLists.txt"}, "fix"},
		{[]string{"requirements.txt"}, "fix"},
		{[]string{"internal/doc/render.go"}, "fix"},
		{nil, "chore"},
	} {
		if got := commitType(devmark.KindTask, tc.code); got != tc.want {
			t.Errorf("task on %v: %q, want %q", tc.code, got, tc.want)
		}
	}
	if got := commitType(devmark.KindFeature, []string{"README.md"}); got != "feat" {
		t.Errorf("feature: %q, want feat", got)
	}
	s := roleCommitter(t)
	if sub := subjects(4, "", "Cap the MQTT payload", devmark.KindTask, nil, []string{"src/mqtt.cpp"}, s); sub.Code != "fix: cap the MQTT payload" || sub.Note == "" {
		t.Errorf("task on shipped code: %+v", sub)
	}
	if sub := subjects(4, "", "Cap the MQTT payload", devmark.KindFix, nil, []string{"src/mqtt.cpp"}, s); sub.Note != "" {
		t.Errorf("a bug's fix says nothing of its type: %q", sub.Note)
	}
}
