package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// rolePageList are the role pages the lot reads: the four READMEs, the pages
// of the reviewer's and the product owner's docs folders, but for the
// status and the archive of tries, which carry no claim about the code.
var rolePageList = func() []string {
	var pages []string
	for _, r := range []string{"developer", "inspector", "tester", "reviewer"} {
		pages = append(pages, "roles/"+r+"/README.md")
	}
	for _, dir := range []string{"roles/reviewer/docs", "roles/product-owner/docs"} {
		files, _ := filepath.Glob(filepath.Join(userRepo, dir, "*.md"))
		sort.Strings(files)
		for _, f := range files {
			if b := filepath.Base(f); b != "status.md" && b != "tried.md" {
				pages = append(pages, dir+"/"+b)
			}
		}
	}
	return pages
}()

// roleFront reads a page: the scalar fields of its front matter (none when
// it has no front matter) and its whole text.
func roleFront(t *testing.T, page string) (fields map[string]string, text string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(userRepo, page))
	if err != nil {
		t.Fatalf("%s: %v", page, err)
	}
	text = string(data)
	fields = map[string]string{}
	if !strings.HasPrefix(text, "---\n") {
		return fields, text
	}
	head, _, _ := strings.Cut(text[4:], "\n---\n")
	for _, l := range strings.Split(head, "\n") {
		if l == "" || l[0] == ' ' || l[0] == '-' {
			continue
		}
		if k, v, ok := strings.Cut(l, ":"); ok {
			fields[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return fields, text
}

// roleDocsRun runs `workline docs --ai none` on a clone of this repository's
// commit, over the whole history, after a commit that rewrites file with
// mutate, and returns what it said.
func roleDocsRun(t *testing.T, file string, mutate func(string) string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Tester")
	t.Setenv("GIT_AUTHOR_EMAIL", "tester@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Tester")
	t.Setenv("GIT_COMMITTER_EMAIL", "tester@example.com")
	for _, k := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI", "WORKLINE_EVENT", "WORKLINE_AI_WAITS"} {
		t.Setenv(k, "")
	}
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
	bin := filepath.Join(t.TempDir(), "workline")
	run(".", "go", "build", "-o", bin, ".")
	top := run(".", "git", "rev-parse", "--show-toplevel")
	clone := filepath.Join(t.TempDir(), "repo")
	run(".", "git", "clone", "-q", top, clone)
	roots := strings.Fields(run(clone, "git", "rev-list", "--max-parents=0", "HEAD"))
	root := roots[len(roots)-1]
	path := filepath.Join(clone, filepath.FromSlash(file))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := mutate(string(data))
	if edited == string(data) {
		t.Fatalf("%s: the edit changed nothing", file)
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	run(clone, "git", "-c", "commit.gpgsign=false", "commit", "-q", "-am", "touch one function")
	cmd := exec.Command(bin, "docs", "--ai", "none", "--since", root)
	cmd.Dir = clone
	out, _ := cmd.CombinedOutput() // a non-zero exit is not the question: the pages listed are
	return string(out)
}

// Lot not suspect
//
// Given the commit of this run,
// When `workline docs --ai none` runs on it,
// Then none of the role READMEs and pages of the lot is listed as suspect.
func TestRolePagesNoneSuspect(t *testing.T) {
	out := userPageDocsRun(t, false)
	for _, m := range userSuspect.FindAllStringSubmatch(out, -1) {
		if slices.Contains(rolePageList, m[1]) {
			t.Errorf("%s is listed as suspect:\n%s", m[1], m[0])
		}
	}
}

// Sources not emptied
//
// Given each page of the lot,
// When its front matter is read,
// Then `sources` lists at least one file or symbol, and each entry exists
// in the repository.
func TestRolePagesSourcesNotEmptied(t *testing.T) {
	for _, page := range rolePageList {
		sources, has, _ := userPageRead(t, page)
		if !has || len(sources) == 0 {
			t.Errorf("%s: `sources` is missing or empty", page)
			continue
		}
		for _, src := range sources {
			path, sym, hasSym := strings.Cut(src, "#")
			info, err := os.Stat(filepath.Join(userRepo, path))
			if err != nil {
				t.Errorf("%s: source %q: %s is not in the repository", page, src, path)
				continue
			}
			if !hasSym || info.IsDir() {
				continue
			}
			parts := strings.Split(sym, ".")
			name := ""
			for _, p := range parts {
				if p != "" {
					name = p
				}
			}
			data, _ := os.ReadFile(filepath.Join(userRepo, path))
			if name == "" || !strings.Contains(string(data), name) {
				t.Errorf("%s: source %q: %q is not in %s", page, src, name, path)
			}
		}
	}
}

// Checked moved
//
// Given each page of the lot,
// When its front matter is read after the work,
// Then `checked` is a commit of this history, and no commit after it
// changed a file the page names in `sources`: the page was read against
// the code at the head of the pull request, not an older commit.
func TestRolePagesCheckedMoved(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", userRepo}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	for _, page := range rolePageList {
		fields, _ := roleFront(t, page)
		checked := fields["checked"]
		if checked == "" {
			t.Errorf("%s: no `checked` in its front matter", page)
			continue
		}
		if _, err := git("rev-parse", "--verify", "-q", checked+"^{commit}"); err != nil {
			t.Errorf("%s: `checked: %s` is no commit of this repository", page, checked)
			continue
		}
		if _, err := git("merge-base", "--is-ancestor", checked, "HEAD"); err != nil {
			t.Errorf("%s: `checked: %s` is not an ancestor of the head", page, checked)
			continue
		}
		sources, _, _ := userPageRead(t, page)
		args := []string{"log", "--format=%h", checked + "..HEAD", "--"}
		for _, src := range sources {
			path, _, _ := strings.Cut(src, "#")
			args = append(args, path)
		}
		if len(sources) == 0 {
			continue
		}
		if out, _ := git(args...); out != "" {
			t.Errorf("%s: `checked: %s` is older than a change to its sources (%s)", page, checked, strings.ReplaceAll(out, "\n", " "))
		}
	}
}

// Verified recorded
//
// Given each page of the lot,
// When its front matter is read after the work,
// Then `verified` is set and names who read the page. (That the pull
// request description lists that reader is for a person to see.)
func TestRolePagesVerifiedRecorded(t *testing.T) {
	for _, page := range rolePageList {
		fields, _ := roleFront(t, page)
		if strings.TrimSpace(fields["verified"]) == "" {
			t.Errorf("%s: `verified` is not set in its front matter", page)
		}
	}
}

// Narrow sources
//
// Given the developer's README, whose `sources` name symbols of the
// developer's package,
// When a commit changes another function of that same package and
// `workline docs --ai none` runs,
// Then the README is not listed as suspect.
func TestRolePagesNarrowSources(t *testing.T) {
	const page = "roles/developer/README.md"
	const pkg = "roles/developer/code"
	sources, _, _ := userPageRead(t, page)
	symbols := map[string]bool{}
	var whole, order []string
	for _, src := range sources {
		path, sym, has := strings.Cut(src, "#")
		if !has {
			whole = append(whole, strings.TrimSuffix(path, "/"))
			continue
		}
		for _, p := range strings.Split(sym, ".") {
			symbols[p] = true
		}
		if strings.HasSuffix(path, ".go") && !slices.Contains(order, path) {
			order = append(order, path)
		}
	}
	files, _ := filepath.Glob(filepath.Join(userRepo, pkg, "*.go"))
	sort.Strings(files)
	for _, f := range files {
		rel := pkg + "/" + filepath.Base(f)
		if !strings.HasSuffix(rel, "_test.go") && !slices.Contains(order, rel) {
			order = append(order, rel)
		}
	}
	covered := func(file string) bool {
		for _, w := range whole {
			if file == w || strings.HasPrefix(file, w+"/") {
				return true
			}
		}
		return false
	}
	funcRe := regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?([A-Za-z_]\w*)\([^\n]*\{\n`)
	file, sig := "", ""
pick:
	for _, f := range order {
		if covered(f) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(userRepo, f))
		if err != nil {
			continue
		}
		for _, m := range funcRe.FindAllStringSubmatch(string(data), -1) {
			if !symbols[m[1]] {
				file, sig = f, m[0]
				break pick
			}
		}
	}
	if file == "" {
		t.Fatalf("%s: every function of %s is named by its sources, a whole file or a folder: they are not narrow", page, pkg)
	}
	out := roleDocsRun(t, file, func(src string) string {
		return strings.Replace(src, sig, sig+"\t_ = 0\n", 1)
	})
	for _, m := range userSuspect.FindAllStringSubmatch(out, -1) {
		if m[1] == page {
			t.Errorf("%s is listed as suspect after a commit that changes a function of %s its sources do not name (%s):\n%s", page, pkg, file, m[0])
		}
	}
}

// No engine internals
//
// Given each page of the lot,
// When the front matter is dropped (awk), the link targets are removed
// (sed) and grep looks for an engine path or a `package.Symbol`,
// Then it prints no line.
//
// A page with no front matter has none to drop: every line is scanned.
func TestRolePagesNoEngineInternals(t *testing.T) {
	target := regexp.MustCompile(`\]\([^)]*\)`)
	internals := regexp.MustCompile(`(internal|roles/[a-z-]+/code)/[a-z]+|\b[a-z]+\.[A-Z][A-Za-z]+\b`)
	for _, page := range rolePageList {
		_, text := roleFront(t, page)
		// n counts the `---` lines passed; the body starts at two. Without
		// front matter the body starts at the first line.
		n := 0
		if !strings.HasPrefix(text, "---\n") {
			n = 2
		}
		for _, l := range strings.Split(text, "\n") {
			if l == "---" && n < 2 {
				n++
				continue
			}
			if n < 2 {
				continue
			}
			if l = target.ReplaceAllString(l, ""); internals.MatchString(l) {
				t.Errorf("%s: names an engine internal: %s", page, strings.TrimSpace(l))
			}
		}
	}
}
