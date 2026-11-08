package tester

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/pathglob"
)

// reportInTree says why a test-report pattern is refused, "" when it
// names files inside the run's tree alone: a pattern leaving it
// (`../**/*.xml`, `/tmp/*.xml`) would read, and remove, files the run
// does not own.
func reportInTree(pattern string) string {
	if !filepath.IsLocal(filepath.FromSlash(pattern)) || strings.HasPrefix(pattern, "/") {
		return fmt.Sprintf("test-report %q leaves the repository: give a path or a pattern from its root, as `junit.xml` or `**/build/test-results/**/*.xml`", pattern)
	}
	return ""
}

// reportFiles are the files test-report names in tree, sorted: a path,
// a pattern one folder deep (`target/surefire-reports/*.xml`), or one
// with `**` spanning folders, as Gradle writes a report per module,
// `**/build/test-results/**/*.xml` (#1175). A `**` pattern walks from
// the folder its fixed part names, the root's `.git` left out. A
// pattern leaving tree names nothing.
func reportFiles(tree, pattern string) []string {
	if reportInTree(pattern) != "" {
		return nil
	}
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
	dotGit := filepath.Join(tree, ".git")
	var found []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // a folder unread holds no report to read
		}
		if d.IsDir() {
			if p == dotGit {
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

// keptFile is a file test-report names that is not the run's to remove,
// as it was before the run: its time and content.
type keptFile struct {
	mod  time.Time
	data []byte
}

// inNestedRepo says whether rel, from tree, lies in a repository nested
// below tree's root, a submodule or a vendored clone: a folder above it
// holds a `.git`. The root's git does not list its files.
func inNestedRepo(tree, rel string) bool {
	for dir := filepath.Dir(rel); dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(tree, dir, ".git")); err == nil {
			return true
		}
	}
	return false
}

// clearReports removes the files test-report names in tree that git
// does not track, a past run's reports (#1185). A tracked one, as
// `pom.xml` under `**/*.xml` or a report committed, or one in a nested
// repository, is never removed: it is kept, its state returned, so
// that, unchanged by the run, it is not read as this run's.
func clearReports(tree, pattern string) (map[string]keptFile, error) {
	found := reportFiles(tree, pattern)
	if len(found) == 0 {
		return nil, nil
	}
	rels := make([]string, len(found))
	for i, f := range found {
		rel, err := filepath.Rel(tree, f)
		if err != nil {
			return nil, err
		}
		rels[i] = filepath.ToSlash(rel)
	}
	tracked := map[string]bool{}
	for start := 0; start < len(rels); start += 500 { // the files found alone, a command line short enough
		args := append([]string{"--literal-pathspecs", "ls-files", "-z", "--"}, rels[start:min(start+500, len(rels))]...)
		out, err := git(tree, args...) // paths from tree, as the run's tree is a checkout's root
		if err != nil {
			return nil, fmt.Errorf("the files git tracks, kept from the stale reports' removal, could not be listed: %v", err)
		}
		for _, f := range strings.Split(out, "\x00") {
			tracked[f] = f != ""
		}
	}
	// Removed through a root that refuses any path leaving tree: a folder
	// linked outside, committed or made by the build, leads nowhere.
	root, err := os.OpenRoot(tree)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	kept := map[string]keptFile{}
	for i, f := range found {
		rel := filepath.FromSlash(rels[i])
		if _, err := root.Lstat(rel); err != nil {
			continue // outside tree through a link: not the run's to remove, nor to read
		}
		if tracked[rels[i]] || inNestedRepo(tree, rel) {
			info, err := os.Stat(f)
			if err != nil {
				return nil, err
			}
			data, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			kept[f] = keptFile{mod: info.ModTime(), data: data}
			continue
		}
		if err := root.Remove(rel); err != nil {
			return nil, err
		}
	}
	return kept, nil
}

// freshReports are the files test-report names in tree after the run,
// but one reached through a link leaving tree, and a kept file the run
// left as it was, or rewrote into no JUnit report (a build bumping
// `pom.xml`'s version).
func freshReports(tree, pattern string, kept map[string]keptFile) []string {
	root, err := os.OpenRoot(tree)
	if err != nil {
		return nil
	}
	defer func() { _ = root.Close() }()
	var out []string
	for _, f := range reportFiles(tree, pattern) {
		rel, err := filepath.Rel(tree, f)
		if err != nil {
			continue
		}
		if _, err := root.Lstat(rel); err != nil {
			continue // outside tree through a link: not this run's report
		}
		if k, ok := kept[f]; ok {
			info, err := os.Stat(f)
			data, rerr := os.ReadFile(f)
			if err != nil || rerr != nil || !bytes.Contains(data, []byte("<testsuite")) ||
				(info.ModTime().Equal(k.mod) && bytes.Equal(data, k.data)) {
				continue
			}
		}
		out = append(out, f)
	}
	return out
}
