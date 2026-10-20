package gate

import (
	"strings"
	"testing"
)

// The merge base's findings are read as main kept them only when they
// stand for a run there: the same command, none of the tool's config
// changed, the same tool by its version or its binary; else why.
func TestKeptBaseOnlyWhenItStandsForARun(t *testing.T) {
	c := Check{ID: "lint", Run: "golangci-lint run --output.sarif.path={out}/l.sarif", Output: "sarif"}
	kept := func(k *keptCheck) *keptFile {
		return &keptFile{V: keptVersion, Commit: "b", Gates: map[string]map[string]*keptCheck{"merge": {"lint": k}}}
	}
	same := &keptCheck{Command: commandOf(c), Tool: "golangci-lint 2.15.0", Binaries: "golangci-lint=ab"}
	plain := &change{files: map[string]bool{"x.go": true}}
	for _, tc := range []struct {
		name       string
		k          *keptFile
		ch         *change
		tool, bins string
		why        string // "" when read
	}{
		{"the same run", kept(same), plain, "golangci-lint 2.15.0", "golangci-lint=ab", ""},
		{"none kept at the base", nil, plain, "golangci-lint 2.15.0", "golangci-lint=ab", "-"},
		{"another check", &keptFile{Gates: map[string]map[string]*keptCheck{}}, plain, "golangci-lint 2.15.0", "golangci-lint=ab", "not kept there"},
		{"another command", kept(&keptCheck{Command: "x", Tool: same.Tool, Binaries: same.Binaries}), plain, same.Tool, same.Binaries, "another command"},
		{"its config changed", kept(same), &change{files: map[string]bool{"sub/.golangci.yml": true}}, same.Tool, same.Binaries, "touches its config, sub/.golangci.yml"},
		{"another version", kept(same), plain, "golangci-lint 2.16.0", same.Binaries, "kept with golangci-lint 2.15.0, the head ran golangci-lint 2.16.0"},
		{"another build", kept(same), plain, same.Tool, "golangci-lint=cd", "another build"},
		{"no version known", kept(&keptCheck{Command: same.Command}), plain, "", "", "version is not known"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, why := keptBase(tc.k, "merge", c, tc.ch, tc.tool, tc.bins)
			switch {
			case tc.why == "" && (got == nil || why != ""):
				t.Errorf("not read: %q", why)
			case tc.why == "-" && (got != nil || why != ""):
				t.Errorf("read %v, why %q; want nothing said", got, why)
			case tc.why != "" && tc.why != "-" && (got != nil || !strings.Contains(why, tc.why)):
				t.Errorf("read %v, why %q; want %q", got, why, tc.why)
			}
		})
	}
}

// A commit's findings are written beside those of other checks and
// gates kept there, and read back; a file of another form is not read.
func TestKeptWrittenAndReadBack(t *testing.T) {
	dir := t.TempDir()
	if err := keep(dir, "c0", "merge", "lint", &keptCheck{Command: "a", Results: []keptResult{{Rule: "r", Path: "x.go", Rest: "m"}}}); err != nil {
		t.Fatal(err)
	}
	if err := keep(dir, "c0", "release", "deps", &keptCheck{Command: "b"}); err != nil {
		t.Fatal(err)
	}
	k := readKept(dir, "c0")
	if k == nil || k.Gates["merge"]["lint"].Results[0].Path != "x.go" || k.Gates["release"]["deps"].Command != "b" {
		t.Fatalf("read back %+v", k)
	}
	if readKept(dir, "c1") != nil || readKept("", "c0") != nil {
		t.Error("read findings kept at another commit, or with no folder")
	}
}
