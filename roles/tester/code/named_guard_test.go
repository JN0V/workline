package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// guardedBase commits two test files in a temporary repository, git's
// global config pointed at the null device, and gives its path: the
// base whose tests a reason may name.
func guardedBase(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	for name, body := range map[string]string{
		"calc/add_test.go":  "package calc\n\nfunc TestAdd(t *testing.T) {}\n",
		"tests/test_mul.py": "def test_mul_keeps_product():\n    assert 2 * 3 == 6\n",
		"calc/add.go":       "package calc\n\nfunc Add(a, b int) int { return a + b }\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@invalid", "commit", "-q", "-m", "tests"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

// Name not found: Given a reason naming a test the base does not hold,
// When the tester writes the Draft, Then the scenario is listed as
// unproven, the name said not found. The reason does not open with
// "kept as today": it is read all the same.
func TestReasonNamingMissingTestIsUnproven(t *testing.T) {
	dir := guardedBase(t)
	said := map[string]string{"Adds": "already checked by the existing `TestNoSuchCase`"}
	guardedSaid(dir, "HEAD", Settings{}, said)
	got := said["Adds"]
	if !strings.Contains(got, "no existing test was found that checks it") || !strings.Contains(got, "`TestNoSuchCase` named, in no test file of the base") {
		t.Errorf("a reason naming a test the base lacks = %q, want it unguarded, the name said not found", got)
	}
}

// Existing test named: Given a reason naming in code a test the base
// holds, When the tester writes the Draft, Then the scenario is not
// listed as unproven. The reason does not open with "kept as today":
// it is read as guarded by the test it names.
func TestReasonNamingExistingTestIsGuarded(t *testing.T) {
	dir := guardedBase(t)
	said := map[string]string{"Adds": "already checked by the existing `TestAdd`"}
	guardedSaid(dir, "HEAD", Settings{}, said)
	got := said["Adds"]
	if strings.Contains(got, "no existing test was found that checks it") {
		t.Errorf("a reason naming a test the base holds = %q, want it not listed as unproven", got)
	}
	if want := "kept as today: guarded by `TestAdd`"; got != want {
		t.Errorf("a reason naming a test the base holds = %q, want %q", got, want)
	}
}
