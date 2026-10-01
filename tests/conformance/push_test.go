package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// The push approval is the person's choice (ADR-0011): off by default, the
// review being on the pull request. Asked for, with no terminal to ask on,
// the push is refused and nothing reaches the remote.
func TestPushNeedsAPerson(t *testing.T) {
	for _, c := range []struct {
		name, config string
		pushed       bool
	}{
		{name: "off by default: the review is on the pull request", pushed: true},
		{name: "asked for, no terminal", config: "approve-push: true\napprove-push-via: [terminal]\n", pushed: false}, // no editor window nor dialog, whoever runs the tests
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

// Without a terminal, as an agent runs it, `workline docs` has the docs the
// commits not pushed yet made suspect judged, and leaves the changes in the
// working tree for a person: nothing is committed.
func TestDocsWithoutTerminal(t *testing.T) {
	work := t.TempDir()
	env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"))
	remote, repo := filepath.Join(work, "remote.git"), filepath.Join(work, "repo")
	if out, err := exec.Command("git", "init", "-q", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	if err := build(repo, "documented", []string{
		"git remote add origin " + remote, "git push -q origin main",
		"sed -i 's/3600/7200/' src/auth/token.go && git commit -qam 'feat(auth): keep users signed in for two hours'",
	}, env); err != nil {
		t.Fatal(err)
	}
	roles, _ := filepath.Abs("../../roles")
	agent, _ := filepath.Abs("fixtures/agents/doc-fixed.yaml")
	cmd := exec.Command(engineBin, "docs", "--ai", "fake:"+agent)
	cmd.Dir, cmd.Env = repo, append(env, "WORKLINE_ROLES="+roles)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "for a person to review") {
		t.Fatalf("%v\n%s", err, out)
	}
	doc, _ := os.ReadFile(filepath.Join(repo, "docs/tech/auth.md"))
	if !strings.Contains(string(doc), "two hours") {
		t.Errorf("the doc was not brought up to date:\n%s", doc)
	}
	if log, _ := exec.Command("git", "-C", repo, "log", "-1", "--format=%s").Output(); !strings.HasPrefix(string(log), "feat(auth)") {
		t.Errorf("something was committed without a person: %s", log)
	}
}

// A run of `workline docs` stopped by a fix the judge refused says so, and
// why, and that docs are left to judge: tried on a solo repository, it said
// only how many docs had changed, and seven were never put before the agent.
func TestDocsSaysWhatStoppedIt(t *testing.T) {
	work := t.TempDir()
	env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"))
	repo := filepath.Join(work, "repo")
	if err := build(repo, "documented", []string{
		"sed -i 's/3600/7200/' src/auth/token.go && git commit -qam 'feat(auth): keep users signed in for two hours'",
	}, env); err != nil {
		t.Fatal(err)
	}
	roles, _ := filepath.Abs("../../roles")
	agent, _ := filepath.Abs("fixtures/agents/doc-fix-not-applying.yaml")
	cmd := exec.Command(engineBin, "docs", "--ai", "fake:"+agent)
	cmd.Dir, cmd.Env = repo, append(env, "WORKLINE_ROLES="+roles)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	out, _ := cmd.CombinedOutput()
	for _, want := range []string{"block", "patch-does-not-apply", "not judged yet"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the output does not say %q:\n%s", want, out)
		}
	}
}

// The push only counts the docs (ADR-0010): one line, how many its commits
// made suspect and how many were before, and the command that judges them;
// never the docs one by one with their commits, never a question.
func TestPushCountsTheDocs(t *testing.T) {
	work := t.TempDir()
	env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"), "GIT_CONFIG_GLOBAL="+filepath.Join(work, "gitconfig"))
	os.WriteFile(filepath.Join(work, "gitconfig"), nil, 0o644)
	remote, repo := filepath.Join(work, "remote.git"), filepath.Join(work, "repo")
	if out, err := exec.Command("git", "init", "-q", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	if err := build(repo, "documented", []string{
		"mkdir -p .workline && printf 'routing:\\n  events: {pre-push: [documentalist]}\\n' > .workline/config.yaml && git add .workline && git commit -qm 'chore: check docs before a push'",
		"git remote add origin " + remote, "git push -q origin main",
		"sed -i 's/3600/7200/' src/auth/token.go && git commit -qam 'feat(auth): keep users signed in for two hours'",
	}, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir, cmd.Env = repo, env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run(engineBin, "hooks", "install", "--global"); err != nil {
		t.Fatal(out)
	}
	out, err := run("git", "push", "origin", "main")
	if err != nil {
		t.Fatalf("the push was stopped:\n%s", out)
	}
	if !strings.Contains(out, "docs suspect: 1 made so by these commits, 0 before") || !strings.Contains(out, "workline docs") {
		t.Errorf("the push does not count the docs in one line:\n%s", out)
	}
	if strings.Contains(out, "changed since it was checked") || strings.Contains(out, "feat(auth): keep users") {
		t.Errorf("the push lists the docs and their commits:\n%s", out)
	}
}

// On a repository pushed to main with no merge request (ADR-0010),
// `workline docs` judges from where the docs were last judged — a ref it
// moves, refs/workline/docs-judged — to HEAD, pushed or not. The ref moves
// once a run passed with nothing left for a person to review.
func TestDocsFromWhereLastJudged(t *testing.T) {
	work := t.TempDir()
	env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"))
	remote, repo := filepath.Join(work, "remote.git"), filepath.Join(work, "repo")
	if out, err := exec.Command("git", "init", "-q", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	if err := build(repo, "documented", []string{
		"git remote add origin " + remote,
		"sed -i 's/3600/7200/' src/auth/token.go && git commit -qam 'feat(auth): keep users signed in for two hours'",
		"git push -q origin main", // pushed: nothing waits, yet the doc is suspect
	}, env); err != nil {
		t.Fatal(err)
	}
	roles, _ := filepath.Abs("../../roles")
	agent, _ := filepath.Abs("fixtures/agents/doc-fixed.yaml")
	docs := func(args ...string) string {
		cmd := exec.Command(engineBin, append([]string{"docs"}, args...)...)
		cmd.Dir, cmd.Env = repo, append(env, "WORKLINE_ROLES="+roles)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		return string(out)
	}
	ref := func() string {
		out, _ := exec.Command("git", "-C", repo, "rev-parse", "-q", "--verify", "refs/workline/docs-judged").Output()
		return strings.TrimSpace(string(out))
	}
	// 0. No agent: nothing judged, the ref stays.
	docs("--ai", "none")
	if ref() != "" {
		t.Fatalf("the ref moved with no agent to judge the docs")
	}
	// 1. Judged although pushed; the fix left for a person: the ref stays.
	if out := docs("--ai", "fake:"+agent); !strings.Contains(out, "for a person to review") {
		t.Fatalf("the pushed commits were not judged:\n%s", out)
	}
	if ref() != "" {
		t.Fatalf("the ref moved before a person reviewed the fix")
	}
	// 2. The person commits it; judged again, nothing to change: the ref moves to HEAD.
	commit := exec.Command("git", "-C", repo, "commit", "-qam", "docs: tokens last two hours")
	commit.Env = append(env, "GIT_CONFIG_GLOBAL="+filepath.Join(work, "gitconfig"))
	os.WriteFile(filepath.Join(work, "gitconfig"), nil, 0o644)
	if out, err := commit.CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	out := docs("--ai", "none")
	head, _ := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if ref() != strings.TrimSpace(string(head)) {
		t.Fatalf("the ref did not move to HEAD once the docs were judged:\n%s", out)
	}
	// 3. Nothing since: nothing judged, and it says so.
	if out := docs("--ai", "fake:"+agent); !strings.Contains(out, "no commit since the docs were last judged") {
		t.Fatalf("judged again what was judged:\n%s", out)
	}
}

// A branch already pushed that merged main sends the merge alone: main's
// commits, GitHub's merge commits among them, are the remote's already, and
// the committer does not judge them again (their committer, noreply@github.com,
// is on no one's list).
func TestPushAfterMergingMain(t *testing.T) {
	work := t.TempDir()
	env := append(hermeticEnv(), "XDG_CONFIG_HOME="+filepath.Join(work, "config"), "GIT_CONFIG_GLOBAL="+filepath.Join(work, "gitconfig"))
	os.WriteFile(filepath.Join(work, "gitconfig"), nil, 0o644)
	os.MkdirAll(filepath.Join(work, "config", "workline"), 0o755)
	os.WriteFile(filepath.Join(work, "config", "workline", "allowed-identities"), []byte("^fixture@example\\.invalid$\n"), 0o644)
	remote, repo := filepath.Join(work, "remote.git"), filepath.Join(work, "repo")
	if out, err := exec.Command("git", "init", "-q", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	if err := build(repo, "documented", []string{
		"mkdir -p .workline && printf 'routing:\\n  events: {pre-push: [committer]}\\n' > .workline/config.yaml && git add .workline && git commit -qm 'chore: check commits before a push'",
		"git remote add origin " + remote, "git push -q origin main",
		"git checkout -qb topic && echo '// topic' >> src/auth/token.go && git commit -qam 'feat(auth): say so' && git push -q origin topic",
		// main moves on by a merge GitHub made
		"git checkout -q main && GIT_COMMITTER_NAME=GitHub GIT_COMMITTER_EMAIL=noreply@github.com git commit -q --allow-empty -m 'Merge pull request #1 from someone/branch' && git push -q origin main",
		"git checkout -q topic && git merge -q --no-edit main",
	}, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir, cmd.Env = repo, env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run(engineBin, "hooks", "install", "--global"); err != nil {
		t.Fatal(out)
	}
	if out, err := run("git", "push", "origin", "topic"); err != nil {
		t.Fatalf("the push was stopped:\n%s", out)
	}
}
