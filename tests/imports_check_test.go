package tests

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "github.com/JN0V/workline"

// checkImports scans every Go file under internal/ and roles/ of root,
// test files included, and returns one line per import that breaks the
// roles' separation: the core importing a role's code, or a role
// importing another role's.
func checkImports(root string) ([]string, error) {
	var found []string
	rolesPrefix := modulePath + "/roles"
	for _, top := range []string{"internal", "roles"} {
		dir := filepath.Join(root, top)
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if path == dir {
					return nil // folder absent from this tree
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			role := ""
			if top == "roles" {
				parts := strings.Split(rel, "/")
				if len(parts) == 2 {
					return nil // roles/builtin.go: the list of the compiled-in roles, no role's code
				}
				role = parts[1]
			}
			for _, spec := range file.Imports {
				imp, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return fmt.Errorf("%s: %w", rel, err)
				}
				if imp != rolesPrefix && !strings.HasPrefix(imp, rolesPrefix+"/") {
					continue
				}
				if top == "internal" {
					found = append(found, fmt.Sprintf("%s: the core imports %s", rel, imp))
					continue
				}
				rest := strings.TrimPrefix(strings.TrimPrefix(imp, rolesPrefix), "/")
				other, _, _ := strings.Cut(rest, "/")
				if other != "" && other != role {
					found = append(found, fmt.Sprintf("%s: role %s imports role %s (%s)", rel, role, other, imp))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(found)
	return found, nil
}
