package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/testdef"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
)

// A rename names no file on its +++ line alone: both sides are held to
// the bounds, so a frozen test cannot leave by a rename.
func TestTouchedHoldsBothSidesOfARename(t *testing.T) {
	diff := "diff --git a/tests/test_mul.sh b/tests/old_mul.sh\nsimilarity index 100%\nrename from tests/test_mul.sh\nrename to tests/old_mul.sh\n"
	got, read := touched(diff)
	for _, want := range []string{"tests/test_mul.sh", "tests/old_mul.sh"} {
		if !read || !slices.Contains(got, want) {
			t.Errorf("touched = %v, %v; want %s in it", got, read, want)
		}
	}
}

// A name holding a space, " b/", or one git quotes is read whole from
// its diff's header; a header whose two names cannot be told apart, and
// no other line naming them, leaves the patch unread.
func TestTouchedReadsNamesWithSpaces(t *testing.T) {
	for _, tc := range []struct {
		name, diff string
		want       []string
		read       bool
	}{
		{"a space", "diff --git a/my file.sh b/my file.sh\nold mode 100644\nnew mode 100755\n", []string{"my file.sh"}, true},
		{"a b/ in the name", "diff --git a/x b/y b/x b/y\nold mode 100644\nnew mode 100755\n", []string{"x b/y"}, true},
		{"quoted", "diff --git \"a/t\\303\\251st.sh\" \"b/t\\303\\251st.sh\"\n--- \"a/t\\303\\251st.sh\"\n+++ \"b/t\\303\\251st.sh\"\n", []string{"tést.sh"}, true},
		{"a rename with spaces", "diff --git a/tests/frozen test.sh b/src/x b/y.sh\nsimilarity index 100%\nrename from tests/frozen test.sh\nrename to src/x b/y.sh\n",
			[]string{"src/x b/y.sh", "tests/frozen test.sh"}, true},
		{"edited, with spaces", "diff --git a/a b/c b/d b/e\n--- a/a b/c\n+++ b/d b/e\n@@ -1 +1 @@\n-x\n+y\n", []string{"a b/c", "d b/e"}, true},
		{"content lines like headers", "diff --git a/src/a.sh b/src/a.sh\n--- a/src/a.sh\n+++ b/src/a.sh\n@@ -1,2 +1,2 @@\n--- a/AGENTS.md\n+++ b/AGENTS.md\n", []string{"src/a.sh"}, true},
		{"cannot be told apart", "diff --git a/a b/c b/d b/e\nold mode 100644\nnew mode 100755\n", nil, false},
	} {
		got, read := touched(tc.diff)
		if read != tc.read {
			t.Errorf("%s: read = %v, want %v", tc.name, read, tc.read)
		}
		if tc.read && len(got) != len(tc.want) {
			t.Errorf("%s: touched = %q, want only %q", tc.name, got, tc.want)
		}
		for _, w := range tc.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: touched = %q, want %q in it", tc.name, got, w)
			}
		}
	}
}

// A patch the bounds cannot hold, naming no file, is refused, never laid;
// with the bounds unread, it is refused as every other.
func TestPatchNamingNoFileIsRefused(t *testing.T) {
	repo := t.TempDir()
	nameless := intent.Intention{Kind: "patch", Value: "@@ -1 +1 @@\n-a\n+b\n"}
	kept, refused, _ := bounded(repo, Bounds{Issue: 3, Scope: []string{"src"}}, nil, []intent.Intention{nameless})
	if len(kept) != 0 || len(refused) != 1 || refused[0].Rule != "bounds-unread" {
		t.Errorf("bounds unread: kept = %v, refused = %+v; want nothing kept, one bounds-unread", kept, refused)
	}
	repo = gitRepo(t, map[string]string{"src/a.sh": "a\n"})
	kept, refused, _ = bounded(repo, Bounds{Issue: 3, Scope: []string{"src"}}, nil, []intent.Intention{nameless})
	if len(kept) != 0 || len(refused) != 1 || refused[0].Rule != "patch-unread" {
		t.Errorf("bounds read: kept = %v, refused = %+v; want nothing kept, one patch-unread", kept, refused)
	}
}

