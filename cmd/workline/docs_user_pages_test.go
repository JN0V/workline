package main_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// docsTestsOnly skips a test that reads this repository's docs as they
// stand: the weekly gardening run sets WORKLINE_DOCS_TESTS=1, a pull
// request's run does not (ADR-0073).
func docsTestsOnly(t *testing.T) {
	t.Helper()
	if os.Getenv("WORKLINE_DOCS_TESTS") == "" {
		t.Skip("reads the docs as they stand: run weekly with WORKLINE_DOCS_TESTS=1 (ADR-0073)")
	}
}

// userPages are the seventeen user pages on the commands and CI.
var userPages = []string{
	"docs/commands.md",
	"docs/commands-setup.md",
	"docs/commands-work.md",
	"docs/commands-follow.md",
	"docs/commands-reconcile.md",
	"docs/ci.md",
	"docs/ci-product-owner.md",
	"docs/ci-github.md",
	"docs/ci-gitlab.md",
	"docs/gitlab-trigger.md",
	"docs/gitlab/without-template.md",
	"docs/github/heartbeat.md",
	"docs/github/own-workflows.md",
	"docs/ci-forgejo.md",
	"docs/ci-github-app.md",
	"docs/ci-gitlab-self-managed.md",
	"docs/ci-other-forges.md",
}

// userRepo is the repository's root, seen from this package's folder.
const userRepo = "../.."

var (
	userSpan     = regexp.MustCompile("`([^`]+)`")
	userWord     = regexp.MustCompile(`^[a-z][a-z-]*$`)
	userFlagName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	userKey      = regexp.MustCompile(`^([a-z][a-z0-9]*(?:[-.][a-z0-9]+)*):(?:\s.*)?$`)
	userBare     = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)+$`)
	userSuspect  = regexp.MustCompile(`(?m)^\s+suspect (\S+?):`)
	userExternal = map[string]bool{"git": true, "gh": true, "glab": true, "curl": true, "claude": true,
		"sh": true, "jq": true, "go": true, "ssh-keygen": true}
)

// userPageRead reads a page: the entries of its `sources` header (and
// whether it has one), and its prose, without the header nor the fenced
// code blocks.
func userPageRead(t *testing.T, page string) (sources []string, hasSources bool, prose string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(userRepo, page))
	if err != nil {
		t.Fatalf("%s: %v", page, err)
	}
	text, head := string(data), ""
	switch {
	case strings.HasPrefix(text, "---\n"):
		head, text, _ = strings.Cut(text[4:], "\n---\n")
	case strings.HasPrefix(text, "<!-- workline"):
		head, text, _ = strings.Cut(text, "-->")
	}
	lines := strings.Split(head, "\n")
	for i, l := range lines {
		rest, ok := strings.CutPrefix(strings.TrimSpace(l), "sources:")
		if !ok {
			continue
		}
		hasSources = true
		if rest = strings.TrimSpace(rest); rest != "" {
			for _, s := range strings.Split(strings.Trim(rest, "[]"), ",") {
				if s = strings.TrimSpace(s); s != "" {
					sources = append(sources, s)
				}
			}
			break
		}
		for _, n := range lines[i+1:] {
			item, ok := strings.CutPrefix(strings.TrimSpace(n), "- ")
			if !ok {
				break
			}
			sources = append(sources, strings.Trim(strings.TrimSpace(item), `"'`))
		}
		break
	}
	var b strings.Builder
	fenced := false
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		if !fenced {
			b.WriteString(l + "\n")
		}
	}
	return sources, hasSources, b.String()
}

