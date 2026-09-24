// Package documentalist holds the deterministic part of the documentalist
// role (roles/documentalist): which docs became suspect because a source
// changed, which are only pending because the cascade is cut, the hygiene
// checks (budgets, duplicates, links, identifiers gone from the code), and the
// judge of the patches the agent proposes for suspect docs.
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
	Budgets    Budgets    `json:"budgets"`
	Duplicates Duplicates `json:"duplicates"`
	AIMaxCalls int        `json:"ai-max-calls"`
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

// header returns a doc's header — YAML between `---` lines, or, where a
// frontmatter would show on the forge (a README), between `<!-- workline` and
// `-->` — and how many lines it takes. A doc without one has no header.
func header(content string) (meta string, lines int) {
	all := strings.Split(content, "\n")
	var end string
	switch {
	case len(all) > 0 && all[0] == "---":
		end = "---"
	case len(all) > 0 && strings.TrimSpace(all[0]) == "<!-- workline":
		end = "-->"
	default:
		return "", 0
	}
	for i := 1; i < len(all); i++ {
		if strings.TrimSpace(all[i]) == end {
			return strings.Join(all[1:i], "\n"), i + 1
		}
	}
	return "", 0
}

// ParseDoc reads a doc's header. A doc without sources is not tracked.
func ParseDoc(path string, content []byte) (*Doc, error) {
	block, n := header(string(content))
	if n == 0 {
		return nil, nil
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
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

// body drops a doc's header: recording who checked a doc is not a change of
// what it says.
func body(content string) string {
	_, n := header(content)
	if n == 0 {
		return content
	}
	return strings.Join(strings.Split(content, "\n")[n:], "\n")
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

// evidenceLines caps what one source's change shows the agent.
const evidenceLines = 80

// evidence shows the agent what changed in a source: the diff of a file, or a
// doc section before and after.
func evidence(dir, checked, rev, path, anchor string) string {
	if !strings.HasSuffix(path, ".md") {
		diff, err := git(dir, "diff", checked, rev, "--", path)
		if err != nil {
			return "(the diff could not be read: " + err.Error() + ")"
		}
		return "```diff\n" + truncate(diff) + "\n```"
	}
	before, _ := git(dir, "show", checked+":"+path)
	after, _ := git(dir, "show", rev+":"+path)
	old, _ := Section(before, anchor)
	now, _ := Section(after, anchor)
	return "Before:\n\n```markdown\n" + truncate(old) + "\n```\n\nNow:\n\n```markdown\n" + truncate(now) + "\n```"
}

func truncate(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= evidenceLines {
		return s
	}
	return strings.Join(lines[:evidenceLines], "\n") + fmt.Sprintf("\n… (%d more lines)", len(lines)-evidenceLines)
}

type projectRepos struct {
	Repos map[string]struct {
		URL    string `yaml:"url"`
		Branch string `yaml:"branch"`
	} `yaml:"repos"`
}

// external is returned when a source repository cannot be read.
type external struct{ error }

// place is where a source's repository can be read, at which revision.
type place struct{ dir, rev string }

// places resolves repository names to where they can be read. Another
// repository is fetched once per run, into a bare copy in the git directory.
type places struct {
	repo, gitDir string
	cfg          projectRepos
	known        map[string]place
}

func (p *places) get(name string) (place, error) {
	if name == "" {
		return place{p.repo, "HEAD"}, nil
	}
	if pl, ok := p.known[name]; ok {
		return pl, nil
	}
	r, ok := p.cfg.Repos[name]
	if !ok {
		return place{}, fmt.Errorf("repository %q is not declared in .workline/config.yaml", name)
	}
	cache := filepath.Join(p.gitDir, "workline", "repos", name)
	if _, err := os.Stat(cache); err == nil {
		if _, err := git(cache, "fetch", "-q", "origin", "+refs/heads/*:refs/heads/*"); err != nil {
			return place{}, external{fmt.Errorf("cannot read repository %q (%s): %v", name, r.URL, err)}
		}
	} else if out, err := exec.Command("git", "clone", "-q", "--bare", r.URL, cache).CombinedOutput(); err != nil {
		os.RemoveAll(cache)
		return place{}, external{fmt.Errorf("cannot read repository %q (%s): %s", name, r.URL, strings.TrimSpace(string(out)))}
	}
	pl := place{cache, orDefault(r.Branch, "main")}
	p.known[name] = pl
	return pl, nil
}

// suspectDoc is a doc to put before the agent, with what made it suspect.
type suspectDoc struct {
	doc      *Doc
	why      []string
	evidence []string
}

// taskMaxChars keeps the task well inside the role's context budget (16000
// tokens, about 64000 characters, facets included). Docs left out stay
// suspect, and are judged by a person or by a later run.
const taskMaxChars = 40000

// Pre finds suspect and pending docs, runs the hygiene checks, and writes the
// suspect docs into task.md for the agent to judge. It exits 3 when a source
// repository cannot be read: a source that was not read has not been checked.
func Pre(runDir, repo string) int {
	var s Settings
	if err := readJSON(filepath.Join(runDir, "in", "settings.json"), &s); err != nil {
		return fail(err)
	}
	tree, err := loadTree(repo, s.Docs)
	if err != nil {
		return fail(err)
	}
	docs, err := trackedDocs(tree, s.Docs)
	if err != nil {
		return fail(err)
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
	pl := &places{repo: repo, gitDir: gitDir, cfg: cfg, known: map[string]place{}}

	var findings []verdict.Finding
	if len(docs) == 0 {
		// Not an error, but never a silent "all good": say what was looked at.
		findings = append(findings, verdict.Finding{Rule: "nothing-tracked",
			Message: fmt.Sprintf("no doc declares its sources under %v, so none can be found suspect", s.Docs)})
	}
	byPath := map[string]*Doc{}
	for _, d := range docs {
		byPath[d.Path] = d
	}
	suspects := map[string]*suspectDoc{}
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
			where, err := pl.get(name)
			var commits string
			if err == nil {
				commits, err = changed(where.dir, checked, where.rev, path, anchor)
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
				why := fmt.Sprintf("%s changed since it was checked:\n%s", src, commits)
				findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path, Message: why})
				sd := suspects[d.Path]
				if sd == nil {
					sd = &suspectDoc{doc: d}
					suspects[d.Path] = sd
				}
				sd.why = append(sd.why, why)
				sd.evidence = append(sd.evidence, fmt.Sprintf("What changed in %s:\n\n%s", src, evidence(where.dir, checked, where.rev, path, anchor)))
			}
		}
	}
	// The cascade is cut: a doc depending on a suspect doc is pending, unless
	// the propagation edge between them says `now`.
	for grew := true; grew; {
		grew = false
		for _, d := range docs {
			if suspects[d.Path] != nil {
				continue
			}
			for _, src := range d.Sources {
				name, path, _ := splitSource(src)
				if name != "" || suspects[path] == nil || byPath[path] == nil {
					continue
				}
				if edge(s, path, d.Path) == "now" {
					why := fmt.Sprintf("it follows %s, which is suspect", path)
					suspects[d.Path], grew = &suspectDoc{doc: d, why: []string{why},
						evidence: []string{fmt.Sprintf("%s is judged in this same task: make this doc agree with it as you leave it.", path)}}, true
					findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path, Message: why})
				} else if !hasPending(findings, d.Path) {
					findings = append(findings, verdict.Finding{Rule: "pending", Where: d.Path,
						Message: fmt.Sprintf("it follows %s, which is suspect; due %s", path, orDefault(edge(s, path, d.Path), "later"))})
				}
			}
		}
	}

	// Hygiene: what needs no judgement is reported by the checks themselves.
	problems := Hygiene(tree, s.Budgets, s.Duplicates)
	gone, err := IdentifiersGone(repo, tree.Docs)
	if err != nil {
		return fail(err)
	}
	for _, p := range append(problems, gone...) {
		f := verdict.Finding{Rule: p.Rule, Where: p.Where, Message: p.Message}
		if p.Rule == "setting-missing" {
			f.Level = "block" // a check that could not run is not a pass
		}
		findings = append(findings, f)
	}

	task, judged, err := suspectTask(suspects, s.AIMaxCalls, pl, repo)
	if err != nil {
		return fail(err)
	}
	for i := range findings {
		if f := &findings[i]; f.Rule == "suspect" && judged[f.Where] == nil && task != "" {
			f.Message += "\n(not put before the agent in this run: over ai-max-calls or the task's size)"
		}
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].Where < findings[j].Where })
	if err := writeYAML(filepath.Join(runDir, "in", "findings.yaml"), findings); err != nil {
		return fail(err)
	}
	if task != "" {
		if err := writeYAML(filepath.Join(runDir, "in", "judged.yaml"), judged); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
			return fail(err)
		}
	}
	return 0
}

