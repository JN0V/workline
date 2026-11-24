package scripts

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGh puts a gh on the PATH that logs each call and answers the lock's
// calls: the copy's main, then a refused creation of the lock tag (it
// exists) when exists is true. Any other call fails.
func fakeGh(t *testing.T, exists bool) (dir, log string) {
	t.Helper()
	dir = t.TempDir()
	log = filepath.Join(dir, "gh.log")
	refuse := "exit 0"
	if exists {
		refuse = "echo 'Reference already exists' >&2; exit 1"
	}
	script := `#!/usr/bin/env bash
echo "$*" >> "` + log + `"
case "$*" in
  "api repos/JN0V/TraceGlass-trial/commits/main --jq .sha") echo 0123456789abcdef ;;
  "api -X POST repos/JN0V/TraceGlass-trial/git/refs "*) ` + refuse + ` ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, log
}

func run(t *testing.T, path string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"try-release.sh"}, args...)...)
	cmd.Env = append(os.Environ(), "PATH="+path+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TRY_REPO=", "TRY_WORKLINE=")
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(out), exit.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func calls(t *testing.T, log string) string {
	t.Helper()
	b, err := os.ReadFile(log)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTryReleaseDryRunPrintsThePlanAndCallsNoForge(t *testing.T) {
	dir, log := fakeGh(t, false)
	out, code := run(t, dir, "--dry-run", "--minutes", "30", "v0.89.1")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	for _, want := range []string{
		"try of v0.89.1 on JN0V/TraceGlass-trial (dry run",
		"tag workline-try-lock",
		"checksums.txt of JN0V/workline v0.89.1",
		"`workline doctor`",
		"pull request workline-v0.89.1",
		"the script never labels",
		"An outline of 65536 by 65536 pixels",
		"wait up to 30 min for a pull request closing it",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the plan misses %q:\n%s", want, out)
		}
	}
	if c := calls(t, log); c != "" {
		t.Errorf("a dry run called the forge:\n%s", c)
	}
}

func TestTryReleaseRefusesWhatIsNotARelease(t *testing.T) {
	dir, log := fakeGh(t, false)
	for _, args := range [][]string{
		{"main"}, {"0.89.1"}, {"v0.89"}, {"--", "--help"}, {"v0.89.1", "v0.89.2"},
		{"--minutes", "soon", "v0.89.1"}, {},
	} {
		out, code := run(t, dir, args...)
		if code != 3 {
			t.Errorf("%q: exit %d, want 3:\n%s", args, code, out)
		}
	}
	if c := calls(t, log); c != "" {
		t.Errorf("a refused run called the forge:\n%s", c)
	}
}

func TestTryReleaseRefusesWhileAnotherTryRuns(t *testing.T) {
	dir, log := fakeGh(t, true)
	out, code := run(t, dir, "v0.89.1")
	if code != 3 {
		t.Fatalf("exit %d, want 3:\n%s", code, out)
	}
	if !strings.Contains(out, "another try is running on JN0V/TraceGlass-trial") {
		t.Errorf("the refusal does not say a try runs:\n%s", out)
	}
	c := calls(t, log)
	if strings.Contains(c, "DELETE") {
		t.Errorf("the running try's lock was removed:\n%s", c)
	}
	if strings.Contains(c, "release download") {
		t.Errorf("the release was downloaded while another try runs:\n%s", c)
	}
}
