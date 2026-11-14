package packs

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// copyRepo copies the repository's files into dst, leaving out its git
// data, the agents' worktrees and the runs' folders.
func copyRepo(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".claude" || d.Name() == ".workline-runs" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Worktrees ignored: given a copy of the repository with a
// .claude/worktrees/x holding a Go file that imports a pack, when
// `go test ./internal/packs/` runs, then it passes.
func TestWorktreesAreNotRead(t *testing.T) {
	copyDir := t.TempDir()
	copyRepo(t, filepath.Join("..", ".."), copyDir)
	wt := filepath.Join(copyDir, ".claude", "worktrees", "x")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package x\n\nimport _ \"github.com/JN0V/workline/internal/packs\"\n"
	if err := os.WriteFile(filepath.Join(wt, "x.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./internal/packs/", "-run", "TestNoVerdictReadsAPack", "-count=1")
	cmd.Dir = copyDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./internal/packs/ in a copy with a worktree: %v\n%s", err, out)
	}
}
