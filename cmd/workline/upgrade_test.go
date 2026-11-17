package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// upgradePin reads a template's pin of the engine: the release the
// templates of this build carry, which `workline upgrade` pins a project on.
var upgradePin = regexp.MustCompile(`(WORKLINE_VERSION:\s*)(v\d+\.\d+\.\d+)`)

// upgradeTemplate reads a GitHub template as the repository keeps it.
func upgradeTemplate(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "defaults", "ci", "github", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// upgradeRun sets up a git repository holding the given workflows (name to
// content), makes the engine the release the templates carry, runs
// `workline <args>` in it, and returns the repository, the output (stdout
// and stderr) and the exit code.
func upgradeRun(t *testing.T, workflows map[string]string, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	wf := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wf, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range workflows {
		if err := os.WriteFile(filepath.Join(wf, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	latest := upgradePin.FindStringSubmatch(upgradeTemplate(t, "workline-bounds.yml"))[2]
	was := version
	version = latest
	t.Cleanup(func() { version = was })
	t.Chdir(dir)

	var out, errOut string
	var code int
	errOut, _ = captureStderr(t, func() int {
		out, code = captureStdout(t, func() int { return dispatch(args) })
		return code
	})
	return dir, out + errOut, code
}

// One command
//
// Given a project pinned on v0.76.0 with old CI files,
// When the person runs `workline upgrade`,
// Then every workflow is pinned on the latest release and matches its template.
func TestUpgradeInOneCommand(t *testing.T) {
	old := map[string]string{}
	for _, name := range []string{"workline-bounds.yml", "workline-apply.yml"} {
		old[name] = upgradePin.ReplaceAllString(upgradeTemplate(t, name), "${1}v0.76.0")
	}
	dir, out, code := upgradeRun(t, old, "upgrade")
	if code != 0 {
		t.Errorf("workline upgrade: exit %d, want 0: %q", code, out)
	}
	for name := range old {
		got, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(got), "v0.76.0") {
			t.Errorf("%s is still pinned on v0.76.0", name)
		}
		if string(got) != upgradeTemplate(t, name) {
			t.Errorf("%s does not match its template, pinned on the latest release", name)
		}
	}
}

// Local edits kept
//
// Given a workflow the project changed by hand,
// When `workline upgrade` runs,
// Then the change is shown, kept or refused by the person, never lost silently.
func TestUpgradeNeverLosesALocalEditSilently(t *testing.T) {
	tmpl := upgradeTemplate(t, "workline-bounds.yml")
	if !strings.Contains(tmpl, "timeout-minutes: 60") {
		t.Fatal("the template no longer holds the line the test edits")
	}
	edited := upgradePin.ReplaceAllString(tmpl, "${1}v0.76.0")
	edited = strings.Replace(edited, "timeout-minutes: 60", "timeout-minutes: 7", 1)
	dir, out, _ := upgradeRun(t, map[string]string{"workline-bounds.yml": edited}, "upgrade")

	if !strings.Contains(out, "workline-bounds.yml") {
		t.Errorf("the file with a local change is not named in the output: %q", out)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "workline-bounds.yml"))
	if err != nil {
		t.Fatal(err)
	}
	kept := strings.Contains(string(got), "timeout-minutes: 7")
	shown := strings.Contains(out, "timeout-minutes: 7")
	if !kept && !shown {
		t.Errorf("the local change is gone from the file and was never shown: output %q", out)
	}
}

// New template added
//
// Given a template the project never copied,
// When `workline upgrade` runs,
// Then it is added and named in the output.
func TestUpgradeAddsANewTemplate(t *testing.T) {
	old := upgradePin.ReplaceAllString(upgradeTemplate(t, "workline-bounds.yml"), "${1}v0.76.0")
	dir, out, _ := upgradeRun(t, map[string]string{"workline-bounds.yml": old}, "upgrade")

	got, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "workline-answer.yml"))
	if err != nil {
		t.Fatalf("workline-answer.yml, a template never copied, was not added: %v (output %q)", err, out)
	}
	if string(got) != upgradeTemplate(t, "workline-answer.yml") {
		t.Errorf("the added workline-answer.yml does not match its template")
	}
	if !strings.Contains(out, "workline-answer.yml") {
		t.Errorf("the added template is not named in the output: %q", out)
	}
}

// Doctor sees drift
//
// Given a workflow that differs from its template,
// When the person runs `workline doctor`,
// Then it says which file drifted and to run `workline upgrade`.
func TestDoctorSeesDrift(t *testing.T) {
	tmpl := upgradeTemplate(t, "workline-bounds.yml")
	drifted := strings.Replace(tmpl, "timeout-minutes: 60", "timeout-minutes: 7", 1)
	if drifted == tmpl {
		t.Fatal("the template no longer holds the line the test edits")
	}
	_, out, _ := upgradeRun(t, map[string]string{"workline-bounds.yml": drifted}, "doctor")

	if !strings.Contains(out, "workline-bounds.yml") {
		t.Errorf("doctor does not say which file drifted: %q", out)
	}
	if !strings.Contains(out, "workline upgrade") {
		t.Errorf("doctor does not say to run `workline upgrade`: %q", out)
	}
}
