// Package documentalist holds the deterministic part of the documentalist
// role (roles/documentalist): which docs became suspect because a source
// changed, and which are only pending because the cascade is cut.
package documentalist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// Settings are the role's settings, as merged by the engine.
type Settings struct {
	Docs        []string `json:"docs"`
	Propagation []struct {
		From string `json:"from"`
		To   string `json:"to"`
		When string `json:"when"`
	} `json:"propagation"`
}

// Doc is a documentation file that declares its sources.
type Doc struct {
	Path    string
	Sources []string
	Checked map[string]string // repository name ("" = this one) -> commit
}

// frontmatter is the YAML block at the top of a doc.
type frontmatter struct {
	Sources []string `yaml:"sources"`
	// A node, not a string: read as YAML, a commit like 11180e1 is a number.
	Checked yaml.Node `yaml:"checked"`
}

// ParseDoc reads a doc's frontmatter. A doc without sources is not tracked.
func ParseDoc(path string, content []byte) (*Doc, error) {
	rest, ok := bytes.CutPrefix(content, []byte("---\n"))
	if !ok {
		return nil, nil
	}
	block, _, ok := bytes.Cut(rest, []byte("\n---"))
	if !ok {
		return nil, nil
	}
	var fm frontmatter
	if err := yaml.Unmarshal(block, &fm); err != nil {
		return nil, fmt.Errorf("%s: frontmatter: %w", path, err)
	}
	if len(fm.Sources) == 0 {
		return nil, nil
	}
	d := &Doc{Path: path, Sources: fm.Sources, Checked: map[string]string{}}
	switch c := fm.Checked; c.Kind {
	case yaml.ScalarNode:
		d.Checked[""] = c.Value
	case yaml.MappingNode:
		for i := 0; i+1 < len(c.Content); i += 2 {
			d.Checked[c.Content[i].Value] = c.Content[i+1].Value
		}
	}
	return d, nil
}

// splitSource turns "api:docs/x.md#part" into ("api", "docs/x.md", "part").
func splitSource(s string) (repo, path, anchor string) {
	s, anchor, _ = strings.Cut(s, "#")
	if r, p, ok := strings.Cut(s, ":"); ok && !strings.Contains(r, "/") {
		return r, p, anchor
	}
	return "", s, anchor
}

// body drops a doc's frontmatter: recording who checked a doc is not a change
// of what it says.
func body(content string) string {
	if rest, ok := strings.CutPrefix(content, "---\n"); ok {
		if _, after, ok := strings.Cut(rest, "\n---"); ok {
			return after
		}
	}
	return content
}

// Section returns the text under the heading whose slug is anchor, up to the
// next heading of the same or a higher level. An empty anchor means the body.
func Section(content, anchor string) (string, bool) {
	text := body(content)
	if anchor == "" {
		return strings.TrimSpace(text), true
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		level := len(l) - len(strings.TrimLeft(l, "#"))
		if level == 0 || level > 6 || slug(strings.TrimSpace(l[level:])) != anchor {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if lv := len(lines[j]) - len(strings.TrimLeft(lines[j], "#")); lv > 0 && lv <= level && strings.HasPrefix(lines[j][lv:], " ") {
				end = j
				break
			}
		}
		return strings.TrimSpace(strings.Join(lines[i:end], "\n")), true
	}
	return "", false
}

