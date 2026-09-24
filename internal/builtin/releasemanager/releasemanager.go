// Package releasemanager holds the deterministic part of the release manager
// role (roles/release-manager): the next version and the changelog, computed
// from conventional commits. The AI only writes the summary for users.
package releasemanager

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// Settings are the role's settings, as merged by the engine.
type Settings struct {
	Flow       string `json:"flow"`
	TagPrefix  string `json:"tag-prefix"`
	Changelog  string `json:"changelog"`
	Versioning struct {
		Scheme       string `json:"scheme"`
		CalverFormat string `json:"calver-format"`
	} `json:"versioning"`
	ChangelogFormat string           `json:"changelog-format"` // "" or keep-a-changelog
	Packages        *PackageSettings `json:"packages"`
	Root            RootSettings     `json:"root"`
}

// Commit is one parsed conventional commit.
type Commit struct {
	Type, Scope, Summary string
	Breaking             bool
}

var header = regexp.MustCompile(`^(\w+)(?:\(([^()]+)\))?(!)?: (.+)$`)

// Parse reads a commit's subject and body. Messages that are not conventional
// are returned with an empty type: they appear nowhere and bump nothing.
func Parse(subject, body string) Commit {
	m := header.FindStringSubmatch(subject)
	if m == nil {
		return Commit{Summary: subject}
	}
	return Commit{Type: m[1], Scope: m[2], Summary: m[4],
		Breaking: m[3] == "!" || strings.Contains(body, "BREAKING CHANGE:") || strings.Contains(body, "BREAKING-CHANGE:")}
}

// Bump is the semver part a set of commits raises: "major", "minor", "patch" or "".
func Bump(commits []Commit) string {
	bump := ""
	for _, c := range commits {
		switch {
		case c.Breaking:
			return "major"
		case c.Type == "feat":
			bump = "minor"
		case (c.Type == "fix" || c.Type == "perf") && bump == "":
			bump = "patch"
		}
	}
	return bump
}

// NextSemver raises a version like "1.2.0" by bump.
func NextSemver(current, bump string) (string, error) {
	p := strings.Split(current, ".")
	if len(p) != 3 {
		return "", fmt.Errorf("%q is not MAJOR.MINOR.PATCH", current)
	}
	n := make([]int, 3)
	for i := range p {
		v, err := strconv.Atoi(p[i])
		if err != nil {
			return "", fmt.Errorf("%q is not MAJOR.MINOR.PATCH", current)
		}
		n[i] = v
	}
	switch bump {
	case "major":
		n = []int{n[0] + 1, 0, 0}
	case "minor":
		n = []int{n[0], n[1] + 1, 0}
	case "patch":
		n[2]++
	}
	return fmt.Sprintf("%d.%d.%d", n[0], n[1], n[2]), nil
}

// NextCalver gives YYYY.MM.MICRO for date, MICRO counting from 0 within the month.
func NextCalver(current string, date time.Time) string {
	prefix := fmt.Sprintf("%d.%02d.", date.Year(), int(date.Month()))
	if micro, ok := strings.CutPrefix(current, prefix); ok {
		if n, err := strconv.Atoi(micro); err == nil {
			return prefix + strconv.Itoa(n+1)
		}
	}
	return prefix + "0"
}

var sections = []struct{ title, kind string }{
	{"Breaking changes", "breaking"}, {"Features", "feat"}, {"Fixes", "fix"}, {"Performance", "perf"},
}

// heading is the first line of a changelog section, in the project's format.
func heading(s Settings, tag, version string, date time.Time) string {
	if s.ChangelogFormat == "keep-a-changelog" {
		return fmt.Sprintf("## [%s] - %s", version, date.Format("2006-01-02"))
	}
	return fmt.Sprintf("## %s — %s", tag, date.Format("2006-01-02"))
}

