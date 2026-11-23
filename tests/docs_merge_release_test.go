package tests

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/docmark"
	"go.yaml.in/yaml/v3"
)

// mergeReleasePages are the seventeen user pages on the merge policy, the
// merge queue, release holds, the weekly sample and the release.
var mergeReleasePages = []string{
	"merge-policy", "merge-queue", "release-holds", "weekly-sample",
	"merge-policy-code", "merge-policy-no-bypass", "merge-policy-normal",
	"merge-policy-reverts", "merge-policy-routine", "merge-queue-commands",
	"release", "release-candidate",
	"merge-policy-holds", "merge-policy-one-head", "merge-policy-verdict",
	"merge-policy-setup", "merge-policy-summary",
}

// suspectLine is how `workline docs` lists a doc made suspect.
var suspectLine = regexp.MustCompile(`(?m)^\s*suspect (docs/[^:\s]+\.md):`)

// mergeReleaseHeader is a page's `sources`, its `checked`, and its body
// (the text after its header).
func mergeReleaseHeader(t *testing.T, page string) (sources []string, checked, body string) {
	t.Helper()
	text := readText(t, "../docs/"+page+".md")
	meta, n := docmark.Header(text)
	var h struct {
		Sources []string  `yaml:"sources"`
		Checked yaml.Node `yaml:"checked"`
	}
	if err := yaml.Unmarshal([]byte(meta), &h); err != nil {
		t.Fatalf("docs/%s.md: header: %v", page, err)
	}
	return h.Sources, h.Checked.Value, strings.Join(strings.Split(text, "\n")[n:], "\n")
}

// mergeReleaseWorkspace builds the engine from this tree and clones this
// repository, git isolated from the machine. With move, the seventeen pages'
// `checked` name the clone's HEAD, in one commit on top of it. It returns
// the binary, the clone, that HEAD, a git runner in the clone and a
// runner of `workline docs --ai none --since <rev>` in the clone.
func mergeReleaseWorkspace(t *testing.T, move bool) (bin, dir, base string, git func(args ...string) string, docs func(since string) string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(t.TempDir(), "workline")
	build := exec.Command("go", "build", "-o", bin, "./cmd/workline")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	dir = filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "clone", "-q", root, dir).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	git = func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	docs = func(since string) string {
		t.Helper()
		cmd := exec.Command(bin, "docs", "--ai", "none", "--since", since)
		cmd.Dir = dir
		out, _ := cmd.CombinedOutput()
		return string(out)
	}
	base = git("rev-parse", "HEAD")
	if move {
		key := regexp.MustCompile(`(?m)^checked:.*$`)
		for _, p := range mergeReleasePages {
			file := filepath.Join(dir, "docs", p+".md")
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			loc := key.FindIndex(data)
			if loc == nil {
				t.Fatalf("docs/%s.md has no `checked`", p)
			}
			moved := string(data[:loc[0]]) + "checked: " + base + string(data[loc[1]:])
			if err := os.WriteFile(file, []byte(moved), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		git("add", "docs")
		git("commit", "-q", "-m", "docs: move checked")
	}
	return bin, dir, base, git, docs
}

func suspectPages(out string) map[string]bool {
	found := map[string]bool{}
	for _, m := range suspectLine.FindAllStringSubmatch(out, -1) {
		found[m[1]] = true
	}
	return found
}

// Scenario: Twelve pages not suspect
// Given the seventeen pages, each with the commit its `checked` names,
// When `workline docs --ai none` runs on the commits since the oldest of them,
// Then none of the seventeen is listed as suspect.
func TestMergeReleasePagesNotSuspect(t *testing.T) {
	docsTestsOnly(t)
	_, _, _, git, docs := mergeReleaseWorkspace(t, false)
	shas := []string{"merge-base", "--octopus"}
	for _, p := range mergeReleasePages {
		_, checked, _ := mergeReleaseHeader(t, p)
		shas = append(shas, git("rev-parse", "--verify", checked+"^{commit}"))
	}
	out := docs(git(shas...))
	listed := suspectPages(out)
	for _, p := range mergeReleasePages {
		if listed["docs/"+p+".md"] {
			t.Errorf("docs/%s.md is listed as suspect", p)
		}
	}
	if t.Failed() {
		t.Logf("workline docs said:\n%s", out)
	}
}

// Scenario: Narrow sources, related change
// Given docs/merge-queue.md checked at a commit, When the next commit changes
// internal/line/seconds.go#Measure, Then the page is listed as suspect.
func TestNarrowSourcesRelatedChange(t *testing.T) {
	_, dir, base, git, docs := mergeReleaseWorkspace(t, true)
	file := filepath.Join(dir, "internal", "line", "seconds.go")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), "\tr.Moment = m\n}", "\tr.Moment = m\n\t_ = m.Budget\n}", 1)
	if changed == string(data) {
		t.Fatal("Measure does not end as the test expects")
	}
	if err := os.WriteFile(file, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	git("commit", "-q", "-am", "line: change Measure")
	out := docs(base)
	if !suspectPages(out)["docs/merge-queue.md"] {
		t.Errorf("docs/merge-queue.md is not listed as suspect after Measure changed:\n%s", out)
	}
}

// Scenario: Sources are narrow and named
// Given each of the seventeen pages, When I read its `sources` header, Then every
// entry is a file path or a `file#symbol` (no folder), and its file name or
// symbol name appears in the page's text.
func TestMergeReleaseSourcesNarrowAndNamed(t *testing.T) {
	docsTestsOnly(t)
	for _, p := range mergeReleasePages {
		sources, _, body := mergeReleaseHeader(t, p)
		if len(sources) == 0 {
			t.Errorf("docs/%s.md declares no sources", p)
		}
		for _, s := range sources {
			file, sym, hasSym := strings.Cut(s, "#")
			info, err := os.Stat(filepath.Join("..", filepath.FromSlash(file)))
			if err != nil || info.IsDir() || strings.HasSuffix(file, "/") {
				t.Errorf("docs/%s.md: source %q is no file path (a folder, or missing)", p, s)
				continue
			}
			named := strings.Contains(body, path.Base(file))
			if hasSym && sym != "" {
				last := sym[strings.LastIndex(sym, ".")+1:]
				named = named || strings.Contains(body, sym) || strings.Contains(body, last)
			}
			if !named {
				t.Errorf("docs/%s.md: source %q: neither its file name nor its symbol appears in the page", p, s)
			}
		}
	}
}
