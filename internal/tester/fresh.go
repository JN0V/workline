package tester

// A fresh tree (ADR-0056, step 3): where the developer's code is proven,
// never in its agent's worktree. A repository of its own, apart: it
// borrows repo's objects, named by their content, so nothing written in
// repo since can change the commit's files; it reads none of repo's
// config, attributes, hooks, refs or replacements, which a worktree of
// repo shares with the agent's, nor the machine's git config. What the
// agent left beside its diff (an ignored file, a cache, a commit put in
// the start commit's place) is not there.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/verdict"
)

// freshTree checks rev out in a repository of its own, apart, and says
// the commit it checked out; done removes it.
func freshTree(repo, rev string) (tree, at string, done func(), f *verdict.Finding) {
	fail := func(err error) (string, string, func(), *verdict.Finding) {
		return "", "", nil, &verdict.Finding{Rule: "check-error", Where: "test", Message: err.Error()}
	}
	at, err := freshGit(repo, "rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return fail(fmt.Errorf("%s could not be read: %w", short(rev), err))
	}
	objects, err := freshGit(repo, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	if err != nil {
		return fail(err)
	}
	tree, err = os.MkdirTemp("", "workline-proof-")
	if err != nil {
		return fail(err)
	}
	done = func() { _ = os.RemoveAll(tree) }
	if _, err := freshGit(tree, "init", "-q", "--template=", "."); err != nil {
		done()
		return fail(err)
	}
	alt := filepath.Join(tree, ".git", "objects", "info", "alternates")
	if err := os.MkdirAll(filepath.Dir(alt), 0o755); err != nil {
		done()
		return fail(err)
	}
	if err := os.WriteFile(alt, []byte(objects+"\n"), 0o644); err != nil {
		done()
		return fail(err)
	}
	if _, err := freshGit(tree, "-c", "advice.detachedHead=false", "checkout", "-q", "--detach", at); err != nil {
		done()
		return fail(fmt.Errorf("%s could not be checked out: %w", short(at), err))
	}
	return tree, at, done, nil
}

// freshGit runs git in dir as a fresh tree needs it: no replacement
// read, no global nor system config, no hook, no GIT_DIR inherited.
func freshGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir, "--no-replace-objects", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = freshEnv()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// freshEnv is the process's environment without what points git
// elsewhere, its global and system config off.
func freshEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_NO_REPLACE_OBJECTS=1")
}
