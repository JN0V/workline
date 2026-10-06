// Package hooks installs workline's git hooks.
//
// Global mode takes core.hooksPath, which git honours for one folder only. So,
// like the forbidden-terms and git-identity-guard tools, it records whatever
// held core.hooksPath before and every hook hands over to it afterwards, then
// to the repository's own hooks. Nothing that ran before stops running.
package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// names are the hooks forwarded down the chain. Taking core.hooksPath would
// otherwise silently disable any of them set up elsewhere.
var names = []string{
	"applypatch-msg", "pre-applypatch", "post-applypatch", "pre-commit",
	"pre-merge-commit", "prepare-commit-msg", "commit-msg", "post-commit",
	"pre-rebase", "post-checkout", "post-merge", "pre-push", "post-rewrite",
	"push-to-checkout", "pre-auto-gc",
}

const dispatcher = `#!/bin/sh
# workline hook dispatcher, written by "workline hooks install".
# Runs workline's check for this hook, then hands over: first to whatever held
# core.hooksPath before workline was installed, else to the repository's hooks.
hook=$(basename "$0")
[ -n "${WORKLINE_HOOK_RUNNING-}" ] && exit 0
export WORKLINE_HOOK_RUNNING=1

# Some hooks read what git sends on standard input (pre-push: the refs being
# pushed). Every link of the chain gets its own copy.
input=""
case "$hook" in
pre-push|post-rewrite)
	input=$(mktemp) || exit 1
	trap 'rm -f "$input"' EXIT
	cat >"$input"
	;;
esac
feed() {
	if [ -n "$input" ]; then "$@" <"$input"; else "$@"; fi
}

bin=%q
[ -x "$bin" ] || bin=$(command -v workline 2>/dev/null)
if [ -n "$bin" ]; then
	feed "$bin" hook "$hook" "$@" || exit $?
else
	echo "workline: binary not found — $hook was NOT checked." >&2
fi

next_file=%q
if [ -r "$next_file" ]; then
	next=$(cat "$next_file")
	if [ -x "$next/$hook" ] && [ ! "$next/$hook" -ef "$0" ]; then
		feed "$next/$hook" "$@"
		exit $?
	fi
fi
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
git_dir=$(git rev-parse --git-dir 2>/dev/null) || exit 0
for d in "$root/.githooks/$hook" "$git_dir/hooks/$hook"; do
	if [ -x "$d" ] && [ ! "$d" -ef "$0" ]; then
		feed "$d" "$@"
		exit $?
	fi
done
exit 0
`

// Paths used by global mode.
type Paths struct {
	Hooks string // folder core.hooksPath points to
	Next  string // file recording the previous core.hooksPath
}

// DefaultPaths returns the folders under the user's config.
func DefaultPaths() (Paths, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	base := filepath.Join(cfg, "workline")
	return Paths{Hooks: filepath.Join(base, "hooks"), Next: filepath.Join(base, "next-hooks-path")}, nil
}

