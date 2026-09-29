package review

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsKeepsWhatIsAccepted(t *testing.T) {
	dir := t.TempDir()
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid")
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.invalid")
	git("init", "-q", "-b", "main")
	for _, f := range []string{"a.md", "b.md", "code.go"} {
		os.WriteFile(filepath.Join(dir, f), []byte("old\n"), 0o644)
	}
	git("add", ".")
	git("commit", "-q", "-m", "first")
	for _, f := range []string{"a.md", "b.md", "code.go"} {
		os.WriteFile(filepath.Join(dir, f), []byte("new\n"), 0o644)
	}
	git("add", "code.go") // the person's own work, staged: not the docs'

	var out bytes.Buffer
	viewed := false
	committed, err := Docs(dir, []string{"a.md", "b.md"}, bufio.NewReader(strings.NewReader("v\ny\nn\n")), &out,
		func(string) error { viewed = true; return nil })
	if err != nil || !committed {
		t.Fatalf("committed %v, %v\n%s", committed, err, out.String())
	}
	if files := git("show", "--name-only", "--format=%s"); files != "docs: bring a.md up to date with the code\n\na.md\n" {
		t.Errorf("the docs commit holds:\n%s", files)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "b.md")); string(b) != "old\n" {
		t.Errorf("the change refused is still there: %q", b)
	}
	if st := git("status", "--porcelain"); st != "M  code.go\n" {
		t.Errorf("the person's staged work was touched: %q", st)
	}
	if !viewed {
		t.Error("v did not open the page")
	}
}
