package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Sealed, git asked for a credential gets none: not from a helper in the
// global config nor in the repository's, nor from a prompt; the person's
// ssh agent, gh's and glab's config are gone, the person's HOME is kept.
func TestSealHandsNoCredential(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	global := filepath.Join(home, "gitconfig")
	helper := "!f() { echo username=someone; echo password=c2VjcmV0; }; f"
	if err := os.WriteFile(global, []byte("[credential]\n\thelper = \""+helper+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "HOME="+home, "GIT_CONFIG_GLOBAL="+global, "GIT_CONFIG_NOSYSTEM=1",
		"SSH_AUTH_SOCK=/tmp/agent.sock", "GH_CONFIG_DIR="+home, "GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=credential.https://github.com.helper", "GIT_CONFIG_VALUE_0="+helper)
	run := func(env []string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Env, cmd.Stdin = env, strings.NewReader("protocol=https\nhost=github.com\n\n")
		out, _ := cmd.Output()
		return string(out)
	}
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	if out, err := exec.Command("git", "-C", repo, "config", "credential.helper", helper).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	if out := run(env, "-C", repo, "credential", "fill"); !strings.Contains(out, "c2VjcmV0") {
		t.Fatalf("unsealed, the helper should answer: %q", out)
	}
	sealed, err := Seal(env, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if out := run(sealed, "-C", repo, "credential", "fill"); strings.Contains(out, "c2VjcmV0") {
		t.Errorf("sealed, git got a credential: %q", out)
	}
	get := func(name string) (string, bool) {
		for _, kv := range slices.Backward(sealed) {
			if k, v, _ := strings.Cut(kv, "="); k == name {
				return v, true
			}
		}
		return "", false
	}
	if _, ok := get("SSH_AUTH_SOCK"); ok {
		t.Error("the ssh agent's socket is left")
	}
	if d, _ := get("GH_CONFIG_DIR"); d == home {
		t.Error("gh reads the person's config")
	}
	if v, _ := get("GH_TOKEN"); v != Placeholder {
		t.Errorf("GH_TOKEN is %q: gh would read the keyring's token", v)
	}
	if h, _ := get("HOME"); h != home {
		t.Errorf("HOME is %q: the agent's own login is lost", h)
	}
	if n := strings.Count(strings.Join(sealed, "\n"), "\nGIT_CONFIG_COUNT="); n != 1 {
		t.Errorf("GIT_CONFIG_COUNT set %d times", n)
	}
}