// InstallGlobal writes the dispatchers and points the global core.hooksPath at
// them, recording what was there before. Running it twice changes nothing.
func InstallGlobal(p Paths, bin string) (string, error) {
	prev, err := gitConfig("--global", "--get", "core.hooksPath")
	if err != nil {
		return "", err
	}
	if prev != "" && filepath.Clean(prev) != filepath.Clean(p.Hooks) {
		if err := os.MkdirAll(filepath.Dir(p.Next), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(p.Next, []byte(prev+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	if err := writeDispatchers(p.Hooks, bin, p.Next); err != nil {
		return "", err
	}
	if _, err := gitConfig("--global", "core.hooksPath", p.Hooks); err != nil {
		return "", err
	}
	msg := "global hooks installed in " + p.Hooks
	if prev != "" && filepath.Clean(prev) != filepath.Clean(p.Hooks) {
		msg += "; they hand over to the previous hooks in " + prev
	}
	return msg, nil
}

// UninstallGlobal gives core.hooksPath back to whatever held it before.
func UninstallGlobal(p Paths) (string, error) {
	cur, err := gitConfig("--global", "--get", "core.hooksPath")
	if err != nil {
		return "", err
	}
	if filepath.Clean(cur) != filepath.Clean(p.Hooks) {
		return "", fmt.Errorf("the global core.hooksPath is %q, not workline's; left untouched", cur)
	}
	prev := ""
	if data, err := os.ReadFile(p.Next); err == nil {
		prev = strings.TrimSpace(string(data))
	}
	if prev != "" {
		_, err = gitConfig("--global", "core.hooksPath", prev)
	} else {
		_, err = gitConfig("--global", "--unset", "core.hooksPath")
	}
	if err != nil {
		return "", err
	}
	os.RemoveAll(p.Hooks)
	os.Remove(p.Next)
	if prev != "" {
		return "global hooks removed; core.hooksPath is back to " + prev, nil
	}
	return "global hooks removed", nil
}

// InstallRepo writes a commit-msg dispatcher in the repository's .githooks/,
// where it can be committed for the whole team.
func InstallRepo(repo, bin string) (string, error) {
	dir := filepath.Join(repo, ".githooks")
	target := filepath.Join(dir, "commit-msg")
	if data, err := os.ReadFile(target); err == nil && !bytes.Contains(data, []byte("workline hook dispatcher")) {
		return "", fmt.Errorf("%s already exists and is not workline's; add `workline hook commit-msg \"$1\"` to it instead", target)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(fmt.Sprintf(dispatcher, bin, "")), 0o755); err != nil {
		return "", err
	}
	msg := "wrote " + target
	local, _ := gitConfigIn(repo, "--local", "--get", "core.hooksPath")
	global, _ := gitConfig("--global", "--get", "core.hooksPath")
	switch {
	case local == ".githooks":
	case global != "":
		msg += fmt.Sprintf("\nnote: a global core.hooksPath (%s) is set; git reaches .githooks/commit-msg only if that chain hands over to it, or if this repository sets `git config core.hooksPath .githooks` (which bypasses the global hooks here)", global)
	default:
		msg += "\nto enable it: git config core.hooksPath .githooks"
	}
	return msg, nil
}

// HandsOver says whether git, running the hook name from dir, reaches
// workline: the file is executable and is workline's dispatcher, or runs
// `workline hook <name>` or workline's global hook of that name as a
// command, before any line that exits. A comment, an echo or a line after
// `exit` is not a hand-over.
func HandsOver(dir, name string, p Paths) bool {
	path := filepath.Join(dir, name)
	if fi, err := os.Stat(path); err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
		return false // git skips a hook it cannot execute
	}
	if p.Hooks != "" && filepath.Clean(dir) == filepath.Clean(p.Hooks) {
		return true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 1 && strings.HasPrefix(lines[1], "# workline hook dispatcher") {
		return true
	}
	global := ""
	if p.Hooks != "" {
		global = filepath.Join(p.Hooks, name)
	}
	for _, line := range lines {
		// An exit at the top level ends the script; one indented in a block
		// may not run.
		if f := strings.Fields(line); len(f) > 0 && f[0] == "exit" && line[0] == 'e' {
			return false
		}
		for _, words := range commands(line) {
			w := unquote(words[0])
			if global != "" && filepath.Clean(expandHome(w)) == global {
				return true
			}
			if filepath.Base(w) == "workline" && len(words) > 2 && words[1] == "hook" && unquote(words[2]) == name {
				return true
			}
		}
	}
	return false
}

// shellOps split a line where a new command may start.
var shellOps = strings.NewReplacer("&&", "\n", "||", "\n", ";", "\n", "|", "\n", "$(", "\n", "(", "\n", ")", "\n", "{", "\n", "}", "\n", "`", "\n")

// commands returns each command a shell line runs, as its words from the
// command's name on: leading keywords, VAR=value assignments and a runner's
// options (`npx --no-install`) are dropped, a trailing comment cut.
func commands(line string) [][]string {
	f := strings.Fields(line)
	for i, w := range f {
		if strings.HasPrefix(w, "#") {
			f = f[:i]
			break
		}
	}
	var out [][]string
	for _, part := range strings.Split(shellOps.Replace(strings.Join(f, " ")), "\n") {
		words := strings.Fields(part)
		for len(words) > 0 {
			w := words[0]
			name, _, assign := strings.Cut(w, "=")
			switch {
			case w == "exec" || w == "command" || w == "npx" || w == "env" || w == "time" || w == "nohup" || w == "!" ||
				w == "if" || w == "then" || w == "do" || w == "else" || w == "elif" || w == "while" || w == "until":
			case strings.HasPrefix(w, "-"):
			case assign && name != "" && !strings.ContainsAny(name, `/"'$`):
			default:
				out = append(out, words)
				words = nil
				continue
			}
			words = words[1:]
		}
	}
	return out
}

func unquote(w string) string { return strings.Trim(w, `"'`) }

// expandHome reads ~, $HOME and $XDG_CONFIG_HOME in a path, as the shell would.
func expandHome(w string) string {
	if strings.HasPrefix(w, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			w = filepath.Join(h, w[2:])
		}
	}
	return os.Expand(w, func(v string) string {
		if v == "HOME" || v == "XDG_CONFIG_HOME" {
			return os.Getenv(v)
		}
		return "$" + v
	})
}

func writeDispatchers(dir, bin, next string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	script := []byte(fmt.Sprintf(dispatcher, bin, next))
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), script, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func gitConfig(args ...string) (string, error) { return gitConfigIn("", args...) }

func gitConfigIn(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"config"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", nil // key not set
	}
	return strings.TrimSpace(string(out)), err
}
