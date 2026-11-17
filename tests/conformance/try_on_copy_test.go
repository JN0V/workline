package conformance

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The try on a copy (ADR-0071 §2): its dry run prints the plan and touches
// nothing on GitHub; a try refuses while another runs, naming it.

const tryScript = "../../scripts/try-on-copy.sh"

// fakeGH puts a gh on PATH that logs each call and answers `repo list`
// with the repositories given.
func fakeGH(t *testing.T, repos string) (path, calls string) {
	t.Helper()
	dir := t.TempDir()
	calls = filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$*\" >> " + calls + "\ncase \"$1 $2\" in\n\"repo list\") printf '%s' '" + repos + "' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir + string(os.PathListSeparator) + os.Getenv("PATH"), calls
}

func runTry(t *testing.T, path string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{tryScript}, args...)...)
	cmd.Env = append(append(os.Environ(), "PATH="+path, "GITHUB_RUN_ID=42"), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(out), exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestTryDryRunTouchesNothing(t *testing.T) {
	path, calls := fakeGH(t, "")
	for _, c := range []struct{ arg, want string }{
		{"v0.78.4", "release v0.78.4 of JN0V/workline, downloaded by the templates as the pin would"},
		{"0.78.4", "release v0.78.4"},
		{"pull/1730/head", "built from pull/1730/head, published as a release engine-<sha> of JN0V/workline-try-42"},
	} {
		out, code := runTry(t, path, nil, "--dry-run", c.arg)
		if code != 0 || !strings.Contains(out, c.want) || !strings.Contains(out, "dry run: nothing done on GitHub") {
			t.Errorf("--dry-run %s: exit %d, want the plan with %q:\n%s", c.arg, code, c.want, out)
		}
		if !strings.Contains(out, "copy of JN0V/TraceGlass-trial at try-base-1") || !strings.Contains(out, "delete JN0V/workline-try-42") {
			t.Errorf("--dry-run %s: the plan names no source, tag or deletion:\n%s", c.arg, out)
		}
	}
	if _, err := os.Stat(calls); err == nil {
		t.Errorf("a dry run called gh")
	}
	if out, code := runTry(t, path, nil, "--dry-run", "v0.78.4", "other/repo"); code != 2 || !strings.Contains(out, "--seed") {
		t.Errorf("another source with no seed: exit %d, want 2 asking for --seed:\n%s", code, out)
	}
}

func TestTryRefusesWhileAnotherRuns(t *testing.T) {
	path, _ := fakeGH(t, "workline-try-41\n")
	out, code := runTry(t, path, []string{"CLAUDE_CODE_OAUTH_TOKEN=x", "WORKLINE_APP_PRIVATE_KEY=x"}, "v0.78.4")
	if code != 3 || !strings.Contains(out, "a try is running, or was kept: JN0V/workline-try-41") {
		t.Errorf("exit %d, want 3 naming the running try:\n%s", code, out)
	}
}
