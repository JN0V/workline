package linters

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProposeAndAdopt(t *testing.T) {
	d := t.TempDir()
	for _, c := range [][]string{{"init", "-q"}} {
		if out, err := exec.Command("git", append([]string{"-C", d}, c...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(d, "run.sh"), []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", d, "add", ".").CombinedOutput(); err != nil {
		t.Fatal(string(out))
	}
	ps, err := Propose(d)
	if err != nil {
		t.Fatal(err)
	}
	var sc *Proposal
	for i := range ps {
		if ps[i].Tool == "shellcheck" {
			sc = &ps[i]
		}
	}
	if sc == nil {
		t.Fatalf("shellcheck not proposed: %+v", ps)
	}
	if _, err := os.Stat(filepath.Join(d, ".workline", "config.yaml")); err == nil {
		t.Fatal("proposing wrote the config")
	}
	added, err := Adopt(d, "lint", []Proposal{*sc})
	if err != nil || len(added) != 1 {
		t.Fatalf("adopt: %v %v", added, err)
	}
	data, _ := os.ReadFile(filepath.Join(d, ".workline", "config.yaml"))
	if !strings.Contains(string(data), "shellcheck") || !strings.Contains(string(data), "gates:") {
		t.Fatalf("config: %s", data)
	}
	again, err := Adopt(d, "lint", []Proposal{*sc})
	if err != nil || len(again) != 0 {
		t.Fatalf("second adopt added %v (%v)", again, err)
	}
}
