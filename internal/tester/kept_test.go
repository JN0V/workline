package tester

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGuardIn holds a named guard to the base's test files (#1359): a
// file, a test it declares, a Go subtest; never a file's extension read
// as a name, nor a name none holds.
func TestGuardIn(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tests/test_add.py", "def test_add_keeps_sum():\n    assert 1 + 1 == 2\n")
	write("src/test/AddTest.java", "import java.util.List;\nclass AddTest { void addsTwo() {} }\n")
	write("calc/add_test.go", "func TestAdd(t *testing.T) { t.Run(\"sub case\", nil) }\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@invalid", "commit", "-q", "-m", "tests"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	files := []string{"tests/test_add.py", "src/test/AddTest.java", "calc/add_test.go"}
	for name, want := range map[string]bool{
		"tests.test_add.test_add_keeps_sum":           true,
		"tests/test_add.py::test_add_keeps_sum[0.00]": true,
		"test_add.py":                       true,
		"add_test.go":                       true,
		"pkg.AddTest.addsTwo":               true,
		"AddTest.java":                      true,
		"TestAdd/sub_case":                  true,
		"NoSuchTest.java":                   false,
		"tests.test_add.test_late_discount": false,
		"java":                              false,
	} {
		if got := guardIn(dir, "HEAD", name, files); got != want {
			t.Errorf("guardIn(%q) = %v, want %v", name, got, want)
		}
	}
}