func TestRuleOrder(t *testing.T) {
	c := checker{b: Bounds{Issue: 3, Scope: []string{"src/lib.go", "AGENTS.md"}, Frozen: map[string]int{"src/lib_test.go": 3}},
		steer: func(p string) bool { return p == "AGENTS.md" }, roles: ".workline/roles"}
	for p, want := range map[string]string{
		"src/lib.go":                "",
		"src/lib_more_test.go":      "",
		"tests/test_lib.sh":         "", // beside AGENTS.md, at the root
		"src/lib_test.go":           "acceptance-test-edit",
		"AGENTS.md":                 "steering-file",
		"roles/developer/policy.md": "steering-file",
		"src/roles/x/persona.md":    "steering-file",
		".workline/config.yaml":     "gate-file",
		".github/workflows/ci.yml":  "gate-file",
		"README.md":                 "outside-scope",
		"../elsewhere.go":           "outside-scope",
		"../tests/test_x.sh":        "outside-scope",
		"src/other.go":              "outside-scope",
	} {
		if got := c.rule(p); got != want {
			t.Errorf("rule(%s) = %q, want %q", p, got, want)
		}
	}
}

// A test folder is beside the Scope only where it sits: in a Scope path's
// folder or under it, not anywhere in the repository.
func TestBesideTestNeedsTheTestFolderBesideTheScope(t *testing.T) {
	scope := []string{"src/lib.go"}
	for p, want := range map[string]bool{
		"src/lib_test.go":         true,
		"src/tests/test_lib.sh":   false, // not in its folder: the fence's when new alone
		"tests/test_lib.sh":       false,
		"other/tests/test_lib.sh": false,
		"other/spec/lib_spec.rb":  false,
		"src/.env.test":           false, // a setting (#745)
		"src/.env.test.local":     false,
		"src/latest.go":           false, // "test" in a word, not a word
		"src/inspect.go":          false,
	} {
		if got := besideTest(p, scope, checker{}.isDir, testdef.Def{}); got != want {
			t.Errorf("besideTest(%s) = %v, want %v", p, got, want)
		}
	}
	// A folder's Scope lets in the tests under it, none at the root.
	for p, want := range map[string]bool{"src/x_test.go": true, "src/tests/test_x.sh": true, "x_test.go": false, "tests/test_x.sh": false} {
		if got := besideTest(p, []string{"src"}, checker{}.isDir, testdef.Def{}); got != want {
			t.Errorf("besideTest(%s) with a Scope src = %v, want %v", p, got, want)
		}
	}
}

// The write fence (ADR-0063, §9): a new file matching the tester's
// test-paths, in or beside a Scope path's folder, is written by rule; the
// trial's #35 and #20 wrote theirs under the component's test/ folder, its
// Scope a header under include/. The same file outside any Scope folder,
// one the commit holds already, or one no test-paths names is refused,
// an ask.
func TestNewTestBesideTheScopeIsKept(t *testing.T) {
	repo := t.TempDir()
	for _, f := range []string{"Storage/include/Storage.h", "Storage/test/test_old/test_old.cpp", "Wifi/include/Wifi.h"} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := checker{repo: repo, steer: func(string) bool { return false },
		b: Bounds{Issue: 20, Scope: []string{"Storage/include/Storage.h"}, TestPaths: []string{"*/test/**", "**/*_test.go"}}}
	for p, want := range map[string]string{
		"Storage/test/test_name_more/test_name_more.cpp": "",              // new, beside: kept
		"Storage/test/test_old/test_old.cpp":             "outside-scope", // the commit holds it: not new
		"Wifi/test/test_wifi/test_wifi.cpp":              "outside-scope", // beside no Scope path
		"Storage/src/helper.cpp":                         "outside-scope", // no test-paths names it
		"Storage/include/Storage.h":                      "",
	} {
		if got := c.rule(p); got != want {
			t.Errorf("rule(%s) = %q, want %q", p, got, want)
		}
	}
	// No test-paths set: the generic ones (ADR-0065); a test they miss
	// is code, outside the Scope: an ask.
	c.b.TestPaths = nil
	if got := c.rule("Storage/test/test_name_more/test_name_more.cpp"); got != "" {
		t.Errorf("with no test-paths, rule = %q, want the generic ones to keep it", got)
	}
	if got := c.rule("Storage/check/check_name.cpp"); got != "outside-scope" {
		t.Errorf("a test the generic patterns miss: rule = %q, want outside-scope", got)
	}
	// besideKept lists only what the fence alone let in, the rules read
	// in a repository.
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "-qm", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	c.b.TestPaths = []string{"*/test/**"}
	patches := []intent.Intention{
		{Kind: "patch", Value: map[string]any{"file": "Storage/include/Storage.h", "content": "y\n"}},
		{Kind: "patch", Value: map[string]any{"file": "Storage/test/test_name_more/test_name_more.cpp", "content": "z\n"}},
	}
	if got := besideKept(repo, c.b, patches); !slices.Equal(got, []string{"Storage/test/test_name_more/test_name_more.cpp"}) {
		t.Errorf("besideKept = %v, want the new test alone", got)
	}
}

