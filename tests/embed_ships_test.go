package tests

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/rolefs"
)

// The binary ships what each role's run reads, named one by one
// (docs/spec/layout.md, "A role folder"): never a role's code/, docs/,
// cases/ nor README.md, and every role keeps its role.yaml.
func TestTheBinaryShipsNoRoleCodeNorDocs(t *testing.T) {
	roles := map[string]bool{}
	err := fs.WalkDir(workline.Roles, "roles", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		parts := strings.Split(p, "/")
		if len(parts) < 3 {
			t.Errorf("%s: embedded, but no role's file", p)
			return nil
		}
		switch {
		case parts[2] == "code", parts[2] == "docs", parts[2] == "cases", parts[2] == "README.md", path.Ext(p) == ".go":
			t.Errorf("%s: embedded, though no run reads it", p)
		case parts[2] == "role.yaml":
			roles[parts[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"committer", "documentalist", "reviewer", "inspector", "tester", "developer", "product-owner", "auditor", "judge"} {
		if !roles[name] {
			t.Errorf("roles/%s/role.yaml is not embedded", name)
		}
	}
}

// shipped says whether a file of a role's folder is one a run reads:
// its role.yaml, pre and post, and all under prompts/ and kinds/.
func shipped(rel string) bool {
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return false
	}
	switch parts[1] {
	case "role.yaml", "pre", "post":
		return len(parts) == 2
	case "prompts", "kinds":
		return len(parts) > 2
	}
	return false
}

// Every file a role's run reads is embedded as it stands in roles/: none
// of its prompts/ nor kinds/ left behind by the embed's patterns, and the
// instruction facet each kind of task names in role.yaml is there. Each
// role loads from the folder the binary extracts.
func TestTheBinaryShipsEveryFileARunReads(t *testing.T) {
	err := filepath.WalkDir("../roles", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel("../roles", p)
		rel = filepath.ToSlash(rel)
		if !shipped(rel) {
			return nil
		}
		want, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		got, err := workline.Roles.ReadFile("roles/" + rel)
		switch {
		case err != nil:
			t.Errorf("roles/%s: not embedded, though a run reads it", rel)
		case !bytes.Equal(got, want):
			t.Errorf("roles/%s: embedded unlike the file in roles/", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	dir, err := rolefs.Dir()
	if err != nil {
		t.Fatal(err)
	}
	names, err := fs.ReadDir(workline.Roles, "roles")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		r, err := role.Load(dir, n.Name())
		if err != nil {
			t.Errorf("%s: does not load from the extracted roles: %v", n.Name(), err)
			continue
		}
		for kind, task := range r.Model.Tasks {
			if task.Instruction == "" {
				continue
			}
			if _, err := fs.Stat(workline.Roles, path.Join("roles", n.Name(), "prompts", task.Instruction)); err != nil {
				t.Errorf("%s: model.tasks.%s names %s, not embedded under prompts/", n.Name(), kind, task.Instruction)
			}
		}
	}
}
