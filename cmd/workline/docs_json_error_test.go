package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docsJSONRepo is a committed repository with one doc.
func docsJSONRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "a.md"), []byte("# A\n\nSome words.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "first")
	return repo
}

// Docs says its error
// Given `workline docs --json` that fails,
// When it exits,
// Then it prints JSON whose rule names the real cause, on every early exit.
func TestDocsJSONSaysItsError(t *testing.T) {
	bin := sinceBuild(t)
	sinceEnv(t)

	broken := sweepBrokenRepo(t)

	notRepo := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notRepo))

	unknownSince := docsJSONRepo(t)

	reviewing := docsJSONRepo(t)
	if err := os.WriteFile(filepath.Join(reviewing, "docs", "a.md"), []byte("# A\n\nSome other words.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, dir string
		args      []string
		rule      string // "" : any rule but config-invalid
		message   string
	}{
		{"a config that is not valid YAML", broken, []string{"--ai", "none"}, "config-invalid", "did not find expected"},
		{"not in a repository", notRepo, []string{"--ai", "none"}, "", "git repository"},
		{"a revision that names no commit", unknownSince, []string{"--since", "v9.9", "--ai", "none"}, "", "v9.9"},
		{"reviewing with no terminal", reviewing, []string{"--review"}, "", "terminal"},
	} {
		args := append([]string{"docs", "--repo", tc.dir, "--json"}, tc.args...)
		stdout, stderr, code := sweepRun(t, bin, tc.dir, args...)
		if code == 0 {
			t.Errorf("%s: exit code = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", tc.name, stdout, stderr)
		}
		var res struct {
			Status   string `json:"status"`
			Findings []struct {
				Rule    string `json:"rule"`
				Message string `json:"message"`
			} `json:"findings"`
		}
		if err := json.Unmarshal([]byte(stdout), &res); err != nil {
			t.Errorf("%s: stdout is not the JSON result: %v\nstdout:\n%s\nstderr:\n%s", tc.name, err, stdout, stderr)
			continue
		}
		if res.Status != "block" {
			t.Errorf("%s: status = %q, want block\n%s", tc.name, res.Status, stdout)
		}
		named := false
		for _, f := range res.Findings {
			if !strings.Contains(f.Message, tc.message) {
				continue
			}
			switch {
			case tc.rule != "" && f.Rule == tc.rule:
				named = true
			case tc.rule == "" && f.Rule != "" && f.Rule != "config-invalid":
				named = true
			}
		}
		if !named {
			t.Errorf("%s: no finding with a rule naming the cause (%q, not config-invalid unless the config) and a message holding %q:\n%s",
				tc.name, tc.rule, tc.message, stdout)
		}
	}
}

// Docs flag documented
// Given the commands' docs,
// When a person looks up `docs`,
// Then `--json` is there.
func TestDocsFlagDocumented(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "commands.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(data), "\n### workline docs\n")
	if !ok {
		t.Fatal("docs/commands.md has no section `workline docs`")
	}
	section, _, _ := strings.Cut(rest, "\n### ")
	if !strings.Contains(section, "--json") {
		t.Errorf("the section `workline docs` of docs/commands.md does not say `--json`:\n%s", section)
	}
}
