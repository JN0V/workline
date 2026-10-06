package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandsOver(t *testing.T) {
	p := Paths{Hooks: filepath.Join(t.TempDir(), "workline", "hooks")}
	for _, c := range []struct {
		name, script string
		mode         os.FileMode
		want         bool
	}{
		{"none", "", 0, false},
		{"foreign", "#!/bin/sh\nnpx commitlint --edit \"$1\"\n", 0o755, false},
		{"calls workline", "#!/bin/sh\nworkline  hook commit-msg \"$1\"\n", 0o755, true},
		{"calls workline, not executable", "#!/bin/sh\nworkline hook commit-msg \"$1\"\n", 0o644, false},
		{"another hook's call", "#!/bin/sh\nworkline hook pre-push \"$@\"\n", 0o755, false},
		{"calls workline by path, after a guard", "#!/bin/sh\n[ -n \"$CI\" ] && exit 0\n\"$HOME/bin/workline\" hook commit-msg \"$@\" || exit $?\n", 0o755, true},
		{"in a comment", "#!/bin/sh\n# workline hook commit-msg \"$1\"\nexit 0\n", 0o755, false},
		{"echoed", "#!/bin/sh\necho workline hook commit-msg\n", 0o755, false},
		{"after exit", "#!/bin/sh\nexit 0\nworkline hook commit-msg \"$1\"\n", 0o755, false},
		{"a longer name", "#!/bin/sh\nworkline hook commit-msg-lint \"$1\"\n", 0o755, false},
		{"dispatcher", "#!/bin/sh\n# workline hook dispatcher, written by \"workline hooks install\".\n", 0o755, true},
		{"dispatcher marker elsewhere", "#!/bin/sh\nexit 0\n# workline hook dispatcher\n", 0o755, false},
		{"global hook", "#!/bin/sh\nexec " + filepath.Join(p.Hooks, "commit-msg") + " \"$@\"\n", 0o755, true},
	} {
		dir := t.TempDir()
		if c.script != "" {
			if err := os.WriteFile(filepath.Join(dir, "commit-msg"), []byte(c.script), c.mode); err != nil {
				t.Fatal(err)
			}
		}
		if got := HandsOver(dir, "commit-msg", p); got != c.want {
			t.Errorf("%s: HandsOver = %v, want %v", c.name, got, c.want)
		}
	}
}