// A test file's home is the folder holding its test folder, else its
// own; the root is a home only to a Scope file at the root.
func TestTestBesideByItsHome(t *testing.T) {
	isDir := checker{}.isDir
	for _, tc := range []struct {
		p     string
		scope []string
		want  bool
	}{
		{"Storage/test/a/test_a.cpp", []string{"Storage/include/Storage.h"}, true},
		{"pkg/foo/new_test.go", []string{"pkg/foo/foo.go"}, true},
		{"pkg/foo/sub/new_test.go", []string{"pkg/foo"}, true},
		{"tests/test_x.py", []string{"lib.py"}, true},
		{"tests/test_x.py", []string{"src/pkg/x.py"}, false},
		{"other/test/test_x.cpp", []string{"Storage/include/Storage.h"}, false},
		{"Storage/test/a/test_a.cpp", []string{"Storage/*.h"}, false}, // a pattern names its files, no folder
	} {
		if got := testBeside(testdef.Def{}.Home(tc.p), tc.scope, isDir); got != tc.want {
			t.Errorf("testBeside(%s, %v) = %v, want %v", tc.p, tc.scope, got, tc.want)
		}
	}
}

// The same set of files asked again waits for a person, whatever the
// head; one file not asked before is a new ask (ADR-0063, §9; #1073).
func TestAskedBeforeIsBySet(t *testing.T) {
	before := []string{"a.h", "b.h"}
	for _, tc := range []struct {
		files []string
		want  bool
	}{
		{[]string{"a.h"}, true},
		{[]string{"b.h", "a.h"}, true},
		{[]string{"a.h", "c.h"}, false},
		{nil, false},
	} {
		if got := askedBefore(tc.files, before); got != tc.want {
			t.Errorf("askedBefore(%v) = %v, want %v", tc.files, got, tc.want)
		}
	}
}

// A Scope naming its files by a pattern bounds what it matches, a new
// file too, as the tester reads it (#801): `src/calc/*.sh`.
func TestUnderAPatternIsWhatItMatches(t *testing.T) {
	for p, want := range map[string]bool{
		"src/calc/add.sh":     true,
		"src/calc/mul.sh":     true, // not there yet
		"src/calc/add.go":     false,
		"src/calc/sub/add.sh": false, // `*` stays in its folder
	} {
		if got := under(p, []string{"src/calc/*.sh"}); got != want {
			t.Errorf("under(%s, src/calc/*.sh) = %v, want %v", p, got, want)
		}
	}
	if !under("src/a/b/c.sh", []string{"src/**/*.sh"}) {
		t.Error("under(src/a/b/c.sh, src/**/*.sh) = false, want true")
	}
}

// A Scope or a source naming the repository's root bounds nothing: it is
// dropped, and never puts every file in bounds.
func TestScopeNamingTheRootIsDropped(t *testing.T) {
	got := scopePaths("## Scope\n\n- `.` and `./`, `src/a.go`.\n", nil)
	if !slices.Equal(got, []string{"src/a.go"}) {
		t.Errorf("scopePaths = %v, want [src/a.go]", got)
	}
	c := checker{b: Bounds{Scope: []string{"."}}, steer: func(string) bool { return false }}
	if got := c.rule("src/other.go"); got != "outside-scope" {
		t.Errorf("rule with a Scope of . = %q, want outside-scope", got)
	}
}

// A Scope naming a file with its line bounds the file: `src/a.go:12`
// lets the developer change src/a.go.
func TestScopeLineIsTheFile(t *testing.T) {
	got := scopePaths("## Scope\n\n- `run` (`src/a.go:12`), `src/b.go:3-9`.\n", nil)
	if !slices.Equal(got, []string{"src/a.go", "src/b.go"}) {
		t.Errorf("scopePaths = %v, want [src/a.go src/b.go]", got)
	}
}

