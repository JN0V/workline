package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// specPageLinesMax is the longest a specification page may be to be judged
// in parts: the longest page kept whole today is under it, the three docs
// that were split (routing, the role contract, refine) over it.
const specPageLinesMax = 400

// specSplitFamilies are the docs that were too long to be judged in
// parts, and the pages that replace them beside them.
var specSplitFamilies = []string{
	"docs/spec/routing",
	"docs/spec/role-contract",
	"docs/spec/backlog-acts/refine",
}

// specSources reads the `sources` a doc declares in its header.
func specSources(t *testing.T, data []byte, path string) []string {
	t.Helper()
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return nil
	}
	head, _, ok := strings.Cut(text[4:], "\n---")
	if !ok {
		t.Errorf("%s: its header is not closed", path)
		return nil
	}
	var h struct {
		Sources []string `yaml:"sources"`
	}
	if err := yaml.Unmarshal([]byte(head), &h); err != nil {
		t.Errorf("%s: its header does not read: %v", path, err)
	}
	return h.Sources
}

// A long specification is judged in parts:
// Given a specification doc split into pages with precise `sources`,
// When gardening runs,
// Then each page is short enough to be judged on its own, and names the
// files or symbols it is judged against, never a whole directory, so the
// agent judges it with no person needed.
func TestLongSpecIsJudgedInParts(t *testing.T) {
	root := ".."
	err := filepath.WalkDir(filepath.Join(root, "docs", "spec"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if n := strings.Count(string(data), "\n"); n > specPageLinesMax {
			t.Errorf("%s: %d lines, over the %d a page may have to be judged in parts: split it into pages", rel, n, specPageLinesMax)
		}
		inFamily := false
		for _, f := range specSplitFamilies {
			if strings.HasPrefix(rel, f) {
				inFamily = true
			}
		}
		if !inFamily {
			return nil
		}
		sources := specSources(t, data, rel)
		if len(sources) == 0 {
			t.Errorf("%s: it names no source, so it cannot be judged against any", rel)
		}
		for _, s := range sources {
			file, _, _ := strings.Cut(s, "#")
			if strings.HasSuffix(file, "/") {
				t.Errorf("%s: source %q is a whole directory: name the files or symbols (`file.go#Symbol`) it describes", rel, s)
				continue
			}
			if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(file))); err == nil && st.IsDir() {
				t.Errorf("%s: source %q is a whole directory: name the files or symbols (`file.go#Symbol`) it describes", rel, s)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