// suspectTask writes the question for the agent: each suspect doc, with line
// numbers, what changed in its sources, and the commits its `checked` must
// name once judged. It returns those commits per doc, for the judge.
func suspectTask(suspects map[string]*suspectDoc, maxDocs int, pl *places, repo string) (string, map[string]map[string]string, error) {
	paths := make([]string, 0, len(suspects))
	for p := range suspects {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	b.WriteString(`Kind: suspect

These docs may no longer be true, because something they depend on changed.
For each one, say whether it still is, with a patch: a unified diff with
context lines, whose hunks cite the doc's lines by the numbers shown here.
If the doc is still true, the patch only sets ` + "`checked`" + `. If not, it also
fixes what is now wrong, and nothing else.

`)
	judged := map[string]map[string]string{}
	for _, p := range paths {
		if maxDocs > 0 && len(judged) >= maxDocs {
			break
		}
		sd := suspects[p]
		want := map[string]string{}
		var short []string
		names := make([]string, 0, len(sd.doc.Checked))
		for n := range sd.doc.Checked {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			where, err := pl.get(n)
			if err != nil {
				return "", nil, err
			}
			full, err := git(where.dir, "rev-parse", where.rev)
			if err != nil {
				return "", nil, err
			}
			want[n] = full
			if n == "" {
				short = append(short, full[:7])
			} else {
				short = append(short, n+": "+full[:7])
			}
		}
		checked := "checked: " + strings.Join(short, "")
		if len(names) > 1 || (len(names) == 1 && names[0] != "") {
			checked = "checked: {" + strings.Join(short, ", ") + "}"
		}
		content, err := os.ReadFile(filepath.Join(repo, p))
		if err != nil {
			return "", nil, err
		}
		var entry strings.Builder
		fmt.Fprintf(&entry, "## %s\n\nWhatever you decide, your patch sets `%s`: the commit this doc is judged against now, not the commit that changed a source.\n\nWhy it is suspect:\n\n", p, checked)
		for _, w := range sd.why {
			fmt.Fprintf(&entry, "- %s\n", strings.ReplaceAll(w, "\n", "\n  "))
		}
		for _, e := range sd.evidence {
			entry.WriteString("\n" + e + "\n")
		}
		entry.WriteString("\nThe doc as it is now, with its line numbers:\n\n```\n")
		for i, l := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
			fmt.Fprintf(&entry, "%4d | %s\n", i+1, l)
		}
		entry.WriteString("```\n\n")
		if len(judged) > 0 && b.Len()+entry.Len() > taskMaxChars {
			break
		}
		b.WriteString(entry.String())
		judged[p] = want
	}
	if len(judged) == 0 {
		return "", nil, nil
	}
	return b.String(), judged, nil
}

