package engine

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/role"
)

// Each step on its own: Given a role with built-in pre code and no post,
// When it runs, Then its code runs for pre and the shared post runs
// after. The role folder holds no pre nor post file.
func TestScriptBuiltinPreThenSharedPost(t *testing.T) {
	const name = "builtin-pre-only-fixture"
	if !slices.Contains(contract.Builtins().Names(), name) {
		contract.Join(name, contract.Role{Pre: func(runDir, repo string) int {
			if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
				return 1
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("from the built-in pre\n"), 0o644); err != nil {
				return 1
			}
			return 0
		}})
	}
	roleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(roleDir, "instruction.md"), []byte("the instruction file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &role.Role{Name: name, Dir: roleDir}
	runDir, repo := t.TempDir(), t.TempDir()
	env := append(os.Environ(), "WORKLINE_RUN_DIR="+runDir)

	code, err := script(r, "pre", repo, env)
	if err != nil || code != 0 {
		t.Fatalf("pre: code %d, error %v; want 0, none", code, err)
	}
	task, _ := os.ReadFile(filepath.Join(runDir, "in", "task.md"))
	if string(task) != "from the built-in pre\n" {
		t.Errorf("pre left in/task.md = %q, want what the built-in pre wrote, not the shared pre's instruction", task)
	}

	code, err = script(r, "post", repo, env)
	if err != nil || code != 0 {
		t.Fatalf("post: code %d, error %v; want 0, none", code, err)
	}
	verdict, _ := os.ReadFile(filepath.Join(runDir, "out", "verdict.yaml"))
	if string(verdict) != "status: pass\n" {
		t.Errorf("post left out/verdict.yaml = %q, want the shared post's pass", verdict)
	}
}

// Built-in code only: Given a role with built-in pre code and a pre file
// beside it that a project script would put there, When its pre runs,
// Then the built-in code runs and the file is not run.
func TestScriptBuiltinFileBesideItIsNotRun(t *testing.T) {
	const name = "builtin-pre-beside-fixture"
	if !slices.Contains(contract.Builtins().Names(), name) {
		contract.Join(name, contract.Role{Pre: func(runDir, repo string) int {
			if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
				return 1
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte("from the built-in pre\n"), 0o644); err != nil {
				return 1
			}
			return 0
		}})
	}
	roleDir, runDir, repo := t.TempDir(), t.TempDir(), t.TempDir()
	ran := filepath.Join(t.TempDir(), "the-file-ran")
	if err := os.WriteFile(filepath.Join(roleDir, "pre"), []byte("#!/bin/sh\ntouch '"+ran+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := &role.Role{Name: name, Dir: roleDir}
	env := append(os.Environ(), "WORKLINE_RUN_DIR="+runDir)

	code, err := script(r, "pre", repo, env)
	if err != nil || code != 0 {
		t.Fatalf("pre: code %d, error %v; want 0, none", code, err)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the pre file beside the built-in code ran")
	}
	if task, _ := os.ReadFile(filepath.Join(runDir, "in", "task.md")); string(task) != "from the built-in pre\n" {
		t.Errorf("pre left in/task.md = %q, want what the built-in pre wrote", task)
	}
}

// Process state restored: Given a built-in step run in this test binary
// through inProcess, with an environment and a folder of its own, When
// the step ends, Then the process's environment and working folder are
// as they were.
func TestInProcessRestoresEnvironmentAndFolder(t *testing.T) {
	t.Setenv("WORKLINE_INPROCESS_PROBE", "before")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	was := os.Environ()
	repo := t.TempDir()
	var inside, probe string
	code := inProcess(func(runDir, repo string) int {
		inside, _ = os.Getwd()
		probe = os.Getenv("WORKLINE_INPROCESS_PROBE")
		return 0
	}, repo, []string{"WORKLINE_RUN_DIR=" + t.TempDir(), "WORKLINE_INPROCESS_PROBE=during"})
	if code != 0 {
		t.Fatalf("inProcess: code %d, want 0", code)
	}
	if real, _ := filepath.EvalSymlinks(repo); inside != repo && inside != real {
		t.Errorf("the step ran in %s, want %s", inside, repo)
	}
	if probe != "during" {
		t.Errorf("the step saw WORKLINE_INPROCESS_PROBE=%q, want the run's own", probe)
	}
	if now, _ := os.Getwd(); now != wd {
		t.Errorf("the working folder is %s after the step, want %s", now, wd)
	}
	now := os.Environ()
	slices.Sort(now)
	slices.Sort(was)
	if !slices.Equal(now, was) {
		t.Error("the process's environment differs after the step")
	}
}