// userPageDocsRun runs `workline docs --ai none` on a clone of this
// repository's commit, over the whole history, and returns what it said.
// With edit, a commit that changes only internal/doctor/doctor.go (the
// body of its `docs` function) is added to the clone first.
func userPageDocsRun(t *testing.T, edit bool) string {
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
	if edit {
		file := filepath.Join(clone, "internal", "doctor", "doctor.go")
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		sig := "func docs(r *Report, o Options, cfg *role.ProjectConfig) {\n"
		if !strings.Contains(string(data), sig) {
			t.Fatalf("internal/doctor/doctor.go has no %q to edit", sig)
		}
		edited := strings.Replace(string(data), sig, sig+"\t_ = o.Repo\n", 1)
		if err := os.WriteFile(file, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		run(clone, "git", "-c", "commit.gpgsign=false", "commit", "-q", "-am", "doctor: touch the docs check")
	}
	cmd := exec.Command(bin, "docs", "--ai", "none", "--since", root)
	cmd.Dir = clone
	out, _ := cmd.CombinedOutput() // a non-zero exit is not the question: the pages listed are
	return string(out)
}

// Suspect list
//
// Given the commit of this run,
// When `workline docs --ai none` runs on it,
// Then none of the seventeen pages is listed as suspect.
func TestUserPagesNoneSuspect(t *testing.T) {
	docsTestsOnly(t)
	out := userPageDocsRun(t, false)
	for _, m := range userSuspect.FindAllStringSubmatch(out, -1) {
		if slices.Contains(userPages, m[1]) {
			t.Errorf("%s is listed as suspect:\n%s", m[1], m[0])
		}
	}
}

// Narrow sources
//
// Given each of the seventeen pages,
// When each `sources` entry is checked,
// Then it is a file path, or `path#name`: never a directory, never a path
// holding a glob character (`*`, `?`, `[`).
func TestUserPagesSourcesAreNarrow(t *testing.T) {
	docsTestsOnly(t)
	for _, page := range userPages {
		sources, has, _ := userPageRead(t, page)
		if !has {
			t.Errorf("%s: no `sources` header", page)
		}
		for _, src := range sources {
			path, _, _ := strings.Cut(src, "#")
			if path == "" || strings.ContainsAny(path, "*?[") {
				t.Errorf("%s: source %q is empty or holds a glob character", page, src)
				continue
			}
			if info, err := os.Stat(filepath.Join(userRepo, path)); err == nil && info.IsDir() {
				t.Errorf("%s: source %q is a directory: name the files", page, src)
			}
		}
	}
}

// Outside commit
//
// Given a commit that edits only `internal/doctor/doctor.go`, which no page
// names in its `sources`,
// When `workline docs --ai none` runs on it,
// Then none of the seventeen pages is listed as suspect.
func TestUserPagesNoneSuspectAfterAnOutsideCommit(t *testing.T) {
	docsTestsOnly(t)
	out := userPageDocsRun(t, true)
	for _, m := range userSuspect.FindAllStringSubmatch(out, -1) {
		if slices.Contains(userPages, m[1]) {
			t.Errorf("%s is listed as suspect after a commit that edits only internal/doctor/doctor.go:\n%s", m[1], m[0])
		}
	}
}

// Names exist in code
//
// Given every command, flag, config key and setting named in backticks in a
// page,
// When each is searched for in `cmd/`, `internal/`, `roles/*/code/`,
// `defaults/` and `.workline/config.yaml`,
// Then each is found, and the pages name no hidden command (`hook`,
// `builtin`).
func TestUserPagesNameOnlyWhatTheCodeHolds(t *testing.T) {
	docsTestsOnly(t)
	var corpus strings.Builder
	add := func(root string) {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, "_test.go") {
				return nil
			}
			if info, err := d.Info(); err != nil || info.Size() > 2<<20 {
				return nil
			}
			if data, err := os.ReadFile(p); err == nil {
				corpus.Write(data)
				corpus.WriteByte('\n')
			}
			return nil
		})
	}
	add(filepath.Join(userRepo, "cmd"))
	add(filepath.Join(userRepo, "internal"))
	add(filepath.Join(userRepo, "defaults"))
	codes, _ := filepath.Glob(filepath.Join(userRepo, "roles", "*", "code"))
	for _, c := range codes {
		add(c)
	}
	add(filepath.Join(userRepo, ".workline", "config.yaml"))
	code := corpus.String()

	table, err := os.ReadFile(filepath.Join(userRepo, "cmd", "workline", "cli.go"))
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{} // command -> hidden
	for _, m := range regexp.MustCompile(`(?m)\{name: "([a-z-]+)",[^\n]*`).FindAllStringSubmatch(string(table), -1) {
		known[m[1]] = strings.Contains(m[0], "hidden: true")
	}

	for _, page := range userPages {
		_, _, prose := userPageRead(t, page)
		for _, m := range userSpan.FindAllStringSubmatch(prose, -1) {
			span := strings.Join(strings.Fields(m[1]), " ")
			fields := strings.Fields(span)
			if len(fields) == 0 {
				continue
			}
			if span == "hook" || span == "builtin" {
				t.Errorf("%s: `%s` is a hidden command, not for a page", page, span)
			}
			if fields[0] == "workline" && len(fields) > 1 && userWord.MatchString(fields[1]) {
				hidden, ok := known[fields[1]]
				switch {
				case ok && hidden:
					t.Errorf("%s: `%s` names the hidden command %s", page, span, fields[1])
				case !ok && (strings.Contains(span, "--") || strings.Contains(span, "<")):
					t.Errorf("%s: `%s` names workline %s, which is no command", page, span, fields[1])
				}
			}
			if !userExternal[fields[0]] {
				for _, f := range fields {
					name, ok := strings.CutPrefix(f, "--")
					if !ok {
						continue
					}
					name, _, _ = strings.Cut(name, "=")
					name = strings.TrimRight(name, ",;.)>")
					if userFlagName.MatchString(name) && !strings.Contains(code, `"`+name+`"`) && !strings.Contains(code, "--"+name) {
						t.Errorf("%s: the flag --%s (in `%s`) is in no code", page, name, span)
					}
				}
			}
			if km := userKey.FindStringSubmatch(span); km != nil {
				for _, part := range strings.Split(km[1], ".") {
					if !strings.Contains(code, part) {
						t.Errorf("%s: the key %q (in `%s`) is in no code", page, part, span)
					}
				}
			} else if userBare.MatchString(span) && !strings.Contains(code, span) {
				t.Errorf("%s: the name `%s` is in no code", page, span)
			}
		}
	}
}