// Post judges the agent's patches, then turns the findings into a verdict.
// Suspect and pending docs are reported for a person, unless a patch handled
// them; a source that could not be judged, or a check that could not run,
// blocks.
func Post(runDir, repo string) int {
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
	judged := map[string]map[string]string{}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "judged.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &judged); err != nil {
			return fail(err)
		}
	}
	var s Settings
	if err := readJSON(filepath.Join(runDir, "in", "settings.json"), &s); err != nil {
		return fail(err)
	}
	refused, patched, err := judgePatches(repo, s, judged, intents)
	if err != nil {
		return fail(err)
	}
	if len(refused) > 0 {
		// Only the refusals: they are what the agent is asked again with, and
		// the docs they name are the suspect ones.
		v := verdict.Verdict{Status: verdict.Block, Summary: "the proposed patch was refused", Findings: refused}
		if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
			return fail(err)
		}
		return 1
	}
	blocking := 0
	var kept []verdict.Finding
	for _, f := range findings {
		if f.Rule == "suspect" && patched[f.Where] {
			continue // judged and patched in this run
		}
		if f.Level == "block" {
			blocking++
			f.Level = ""
		}
		kept = append(kept, f)
	}
	v := verdict.Verdict{Status: verdict.Pass, Findings: kept}
	switch {
	case blocking > 0:
		v.Status, v.Summary = verdict.Block, "some sources could not be judged, or some checks could not run"
	case len(intents) > 0:
		v.Summary = "docs updated"
	case len(kept) > 0:
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

// loadTree reads the docs the role looks after — the Markdown files matching
// the docs globs, and the agents' entry points at the root — and lists every
// tracked file.
func loadTree(repo string, globs []string) (Tree, error) {
	t := Tree{Docs: map[string]string{}, Files: map[string]bool{}}
	files, err := git(repo, "ls-files")
	if err != nil {
		return t, err
	}
	for _, f := range strings.Split(files, "\n") {
		if f == "" {
			continue
		}
		t.Files[f] = true
		isAgentFile := false
		for _, a := range rootAgentFiles {
			isAgentFile = isAgentFile || f == a
		}
		if !isAgentFile && (!strings.HasSuffix(f, ".md") || !matchAny(globs, f)) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo, f))
		if os.IsNotExist(err) {
			continue // deleted in the working tree
		}
		if err != nil {
			return t, err
		}
		t.Docs[f] = string(data)
	}
	return t, nil
}

// trackedDocs lists the docs that declare their sources.
func trackedDocs(t Tree, globs []string) ([]*Doc, error) {
	paths := make([]string, 0, len(t.Docs))
	for p := range t.Docs {
		if matchAny(globs, p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var docs []*Doc
	for _, p := range paths {
		d, err := ParseDoc(p, []byte(t.Docs[p]))
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
	return gitIn(dir, "", args...)
}

// gitIn runs git with stdin.
func gitIn(dir, stdin string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdin = strings.NewReader(stdin)
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
