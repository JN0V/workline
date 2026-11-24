package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// roleBumpTemplate is a CI template whose version line a release bumps.
const roleBumpTemplate = "defaults/ci/github/workline-develop.yml"

// roleBumpVersions returns the version line roleBumpTemplate holds now, and
// the line a release would put in its place: the tests must not hang on the
// version a release just bumped.
func roleBumpVersions(t *testing.T) (now, next string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(roleBumpTemplate)))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`WORKLINE_VERSION: (v[0-9][^\s]*)`).FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("%s holds no WORKLINE_VERSION line", roleBumpTemplate)
	}
	next = "v99.0.0"
	if m[1] == next {
		next = "v99.0.1"
	}
	return m[0], "WORKLINE_VERSION: " + next
}

// roleBumpClone clones this repository's history, brings the working tree's
// own changes into it, then gives each of pages a header whose `sources`
// name roleBumpTemplate and whose `checked` is the commit it stands on, with
// pinned's body saying the version the template holds as the version now. It
// returns the clone, and a function that replaces old by new in a file of it
// in a commit of its own.
func roleBumpClone(t *testing.T, pages []string, pinned string) (clone string, commit func(file, old, new string)) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	run := func(dir, name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	top := run(".", "git", "rev-parse", "--show-toplevel")
	clone = filepath.Join(t.TempDir(), "repo")
	run(".", "git", "clone", "-q", top, clone)
	write := func(file, text string, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(clone, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), mode); err != nil {
			t.Fatal(err)
		}
	}
	// The work not committed yet (the test file itself, maybe) is what the
	// clone must run.
	mine := run(top, "git", "-c", "core.quotePath=off", "diff", "--name-only", "HEAD") + "\n" +
		run(top, "git", "-c", "core.quotePath=off", "ls-files", "-o", "--exclude-standard")
	for _, name := range strings.Split(mine, "\n") {
		if name == "" || strings.HasPrefix(name, ".workline-runs/") {
			continue
		}
		src := filepath.Join(top, filepath.FromSlash(name))
		data, err := os.ReadFile(src)
		if err != nil {
			if err := os.Remove(filepath.Join(clone, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			continue
		}
		info, err := os.Stat(src)
		if err != nil {
			t.Fatal(err)
		}
		write(name, string(data), info.Mode().Perm())
	}
	run(clone, "git", "add", "-A")
	run(clone, "git", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "the work in progress")
	checked := run(clone, "git", "rev-parse", "HEAD")
	for _, page := range pages {
		data, err := os.ReadFile(filepath.Join(clone, filepath.FromSlash(page)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.HasPrefix(text, "---\n") {
			t.Fatalf("%s: no front matter to rewrite", page)
		}
		_, body, _ := strings.Cut(text[4:], "\n---\n")
		if page == pinned {
			now, _ := roleBumpVersions(t)
			body += "\n```yaml\n" + now + "\n```\n"
		}
		write(page, "---\nsources:\n  - "+roleBumpTemplate+"\nchecked: "+checked+"\nverified: human:Tester\n---\n"+body, 0o644)
	}
	run(clone, "git", "add", "-A")
	run(clone, "git", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "the pages read against the template")
	commit = func(file, old, new string) {
		t.Helper()
		path := filepath.Join(clone, filepath.FromSlash(file))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), old) {
			t.Fatalf("%s has no %q to replace", file, old)
		}
		write(file, strings.Replace(string(data), old, new, 1), 0o644)
		run(clone, "git", "add", file)
		run(clone, "git", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "change "+file)
	}
	return clone, commit
}

// roleBumpRun runs the "Checked moved" test of the role pages, as the
// weekly gardening run does, in clone, and returns what it printed: a page
// it counts as older than its sources is named in it.
func roleBumpRun(t *testing.T, clone string) string {
	t.Helper()
	cmd := exec.Command("go", "test", "-v", "-count=1", "-run", "^TestRolePagesCheckedMoved$", "./cmd/workline")
	cmd.Dir = clone
	cmd.Env = append(os.Environ(), "WORKLINE_DOCS_TESTS=1")
	out, _ := cmd.CombinedOutput() // a failure is not the question: the pages it names are
	text := string(out)
	if !strings.Contains(text, "--- PASS: TestRolePagesCheckedMoved") && !strings.Contains(text, "--- FAIL: TestRolePagesCheckedMoved") {
		t.Fatalf("the Checked moved test did not run in the clone:\n%s", text)
	}
	return text
}

// A release bump leaves a page checked, a real change still counts
//
// Given a page whose `sources` name a CI template, and whose `checked` is
// before a release,
// When the release only changes that template's `x-release-please-version`
// line, and then a later commit changes another line of the template,
// Then after the bump the page counts as checked against the head, and
// after the other line it counts as older than its sources.
func TestRolePagesCheckedMovedBumpThenRealChange(t *testing.T) {
	const page = "roles/developer/README.md"
	clone, commit := roleBumpClone(t, []string{page}, "")
	now, next := roleBumpVersions(t)
	commit(roleBumpTemplate, now, next)
	if out := roleBumpRun(t, clone); strings.Contains(out, page) {
		t.Errorf("%s counts as older than its sources after a release only bumped the template's version:\n%s", page, out)
	}
	const runs = "WORKLINE_RUNS_DIR: ${{ github.workspace }}/.workline-runs"
	commit(roleBumpTemplate, runs, runs+"-kept")
	if out := roleBumpRun(t, clone); !strings.Contains(out, page) {
		t.Errorf("%s does not count as older than its sources after a commit changed another line of the template:\n%s", page, out)
	}
}

// A pin the page states still counts
//
// Given a page that says the version the release replaced, as the version
// now, beside a page that says none,
// When the release bumps that version in their source,
// Then the page that pins it counts as older than its sources, the other
// does not.
func TestRolePagesCheckedMovedPinnedVersion(t *testing.T) {
	const pinned, other = "roles/developer/README.md", "roles/tester/README.md"
	clone, commit := roleBumpClone(t, []string{pinned, other}, pinned)
	now, next := roleBumpVersions(t)
	commit(roleBumpTemplate, now, next)
	out := roleBumpRun(t, clone)
	if !strings.Contains(out, pinned) {
		t.Errorf("%s says the replaced version as the version now, yet does not count as older than its sources:\n%s", pinned, out)
	}
	if strings.Contains(out, other) {
		t.Errorf("%s says no version, yet counts as older than its sources after a release only bumped the template's version:\n%s", other, out)
	}
}