// A Scope's paths in plain words are in bounds when the repository holds
// them; a word in backticks the repository has no folder for is not a
// path (#901, on #79's and #244's Scope).
func TestScopePlainPathsAreBounds(t *testing.T) {
	files := []string{"docs/ci.md", "ci/github/workflow.yml", "ci/gitlab/pipeline.yml", ".github/workflows/ci.yml",
		"internal/doctor/doctor.go", "internal/gate/gate.go", "internal/tools/tools.go", "docs/spec/gates.md", "docs/install.md", "src/a.go"}
	for _, c := range []struct {
		name, scope string
		want        []string
	}{
		{"#79", "docs/ci.md, ci/github, ci/gitlab, .github/workflows/ci.yml. The way, in the ecosystem's words: a first job tells what changed (`dorny/paths-filter`, GitLab's `rules: changes`), the heavy jobs skipped with `if:`.",
			[]string{"docs/ci.md", "ci/github", "ci/gitlab", ".github/workflows/ci.yml"}},
		{"#244", "internal/doctor, `workline init`, internal/gate (a \"linted\" rule), a table language → linter → pinned install (internal/tools), docs/spec/gates.md, docs/install.md. Related: #202.",
			[]string{"internal/doctor", "internal/gate", "internal/tools", "docs/spec/gates.md", "docs/install.md"}},
		{"new file in a known folder", "- `src/b.go`, `src/new/c.go`, and/or src/a.go:12.", []string{"src/b.go", "src/new/c.go", "src/a.go"}},
		{"new folder", "- `pkg/__init__.py`, `lib/`, `actions/checkout`.", []string{"pkg/__init__.py", "lib"}},
		{"an Action's version", "- `actions/checkout@v4.1`, `golangci/lint-action@v6.1.0`, `pkg/x.go`.", []string{"pkg/x.go"}},
	} {
		got := scopePaths("## Need\n\nA need.\n\n## Scope\n\n"+c.scope+"\n", files)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: scopePaths = %q, want %q", c.name, got, c.want)
		}
	}
	if got := scopePaths("## Scope\n\n`dorny/paths-filter`, docs/ci.md\n", nil); !slices.Equal(got, []string{"dorny/paths-filter"}) {
		t.Errorf("files unread: scopePaths = %q, want the words in backticks as written", got)
	}
	ch := checker{b: Bounds{Scope: scopePaths("## Scope\n\ninternal/doctor\n", files)}, steer: func(string) bool { return false }}
	if r := ch.rule("internal/doctor/new.go"); r != "" {
		t.Errorf("rule(internal/doctor/new.go) = %q, want in bounds", r)
	}
}

// The pick passes over an issue whose plain Scope overlaps one taken.
func TestPickReadsPlainScope(t *testing.T) {
	s := SettingsFrom(nil)
	issues := []forge.Issue{
		{ID: 1, Labels: []string{"workline:ready", s.ClaimLabel}, Body: "## Scope\n\ninternal/doctor, docs/install.md\n"},
		{ID: 2, Labels: []string{"workline:ready"}, Body: "## Scope\n\ninternal/doctor/doctor.go\n"},
	}
	// #1 holds its Scope by its open pull request; with none, its agent
	// wrote no code, and it holds nothing (#952).
	if id, _ := pick(issues, nil, s, 0, false, nil, nil, []string{"internal/doctor/doctor.go", "docs/install.md"}); id != 2 {
		t.Errorf("pick beside a claim with no code: %d, want #2", id)
	}
	coded := []forge.MergeRequest{{ID: 7, Title: "fix: doctor", Body: "Closes #1"}}
	id, passed := pick(issues, coded, s, 0, false, nil, nil, []string{"internal/doctor/doctor.go", "docs/install.md"})
	if id != 0 || len(passed) != 2 || !strings.Contains(passed[1].Why, "overlaps #1") {
		t.Errorf("pick: %d, %v", id, passed)
	}
	// A bare name is the one file of that name, as the bounds read it.
	issues[0].Body, issues[1].Body = "## Scope\n\n- `doctor.go`\n", "## Scope\n\n- `internal/doctor/doctor.go`\n"
	if id, passed := pick(issues, coded, s, 0, false, nil, nil, []string{"internal/doctor/doctor.go"}); id != 0 || len(passed) != 2 {
		t.Errorf("pick on a bare name: %d, %v", id, passed)
	}
}

