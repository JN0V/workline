package simulator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// suiteRun runs the fixture's tools/workline-tests.sh on one component
// with a g++ that counts its calls, the suite cache folder given to the
// script; it returns how many times g++ ran, the report and the error.
func suiteRun(t *testing.T, repo, shim, cache, log string) (int, string, error) {
	report := filepath.Join(t.TempDir(), "report.xml")
	before, _ := os.ReadFile(log)
	cmd := exec.Command("tools/workline-tests.sh", report, "DomoticsCore-Core/test")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"), "WORKLINE_SUITE_CACHE="+cache)
	out, err := cmd.CombinedOutput()
	after, _ := os.ReadFile(log)
	data, _ := os.ReadFile(report)
	_ = out
	return strings.Count(string(after), "g++\n") - strings.Count(string(before), "g++\n"), string(data), err
}

func suiteWorld(t *testing.T) (repo, shim, cache, log string) {
	work := t.TempDir()
	hermetic(t, work)
	repo = filepath.Join(work, "repo")
	if out, err := exec.Command("cp", "-a", repoTemplate, repo).CombinedOutput(); err != nil {
		t.Fatalf("the repository: %v\n%s", err, out)
	}
	real, err := exec.LookPath("g++")
	if err != nil {
		t.Fatal(err)
	}
	shim, cache, log = filepath.Join(work, "shim"), filepath.Join(work, "cache"), filepath.Join(work, "gpp.log")
	if err := os.MkdirAll(shim, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho 'g++' >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shim, "g++"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo, shim, cache, log
}

// Same sources, one build: Given two simulator runs of a suite with the
// same sources, When the second runs, Then it uses the first's binary and
// starts no g++.
func TestSameSourcesOneBuild(t *testing.T) {
	repo, shim, cache, log := suiteWorld(t)
	n, report, err := suiteRun(t, repo, shim, cache, log)
	if err != nil || n != 1 || !strings.Contains(report, `name="test_subscribe_receives"`) {
		t.Fatalf("first run: %d g++, %v\n%s", n, err, report)
	}
	n, report, err = suiteRun(t, repo, shim, cache, log)
	if err != nil || n != 0 {
		t.Errorf("second run: %d g++ started, want 0 (%v)", n, err)
	}
	if !strings.Contains(report, `name="test_subscribe_receives"`) {
		t.Errorf("second run lost its case:\n%s", report)
	}
}

// Changed source, a new build: Given a source file changed, When the suite
// is needed, Then it is built again, and a broken source still fails.
func TestChangedSourceNewBuild(t *testing.T) {
	repo, shim, cache, log := suiteWorld(t)
	src := filepath.Join(repo, "DomoticsCore-Core", "test", "test_eventbus", "test_eventbus.cpp")
	orig, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if n, _, err := suiteRun(t, repo, shim, cache, log); err != nil || n != 1 {
		t.Fatalf("first run: %d g++, %v", n, err)
	}
	if n, _, _ := suiteRun(t, repo, shim, cache, log); n != 0 {
		t.Fatalf("unchanged run started %d g++: nothing cached, so a changed source proves no new key", n)
	}
	if err := os.WriteFile(src, append(orig, []byte("// changed\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	n, report, err := suiteRun(t, repo, shim, cache, log)
	if err != nil || n != 1 || !strings.Contains(report, `name="test_subscribe_receives"`) {
		t.Errorf("changed source: %d g++, want 1 (%v)\n%s", n, err, report)
	}
	if err := os.WriteFile(src, append(orig, []byte("int broken(\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := suiteRun(t, repo, shim, cache, log); err == nil {
		t.Error("a broken source passed")
	}
}