// Section renders a changelog section under the given heading line.
func Section(head string, commits []Commit) string {
	var b strings.Builder
	b.WriteString(head + "\n")
	for _, s := range sections {
		var lines []string
		for _, c := range commits {
			if (s.kind == "breaking" && c.Breaking) || (s.kind != "breaking" && !c.Breaking && c.Type == s.kind) {
				line := "- " + c.Summary
				if c.Scope != "" {
					line = fmt.Sprintf("- **%s:** %s", c.Scope, c.Summary)
				}
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			fmt.Fprintf(&b, "\n### %s\n\n%s\n", s.title, strings.Join(lines, "\n"))
		}
	}
	return b.String()
}

// Insert puts a section at the top of a changelog, under its title.
func Insert(changelog, section string) string {
	if strings.TrimSpace(changelog) == "" {
		return "# Changelog\n\n" + section
	}
	if i := strings.Index(changelog, "\n## "); i >= 0 {
		return changelog[:i+1] + section + "\n" + changelog[i+1:]
	}
	return strings.TrimRight(changelog, "\n") + "\n\n" + section
}

// Pre works out the next version and its changelog. With nothing to release,
// it exits 10. It proposes the release itself as the fallback, and asks the
// agent only for a summary users can read.
func Pre(runDir, repo string) int {
	var s Settings
	if err := readJSON(filepath.Join(runDir, "in", "settings.json"), &s); err != nil {
		return fail(err)
	}
	last, _ := git(repo, "describe", "--tags", "--abbrev=0", "--match", s.TagPrefix+"*")
	rangeArg := "HEAD"
	if last != "" {
		rangeArg = last + "..HEAD"
	}
	log, err := git(repo, "log", "--format=%x1e%s%x1f%b%x1f", "--name-only", rangeArg)
	if err != nil {
		return fail(err)
	}
	var detailed []commit
	var commits []Commit
	for _, rec := range strings.Split(log, "\x1e") {
		parts := strings.SplitN(rec, "\x1f", 3)
		if len(parts) < 3 {
			continue
		}
		c := commit{Commit: Parse(strings.TrimSpace(parts[0]), parts[1])}
		for _, f := range strings.Split(strings.TrimSpace(parts[2]), "\n") {
			if f != "" {
				c.files = append(c.files, f)
			}
		}
		detailed = append(detailed, c)
		commits = append(commits, c.Commit)
	}
	dateText, err := git(repo, "log", "-1", "--format=%cI")
	if err != nil {
		return fail(err)
	}
	date, _ := time.Parse(time.RFC3339, dateText)
	if s.Packages != nil {
		return prePackages(runDir, repo, s, last, detailed, date)
	}
	bump := Bump(commits)
	if bump == "" {
		return 10 // nothing a user would notice since the last version
	}
	current := strings.TrimPrefix(last, s.TagPrefix)
	var next string
	switch s.Versioning.Scheme {
	case "calver":
		next = NextCalver(current, date)
	case "semver", "":
		if current == "" {
			current = "0.0.0"
		}
		if next, err = NextSemver(current, bump); err != nil {
			return fail(err)
		}
	default:
		return fail(fmt.Errorf("unknown versioning scheme %q", s.Versioning.Scheme))
	}
	version := s.TagPrefix + next
	return finish(runDir, repo, s, version, Section(heading(s, version, next, date), commits), nil)
}

// finish adds the changelog and the release to the fallback proposals, and
// asks the agent for the notes.
func finish(runDir, repo string, s Settings, version, section string, fallback []intent.Intention) int {
	old, _ := os.ReadFile(filepath.Join(repo, s.Changelog))
	fallback = append(fallback,
		intent.Intention{Kind: "patch", Value: map[string]any{"file": s.Changelog, "content": Insert(string(old), section)}},
		intent.Intention{Kind: "release", Value: map[string]any{"version": version, "notes": section}},
	)
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "version"), []byte(version), 0o644); err != nil {
		return fail(err)
	}
	task := fmt.Sprintf("The next version is %s. Its changelog, generated from the commits:\n\n%s\n"+
		"Write the release notes: a short summary for users on top of this changelog, which stays below it unchanged.\n", version, section)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// Post checks what the AI must not decide: the version.
func Post(runDir string) int {
	want, err := os.ReadFile(filepath.Join(runDir, "in", "version"))
	if err != nil {
		return fail(err)
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	fallback, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return fail(err)
	}
	for _, in := range intents {
		if in.Kind == "patch" && !samePatch(in, fallback) {
			return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the changelog is generated, not written by hand",
				Findings: []verdict.Finding{{Rule: "changelog-edited", Where: "patch",
					Message: "only the generated changelog entry may be applied; put the summary in the release notes"}}})
		}
	}
	for _, in := range intents {
		if in.Kind != "release" {
			continue
		}
		m, _ := in.Value.(map[string]any)
		if got, _ := m["version"].(string); got != string(want) {
			return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the proposed release has the wrong version",
				Findings: []verdict.Finding{{Rule: "version-mismatch", Where: "release",
					Message: fmt.Sprintf("the commits give %s; %q was proposed", want, got)}}})
		}
		if notes, _ := m["notes"].(string); strings.TrimSpace(notes) == "" {
			return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the release has no notes",
				Findings: []verdict.Finding{{Rule: "empty-notes", Where: "release"}}})
		}
		return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "release " + string(want)})
	}
	return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no release was proposed",
		Findings: []verdict.Finding{{Rule: "no-release"}}})
}

func git(repo string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	if v.Status == verdict.Pass {
		return 0
	}
	return 1
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "release-manager:", err)
	return 99
}

// samePatch reports whether a proposed patch is one pre generated.
func samePatch(p intent.Intention, fallback []intent.Intention) bool {
	for _, f := range fallback {
		if f.Kind == "patch" && fmt.Sprint(f.Value) == fmt.Sprint(p.Value) {
			return true
		}
	}
	return false
}