// slug is the anchor a heading gets on GitHub and GitLab.
func slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r > 127:
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// changed says what changed in a source since checked, as text for a person;
// empty means nothing that matters. dir is the repository holding the source,
// rev the revision to compare with (HEAD, or a branch for another repository).
func changed(dir, checked, rev, path, anchor string) (string, error) {
	if !strings.HasSuffix(path, ".md") {
		return git(dir, "log", "--format=%h %s", checked+".."+rev, "--", path)
	}
	before, err := git(dir, "show", checked+":"+path)
	if err != nil {
		return "", err
	}
	after, err := git(dir, "show", rev+":"+path)
	if err != nil {
		return "", fmt.Errorf("%s no longer exists", path)
	}
	old, okOld := Section(before, anchor)
	now, okNow := Section(after, anchor)
	if !okOld {
		return "", fmt.Errorf("section #%s did not exist in %s at %s", anchor, path, checked)
	}
	if !okNow {
		return "section #" + anchor + " is gone", nil
	}
	if old == now {
		return "", nil
	}
	return git(dir, "log", "--format=%h %s", checked+".."+rev, "--", path)
}

type projectRepos struct {
	Repos map[string]struct {
		URL    string `yaml:"url"`
		Branch string `yaml:"branch"`
	} `yaml:"repos"`
}

// external is returned when a source repository cannot be read.
type external struct{ error }

// Pre finds suspect and pending docs. It exits 3 when a source repository
// cannot be read: a source that was not read has not been checked.
func Pre(runDir, repo string) int {
	var s Settings
	if err := readJSON(filepath.Join(runDir, "in", "settings.json"), &s); err != nil {
		return fail(err)
	}
	docs, err := findDocs(repo, s.Docs)
	if err != nil {
		return fail(err)
	}
	if len(docs) == 0 {
		// Not an error, but never a silent "all good": say what was looked at.
		v := verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("no doc declares its sources under %v — nothing is tracked yet", s.Docs)}
		if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
			return fail(err)
		}
		return 10
	}
	var cfg projectRepos
	if data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fail(err)
		}
	}
	gitDir, err := git(repo, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return fail(err)
	}

	byPath := map[string]*Doc{}
	for _, d := range docs {
		byPath[d.Path] = d
	}
	var findings []verdict.Finding
	suspect := map[string]bool{}
	// A doc is suspect when one of its own sources changed since it was checked.
	for _, d := range docs {
		for _, src := range d.Sources {
			name, path, anchor := splitSource(src)
			checked := d.Checked[name]
			if checked == "" || checked == "HEAD" {
				findings = append(findings, verdict.Finding{Rule: "unchecked", Where: d.Path,
					Message: fmt.Sprintf("no commit recorded in `checked` for %s", src)})
				continue
			}
			var commits string
			var err error
			if name == "" {
				commits, err = changed(repo, checked, "HEAD", path, anchor)
			} else {
				commits, err = remoteChanged(gitDir, name, cfg, checked, path, anchor)
			}
			if _, ok := err.(external); ok {
				fmt.Fprintln(os.Stderr, "documentalist:", err)
				return 3
			}
			if err != nil {
				findings = append(findings, verdict.Finding{Rule: "unknown", Where: d.Path, Level: "block",
					Message: fmt.Sprintf("cannot tell whether %s changed since %s: %v", src, checked, err)})
				continue
			}
			if commits != "" {
				suspect[d.Path] = true
				findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path,
					Message: fmt.Sprintf("%s changed since it was checked:\n%s", src, commits)})
			}
		}
	}
	// The cascade is cut: a doc depending on a suspect doc is pending, unless
	// the propagation edge between them says `now`.
	for grew := true; grew; {
		grew = false
		for _, d := range docs {
			if suspect[d.Path] {
				continue
			}
			for _, src := range d.Sources {
				name, path, _ := splitSource(src)
				if name != "" || !suspect[path] || byPath[path] == nil {
					continue
				}
				if edge(s, path, d.Path) == "now" {
					suspect[d.Path], grew = true, true
					findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path,
						Message: fmt.Sprintf("it follows %s, which is suspect", path)})
				} else if !hasPending(findings, d.Path) {
					findings = append(findings, verdict.Finding{Rule: "pending", Where: d.Path,
						Message: fmt.Sprintf("it follows %s, which is suspect; due %s", path, orDefault(edge(s, path, d.Path), "later"))})
				}
			}
		}
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].Where < findings[j].Where })
	if err := writeYAML(filepath.Join(runDir, "in", "findings.yaml"), findings); err != nil {
		return fail(err)
	}
	var task strings.Builder
	for _, f := range findings {
		if f.Rule == "suspect" {
			fmt.Fprintf(&task, "## %s\n\n%s\n\n", f.Where, f.Message)
		}
	}
	if task.Len() > 0 {
		body := "These docs may no longer be true. For each, say whether it still is; if not, fix only what is now wrong.\n\n" + task.String()
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(body), 0o644); err != nil {
			return fail(err)
		}
	}
	return 0
}