// A repository with no file reads as files known, none held.
func TestRepoFilesOfAnEmptyRepository(t *testing.T) {
	repo := t.TempDir()
	if err := exec.Command("git", "-C", repo, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	if got := repoFiles(repo); got == nil || len(got) != 0 {
		t.Errorf("repoFiles = %q, want none, known", got)
	}
}

// A Scope naming a file by its bare name bounds the one file of that
// name; a folder, or a name two files have, stays as written.
func TestScopeBareNameIsTheFile(t *testing.T) {
	files := []string{"internal/b/parts.go", "x/a.go", "y/a.go", "src/c d.go"}
	got := inRepo([]string{"parts.go", "a.go", "internal/b", "c d.go"}, files)
	want := []string{"internal/b/parts.go", "a.go", "internal/b", "src/c d.go"}
	if !slices.Equal(got, want) {
		t.Errorf("inRepo = %q, want %q", got, want)
	}
	c := checker{b: Bounds{Scope: got}, steer: func(string) bool { return false }}
	if r := c.rule("internal/b/parts.go"); r != "" {
		t.Errorf("rule(internal/b/parts.go) = %q, want in bounds", r)
	}
}

// A gate's config is compared cleaned: ./x, a trailing slash or a folder
// still refuse the files they name.
func TestGateConfigPathsAreCleaned(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "gates:\n  lint:\n    checks:\n      - run: golangci-lint run\n        config: [./.golangci.yml, lint/rules/, \".\", ../up.yml]\n"
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	gates, err := gateFiles(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	c := checker{b: Bounds{Scope: []string{".golangci.yml", "lint", "src"}}, gates: gates, steer: func(string) bool { return false }}
	for p, want := range map[string]string{".golangci.yml": "gate-file", "lint/rules/x.yml": "gate-file", "src/a.go": ""} {
		if got := c.rule(p); got != want {
			t.Errorf("rule(%s) = %q, want %q", p, got, want)
		}
	}
}

// When the steering files or the gates cannot be read, every patch is
// refused and the refusal says why: the bounds fail closed.
func TestUnreadBoundsRefuseEveryPatch(t *testing.T) {
	repo := t.TempDir() // not a git repository: the steering files cannot be read
	patch := intent.Intention{Kind: "patch", Value: map[string]any{"file": "AGENTS.md", "content": "x"}}
	kept, refused, _ := bounded(repo, Bounds{Issue: 3, Scope: []string{"AGENTS.md"}}, nil, []intent.Intention{patch})
	if len(kept) != 0 {
		t.Errorf("kept = %v, want nothing", kept)
	}
	if len(refused) != 1 || refused[0].Rule != "bounds-unread" || !strings.Contains(why(refused[0]), "steer agents") {
		t.Errorf("refused = %+v, want one bounds-unread saying the steering files could not be read", refused)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("gates: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gateFiles(repo, ""); err == nil {
		t.Error("gateFiles of a config that does not parse: no error")
	}
}

// An objection needs the Draft's commit to sit on a line: without it, it
// is refused and said, never posted with no head.
func TestObjectionWithoutTheDraftsHeadIsRefused(t *testing.T) {
	b := Bounds{Issue: 3, Frozen: map[string]int{"tests/t.sh": 3}, Draft: 1}
	o := intent.Intention{Kind: "comment", Value: map[string]any{"objection": map[string]any{
		"test": "tests/t.sh", "line": 2, "scenario": "S", "kind": "wrong", "why": "W"}}}
	kept, _, said := bounded(t.TempDir(), b, nil, []intent.Intention{o})
	if len(kept) != 0 || len(said) != 1 || !strings.Contains(said[0], "could not be read") {
		t.Errorf("kept = %v, said = %v; want nothing kept, one refusal said", kept, said)
	}
}

// Second objection, same file (#899): Given two objections to one frozen
// test on two lines, When bounded keeps them, Then each has its own key,
// so one still open on its line does not hold back the other.
func TestObjectionKeyIsPerLine(t *testing.T) {
	b := Bounds{Issue: 3, Frozen: map[string]int{"tests/t.sh": 3}, Draft: 1, Head: "abc"}
	on := func(line int) intent.Intention {
		return intent.Intention{Kind: "comment", Value: map[string]any{"objection": map[string]any{
			"test": "tests/t.sh", "line": line, "scenario": "S", "kind": "wrong", "why": "W"}}}
	}
	kept, _, said := bounded(t.TempDir(), b, nil, []intent.Intention{on(2), on(5)})
	if len(kept) != 1 {
		t.Fatalf("kept = %v, said = %v; want one comment holding the review", kept, said)
	}
	review := kept[0].Value.(map[string]any)["review"].(map[string]any)
	comments := review["comments"].([]forge.LineComment)
	if len(comments) != 2 || comments[0].Key == comments[1].Key {
		t.Errorf("comments = %+v; want two, each its own key", comments)
	}
}

// Test name given (#913): Given an issue's scenarios, When an objection
// names the Go test for one, Then it maps to that scenario; a name that
// spells none, or two, maps to nothing and is not posted.
func TestScenarioOfMapsATestsName(t *testing.T) {
	vouched := []string{"Not vouched for", "Vouched for"}
	for _, c := range []struct {
		name string
		in   []string
		want string
	}{
		{"Not vouched for", vouched, "Not vouched for"},
		{"TestNotVouchedFor", vouched, "Not vouched for"},
		{"TestVouchedFor", vouched, "Vouched for"},
		{"TestNoBrewNoCargo", []string{"No brew, no cargo"}, "No brew, no cargo"},
		{"test_no_brew_no_cargo", []string{"No brew, no cargo"}, "No brew, no cargo"},
		{"TestLychee/No_brew,_no_cargo", []string{"No brew, no cargo"}, "No brew, no cargo"},
		{"TestSomethingElse", vouched, ""},
		{"TestByZero", []string{"By zero", "Two by three"}, "By zero"},
		{"TestMulByZero", []string{"By zero", "Two by three"}, ""},
		{"TestNonZero", []string{"Zero", "One"}, ""},
		{"Test", []string{"Test", "Other"}, "Test"},
		{"Anything", nil, "Anything"},
	} {
		if got := scenarioOf(c.name, c.in); got != c.want {
			t.Errorf("scenarioOf(%q, %q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// Scenario named (#913): Given an issue's scenarios, When the question
// shows the objection's form, Then it lists them whole, a comma in one
// kept; with none, it points at the issue's Validation.
func TestScenarioHintListsTheIssues(t *testing.T) {
	if got := scenarioHint([]string{"No brew, no cargo", "Other"}); got != "<one of: No brew, no cargo | Other>" {
		t.Errorf("scenarioHint = %q", got)
	}
	if got := scenarioHint(nil); !strings.Contains(got, "Validation") {
		t.Errorf("scenarioHint(nil) = %q, want it to name the Validation", got)
	}
}

// Test name given (#913): Given an objection whose scenario spells none
// of the issue's, When bounded holds it, Then it is said, not posted.
func TestObjectionWithAnUnknownScenarioIsSaid(t *testing.T) {
	b := Bounds{Issue: 3, Frozen: map[string]int{"tests/t.sh": 3}, Draft: 1, Head: "abc", Scenarios: []string{"Not vouched for", "Vouched for"}}
	o := intent.Intention{Kind: "comment", Value: map[string]any{"objection": map[string]any{
		"test": "tests/t.sh", "line": 2, "scenario": "TestSomethingElse", "kind": "interface", "why": "W"}}}
	kept, _, said := bounded(t.TempDir(), b, nil, []intent.Intention{o})
	if len(kept) != 0 || len(said) != 1 || !strings.Contains(said[0], `"TestSomethingElse" is none of #3's (Not vouched for, Vouched for)`) {
		t.Errorf("kept = %v, said = %v; want nothing posted, the scenarios said", kept, said)
	}
}

// The Scope's paths and globs are the bounds, nothing guessed from a
// name (ADR-0065, change 2; #1228): a class or a module word bounds no
// file; a path, a glob, a new file's path do.
func TestScopePathsOnlyNoNameGuessed(t *testing.T) {
	files := []string{
		"core/session/build.gradle.kts",
		"core/session/src/main/kotlin/s/SettingsRepository.kt",
		"feature/tracing/src/main/kotlin/t/TracingViewModel.kt",
	}
	for _, c := range []struct {
		name, scope string
		want        []string
	}{
		{"class", "- core:session `SettingsRepository`.", nil},
		{"module word", "- `core:session`.", nil},
		{"path", "- `feature/tracing/src/main/kotlin/t/TracingViewModel.kt`.", []string{"feature/tracing/src/main/kotlin/t/TracingViewModel.kt"}},
		{"glob", "- `core/session/**`.", []string{"core/session/**"}},
		{"narrow glob, never its folder", "- `core/session/*.kts`.", []string{"core/session/*.kts"}},
		{"new file", "- New: `core/overlay/src/main/kotlin/o/CompositionGrid.kt`.", []string{"core/overlay/src/main/kotlin/o/CompositionGrid.kt"}},
	} {
		body := "## Need\n\nA need.\n\n## Scope\n\n" + c.scope + "\n"
		if got := inRepo(scopePaths(body, files), files); !slices.Equal(got, c.want) {
			t.Errorf("%s: scope = %q, want %q", c.name, got, c.want)
		}
	}
}

// A new test file, a fake under a module's src/test beside a Scope file
// under its src/main, is the write fence's (#1207, ADR-0065): the
// project's test definition says it is a test, its test-homes where its
// code lives. An existing helper or fixture a frozen test may read, a
// test of another module, are not. The generic definition calls a
// production package named test a test; the project's test-paths, set,
// do not.
func TestNewTestFileByTheDefinition(t *testing.T) {
	repo := t.TempDir()
	for _, f := range []string{
		"mul.sh", "tests/lib.sh", "tests/fixtures/expected.txt", "tests/test_mul.sh",
		"app/src/main/kotlin/a/App.kt", "app/src/test/resources/golden.json",
		"app/src/test/kotlin/a/FakeStore.kt", "feature/x/src/main/kotlin/x/X.kt",
	} {
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, f), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c := checker{repo: repo, steer: func(string) bool { return false },
		b: Bounds{Issue: 5, Scope: []string{"mul.sh", "app/src/main/kotlin/a/App.kt", "feature"}}}
	gradle := c
	gradle.b.TestPaths = []string{"**/src/test/**", "tests/**"}
	gradle.b.TestHomes = []testdef.Home{{Tests: "**/src/test/kotlin/**", Code: "**/src/main/kotlin/**"}}
	for p, want := range map[string][2]string{ // generic, gradle
		"app/src/test/kotlin/a/FakeGrid.kt":    {"", ""},                           // new, beside
		"app/src/test/kotlin/a/FakeStore.kt":   {"outside-scope", "outside-scope"}, // exists: a frozen test may read it
		"app/src/test/resources/golden.json":   {"outside-scope", "outside-scope"}, // exists
		"tests/lib.sh":                         {"outside-scope", "outside-scope"}, // exists
		"tests/fixtures/expected.txt":          {"outside-scope", "outside-scope"},
		"tests/helper.sh":                      {"", "outside-scope"},              // new, beside a root Scope file; test-homes set, the generic ones not read
		"feature/x/src/test/kotlin/x/FakeX.kt": {"", ""},                           // under the Scope folder itself
		"other/src/test/kotlin/o/FakeO.kt":     {"outside-scope", "outside-scope"}, // another module
		"app/src/test/kotlin/b/FakeB.kt":       {"", "outside-scope"},              // another package: its home app/src/main/kotlin/b
		"app/src/main/kotlin/a/test/Other.kt":  {"", "outside-scope"},              // a production package named test
		"app/src/check/kotlin/a/CheckApp.kt":   {"outside-scope", "outside-scope"}, // a test no definition names: code, asked
	} {
		if got := c.rule(p); got != want[0] {
			t.Errorf("generic: rule(%s) = %q, want %q", p, got, want[0])
		}
		if got := gradle.rule(p); got != want[1] {
			t.Errorf("gradle: rule(%s) = %q, want %q", p, got, want[1])
		}
	}
	// Kept by an earlier round: still the fence's on the branch's head.
	c.b.Beside = []string{"app/src/test/kotlin/a/FakeStore.kt"}
	if got := c.rule("app/src/test/kotlin/a/FakeStore.kt"); got != "" {
		t.Errorf("a helper kept before: rule = %q, want in bounds", got)
	}
}
