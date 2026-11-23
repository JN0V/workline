package tests

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/docmark"
)

// docsTestsOnly skips a test that reads this repository's docs as they
// stand: the weekly gardening run sets WORKLINE_DOCS_TESTS=1, a pull
// request's run does not.
func docsTestsOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("WORKLINE_DOCS_TESTS") == "" {
		t.Skip("reads the docs as they stand: run weekly with WORKLINE_DOCS_TESTS=1")
	}
}

// checkedGit runs git in dir, git isolated from the machine, and gives its
// output and its error.
func checkedGit(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Scenario: Squash keeps checked
// Given a docs pull request that sets `checked` to a commit on its branch,
// When it is squash-merged,
// Then no page on `main` names a commit `main` lacks.
//
// Scenario: Existing headers
// Given today's `main`,
// When the list of pages with a non-ancestor `checked` is computed,
// Then it is empty.
//
// Every page's `checked` (front matter or comment header, this repository's
// commit) must be a commit that `git merge-base --is-ancestor` finds in the
// history of HEAD, the tip of main.
func TestEveryCheckedIsAnAncestorOfMain(t *testing.T) {
	docsTestsOnly(t)
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var pages int
	var bad []string
	walk := func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == ".git" || rel == "tests/conformance/fixtures" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		doc, err := docmark.ParseDoc(rel, data)
		if err != nil || doc == nil || doc.Checked[""] == "" {
			return nil
		}
		pages++
		checked := doc.Checked[""]
		sha, err := checkedGit(t, root, "rev-parse", "--verify", "-q", checked+"^{commit}")
		if err != nil {
			bad = append(bad, rel+": checked "+checked+" is no commit of this clone")
			return nil
		}
		if _, err := checkedGit(t, root, "merge-base", "--is-ancestor", sha, "HEAD"); err != nil {
			bad = append(bad, rel+": checked "+checked+" is not an ancestor of main")
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatal(err)
	}
	if pages == 0 {
		t.Fatal("no page with a `checked` was found")
	}
	if len(bad) > 0 {
		t.Errorf("%d of %d pages name a commit main lacks:\n%s", len(bad), pages, strings.Join(bad, "\n"))
	}
}

// Scenario: Check refuses
// Given a pull request whose page has a `checked` that is not an ancestor of the base,
// When the docs check runs,
// Then it fails and names the page and the commit.
func TestDocsCheckRefusesCheckedNotAnAncestor(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "workline")
	build := exec.Command("go", "build", "-o", bin, "./cmd/workline")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := checkedGit(t, dir, args...)
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return out
	}
	write := func(rel, text string) {
		t.Helper()
		file := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// main holds the source; a side branch holds a commit main never gets.
	git("init", "-q", "-b", "main")
	write("src/a.go", "package a\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")
	git("checkout", "-q", "-b", "side")
	write("other.txt", "elsewhere\n")
	git("add", "-A")
	git("commit", "-q", "-m", "only on the side")
	side := git("rev-parse", "HEAD")
	git("checkout", "-q", "main")
	// The pull request: a page whose `checked` names the side commit.
	write("docs/page.md", "---\nsources: [src/a.go]\nchecked: "+side+"\n---\n# Page\n\nSome words.\n")
	git("add", "-A")
	git("commit", "-q", "-m", "docs: add a page")

	cmd := exec.Command(bin, "run-role", "documentalist", "--event", "merge-request", "--repo", dir,
		"--ai", "none", "--forge", "none", "--no-apply", "--input", "range="+base+"..HEAD")
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err == nil {
		t.Errorf("the docs check passed on a `checked` that is not an ancestor of the base:\n%s", out)
	} else if !errors.As(err, &exit) {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "docs/page.md") {
		t.Errorf("the docs check does not name the page docs/page.md:\n%s", out)
	}
	if !strings.Contains(string(out), side[:7]) {
		t.Errorf("the docs check does not name the commit %s:\n%s", side[:7], out)
	}
}
