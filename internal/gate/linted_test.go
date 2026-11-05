package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
