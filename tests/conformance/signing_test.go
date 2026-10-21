package conformance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Every GitLab job that pushes commits names the bot through .workline:bot,
// which, given an SSH key as the File variable WORKLINE_SIGNING_KEY, sets
// git to sign with it: the key made readable by its owner alone and ended
// with a newline, as ssh-keygen asks; no key, git left unsigned, as
// before; a key given as a plain variable, the job fails without printing
// it (#549).
func TestGitLabBotSignsWithItsKey(t *testing.T) {
	data, err := os.ReadFile("../../ci/gitlab/workline.gitlab-ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := yaml.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	bot, _ := top[".workline:bot"].([]any)
	if len(bot) == 0 {
		t.Fatal(".workline:bot: missing, or not a script")
	}
	writers := 0
	for name, v := range top {
		job, ok := v.(map[string]any)
		script := fmt.Sprint(job["script"])
		if !ok || strings.HasPrefix(name, ".") || !strings.Contains(script, "git remote set-url origin") {
			continue // pushes nothing
		}
		writers++
		if !strings.Contains(script, fmt.Sprint(bot[len(bot)-1])) {
			t.Errorf("%s: pushes commits, but not as .workline:bot: they are never signed", name)
		}
	}
	if writers < 5 {
		t.Fatalf("%d jobs push commits, want apply, merge, garden:apply, follow and sample:write", writers)
	}
	var lines strings.Builder
	for _, l := range bot {
		fmt.Fprintln(&lines, l)
	}
	run := func(key string) (string, string, error) {
		t.Helper()
		repo := t.TempDir()
		if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
			t.Fatalf("git init: %s", out)
		}
		cmd := exec.Command("sh", "-ec", lines.String())
		cmd.Dir = repo
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + repo, "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
			"CI_SERVER_HOST=gitlab.example.com", "WORKLINE_SIGNING_KEY=" + key}
		out, err := cmd.CombinedOutput()
		var set strings.Builder
		for _, k := range []string{"user.name", "gpg.format", "user.signingkey", "commit.gpgsign"} {
			if v, err := exec.Command("git", "-C", repo, "config", "--local", "--get", k).Output(); err == nil {
				fmt.Fprintf(&set, "%s %s", k, v)
			}
		}
		return string(out), set.String(), err
	}
	if out, set, err := run(""); err != nil || set != "user.name workline\n" {
		t.Errorf("no key: %v %s; git set to %q, want the bot's name alone", err, out, set)
	}
	key := filepath.Join(t.TempDir(), "WORKLINE_SIGNING_KEY")
	if err := os.WriteFile(key, []byte("not a real key"), 0o666); err != nil {
		t.Fatal(err)
	}
	os.Chmod(key, 0o666) // as GitLab writes a File variable
	want := "user.name workline\ngpg.format ssh\nuser.signingkey " + key + "\ncommit.gpgsign true\n"
	if out, set, err := run(key); err != nil || set != want {
		t.Errorf("a key: %v %s; git set to %q, want %q", err, out, set, want)
	}
	if fi, err := os.Stat(key); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("a key: its mode is %v (%v), want 0600, which ssh-keygen asks", fi.Mode().Perm(), err)
	}
	if got, _ := os.ReadFile(key); string(got) != "not a real key\n" {
		t.Errorf("a key with no last newline, which ssh-keygen does not read: %q, want one added", got)
	}
	if _, _, err := run(key); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(key); string(got) != "not a real key\n" {
		t.Errorf("a key ending with a newline: %q, want it as it was", got)
	}
	text := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----"
	if out, set, err := run(text); err == nil || strings.Contains(out, "secret") || strings.Contains(set, "gpgsign") || !strings.Contains(out, "File") {
		t.Errorf("a key as a plain variable: %v; printed %q, git set to %q; want a failure, the key unprinted, saying to make it a File variable", err, out, set)
	}
}
