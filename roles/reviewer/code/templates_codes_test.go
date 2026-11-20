package reviewer

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShippedTemplatesHoldNoInternalCode: a project copies the CI
// templates of defaults/ci/ as they are, and its pull request goes
// through its reviewer; a code of workline's own in their comments
// (a decision's number) would block it there.
func TestShippedTemplatesHoldNoInternalCode(t *testing.T) {
	t.Setenv("WORKLINE_ROLES_DIR", filepath.Join("..", "..", "..", "roles"))
	codes, err := committerSettings(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "..", "defaults", "ci")
	c := Change{Added: map[string][]Line{}}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(p, ".go") {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		for i, text := range strings.Split(string(data), "\n") {
			c.Added[rel] = append(c.Added[rel], Line{rel, i + 1, text})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Added) == 0 {
		t.Fatal("no template read under " + root)
	}
	found, err := mechanical(c, Settings{}, codes)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		if f.Rule == "internal-code" {
			t.Errorf("defaults/ci/%s: %s", f.Where, f.Message)
		}
	}
}
