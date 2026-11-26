package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Verdict not kept
// Given a forge command that does not know `keep-verdict`,
// When `workline follow --verdict verdict.json` runs,
// Then it warns the verdict was not kept and exits 0.
func TestFollowVerdictOnForgeWithoutKeepVerdictWarnsAndExitsZero(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	bin := filepath.Join(t.TempDir(), "workline")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	repo := t.TempDir()
	env := append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com",
		"GITHUB_ACTIONS=", "GITHUB_REPOSITORY=", "GITHUB_TOKEN=", "GH_TOKEN=",
		"GITLAB_CI=", "GITLAB_TOKEN=", "CI=")
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=" + os.DevNull}, args...)...)
		cmd.Dir = repo
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("commit", "-q", "--allow-empty", "-m", "init")
	head := git("rev-parse", "HEAD")

	// The project sets a merge policy, so the verdict is to be kept.
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("merge: {autonomy: normal}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	verdictFile := filepath.Join(repo, "verdict.json")
	if err := os.WriteFile(verdictFile, []byte(`{"head":"`+head+`","status":"pass"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A forge command that knows every operation but keep-verdict.
	script := `cat >/dev/null; case "$WORKLINE_FORGE_OPERATION" in keep-verdict) echo '{"error":"unknown operation keep-verdict"}';; *) echo '{}';; esac`
	cmd := exec.Command(bin, "follow", "--forge", "cmd:"+script, "--verdict", "verdict.json")
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		code = 1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	if code != 0 {
		t.Errorf("exit %d, want 0:\n%s", code, out)
	}
	low := strings.ToLower(string(out))
	if !strings.Contains(low, "not kept") {
		t.Errorf("the output does not say the verdict was not kept:\n%s", out)
	}
	if !strings.Contains(low, "(warn)") {
		t.Errorf("the verdict not kept is not a warning:\n%s", out)
	}
}
