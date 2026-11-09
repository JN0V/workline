package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func sinceEnv(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
}

func sinceBuild(t *testing.T) string {
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// sinceRepo makes a repository of two commits, with a tag v1 on the first,
// and a config giving the tester a test command.
func sinceRepo(t *testing.T) string {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	for p, c := range map[string]string{
		"docs/a.md":             "# A\n\nSome words.\n",
		"main.go":               "package main\n",
		".workline/config.yaml": "roles:\n  tester:\n    settings:\n      test-command: \"true {report}\"\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, p), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-m", "first")
	gitIn(t, repo, "tag", "v1")
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\n\nfunc x() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "-am", "second")
	return repo
}

func runIn(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

// phaseLine is the line CI's job log gets as a role's step starts, with the
// time it started (the engine's phase, #1197): two runs on different
// seconds differ by it alone (#1234).
var phaseLine = regexp.MustCompile(`(?m)^workline: \S+: \S+ \(\d{2}:\d{2}:\d{2} UTC\)\n`)

// sinceFindings is the output with the revision given as REV, and without
// the phase lines.
func sinceFindings(out, rev string) string {
	return strings.ReplaceAll(phaseLine.ReplaceAllString(out, ""), rev, "REV")
}

func TestSinceFindingsIgnoresPhaseTimes(t *testing.T) {
	run := func(rev, at, verdict string) string {
		return "workline: judging the docs made suspect by the commits since " + rev + "\n" +
			"workline: documentalist: pre (" + at + " UTC)\n" +
			"workline: " + verdict + "\n"
	}
	same := run("v1", "09:13:26", "pass — no doc changed")
	if sinceFindings(same, "v1") != sinceFindings(run("df14f5c", "09:13:25", "pass — no doc changed"), "df14f5c") {
		t.Error("same findings on different seconds compare different")
	}
	if sinceFindings(same, "v1") == sinceFindings(run("df14f5c", "09:13:26", "block — docs/a.md suspect"), "df14f5c") {
		t.Error("different findings compare the same")
	}
}

func TestSinceKnownRevisions(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := sinceRepo(t)
	first := strings.TrimSpace(func() string {
		o, _ := exec.Command("git", "-C", repo, "rev-list", "--max-parents=0", "HEAD").Output()
		return string(o)
	}())
	byCommit, code := runIn(t, bin, repo, "docs", "--since", first, "--ai", "none")
	if code != 0 {
		t.Fatalf("first commit: exit %d\n%s", code, byCommit)
	}
	if strings.Contains(byCommit, "names no commit") || regexp.MustCompile(`(?m)block\s+—\s*$`).MatchString(byCommit) {
		t.Errorf("first commit rejected:\n%s", byCommit)
	}
	byTag, code := runIn(t, bin, repo, "docs", "--since", "v1", "--ai", "none")
	if code != 0 {
		t.Fatalf("tag: exit %d\n%s", code, byTag)
	}
	if sinceFindings(byTag, "v1") != sinceFindings(byCommit, first) {
		t.Errorf("tag and its commit differ:\n%s\n--\n%s", byTag, byCommit)
	}
}

func TestSinceNotAnAncestor(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := sinceRepo(t)
	gitIn(t, repo, "checkout", "-q", "--orphan", "other")
	gitIn(t, repo, "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "other")
	gitIn(t, repo, "checkout", "-q", "master")
	other, _ := exec.Command("git", "-C", repo, "rev-parse", "other").Output()
	out, _ := runIn(t, bin, repo, "docs", "--since", strings.TrimSpace(string(other)), "--ai", "none")
	if regexp.MustCompile(`(?m)block\s+—\s*$`).MatchString(out) {
		t.Errorf("a block line with an empty reason:\n%s", out)
	}
}

func TestTestCheckAndDocsAgreeOnUnknownRevision(t *testing.T) {
	sinceEnv(t)
	bin := sinceBuild(t)
	repo := sinceRepo(t)
	dOut, dCode := runIn(t, bin, repo, "docs", "--since", "v9.9", "--ai", "none")
	tOut, tCode := runIn(t, bin, repo, "test", "check", "--base", "v9.9")
	if dCode == 0 || dCode != tCode {
		t.Errorf("want the same non-zero exit, docs %d, test check %d\n%s\n--\n%s", dCode, tCode, dOut, tOut)
	}
	for name, o := range map[string]string{"docs": dOut, "test check": tOut} {
		if !strings.Contains(o, "v9.9") {
			t.Errorf("%s does not name v9.9:\n%s", name, o)
		}
	}
}
