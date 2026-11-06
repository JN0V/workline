package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func lintRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, a := range [][]string{{"init", "-q", "-b", "main"}, {"add", "run.sh"}} {
		if a[0] == "add" {
			if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("git", a...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", a, err, out)
		}
	}
	return dir
}

func TestProposeLintersWritesOnlyOnYes(t *testing.T) {
	dir := lintRepo(t)
	file := filepath.Join(dir, ".workline", "config.yaml")

	note, err := proposeLinters(dir, "", false) // not asked, no answer: only said
	if err != nil || !strings.Contains(note, "shellcheck") || !strings.Contains(note, "nothing written") {
		t.Fatalf("note %q, err %v", note, err)
	}
	if _, err := os.Stat(file); err == nil {
		t.Fatal("config written without the person's agreement")
	}
	if _, err := proposeLinters(dir, "no", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err == nil {
		t.Fatal("config written after no")
	}

	note, err = proposeLinters(dir, "yes", false)
	if err != nil || !strings.Contains(note, "added") {
		t.Fatalf("note %q, err %v", note, err)
	}
	data, _ := os.ReadFile(file)
	if !strings.Contains(string(data), "shellcheck") {
		t.Fatalf("shellcheck check not written: %s", data)
	}
}