// Post turns the findings into a verdict. Suspect and pending docs are reported
// for a person (or were handled by the agent's patch); a source that could not
// be judged blocks.
func Post(runDir string) int {
	var findings []verdict.Finding
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "findings.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &findings); err != nil {
			return fail(err)
		}
	}
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	blocking := 0
	for i := range findings {
		if findings[i].Level == "block" {
			blocking++
			findings[i].Level = ""
		}
	}
	v := verdict.Verdict{Status: verdict.Pass, Findings: findings}
	switch {
	case blocking > 0:
		v.Status, v.Summary = verdict.Block, "some sources could not be judged"
	case len(intents) > 0:
		v.Summary = "docs updated"
	case len(findings) > 0:
		v.Summary = "docs to check — see the findings"
	default:
		v.Summary = "docs are up to date"
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	if v.Status == verdict.Block {
		return 1
	}
	return 0
}

// remoteChanged is changed, for a source in a declared repository. It keeps a
// bare copy of that repository in the git directory.
func remoteChanged(gitDir, name string, cfg projectRepos, checked, path, anchor string) (string, error) {
	r, ok := cfg.Repos[name]
	if !ok {
		return "", fmt.Errorf("repository %q is not declared in .workline/config.yaml", name)
	}
	branch := orDefault(r.Branch, "main")
	cache := filepath.Join(gitDir, "workline", "repos", name)
	if _, err := os.Stat(cache); err == nil {
		if _, err := git(cache, "fetch", "-q", "origin", "+refs/heads/*:refs/heads/*"); err != nil {
			return "", external{fmt.Errorf("cannot read repository %q (%s): %v", name, r.URL, err)}
		}
	} else if out, err := exec.Command("git", "clone", "-q", "--bare", r.URL, cache).CombinedOutput(); err != nil {
		os.RemoveAll(cache)
		return "", external{fmt.Errorf("cannot read repository %q (%s): %s", name, r.URL, strings.TrimSpace(string(out)))}
	}
	return changed(cache, checked, branch, path, anchor)
}

func edge(s Settings, from, to string) string {
	for _, e := range s.Propagation {
		if match(e.From, from) && match(e.To, to) {
			return e.When
		}
	}
	return ""
}

func hasPending(f []verdict.Finding, where string) bool {
	for _, x := range f {
		if x.Rule == "pending" && x.Where == where {
			return true
		}
	}
	return false
}

// findDocs lists the tracked docs: files matching the docs globs that declare sources.
func findDocs(repo string, globs []string) ([]*Doc, error) {
	files, err := git(repo, "ls-files")
	if err != nil {
		return nil, err
	}
	var docs []*Doc
	for _, f := range strings.Split(files, "\n") {
		if f == "" || !strings.HasSuffix(f, ".md") || !matchAny(globs, f) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo, f))
		if err != nil {
			return nil, err
		}
		d, err := ParseDoc(f, data)
		if err != nil {
			return nil, err
		}
		if d != nil {
			docs = append(docs, d)
		}
	}
	return docs, nil
}

func matchAny(globs []string, path string) bool {
	for _, g := range globs {
		if match(g, path) {
			return true
		}
	}
	return false
}

func match(glob, path string) bool { return pathglob.Match(glob, path) }

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "documentalist:", err)
	return 99
}
