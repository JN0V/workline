package developer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
		"src/tests/test_lib.sh":   true,
		"tests/test_lib.sh":       false,
		"other/tests/test_lib.sh": false,
		"other/spec/lib_spec.rb":  false,
	} {
		if got := besideTest(p, scope, checker{}.isDir); got != want {
			t.Errorf("besideTest(%s) = %v, want %v", p, got, want)
		}
	}
	// A folder's Scope lets in the tests under it, none at the root.
	for p, want := range map[string]bool{"src/x_test.go": true, "src/tests/test_x.sh": true, "x_test.go": false, "tests/test_x.sh": false} {
		if got := besideTest(p, []string{"src"}, checker{}.isDir); got != want {
			t.Errorf("besideTest(%s) with a Scope src = %v, want %v", p, got, want)
		}
	}
}

// A Scope or a source naming the repository's root bounds nothing: it is
// dropped, and never puts every file in bounds.
func TestScopeNamingTheRootIsDropped(t *testing.T) {
	got := scopePaths("## Scope\n\n- `.` and `./`, `src/a.go`.\n")
	if !slices.Equal(got, []string{"src/a.go"}) {
		t.Errorf("scopePaths = %v, want [src/a.go]", got)
	}
	c := checker{b: Bounds{Scope: []string{"."}}, steer: func(string) bool { return false }}
	if got := c.rule("src/other.go"); got != "outside-scope" {
		t.Errorf("rule with a Scope of . = %q, want outside-scope", got)
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
