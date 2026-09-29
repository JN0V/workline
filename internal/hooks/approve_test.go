package hooks

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func pushRepo(t *testing.T) (string, []Ref) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"update-ref", "refs/remotes/origin/main", "HEAD"},
		{"commit", "-q", "--allow-empty", "-m", "feat: the change to push"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir, []Ref{{Local: "refs/heads/main", Remote: "refs/heads/main", Range: "origin/main..main"}}
}

func TestAsk(t *testing.T) {
	repo, refs := pushRepo(t)
	for _, c := range []struct {
		typed  string
		want   bool
		viewed bool
	}{
		{typed: "y\n", want: true},
		{typed: "\n", want: false}, // the default is no
		{typed: "", want: false},   // no answer
		{typed: "v\ny\n", want: true, viewed: true},
	} {
		var out bytes.Buffer
		viewed := ""
		got := ask(Push{Repo: repo, Remote: "origin", Refs: refs}, strings.NewReader(c.typed), &out, func(p string) error { viewed = p; return nil })
		if got != c.want || (viewed != "") != c.viewed {
			t.Errorf("typed %q: approved %v, page %q; want %v, viewed %v\n%s", c.typed, got, viewed, c.want, c.viewed, out.String())
		}
		if !strings.Contains(out.String(), "feat: the change to push") {
			t.Errorf("the commits pushed are not listed:\n%s", out.String())
		}
		if viewed != "" {
			page, _ := os.ReadFile(viewed)
			if !strings.Contains(string(page), "feat: the change to push") {
				t.Errorf("the page does not show the commit")
			}
		}
	}
	var out bytes.Buffer
	if !ask(Push{Repo: repo, Remote: "origin", Refs: []Ref{{Local: "refs/heads/main", Remote: "refs/heads/main"}}}, strings.NewReader(""), &out, nil) {
		t.Errorf("nothing to push was not let through: %s", out.String())
	}
}
