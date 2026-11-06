package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

func lintedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-qm", "base")
	run("checkout", "-qb", "feat")
	if err := os.WriteFile(filepath.Join(dir, "deploy.sh"), []byte("echo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-qm", "shell")
	return dir
}

func TestLintedRefusesAnUnlintedLanguageTheChangeTouches(t *testing.T) {
	dir := lintedRepo(t)
	ch, err := changeSince(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	g := &Gate{Enforce: map[string]string{"linted": "block"}}
	got := linted(dir, g, ch)
	if len(got) != 1 || got[0].Where != "linted/shell" || got[0].Level != "" ||
		!strings.Contains(got[0].Message, "shellcheck") {
		t.Fatalf("want one blocking finding naming shellcheck, got %+v", got)
	}
	g.Enforce["linted/shell"] = "exempt"
	got = linted(dir, g, ch)
	if len(got) != 1 || got[0].Level != "warn" || !strings.Contains(got[0].Message, "exempted") {
		t.Fatalf("want the exemption said as a warning, got %+v", got)
	}
	if got := linted(dir, &Gate{}, ch); got != nil {
		t.Errorf("rule unset, nothing to say: %+v", got)
	}
}

func TestLintedIsEnforcedByRunWith(t *testing.T) {
	dir := lintedRepo(t)
	write := func(enforce string) {
		t.Helper()
		cfg := "gates:\n  g:\n    checks:\n      - id: ok\n        run: \"true\"\n        output: exit\n    enforce:\n" + enforce
		if err := os.MkdirAll(filepath.Join(dir, ".workline"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("      linted: block\n")
	v, _ := RunWith(dir, "g", Options{Since: "main"})
	if v.Status != verdict.Block {
		t.Fatalf("want blocked, got %+v", v)
	}
	var seen bool
	for _, f := range v.Findings {
		seen = seen || (f.Where == "linted/shell" && f.Level == "")
	}
	if !seen {
		t.Errorf("want a blocking linted/shell finding, got %+v", v.Findings)
	}
	write("      linted: block\n      linted/shell: exempt\n")
	v, _ = RunWith(dir, "g", Options{Since: "main"})
	if v.Status != verdict.Pass {
		t.Errorf("an exempted language passes, got %+v", v)
	}
}
