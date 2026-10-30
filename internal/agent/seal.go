package agent

// No credential in an agent's reach on a person's machine as in CI
// (ADR-0056, step 2, principle 6): an agent working in a worktree runs
// git, and may run gh or glab, which would hand it the person's
// credentials. Its environment seals them, with the tools' own switches:
// git reads no system nor global config and runs no credential helper,
// sends no header the environment could reset, prompts no one and offers
// ssh no key; gh and glab read an empty config folder. The engine checks
// with git, under this environment, that nothing is left (developer's
// credentialsInReach). Its own login (HOME, ~/.claude) is left.

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// sealDrop are the variables that would undo the seal, or carry the
// person's way to a credential: dropped before the seal's own are set.
var sealDrop = []string{
	"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG", "GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT",
	"GIT_ASKPASS", "SSH_ASKPASS", "SSH_ASKPASS_REQUIRE", "GIT_TERMINAL_PROMPT",
	"GIT_SSH", "GIT_SSH_COMMAND", "GIT_SSH_VARIANT", "SSH_AUTH_SOCK",
	"GH_CONFIG_DIR", "GLAB_CONFIG_DIR",
	"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN", "GITLAB_TOKEN", "GLAB_TOKEN",
}

// sshSealed is ssh with no key: no user config (-F), no agent's key
// (IdentitiesOnly, SSH_AUTH_SOCK dropped), no default key file
// (IdentityFile), no prompt (BatchMode). It overrides core.sshCommand.
const sshSealed = "ssh -F /dev/null -o IdentitiesOnly=yes -o IdentityFile=/dev/null -o BatchMode=yes"

// Seal is environ for an agent that works in a worktree: no way left to a
// credential git, ssh, gh or glab would find for it. dir is a folder the
// caller owns and removes after the call: the empty global config and the
// tools' empty config folders are there.
func Seal(environ []string, dir string) ([]string, error) {
	global := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(global, nil, 0o600); err != nil {
		return nil, err
	}
	gh, glab := filepath.Join(dir, "gh"), filepath.Join(dir, "glab")
	for _, d := range []string{gh, glab} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	env := make([]string, 0, len(environ)+16)
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(sealDrop, name) || strings.HasPrefix(name, "GIT_CONFIG_KEY_") || strings.HasPrefix(name, "GIT_CONFIG_VALUE_") {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		// git: no system config, an empty global one; then, in the
		// command's scope read last, an empty helper and header, which
		// reset the lists before them (git-config(1)).
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+global,
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=credential.helper", "GIT_CONFIG_VALUE_0=",
		"GIT_CONFIG_KEY_1=http.extraheader", "GIT_CONFIG_VALUE_1=",
		// No one asked: a prompt fails, as an askpass that fails.
		"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=false", "SSH_ASKPASS=false",
		"GIT_SSH_COMMAND="+sshSealed,
		"GH_CONFIG_DIR="+gh, "GLAB_CONFIG_DIR="+glab,
		// gh, its config empty, still reads the keyring's token for its
		// host (`gh auth token`, gh 2.103): a token in the environment
		// comes first, so a placeholder that opens nothing is given.
		"GH_TOKEN="+Placeholder, "GH_ENTERPRISE_TOKEN="+Placeholder, "GITLAB_TOKEN="+Placeholder,
	), nil
}

// Placeholder is the value of the tokens a sealed agent is given: none
// that opens anything, so gh and glab look for no other.
const Placeholder = "workline-sealed"

// sealed is env sealed when the agent works in a worktree, and what
// removes the seal's folder after its call.
func sealed(env []string, tree string) ([]string, func(), error) {
	if tree == "" {
		return env, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "workline-seal-*")
	if err != nil {
		return nil, nil, err
	}
	done := func() { _ = os.RemoveAll(dir) }
	env, err = Seal(env, dir)
	if err != nil {
		done()
		return nil, nil, err
	}
	return env, done, nil
}
