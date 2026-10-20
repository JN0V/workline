package hooks

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOutdated(t *testing.T) {
	dir := t.TempDir()
	p := Paths{Hooks: filepath.Join(dir, "hooks"), Next: filepath.Join(dir, "next-hooks-path")}
	if err := writeDispatchers(p.Hooks, "/opt/bin/work line", p.Next); err != nil {
		t.Fatal(err)
	}
	if old := OutdatedGlobal(p); len(old) != 0 {
		t.Fatalf("dispatchers just written are outdated: %v", old)
	}
	// What an older workline wrote: the git dir's hooks, which a linked
	// worktree does not have.
	file := filepath.Join(p.Hooks, "pre-push")
	data, _ := os.ReadFile(file)
	older := strings.Replace(string(data), `--git-common-dir`, `--git-dir`, 1)
	if older == string(data) {
		t.Fatal("the dispatcher no longer reads --git-common-dir: update this test")
	}
	os.WriteFile(file, []byte(older), 0o755)
	if old := OutdatedGlobal(p); !reflect.DeepEqual(old, []string{"pre-push"}) {
		t.Errorf("outdated = %v, want [pre-push]", old)
	}
	// Another tool's hook, or none, is not workline's to rewrite.
	os.WriteFile(filepath.Join(p.Hooks, "commit-msg"), []byte("#!/bin/sh\nexec lint \"$1\"\n"), 0o755)
	os.Remove(filepath.Join(p.Hooks, "post-commit"))
	if old := OutdatedGlobal(p); !reflect.DeepEqual(old, []string{"pre-push"}) {
		t.Errorf("outdated = %v, want [pre-push]", old)
	}
	// A dispatcher recording no binary is outdated.
	os.WriteFile(filepath.Join(p.Hooks, "post-merge"), []byte("#!/bin/sh\n# workline hook dispatcher, written by an old one.\n"), 0o755)
	if old := OutdatedGlobal(p); !reflect.DeepEqual(old, []string{"post-merge", "pre-push"}) {
		t.Errorf("outdated = %v, want [post-merge pre-push]", old)
	}
}
