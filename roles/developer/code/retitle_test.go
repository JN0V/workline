package developer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
	"github.com/JN0V/workline/internal/kit/verdict"
	"go.yaml.in/yaml/v3"
)

// greenTests makes the project's tests green on whatever the proof runs,
// through a fixture tester joined to a registry of its own, and isolates
// git from the machine's settings.
func greenTests(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("WORKLINE_FORGE", "")
	t.Setenv("WORKLINE_ROLES_DIR", "../..")
	r := contract.NewRegistry()
	r.Join("tester", contract.Role{Tests: &contract.Tests{
		CommandRuns: func(string, string, map[string]any) (string, *verdict.Finding, error) {
			return "fake", nil, nil
		},
		Settings: func(map[string]any) (contract.TestSettings, error) {
			return contract.TestSettings{Command: "fake"}, nil
		},
		OnChange: func(_, rev string, _ []intent.Intention, _ contract.TestSettings, _ time.Time, _ contract.TestScope) (contract.TestOutcome, *verdict.Finding) {
			return contract.TestOutcome{Passed: []string{"T.ok"}, At: rev, Runs: 1}, nil
		},
	}})
	t.Cleanup(contract.UseForTest(r))
}

// gitIn runs git in repo with its own name and email, and returns what it printed.
func gitIn(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo, "-c", "core.hooksPath=/dev/null", "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// titleOf is the title out/merge-request.yaml gives its pull request.
func titleOf(t *testing.T, runDir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var mr struct {
		Title string `yaml:"title"`
	}
	if err := yaml.Unmarshal(data, &mr); err != nil {
		t.Fatal(err)
	}
	return mr.Title
}

// Round adds shipped code
// Given a task's pull request opened as `chore` (its record lists a doc
// alone),
// When a review round changes a shipped file,
// Then its title becomes `fix`: the round's merge-request answer retitles it.
func TestRoundRetypesATaskThatNowShipsCode(t *testing.T) {
	greenTests(t)
	repo := gitRepo(t, map[string]string{"src/lib.go": "package lib\n", "docs/notes.md": "notes\n"})
	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON := func(name string, v any) {
		t.Helper()
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON(roundFile, round{MR: 5, Branch: "workline/developer/3", At: "HEAD",
		Record: devmark.Act{Act: "code", Issue: 3, Code: []string{"docs/notes.md"}},
		Asked:  []devmark.Asked{{Key: "correctness|src/lib.go|wrong", Lens: "correctness", Where: "src/lib.go:1", Holds: "blocks", Text: "wrong"}}})
	writeJSON(boundsFile, Bounds{Issue: 3, Kind: devmark.KindTask, Scope: []string{"src/lib.go"}})
	diff := "diff --git a/src/lib.go b/src/lib.go\n--- a/src/lib.go\n+++ b/src/lib.go\n@@ -1 +1,2 @@\n package lib\n+var X = 1\n"
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), []intent.Intention{{Kind: "patch", Value: diff}}); err != nil {
		t.Fatal(err)
	}

	if code := reviewPost(runDir, repo, Settings{Rounds: 3}); code != 0 {
		t.Fatalf("reviewPost exited %d", code)
	}

	if title := titleOf(t, runDir); !strings.HasPrefix(title, "fix") {
		t.Errorf("title after a round that changes a shipped file: %q, want a fix", title)
	}
}

// Opened from a person's Draft
// Given a task whose tester's Draft a person made green (a shipped file
// changed on it), its agent writing nothing,
// When the developer opens its pull request,
// Then the type comes from the Draft's changed files: `fix`, not `chore`.
func TestPullRequestFromAGreenDraftIsTypedByItsFiles(t *testing.T) {
	greenTests(t)
	repo := gitRepo(t, map[string]string{"src/lib.go": "package lib\n"})
	base := gitIn(t, repo, "rev-parse", "HEAD")
	gitIn(t, repo, "checkout", "-q", "-b", "workline/tester/issue-3")
	if err := os.WriteFile(filepath.Join(repo, "src", "lib_test.go"), []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "src", "lib.go"), []byte("package lib\n\nvar X = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "the Draft, made green by a person")
	draft := gitIn(t, repo, "rev-parse", "HEAD")
	gitIn(t, repo, "checkout", "-q", "main")

	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON := func(name string, v any) {
		t.Helper()
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON("settings.json", map[string]any{})
	writeJSON(pickFile, devmark.Act{Act: "code", Issue: 3, At: base, From: draft, Acceptance: []string{"src/lib_test.go"}})
	writeJSON(boundsFile, Bounds{Issue: 3, Kind: devmark.KindTask, Scope: []string{"src/lib.go"}})

	if code := Post(runDir, repo); code != 0 {
		t.Fatalf("Post exited %d", code)
	}

	if title := titleOf(t, runDir); !strings.HasPrefix(title, "fix") {
		t.Errorf("title of the pull request opened from the green Draft: %q, want a fix: the Draft changes a shipped file", title)
	}
}
