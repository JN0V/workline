package tester

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/pathglob"
)

// reportFiles are the files test-report names in tree, sorted: a path,
// a pattern one folder deep (`target/surefire-reports/*.xml`), or one
// with `**` spanning folders, as Gradle writes a report per module,
// `**/build/test-results/**/*.xml` (#1175). A `**` pattern walks from
// the folder its fixed part names, `.git` left out.
func reportFiles(tree, pattern string) []string {
	if !strings.Contains(pattern, "**") {
		found, _ := filepath.Glob(filepath.Join(tree, filepath.FromSlash(pattern)))
		return found
	}
	clean := path.Clean(filepath.ToSlash(pattern))
	parts := strings.Split(clean, "/")
	fixed := 0
	for fixed < len(parts) && !strings.ContainsAny(parts[fixed], `*?[\`) {
		fixed++
	}
	root := filepath.Join(tree, filepath.FromSlash(path.Join(parts[:fixed]...)))
	var found []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // a folder unread holds no report to read
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(tree, p)
		if err == nil && pathglob.Match(clean, filepath.ToSlash(rel)) {
			found = append(found, p)
		}
		return nil
	})
	return found
}
