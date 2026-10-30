package tester

// A fresh tree (ADR-0056, step 3; roles/developer/docs/proof.md): where
// the developer's code is proven, never in its agent's worktree. A
// repository of its own, in repo's object format, borrowing only repo's
// objects: none of its config, attributes, hooks, refs or replacements,
// nor the machine's git config. A commit holding a submodule or a
// filtered file is refused, not checked out half.

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
	format, err := freshGit(repo, "rev-parse", "--show-object-format")
	if err != nil {
		return fail(err)
	}
	tree, err = os.MkdirTemp("", "workline-proof-")
	if err != nil {
		return fail(err)
	}
	done = func() { _ = os.RemoveAll(tree) }
	if _, err := freshGit(tree, "init", "-q", "--template=", "--object-format="+format, "."); err != nil {
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
	if why, err := notWhole(tree); err != nil || why != "" {
		done()
		if err == nil {
			err = fmt.Errorf("%s is not whole in a fresh tree: %s", short(at), why)
		}
		return fail(err)
	}
	return tree, at, done, nil
}

// notWhole says what of the commit checked out in tree a fresh tree does
// not give whole: a submodule, left empty, or a file whose attributes
// name a filter (git-lfs, an encryption), checked out as stored, never
// smudged; "" when nothing.
func notWhole(tree string) (string, error) {
	staged, err := freshGit(tree, "ls-files", "--stage", "-z")
	if err != nil {
		return "", err
	}
	var files []string
	for _, e := range strings.Split(staged, "\x00") {
		mode, name, ok := strings.Cut(e, "\t")
		if !ok {
			continue
		}
		if strings.HasPrefix(mode, "160000 ") {
			return "submodule " + name, nil
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		return "", nil
	}
	cmd := exec.Command("git", "-C", tree, "--no-replace-objects", "-c", "core.hooksPath=/dev/null", "check-attr", "-z", "--stdin", "filter")
	cmd.Env = freshEnv()
	cmd.Stdin = strings.NewReader(strings.Join(files, "\x00") + "\x00")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git check-attr filter: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	// -z: path, attribute, value, each ended by NUL.
	f := strings.Split(string(out), "\x00")
	for i := 0; i+2 < len(f); i += 3 {
		if v := f[i+2]; v != "unspecified" && v != "unset" {
			return fmt.Sprintf("%s names filter %s", f[i], v), nil
		}
	}
	return "", nil
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
