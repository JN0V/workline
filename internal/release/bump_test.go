package release

import (
	"errors"
	"testing"
)

func TestOnlyBumps(t *testing.T) {
	head := "diff --git a/f b/f\nindex 1..2 100644\n--- a/f\n+++ b/f\n"
	block := "#!/bin/sh\n# x-release-please-start-version\nVERSION=1.3.0\n# x-release-please-end\nNAME=tool-1.3.0\n# x-release-please-start-major\nMAJOR=2\n# x-release-please-end\n"
	for _, c := range []struct {
		name, path, diff string
		want             bool
	}{
		{"a marked line", "ci/job.yml", head + "@@ -3 +3 @@\n-  V: v0.27.0  # x-release-please-version\n+  V: v0.28.0  # x-release-please-version\n", true},
		{"two hunks", "ci/job.yml", head + "@@ -3 +3 @@\n-a 1.2.0 # x-release-please-version\n+a 1.3.0 # x-release-please-version\n@@ -9 +9 @@\n-b 1.2.0 # x-releaser-pleaser-version\n+b 1.3.0 # x-releaser-pleaser-version\n\\ No newline at end of file\n", true},
		{"a marked line changed besides", "ci/job.yml", head + "@@ -3 +3 @@\n-  V: v0.27.0  # x-release-please-version\n+  V: v0.28.0  # x-release-please-version (latest)\n", false},
		{"a line no tool marks", "go.mod", head + "@@ -3 +3 @@\n-require x v1.2.0\n+require x v1.3.0\n", false},
		{"a marked line and another", "ci/job.yml", head + "@@ -3,2 +3,2 @@\n-V: 1.2.0 # x-release-please-version\n-N: 3\n+V: 1.3.0 # x-release-please-version\n+N: 4\n", false},
		{"a line added", "ci/job.yml", head + "@@ -3,0 +4 @@\n+V: 1.3.0 # x-release-please-version\n", false},
		{"a line in a block", "install.sh", head + "@@ -3 +3 @@\n-VERSION=1.2.0\n+VERSION=1.3.0\n", true},
		{"a line out of a block", "install.sh", head + "@@ -5 +5 @@\n-NAME=tool-1.2.0\n+NAME=tool-1.3.0\n", false},
		{"a major in its block", "install.sh", head + "@@ -7 +7 @@\n-MAJOR=1\n+MAJOR=2\n", true},
		{"a major marked inline", "x.go", head + "@@ -2 +2 @@\n-const Major = 1 // x-release-please-major\n+const Major = 2 // x-release-please-major\n", true},
		{"the manifest", ".release-please-manifest.json", head + "@@ -2 +2 @@\n-  \".\": \"0.27.0\"\n+  \".\": \"0.28.0\"\n", true},
		{"the manifest, a package added", ".release-please-manifest.json", head + "@@ -2 +2,2 @@\n-  \".\": \"0.27.0\"\n+  \".\": \"0.28.0\",\n+  \"cli\": \"1.0.0\"\n", false},
		{"a file added", "v.go", "diff --git a/v.go b/v.go\nnew file mode 100644\nindex 0..1\n--- /dev/null\n+++ b/v.go\n@@ -0,0 +1 @@\n+const V = \"1.0.0\" // x-release-please-version\n", false},
		{"a mode changed", "ci/job.yml", "diff --git a/f b/f\nold mode 100644\nnew mode 100755\n" + "index 1..2\n--- a/f\n+++ b/f\n@@ -3 +3 @@\n-a 1.2.0 # x-release-please-version\n+a 1.3.0 # x-release-please-version\n", false},
		{"nothing", "ci/job.yml", "", false},
	} {
		now := func() (string, error) { return "module x\n\ngo 1.22\nrequire x v1.3.0\n", nil }
		if c.path == "install.sh" {
			now = func() (string, error) { return block, nil }
		}
		if got := OnlyBumps(c.path, c.diff, now); got != c.want {
			t.Errorf("%s: OnlyBumps = %v, want %v", c.name, got, c.want)
		}
	}
	unread := func() (string, error) { return "", errors.New("gone") }
	if OnlyBumps("install.sh", head+"@@ -3 +3 @@\n-VERSION=1.2.0\n+VERSION=1.3.0\n", unread) {
		t.Error("a file that cannot be read is taken for a block")
	}
}
