package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A person approves every push (ADR-0007). Run with no terminal, as an agent
// or an editor's button pushes, the push is refused and nothing reaches the
// remote; the person's own config turns the approval off.
func TestPushNeedsAPerson(t *testing.T) {
	for _, c := range []struct {
		name, config string
		pushed       bool
	}{
		{name: "no terminal", pushed: false},
		{name: "turned off by the person", config: "approve-push: false\n", pushed: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			work := t.TempDir()
			env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"), "GIT_CONFIG_GLOBAL="+filepath.Join(work, "gitconfig"))
			run := func(dir string, args ...string) (string, error) {
				cmd := exec.Command(args[0], args[1:]...)
				cmd.Dir, cmd.Env = dir, env
				cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // no controlling terminal
				out, err := cmd.CombinedOutput()
				return string(out), err
			}
			os.WriteFile(filepath.Join(work, "gitconfig"), nil, 0o644)
			if c.config != "" {
				os.MkdirAll(filepath.Join(work, "config", "workline"), 0o755)
				os.WriteFile(filepath.Join(work, "config", "workline", "config.yaml"), []byte(c.config), 0o644)
			}
			remote, repo := filepath.Join(work, "remote.git"), filepath.Join(work, "repo")
			if out, err := run(work, "git", "init", "-q", "--bare", remote); err != nil {
				t.Fatal(out)
			}
			if err := build(repo, "basic", []string{"git remote add origin " + remote, "git push -q origin main"}, env); err != nil {
				t.Fatal(err)
			}
			if out, err := run(repo, engineBin, "hooks", "install", "--global"); err != nil {
				t.Fatal(out)
			}
			if out, err := run(repo, "git", "commit", "-q", "--allow-empty", "-m", "docs: say nothing new"); err != nil {
				t.Fatal(out)
			}
			out, err := run(repo, "git", "push", "origin", "main")
			head, _ := run(repo, "git", "rev-parse", "HEAD")
			there, _ := run(repo, "git", "ls-remote", remote, "refs/heads/main")
			pushed := strings.HasPrefix(there, strings.TrimSpace(head))
			if pushed != c.pushed || (err == nil) != c.pushed {
				t.Fatalf("pushed = %v (exit error %v), want %v\n%s", pushed, err, c.pushed, out)
			}
			if !c.pushed && !strings.Contains(out, "a person approves every push") {
				t.Errorf("the refusal does not say why:\n%s", out)
			}
		})
	}
}
