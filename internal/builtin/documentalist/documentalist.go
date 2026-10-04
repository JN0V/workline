// Package documentalist holds the deterministic part of the documentalist
// role (roles/documentalist): which docs became suspect because a source
// changed, which are only pending because the cascade is cut, the hygiene
// checks (budgets, duplicates, links, identifiers gone from the code), and the
// judge of the patches the agent proposes for suspect docs.
package documentalist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/tools"
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
	Freshness  Freshness  `json:"freshness"`
	// MaxOpenMergeRequests stops gardening while this many of the role's
	// merge requests wait for review (ADR-0006).
	MaxOpenMergeRequests int `json:"max-open-merge-requests"`
	Truth                struct {
		Doc []string `json:"doc"` // docs the code follows: never rewritten to match it
	} `json:"truth"`
	AIMaxCalls int               `json:"ai-max-calls"`
	Derive     map[string]string `json:"derive"` // name -> command giving a derived block
	// Documented: the code the project wants described; a file of it no
	// doc names in its sources is reported.
	Documented []string `json:"documented"`
	// JudgeInParts judges a doc whose sources do not fit a task in parts,
	// instead of leaving it to a person (ADR-0009); PartsMax caps the parts
	// of one doc, PartsMaxPerRun those asked in one run.
	JudgeInParts   bool `json:"judge-in-parts"`
	PartsMax       int  `json:"parts-max"`
	PartsMaxPerRun int  `json:"parts-max-per-run"`
	// For measuring whole against parts on the same doc (tests/evaluation):
	// PartsAlways judges in parts even a doc that fits whole, PartChars
	// caps the sources' share of a part, so that small sources still split.
	PartsAlways bool `json:"parts-always"`
	PartChars   int  `json:"part-chars"`
	// History and Decisions: the project's globs of history docs and of
	// decision records, added to the names the usual tools give them
	// (records.go).
	History   []string `json:"history"`
	Decisions []string `json:"decisions"`
	// Language: the docs' language, whose glue and fact words the removal
	// rule reads ("en", "fr"); "" reads each doc's own (words.go).
	Language string `json:"language"`
	// Versions: how the project writes a version, when not in three parts
	// (a regexp), and the files saying its version (values.go).
	Versions struct {
		Pattern string   `json:"pattern"`
		Files   []string `json:"files"`
	} `json:"versions"`
	// WholeChars caps the characters a doc's sources may take to be judged
	// whole; past it, a person judges the doc, or it is judged in parts.
	WholeChars int `json:"whole-chars"`
	// Sample: who reads the weekly sample of the docs vouched for
	// (ADR-0014, step 4; workline sample).
	Sample SampleSettings `json:"sample"`
}

// Doc is a documentation file that declares its sources.
type Doc struct {
	Path    string
	Sources []string
	Checked map[string]string // repository name ("" = this one) -> commit
	// JudgedInParts is the commit the doc was last judged in parts at: it is
	// not put before an agent again until a source changes after it.
	JudgedInParts string
	// Judged is the commit an agent judged the doc whole at without vouching
	// for every sentence: held the same way.
	Judged string
}

// frontmatter is the YAML block at the top of a doc.
type frontmatter struct {
	Sources []string `yaml:"sources"`
	// A node, not a string: read as YAML, a commit like 11180e1 is a number.
	Checked       yaml.Node `yaml:"checked"`
	JudgedInParts yaml.Node `yaml:"judged-in-parts"`
	Judged        yaml.Node `yaml:"judged"`
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
	d := &Doc{Path: path, Sources: fm.Sources, Checked: map[string]string{}, JudgedInParts: fm.JudgedInParts.Value, Judged: fm.Judged.Value}
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
		out, err := git(dir, "log", "--format=%H %h %s", checked+".."+rev, "--", path)
		if err != nil || out == "" {
			return out, err
		}
		var kept []string
		for _, l := range strings.Split(out, "\n") {
			full, rest, _ := strings.Cut(l, " ")
			if !headersOnly(dir, full, path) {
				kept = append(kept, rest)
			}
		}
		return strings.Join(kept, "\n"), nil
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

// onMain returns the commit to compare a doc's sources with: the one its
// `checked` names, when rev holds it; else the commit of rev that brought
// that name into the doc — the squash or the rebased commit of the merge
// request where the doc was judged. ok is false when rev has neither.
func onMain(repo, rev, doc, checked string) (string, bool) {
	key := strings.Join([]string{repo, rev, doc, checked}, "\x00")
	if a, ok := onMainSeen[key]; ok {
		return a.commit, a.ok
	}
	a := anchored{checked, true}
	if _, err := git(repo, "merge-base", "--is-ancestor", checked, rev); err != nil {
		out, err := git(repo, "log", "--reverse", "--format=%H", "-S"+checked, rev, "--", doc)
		first, _, _ := strings.Cut(out, "\n")
		a = anchored{first, err == nil && first != ""}
	}
	onMainSeen[key] = a
	return a.commit, a.ok
}

// anchored is what onMain found, kept for the doc's other sources.
type anchored struct {
	commit string
	ok     bool
}

var onMainSeen = map[string]anchored{}

// headersOnly says whether a commit changed, under path, only the headers of
// docs: recording who checked a doc changes nothing it says, and would
// otherwise make the docs naming a folder of docs suspect at every check.
// semver is a version number, as release tools write and bump them.
var semver = regexp.MustCompile(`v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?`)

// versionsOnly says whether a commit changed path only by putting one
// version number in place of another, line for line — a release tool's
// bump (release-please's `x-release-please-version`, a manifest) — and
// which versions it replaced.
func versionsOnly(dir, commit, path string) (bool, []string) {
	out, err := git(dir, "show", "--format=", "-U0", "--no-color", commit, "--", path)
	if err != nil || out == "" {
		return false, nil
	}
	var removed, added, old []string
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "---"), strings.HasPrefix(l, "+++"):
		case strings.HasPrefix(l, "-"):
			old = append(old, semver.FindAllString(l[1:], -1)...)
			removed = append(removed, semver.ReplaceAllString(l[1:], "V"))
		case strings.HasPrefix(l, "+"):
			added = append(added, semver.ReplaceAllString(l[1:], "V"))
		}
	}
	if len(removed) == 0 || len(removed) != len(added) {
		return false, nil
	}
	for i := range removed {
		if removed[i] != added[i] || !strings.Contains(removed[i], "V") {
			return false, nil
		}
	}
	return true, old
}

// dropBumps leaves out of a doc's commits ("%h %s" lines), on a release
// tool's pull request, those that only bumped a version in path, unless the
// doc quotes a version they replaced: the tool's own bump says nothing else
// a doc describes, and would make the release's docs suspect at every
// release. Elsewhere a bump is a change like any: the docs it touches are
// judged (the evaluation's drifted-version-bump).
func dropBumps(dir, commits, path, doc string) string {
	if commits == "" || strings.HasPrefix(commits, "(") {
		return commits
	}
	var kept []string
	for _, l := range strings.Split(commits, "\n") {
		hash, _, _ := strings.Cut(l, " ")
		bump, old := versionsOnly(dir, hash, path)
		quoted := slices.ContainsFunc(old, func(v string) bool { return strings.Contains(doc, strings.TrimPrefix(v, "v")) })
		if !bump || quoted {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

func headersOnly(dir, commit, path string) bool {
	under := strings.TrimSuffix(path, "/")
	files := commitFiles(dir, commit)
	found := false
	for i := range files {
		f := &files[i]
		if f.path != under && !strings.HasPrefix(f.path, under+"/") {
			continue
		}
		if !strings.HasSuffix(f.path, ".md") {
			return false
		}
		if f.headerOnly == nil {
			before, err1 := git(dir, "show", commit+"^:"+f.path)
			after, err2 := git(dir, "show", commit+":"+f.path)
			same := err1 == nil && err2 == nil && body(before) == body(after)
			f.headerOnly = &same
		}
		if !*f.headerOnly {
			return false
		}
		found = true
	}
	return found
}

// changedFile is a file a commit changed; headerOnly, once read, says
// whether it is a doc whose header alone changed.
type changedFile struct {
	path       string
	headerOnly *bool
}

// commitFilesSeen keeps what each commit changed: the same commits come back
// for every source of every doc, thousands of times on a large repository.
var commitFilesSeen = map[string][]changedFile{}

func commitFiles(dir, commit string) []changedFile {
	key := dir + "\x00" + commit
	if files, ok := commitFilesSeen[key]; ok {
		return files
	}
	var files []changedFile
	if out, err := git(dir, "diff-tree", "-z", "--no-commit-id", "--name-only", "-r", commit); err == nil && out != "" {
		for _, f := range pathList(out) {
			files = append(files, changedFile{path: f})
		}
	}
	commitFilesSeen[key] = files
	return files
}

// evidenceLines caps what one source's change shows the agent.
const evidenceLines = 80

// evidence shows the agent what changed in a source: the diff of a file, or a
// doc section before and after.
func evidence(dir, checked, rev, path, anchor string, max int) string {
	if !strings.HasSuffix(path, ".md") {
		diff, err := git(dir, "diff", checked, rev, "--", path)
		if err != nil {
			return "(the diff could not be read: " + err.Error() + ")"
		}
		return "```diff\n" + truncate(diff, max) + "\n```"
	}
	before, _ := git(dir, "show", checked+":"+path)
	after, _ := git(dir, "show", rev+":"+path)
	old, _ := Section(before, anchor)
	now, _ := Section(after, anchor)
	return "Before:\n\n```markdown\n" + truncate(old, max) + "\n```\n\nNow:\n\n```markdown\n" + truncate(now, max) + "\n```"
}

func truncate(s string, max int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "\n") + fmt.Sprintf("\n… (%d more lines)", len(lines)-max)
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
		if pl, ok := p.known[""]; ok { // this repository read at another revision
			return pl, nil
		}
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
	// tooLarge: the doc, with what changed in its sources, does not fit in a
	// task alone; judged by a person, never confirmed unread.
	tooLarge bool
	// whole: every source of the doc is given whole in its task, as it is
	// now. Only then may `checked` move (ADR-0014): a doc judged on diffs,
	// or beside a doc of the same task, is fixed and records `judged`.
	whole  bool
	shown  int  // lines of evidence shown so far, within docEvidenceLines
	capped bool // more changed than docEvidenceLines shows
	// note says what became of a doc judged in parts, or why it was not:
	// such a doc is not put in a task of its own.
	note     string
	deferred bool // left for the run's next round
	// earlier: judged by an earlier round of this run, whose merge request
	// is proposed, not opened yet: that round's finding stands.
	earlier bool
}

// docEvidenceLines is what one doc shows of what changed in all its sources:
// one budget for the doc, so naming more files does not grow its task.
// Measured on DomoticsCore (2026-09-29): 80 lines a source made a doc naming
// ten files cost more than one naming their folder.
const docEvidenceLines = 240

// evidenceFor is what changed in one source, within what is left of the
// doc's budget; past it, how much changed, and how to see it.
func (sd *suspectDoc) evidenceFor(dir, checked, rev, path, anchor string) string {
	left := min(evidenceLines, docEvidenceLines-sd.shown)
	if left <= 0 {
		sd.capped = true
		stat, _ := git(dir, "diff", "--shortstat", checked, rev, "--", path)
		return fmt.Sprintf("(not shown, past this doc's share of the task: %s; `git diff %s %s -- %s` shows it)", strings.TrimSpace(stat), checked, rev, path)
	}
	e := evidence(dir, checked, rev, path, anchor, left)
	sd.shown += min(left, strings.Count(e, "\n"))
	sd.capped = sd.capped || strings.Contains(e, " more lines)")
	return e
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
	useRecords(s)
	useLanguage(s)
	if err := useVersions(s); err != nil {
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

	// Given the commits of a push or a merge request, only the docs they
	// made suspect are the task; the others are reported, for gardening.
	touched, ranged, err := rangeFiles(runDir, repo)
	if err != nil {
		return fail(err)
	}
	// The release is every doc made suspect since the last one, not only
	// those the release pull request's own commits touched.
	if os.Getenv("WORKLINE_EVENT") == "release" {
		ranged = false
	}
	var findings []verdict.Finding
	if len(docs) == 0 && os.Getenv("WORKLINE_EVENT") != "init" {
		// Not an error, but never a silent "all good": say what was looked at.
		// On init, each doc is said with the header that would track it.
		findings = append(findings, verdict.Finding{Rule: "nothing-tracked",
			Message: fmt.Sprintf("no doc declares its sources under %v, so none can be found suspect", s.Docs)})
	}
	byPath := map[string]*Doc{}
	for _, d := range docs {
		byPath[d.Path] = d
	}
	suspects := map[string]*suspectDoc{}
	var stale map[string]*suspectDoc      // docs confirmed too long ago, whose sources fit in a task
	propagate := map[string]*suspectDoc{} // docs due now, at the moment their edge names
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
			// A doc fixed on a merge request names a commit of its branch,
			// which a squash or a rebase leaves out of main.
			unanchored := false
			if err == nil && name == "" {
				if c, ok := onMain(repo, where.rev, d.Path, checked); ok {
					checked = c
				} else {
					unanchored = true
				}
			}
			gone := name == "" && !tree.exists(strings.TrimSuffix(path, "/"))
			// A doc the commits did not touch is only left for gardening:
			// once it is found suspect, its other sources need no reading.
			if !gone && ranged && hasSuspect(findings, d.Path) && !docTouched(d, touched) {
				continue
			}
			// A source gone is said, and what removed it makes the doc suspect.
			if err == nil && gone {
				findings = append(findings, verdict.Finding{Rule: "source-gone", Where: d.Path,
					Message: fmt.Sprintf("names %s as a source, which no longer exists: name what replaced it, or drop it", src)})
				commits, err = git(where.dir, "log", "--format=%h %s", checked+".."+where.rev, "--", path)
			} else if err == nil && unanchored {
				commits = fmt.Sprintf("(`checked` names %s, which %s does not hold, and no commit brought it there)", checked, where.rev)
			} else if err == nil {
				commits, err = changed(where.dir, checked, where.rev, path, anchor)
				if err == nil && name == "" && os.Getenv("WORKLINE_RELEASE_BRANCH") != "" {
					commits = dropBumps(where.dir, commits, path, tree.Docs[d.Path])
				}
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
			if commits != "" && ranged && !docTouched(d, touched) {
				if !hasSuspect(findings, d.Path) {
					findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path,
						Message: fmt.Sprintf("%s changed since it was checked, but not changed by these commits: left for gardening\n%s", src, commits)})
				}
				continue
			}
			if commits != "" {
				why := fmt.Sprintf("%s changed since it was checked:\n%s", src, commits)
				// A doc following another doc along an edge due later waits for
				// that moment (a product doc, for the release), then is propagated.
				into := suspects
				if when := edge(s, path, d.Path); name == "" && strings.HasSuffix(path, ".md") && when != "" && when != "now" {
					if os.Getenv("WORKLINE_EVENT") != when {
						if !hasPending(findings, d.Path) {
							findings = append(findings, verdict.Finding{Rule: "pending", Where: d.Path,
								Message: fmt.Sprintf("%s changed since it was checked; due at %s", src, when)})
						}
						continue
					}
					findings = append(findings, verdict.Finding{Rule: "due", Where: d.Path, Level: "block",
						Message: fmt.Sprintf("%s changed since it was checked, and this %s is when it follows: until it does, the %s waits", src, when, when)})
					into = propagate
				} else {
					findings = append(findings, verdict.Finding{Rule: "suspect", Where: d.Path, Message: why})
				}
				sd := into[d.Path]
				if sd == nil {
					sd = &suspectDoc{doc: d}
					into[d.Path] = sd
				}
				sd.why = append(sd.why, why)
				if unanchored { // nothing to compare with: judged against its sources as they are now
					sd.capped = true
				} else {
					sd.evidence = append(sd.evidence, fmt.Sprintf("What changed in %s:\n\n%s", src, sd.evidenceFor(where.dir, checked, where.rev, path, anchor)))
				}
			}
		}
	}
	// A doc far behind its sources is judged against them as they are now:
	// what changed would not fit, and reading months of diffs cut short
	// lets nobody vouch for it. Sources too large for that go to a person.
	for _, into := range []map[string]*suspectDoc{suspects, propagate} {
		for _, sd := range into {
			if !sd.capped {
				// Vouching for every sentence the doc keeps needs its sources,
				// not only what changed in them (ADR-0012): beside the diffs,
				// when they fit.
				if now, ok := sourcesNow(sd.doc, pl, wholeChars(s)); ok && len(now) > 0 {
					sd.evidence = append(sd.evidence, append([]string{"Its sources as they are now, in full, beside what changed: judge each sentence of the doc against them, not only the lines that changed."}, now...)...)
					sd.whole = true
				}
				continue
			}
			now, ok := sourcesNow(sd.doc, pl, wholeChars(s))
			if !ok {
				sd.tooLarge = true
				continue
			}
			sd.evidence = append([]string{"More changed in its sources since it was checked than a task can show. Here they are as they are now, in full: judge each sentence of the doc against them."}, now...)
			sd.whole = len(now) > 0
		}
	}
	holdJudgedInParts(suspects, repo)

	if s.JudgeInParts && s.PartsAlways { // measuring: every suspect doc in parts
		for _, into := range []map[string]*suspectDoc{suspects, propagate} {
			for _, sd := range into {
				sd.tooLarge = true
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

	var added map[string]bool
	if ranged {
		if added, err = addedFiles(runDir, repo); err != nil {
			return fail(err)
		}
	}
	findings = append(findings, undocumented(tree, docs, s.Documented, added, ranged)...)
	// Docs the role does not read: said when gardening and when adopting,
	// not on each change, which adds nothing to them.
	if ev := os.Getenv("WORKLINE_EVENT"); ev == "schedule" || ev == "init" {
		if msg := unreadMessage(unreadDocs(tree.Files, s.Docs)); msg != "" {
			findings = append(findings, verdict.Finding{Rule: "docs-not-read", Message: msg})
		}
	}

	// Hygiene: what needs no judgement is reported by the checks themselves.
	problems := Hygiene(tree, s.Budgets, s.Duplicates)
	var gone []Problem
	if ranged {
		gone, err = IdentifiersRemoved(repo, rangeOf(runDir), tree.Docs)
	} else {
		gone, err = IdentifiersGone(repo, tree.Docs)
	}
	if err != nil {
		return fail(err)
	}
	// Links to other sites and freshness: checked when gardening only, since
	// they go wrong with no change to the repository.
	if os.Getenv("WORKLINE_EVENT") == "schedule" {
		if problems, err = withExternalLinks(repo, tree, problems); err != nil {
			return fail(err)
		}
		findings = append(findings, staleDocs(docs, pl, s.Freshness, time.Now(), func(p string) bool {
			return suspects[p] != nil || hasPending(findings, p)
		})...)
		stale = staleForAgent(findings, byPath, pl, s)
	}
	// Line counts off: counted by the engine, never vouched for (ADR-0014).
	counts := CountsOff(repo, tree, docs)
	for _, p := range append(append(problems, gone...), counts...) {
		f := verdict.Finding{Rule: p.Rule, Where: p.Where, Message: p.Message}
		if p.Rule == "setting-missing" {
			f.Level = "block" // a check that could not run is not a pass
		}
		findings = append(findings, f)
	}

	// Derived blocks: regenerated by the role itself, no judgement needed.
	derivedFindings, fixes := Derive(repo, tree.Docs, s.Derive)
	findings = append(findings, derivedFindings...)
	var fallback []intent.Intention
	for _, p := range sortedKeys(fixes) {
		// A diff of the derived lines alone: the agent's patch of the same doc
		// then applies beside it, instead of replacing it (intent.Merge).
		diff, err := zeroContextDiff(p, tree.Docs[p], fixes[p])
		if err != nil {
			return fail(err)
		}
		fallback = append(fallback, intent.Intention{Kind: "patch", Value: diff})
	}
	if c, ok := checklist(findings); ok {
		fallback = append(fallback, c)
	}
	if i, ok := trackingIssue(findings); ok {
		fallback = append(fallback, i)
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}

	// Adopting a repository: the docs saying nothing of their sources are the
	// task; the suspect ones wait for the next run, once the person has
	// reviewed what this one proposes.
	adopting := os.Getenv("WORKLINE_EVENT") == "init"
	holdWaitingTasks(suspects, stale)
	var task string
	var judged map[string]map[string]string
	if adopting {
		var more []verdict.Finding
		if more, task, judged, err = adoption(runDir, repo, tree, s); err != nil {
			return fail(err)
		}
		findings = append(findings, more...)
	} else if task, judged, err = suspectTask(suspects, s, pl, repo); err != nil {
		return fail(err)
	}
	// Docs too large to be judged whole are judged in parts, when the
	// project says so (ADR-0009): the parts first, then one call to fix. A
	// run asking parts asks nothing else: the rest waits for the next.
	inParts := map[string]string{}
	partsAsked := 0
	var partsFallback []intent.Intention
	inPartsTask := func(docs map[string]*suspectDoc) error {
		if task != "" || adopting || !s.JudgeInParts {
			return nil
		}
		var cands []*suspectDoc
		for _, p := range sortedDocs(docs) {
			if sd := docs[p]; sd.tooLarge && sd.note == "" {
				cands = append(cands, sd)
			}
		}
		t, j, fb, ip, err := partsFor(runDir, repo, cands, s, &partsAsked, &findings)
		if err != nil {
			return err
		}
		task, judged, partsFallback = t, j, append(partsFallback, fb...)
		for p, at := range ip {
			inParts[p] = at
		}
		if task != "" {
			return os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("fix\n"), 0o644)
		}
		return nil
	}
	if err := inPartsTask(suspects); err != nil {
		return fail(err)
	}
	// Due now (at the release): each doc is brought up to date for its reader.
	if task == "" && partsAsked == 0 && len(propagate) > 0 {
		if task, judged, err = docTask(`Kind: propagate

These docs follow technical docs that changed, and are updated now, at the
moment their edge names. For each, update it for its reader — what they can
do, what changed for them — from what changed in its sources below, and say
nothing those changes do not back. A patch: a unified diff with context lines,
whose hunks cite the doc's lines by the numbers shown here. It may grow the
doc when its reader really gained something to know.

`, propagate, s, pl, repo); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("propagate\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	// Backpressure: while enough of the role's merge requests wait for review,
	// gardening proposes nothing more; what it finds is still reported.
	gardening := os.Getenv("WORKLINE_EVENT") == "schedule"
	if n, err := strconv.Atoi(os.Getenv("WORKLINE_OPEN_MERGE_REQUESTS")); err == nil && gardening &&
		s.MaxOpenMergeRequests > 0 && n >= s.MaxOpenMergeRequests {
		task, judged, stale, gardening, inParts, partsFallback, partsAsked = "", nil, nil, false, map[string]string{}, nil, 0
		os.RemoveAll(filepath.Join(runDir, "in", "parts"))
		var kept []intent.Intention // derived blocks too would open one more merge request
		for _, f := range fallback {
			if f.Kind != "patch" {
				kept = append(kept, f)
			}
		}
		fallback = kept
		os.Remove(filepath.Join(runDir, "in", "fallback.yaml"))
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), kept); err != nil {
			return fail(err)
		}
		findings = append(findings, verdict.Finding{Rule: "gardening-paused",
			Message: fmt.Sprintf("%d merge requests of the documentalist wait for review (max-open-merge-requests: %d): no task proposed until fewer wait", n, s.MaxOpenMergeRequests)})
	}
	// Gardening, with no suspect doc to judge: read stale docs again, else
	// merge a repeated passage (two copies drift apart), else condense the
	// doc most over its budget. A merge request or a push never turns into a
	// rewrite of the docs.
	holdJudgedInParts(stale, repo)
	if task == "" && partsAsked == 0 && len(stale) > 0 {
		if task, judged, err = staleTask(stale, s, pl, repo); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("stale\n"), 0o644); err != nil {
			return fail(err)
		}
	}
	if err := inPartsTask(stale); err != nil {
		return fail(err)
	}
	if len(partsFallback) > 0 { // the header of each doc judged in parts records when
		os.Remove(filepath.Join(runDir, "in", "fallback.yaml"))
		if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), append(fallback, partsFallback...)); err != nil {
			return fail(err)
		}
	}
	if len(inParts) > 0 { // their fixes are judged as fixes in parts: `checked` stays
		if err := writeYAML(filepath.Join(runDir, "in", "in-parts.yaml"), inParts); err != nil {
			return fail(err)
		}
	}
	if task == "" && partsAsked == 0 && gardening {
		if d := pickDedupe(problems); d != nil {
			task = writeDedupeTask(d, problems, tree)
			if err := writeYAML(filepath.Join(runDir, "in", "dedupe.yaml"), d); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("duplicates\n"), 0o644); err != nil {
				return fail(err)
			}
		}
	}
	if task == "" && partsAsked == 0 && gardening {
		if c := pickCondense(problems); c != nil {
			task = writeCondenseTask(c, problems, tree)
			if err := writeYAML(filepath.Join(runDir, "in", "condense.yaml"), c); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("condense\n"), 0o644); err != nil {
				return fail(err)
			}
		}
	}
	// A card too long holds more than one concept: split into cards.
	if task == "" && partsAsked == 0 && gardening {
		if c := pickSplit(problems); c != nil {
			task = writeSplitTask(c, problems, tree)
			if err := writeYAML(filepath.Join(runDir, "in", "condense.yaml"), c); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("split\n"), 0o644); err != nil {
				return fail(err)
			}
		}
	}
	// Last, a card too short to stand alone goes into the one it belongs with.
	if task == "" && partsAsked == 0 && gardening {
		if m := pickMergeCard(problems, tree); m != nil {
			task = writeMergeCardTask(m, problems, tree)
			if err := writeYAML(filepath.Join(runDir, "in", "merge-card.yaml"), m); err != nil {
				return fail(err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("merge-card\n"), 0o644); err != nil {
				return fail(err)
			}
		}
	}
	if os.Getenv("WORKLINE_EVENT") == "release" {
		if findings, err = holdTheRelease(findings, suspects, repo); err != nil {
			return fail(err)
		}
	}
	var left strings.Builder
	for i := range findings {
		if adopting {
			break // adoption said itself what it left for the next round
		}
		f := &findings[i]
		// Only the docs this run could put before the agent are left for its
		// next round: not those left for gardening, nor those too large.
		var sd *suspectDoc
		switch f.Rule {
		case "suspect":
			sd = suspects[f.Where]
		case "due":
			sd = propagate[f.Where]
		case "stale":
			sd = stale[f.Where]
		}
		if sd == nil || judged[f.Where] != nil && sd.note == "" {
			continue
		}
		if sd.earlier {
			f.Rule = "" // what the earlier round said of it stands: left out below
			continue
		}
		if sd.note != "" {
			f.Message += "\n" + sd.note
			if sd.deferred {
				fmt.Fprintf(&left, "%s %s\n", f.Rule, f.Where)
			}
			continue
		}
		if sd.tooLarge && s.JudgeInParts && (task != "" || partsAsked > 0) && f.Rule != "due" {
			f.Message += "\n(to be judged in parts in a later round: this round asks something else)"
			fmt.Fprintf(&left, "%s %s\n", f.Rule, f.Where)
			continue
		}
		if sd.tooLarge {
			f.Message += "\n(too large to be put before the agent — the doc with what changed in its sources, or with its sources as they are now: a person judges it)"
			continue
		}
		if task != "" || partsAsked > 0 {
			f.Message += "\n(not put before the agent in this round: over ai-max-calls or the task's size, or the round judges docs in parts)"
			fmt.Fprintf(&left, "%s %s\n", f.Rule, f.Where)
		}
	}
	findings = slices.DeleteFunc(findings, func(f verdict.Finding) bool { return f.Rule == "" })
	// Once each finding says what became of its doc: those left to a person.
	if i, ok := personIssue(findings, repo); ok {
		file := filepath.Join(runDir, "in", "fallback.yaml")
		fallback, err := intent.Read(file)
		if err != nil {
			return fail(err)
		}
		if err := intent.Write(file, append(fallback, i)); err != nil {
			return fail(err)
		}
	}
	if left.Len() > 0 { // the engine runs another round, once this one is applied
		if err := os.WriteFile(filepath.Join(runDir, "in", "more"), []byte(left.String()), 0o644); err != nil {
			return fail(err)
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
		// The docs whose sources went whole into the task: only theirs may
		// have `checked` moved (ADR-0014). The judge refuses any other move.
		whole := []string{}
		for p := range judged {
			for _, m := range []map[string]*suspectDoc{suspects, propagate, stale} {
				if sd := m[p]; sd != nil && sd.whole && !sd.tooLarge {
					whole = append(whole, p)
					break
				}
			}
		}
		sort.Strings(whole)
		if err := writeYAML(filepath.Join(runDir, "in", "read-whole.yaml"), whole); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(task), 0o644); err != nil {
			return fail(err)
		}
	}
	return 0
}

// holdTheRelease has the docs made suspect since the last release hold the
// release until they are judged (ADR-0010): where there is no merge
// request, the release is when docs are made true; on a release tool's pull
// request, before it is merged (ADR-0017). A doc already suspect at the
// release was let through then; one judged without being vouched for, or in
// parts, waits for a person, and does not hold it. Without a release, every
// suspect doc holds the first one; in a shallow clone, which may lack it,
// the release is held.
func holdTheRelease(findings []verdict.Finding, suspects map[string]*suspectDoc, repo string) ([]verdict.Finding, error) {
	rs, err := release.Load(repo)
	if err != nil {
		return findings, err
	}
	tag, err := release.Last(repo, rs.Tags)
	if errors.Is(err, release.ErrShallow) {
		return append(findings, verdict.Finding{Rule: "shallow-clone", Level: "block",
			Message: "the release waits: " + err.Error()}), nil
	}
	if err != nil {
		return findings, err
	}
	since, on := "the first release", ""
	if tag != "" {
		since = tag
	}
	if b := os.Getenv("WORKLINE_RELEASE_BRANCH"); b != "" {
		on = " (this merge request comes from " + b + ", a release tool's branch: it is the release)"
	}
	for i := range findings {
		f := &findings[i]
		sd := suspects[f.Where]
		if f.Rule != "suspect" || sd == nil || held(sd.doc, sd.doc.Judged, repo) || held(sd.doc, sd.doc.JudgedInParts, repo) {
			continue
		}
		if tag != "" && !changedSince(repo, sd.doc, tag) {
			continue
		}
		f.Level = "block"
		f.Message += "\n(made suspect since " + since + ": the release waits until it is judged — by the agent, `workline docs`, or a person moving `checked`" + on + ")"
	}
	return findings, nil
}

// changedSince says whether a source of the doc, in this repository,
// changed after the commit given.
func changedSince(repo string, d *Doc, commit string) bool {
	for _, src := range d.Sources {
		name, p, anchor := splitSource(src)
		if name != "" {
			continue
		}
		if commits, err := changed(repo, commit, "HEAD", p, anchor); err != nil || commits != "" {
			return true
		}
	}
	return false
}

// holdWaitingTasks keeps from the agent, when gardening, the docs a task
// would judge while that task's merge request waits for review: judged
// again, they would only propose what is already proposed. A doc judged
// whole goes to its kind's task; one judged in parts, to `fix`.
func holdWaitingTasks(maps ...map[string]*suspectDoc) {
	if os.Getenv("WORKLINE_EVENT") != "schedule" {
		return
	}
	open, proposed := map[string]bool{}, map[string]bool{}
	for _, t := range strings.Fields(os.Getenv("WORKLINE_OPEN_MERGE_REQUEST_TASKS")) {
		open[t] = true
	}
	for _, t := range strings.Fields(os.Getenv("WORKLINE_PROPOSED_TASKS")) {
		proposed[t] = true
	}
	for i, docs := range maps {
		kind := []string{"suspect", "stale"}[i]
		for _, sd := range docs {
			task := kind
			if sd.tooLarge {
				task = "fix"
			}
			switch {
			case sd.note != "":
			case proposed[task]: // an earlier round of this run judged it
				sd.note, sd.earlier = "(judged in an earlier round of this run)", true
			case open[task]:
				sd.note = "(its task's merge request, workline/documentalist/" + task + ", waits for review: not put before an agent again until it is merged or closed)"
			}
		}
	}
}

// suspectTask writes the question for the agent: each suspect doc, with line
// numbers, what changed in its sources, and the commits its `checked` must
// name once judged. It returns those commits per doc, for the judge.
func suspectTask(suspects map[string]*suspectDoc, s Settings, pl *places, repo string) (string, map[string]map[string]string, error) {
	return docTask(`Kind: suspect

These docs may no longer be true, because something they depend on changed.
For each one, say whether it still is, with a patch: a unified diff with
context lines, whose hunks cite the doc's lines by the numbers shown here.
If the doc is still true, the patch only sets `+"`checked`"+`. If not, it also
fixes what is now wrong, and nothing else.

`, suspects, s, pl, repo)
}

// docTask writes a task putting docs before the agent, each with why it is
// there and what it is judged against, under the header saying the kind.
func docTask(header string, suspects map[string]*suspectDoc, s Settings, pl *places, repo string) (string, map[string]map[string]string, error) {
	header += citeTask
	maxDocs := s.AIMaxCalls
	paths := make([]string, 0, len(suspects))
	for p := range suspects {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	b.WriteString(header)
	judged := map[string]map[string]string{}
	files, err := trackedFiles(repo)
	if err != nil {
		return "", nil, err
	}
	for _, p := range paths {
		if maxDocs > 0 && len(judged) >= maxDocs {
			break
		}
		sd := suspects[p]
		if sd.tooLarge || sd.note != "" {
			continue // a person judges it, or it is judged in parts
		}
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
		if sd.whole {
			fmt.Fprintf(&entry, "## %s\n\nIf you vouch for every sentence it keeps, your patch sets `%s`: the commit this doc is judged against now, not the commit that changed a source.", p, checked)
			if full := want[""]; full != "" {
				fmt.Fprintf(&entry, " If you cannot, it leaves `checked` as it is and sets `judged: %s` instead.", full[:7])
			}
		} else {
			// Not every source could be given whole: nobody can vouch for
			// the doc from this task, so `checked` stays (ADR-0014).
			fmt.Fprintf(&entry, "## %s\n\nNot every source of this doc could be given whole in this task, so `checked` cannot move: your patch fixes what you find wrong, leaves `checked` as it is", p)
			if full := want[""]; full != "" {
				fmt.Fprintf(&entry, " and sets `judged: %s`, the commit it is judged at", full[:7])
			}
			entry.WriteString(". A person reads it against its sources, then moves `checked`.")
		}
		entry.WriteString("\n\nWhy it is here:\n\n")
		for _, w := range sd.why {
			fmt.Fprintf(&entry, "- %s\n", strings.ReplaceAll(w, "\n", "\n  "))
		}
		// Line counts the engine counted (ADR-0014, step 2): told, so the
		// agent neither vouches for them nor counts them itself.
		if off := countsOff(repo, files, sd.doc, string(content)); len(off) > 0 {
			entry.WriteString("\nLine counts this doc states wrong, counted by the engine in the files as they are now. Bring each to the engine's number: that change needs no `claim`, the engine's count being the evidence (a count is not words a file holds, so it cannot be quoted from one). `checked` cannot move while one stands.\n\n")
			for _, c := range off {
				fmt.Fprintf(&entry, "- %s\n", c.message())
			}
		}
		if isAuthority(s, p) {
			fmt.Fprintf(&entry, "\nThis doc is an authority (`truth: doc`): the code follows it, not the reverse. Do not change what it says. "+
				"If the code now disagrees with it, return an `issue` titled %q, saying where they disagree and quoting both; "+
				"your patch then only sets %s, the disagreement being tracked by the issue.\n", "The code disagrees with "+p, map[bool]string{true: "`checked` and `verified`", false: "`judged`"}[sd.whole])
		}
		for _, e := range sd.evidence {
			entry.WriteString("\n" + e + "\n")
		}
		entry.WriteString("\nThe doc as it is now, with its line numbers:\n\n```\n")
		for i, l := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
			fmt.Fprintf(&entry, "%4d | %s\n", i+1, l)
		}
		entry.WriteString("```\n\n")
		if len(header)+entry.Len() > taskChars(s) {
			sd.tooLarge = true
			continue
		}
		if b.Len()+entry.Len() > taskChars(s) {
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
	// The docs whose sources all went whole into the task; a file missing
	// means none did: `checked` moves for no doc (ADR-0014).
	whole := map[string]bool{}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "read-whole.yaml")); err == nil {
		var paths []string
		if err := yaml.Unmarshal(data, &paths); err != nil {
			return fail(err)
		}
		for _, p := range paths {
			whole[p] = true
		}
	}
	var s Settings
	if err := readJSON(filepath.Join(runDir, "in", "settings.json"), &s); err != nil {
		return fail(err)
	}
	useRecords(s)
	useLanguage(s)
	if err := useVersions(s); err != nil {
		return fail(err)
	}
	fallback, err := intent.Read(filepath.Join(runDir, "in", "fallback.yaml"))
	if err != nil {
		return fail(err)
	}
	var refused, reported, withheld []verdict.Finding
	var judgedFix *judgedFixes
	inParts := map[string]string{} // the docs judged in parts, by the commit they were judged at
	patched, partly := map[string]bool{}, map[string]bool{}
	resolved := map[string]bool{} // "rule where" of budget problems a condense patch resolves
	var mergedPair []string       // the two docs a duplicates patch merged
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "condense.yaml")); err == nil {
		var c condenseTask
		if err := yaml.Unmarshal(data, &c); err != nil {
			return fail(err)
		}
		var material string
		refused, material, err = judgeCondense(repo, s, &c, intents, fallback)
		if err != nil {
			return fail(err)
		}
		if material != "" { // the engine asks a judge what no check can tell
			q := map[string]string{"material": material,
				"question": "Does each of these cards hold one concept, the one its title names? A sentence explaining that concept by what it depends on stays within it; " +
					"a second concept is a part that could stand as a card of its own, under a title of its own."}
			if err := writeYAML(filepath.Join(runDir, "out", "judge.yaml"), q); err != nil {
				return fail(err)
			}
		}
		for _, k := range c.Keys {
			resolved[k] = len(refused) == 0 && proposedPatch(intents, fallback)
		}
	} else if data, err := os.ReadFile(filepath.Join(runDir, "in", "merge-card.yaml")); err == nil {
		var m mergeCardTask
		if err := yaml.Unmarshal(data, &m); err != nil {
			return fail(err)
		}
		if refused, err = judgeMergeCard(repo, s, &m, intents, fallback); err != nil {
			return fail(err)
		}
		resolved["card-too-short "+m.Card] = len(refused) == 0 && proposedPatch(intents, fallback)
	} else if data, err := os.ReadFile(filepath.Join(runDir, "in", "dedupe.yaml")); err == nil {
		var d dedupeTask
		if err := yaml.Unmarshal(data, &d); err != nil {
			return fail(err)
		}
		if refused, err = judgeDedupe(repo, s, &d, intents, fallback); err != nil {
			return fail(err)
		}
		if len(refused) == 0 && proposedPatch(intents, fallback) {
			mergedPair = d.Docs
		}
	} else {
		kind, _ := os.ReadFile(filepath.Join(runDir, "in", "task-kind"))
		if strings.TrimSpace(string(kind)) == "sources" {
			refused, patched, err = judgeSources(repo, s, judged, intents, fallback)
		} else {
			if data, err := os.ReadFile(filepath.Join(runDir, "in", "in-parts.yaml")); err == nil {
				if err := yaml.Unmarshal(data, &inParts); err != nil {
					return fail(err)
				}
			}
			var j patchJudgement
			j, err = judgePatches(repo, s, judged, inParts, whole, intents, fallback, strings.TrimSpace(string(kind)) == "propagate")
			refused, patched, partly, judgedFix, reported, withheld = j.refused, j.patched, j.partly, j.fix, j.reported, j.withheld
			// The patches narrowed to what holds are what the engine applies
			// (ADR-0014, step 4).
			if err == nil && len(refused) == 0 && j.intents != nil {
				intents = j.intents
				err = writeIntents(filepath.Join(runDir, "out", "intentions.yaml"), intents)
			}
		}
		if err != nil {
			return fail(err)
		}
	}
	// A comment given as the only reason for a fix is reported for a person,
	// whatever the agent answers when asked again (ADR-0014, step 2): kept
	// beside the run's answers until the run ends.
	commentsFile := filepath.Join(runDir, "out", "comments-disagree.yaml")
	var comments []verdict.Finding
	if data, err := os.ReadFile(commentsFile); err == nil {
		if err := yaml.Unmarshal(data, &comments); err != nil {
			return fail(err)
		}
	}
	if len(reported) > 0 {
		comments = sortedFindings(append(comments, reported...))
		if err := writeYAML(commentsFile, comments); err != nil {
			return fail(err)
		}
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
	blocking, due := 0, 0
	var kept []verdict.Finding
	said := map[string]bool{}
	for _, f := range findings {
		if _, ok := inParts[f.Where]; ok && strings.Contains(f.Message, partsFixAsked) {
			// What the fix call made of what the parts found, now known.
			if n := judgedFix.placesFixed(f.Where); n > 0 {
				f.Message = strings.Replace(f.Message, partsFixAsked, "what they found wrong is fixed in this run, in "+nPlaces(n), 1)
			} else {
				f.Message = strings.Replace(f.Message, partsFixAsked, "but the fix changed nothing of what they found wrong: a person reads those places", 1)
			}
		}
		if v, ok := patched[f.Where]; ok && !v && (f.Rule == "suspect" || f.Rule == "stale") {
			f.Level = "" // judged, its header says when: a person's now, as one judged in parts
			if !said[f.Where] {
				said[f.Where] = true
				f.Message += "\n" + notVouched(judgedFix.placesFixed(f.Where), partly[f.Where], whole[f.Where])
			}
		}
		if (f.Rule == "suspect" || f.Rule == "stale" || f.Rule == "due" || f.Rule == "no-sources") && patched[f.Where] || resolved[f.Rule+" "+f.Where] ||
			f.Rule == "duplicate" && len(mergedPair) == 2 && contains(mergedPair, f.Where) && strings.Contains(f.Message, " "+otherOf(mergedPair, f.Where)+" ") {
			continue // judged and patched, or condensed, in this run
		}
		if judgedFix.countFixed(f) {
			f.Message += " (fixed in this run)"
		}
		if f.Level == "block" {
			blocking++
			f.Level = ""
		}
		if f.Rule == "due" {
			due++
		}
		kept = append(kept, f)
	}
	// A version the fixes replace, still said beside them: reported, for a
	// person (ADR-0014, step 2).
	if judgedFix != nil {
		kept = append(kept, valuesLeft(judgedFix.fixes, judgedFix.after, judgedFix.files, func(p string) string {
			data, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
			return string(data)
		})...)
	}
	kept = append(kept, withheld...)
	kept = append(kept, comments...)
	if proposedPatch(intents, fallback) || len(fallback) > 0 {
		outcome := outcomes(judged, inParts, patched, partly, judgedFix)
		if err := writeYAML(filepath.Join(runDir, "out", "merge-request.yaml"), mergeRequest(runDir, judged, proposedPatch(intents, fallback), withheld, outcome)); err != nil {
			return fail(err)
		}
	}
	v := verdict.Verdict{Status: verdict.Pass, Findings: kept}
	// Nothing of the answer was refused: what blocks — a release held, docs
	// due, a check that could not run — is not the answer's, and asking the
	// agent again would change none of it.
	switch {
	case due > 0 && due == blocking:
		v.Status, v.Summary, v.Final = verdict.Block, fmt.Sprintf("%d docs due at this moment are not up to date yet: they wait for an agent, or a person", due), true
	case blocking > 0:
		v.Status, v.Summary, v.Final = verdict.Block, "some sources could not be judged, or some checks could not run", true
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

func sortedDocs(m map[string]*suspectDoc) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
	files, err := git(repo, "ls-files", "-z")
	if err != nil {
		return t, err
	}
	for _, f := range pathList(files) {
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

// trackedFiles lists the files the repository tracks.
func trackedFiles(repo string) (map[string]bool, error) {
	out := map[string]bool{}
	files, err := git(repo, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	for _, f := range pathList(files) {
		if f != "" {
			out[f] = true
		}
	}
	return out, nil
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

// rangeFiles returns the files the commits of the run's `range` input
// touched; ranged is false when the run was given no range.
func rangeFiles(runDir, repo string) (touched map[string]bool, ranged bool, err error) {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "input", "range"))
	if os.IsNotExist(err) || err == nil && strings.TrimSpace(string(data)) == "" {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	out, err := git(repo, append([]string{"log", "--name-only", "--format=commit %H"}, gitrange.Args(string(data))...)...)
	if err != nil {
		return nil, false, err
	}
	// A doc whose commits changed only its header, who checked it, is not
	// touched: what the docs following it say did not change.
	touched = map[string]bool{}
	commit := ""
	for _, f := range strings.Split(out, "\n") {
		if !strings.HasPrefix(f, "commit ") {
			f = unquotePath(f)
		}
		switch {
		case strings.HasPrefix(f, "commit "):
			commit = strings.TrimPrefix(f, "commit ")
		case f == "" || touched[f]:
		case strings.HasSuffix(f, ".md"):
			before, err1 := git(repo, "show", commit+"^:"+f)
			after, err2 := git(repo, "show", commit+":"+f)
			touched[f] = err1 != nil || err2 != nil || body(before) != body(after)
		default:
			touched[f] = true
		}
	}
	return touched, true, nil
}

// rangeOf is the range of commits the run was given.
func rangeOf(runDir string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", "range"))
	return strings.TrimSpace(string(data))
}

// addedFiles returns the files the commits of the run's range added.
func addedFiles(runDir, repo string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(runDir, "in", "input", "range"))
	if err != nil {
		return nil, err
	}
	out, err := git(repo, append([]string{"log", "--diff-filter=A", "--name-only", "--format="}, gitrange.Args(string(data))...)...)
	if err != nil {
		return nil, err
	}
	added := map[string]bool{}
	for _, f := range strings.Split(out, "\n") {
		if f != "" {
			added[unquotePath(f)] = true
		}
	}
	return added, nil
}

// touches says whether one of the files is path, or under it.
func touches(files map[string]bool, path string) bool {
	dir := strings.TrimSuffix(path, "/")
	for f, yes := range files {
		if yes && (f == dir || strings.HasPrefix(f, dir+"/")) {
			return true
		}
	}
	return false
}

// undocumented reports, by folder, the files matching the documented globs
// that no doc names in its sources: those the commits add, given a range,
// else all of them.
func undocumented(t Tree, docs []*Doc, globs []string, added map[string]bool, ranged bool) []verdict.Finding {
	if len(globs) == 0 {
		return nil
	}
	var sources []string
	for _, d := range docs {
		for _, src := range d.Sources {
			if name, path, _ := splitSource(src); name == "" {
				sources = append(sources, strings.TrimSuffix(path, "/"))
			}
		}
	}
	byDir := map[string][]string{}
	for f := range t.Files {
		if ranged && !added[f] || !matchAny(globs, f) {
			continue
		}
		covered := false
		for _, s := range sources {
			covered = covered || f == s || strings.HasPrefix(f, s+"/")
		}
		if !covered {
			byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], filepath.Base(f))
		}
	}
	var out []verdict.Finding
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		files := byDir[d]
		sort.Strings(files)
		out = append(out, verdict.Finding{Rule: "undocumented", Where: d,
			Message: fmt.Sprintf("no doc names these in its sources, so no doc goes wrong when they change: %s; add them to the sources of the doc that describes them, or write one", strings.Join(files, ", "))})
	}
	return out
}

// docTouched says whether the commits touched one of the doc's own sources.
// Such a doc is judged whole, every changed source with it: moving
// `checked` vouches for all of them.
func docTouched(d *Doc, touched map[string]bool) bool {
	for _, src := range d.Sources {
		if name, path, _ := splitSource(src); name == "" && touches(touched, path) {
			return true
		}
	}
	return false
}

// hasSuspect says whether a doc is reported suspect already.
func hasSuspect(f []verdict.Finding, where string) bool {
	for _, x := range f {
		if x.Rule == "suspect" && x.Where == where {
			return true
		}
	}
	return false
}

// Coverage sorts the docs under the docs globs: those declaring their
// sources, which alone can be found suspect; those declaring none
// (`sources: []`), which describe no code; and those declaring nothing.
func Coverage(repo string, globs []string) (tracked, none, untracked []string, err error) {
	t, err := loadTree(repo, globs)
	if err != nil {
		return nil, nil, nil, err
	}
	for p, content := range t.Docs {
		if !matchAny(globs, p) {
			continue
		}
		declared, sources := declaresSources(content)
		switch {
		case len(sources) > 0:
			tracked = append(tracked, p)
		case declared:
			none = append(none, p)
		default:
			untracked = append(untracked, p)
		}
	}
	sort.Strings(tracked)
	sort.Strings(none)
	sort.Strings(untracked)
	return tracked, none, untracked, nil
}

// declaresSources says whether a doc's header has a `sources` key, even an
// empty one, and what it lists. A header that does not parse declares nothing.
func declaresSources(content string) (bool, []string) {
	block, n := header(content)
	if n == 0 {
		return false, nil
	}
	var fm struct {
		Sources *[]string `yaml:"sources"`
	}
	if yaml.Unmarshal([]byte(block), &fm) != nil || fm.Sources == nil {
		return false, nil
	}
	return true, *fm.Sources
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

// pathList splits the paths git lists with -z, which it never quotes: a
// path that is not ASCII, quoted, was not found and its doc skipped in
// silence (docs/research/documentalist-genericity.md).
func pathList(out string) []string {
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

// unquotePath reads a path git listed one a line: with core.quotePath off
// (every git call here), only a path holding a quote, a backslash or a
// control character is still quoted, in C's way.
func unquotePath(p string) string {
	if len(p) > 1 && p[0] == '"' && p[len(p)-1] == '"' {
		if u, err := strconv.Unquote(p); err == nil {
			return u
		}
	}
	return p
}

func git(dir string, args ...string) (string, error) {
	return gitIn(dir, "", args...)
}

// gitIn runs git with stdin.
func gitIn(dir, stdin string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.quotePath=off"}, args...)...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// zeroContextDiff is the diff from old to now of the file at path, with no
// line of context: it applies wherever those lines still read as they did.
func zeroContextDiff(path, old, now string) (string, error) {
	return unifiedDiff(path, old, now, 0)
}

// unifiedDiff is the diff from old to now of the file at path, with that
// many lines of context; empty when they are the same.
func unifiedDiff(path, old, now string, context int) (string, error) {
	dir, err := os.MkdirTemp("", "workline-diff-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	for side, content := range map[string]string{"a": old, "b": now} {
		file := filepath.Join(dir, side, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return "", err
		}
	}
	cmd := exec.Command("git", "-c", "core.quotePath=off", "diff", "--no-index", "--no-prefix", "--no-color", fmt.Sprintf("-U%d", context), "a/"+path, "b/"+path)
	cmd.Dir = dir
	out, err := cmd.Output()
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		err = nil // the files differ, as they should
	}
	return string(out), err
}

// writeIntents writes a set of intentions, an empty one as such.
func writeIntents(path string, in []intent.Intention) error {
	if len(in) == 0 {
		return os.WriteFile(path, []byte("[]\n"), 0o644)
	}
	return intent.Write(path, in)
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

// withExternalLinks replaces the count of links to other sites with what
// lychee finds of them; without lychee, the count says it is missing.
func withExternalLinks(repo string, tree Tree, problems []Problem) ([]Problem, error) {
	docs := make([]string, 0, len(tree.Docs))
	for p := range tree.Docs {
		docs = append(docs, p)
	}
	found, ok, err := ExternalLinks(repo, docs)
	if err != nil {
		return nil, err
	}
	var out []Problem
	for _, p := range problems {
		if p.Rule != "links-not-checked" {
			out = append(out, p)
		} else if !ok {
			p.Message = fmt.Sprintf("%d link(s) to other sites not checked: lychee is not installed; to install it: %s", p.Size, tools.Lookup("lychee").Install())
			out = append(out, p)
		}
	}
	return append(out, found...), nil
}

// isAuthority says whether the code follows this doc (settings.truth.doc):
// when they disagree, an issue is opened, and the doc is never rewritten.
func isAuthority(s Settings, path string) bool {
	return matchAny(s.Truth.Doc, path)
}

// checklist is the comment that lists, on a merge request run without an
// agent, the docs a person must read again: one comment, edited on each run.
// A doc leaves it when its `checked` moves; with none left, the comment says
// so, and none is opened when there never was one.
func checklist(findings []verdict.Finding) (intent.Intention, bool) {
	if os.Getenv("WORKLINE_AI") != "none" || !strings.HasPrefix(os.Getenv("WORKLINE_TARGET"), "merge-request:") {
		return intent.Intention{}, false
	}
	var now, later strings.Builder
	for _, f := range findings {
		why, rest, _ := strings.Cut(f.Message, "\n")
		var commits []string // the lines after "… changed since it was checked:" start with a commit
		for _, l := range strings.Split(rest, "\n") {
			if fs := strings.Fields(l); len(fs) > 0 && !strings.HasPrefix(l, "(") {
				commits = append(commits, fs[0])
			}
		}
		if len(commits) > 0 {
			why = strings.TrimSuffix(why, ":") + " (" + strings.Join(commits, ", ") + ")"
		}
		switch f.Rule {
		case "suspect":
			fmt.Fprintf(&now, "- [ ] `%s` — %s\n", f.Where, why)
		case "pending":
			fmt.Fprintf(&later, "- `%s` — %s\n", f.Where, why)
		}
	}
	value := map[string]any{"sticky": "checklist"}
	if now.Len() == 0 && later.Len() == 0 {
		value["body"] = "**Docs to check**: none left."
		value["update-only"] = true
		return intent.Intention{Kind: "comment", Value: value}, true
	}
	var b strings.Builder
	b.WriteString("**Docs to check** — the documentalist ran without an agent. Read each doc against this change; " +
		"once it is right, move its `checked` to the commit it was read against. This list is rebuilt on each push.\n\n")
	if now.Len() > 0 {
		b.WriteString(now.String())
	}
	if later.Len() > 0 {
		b.WriteString("\nDue later, not in this merge request:\n\n" + later.String())
	}
	value["body"] = b.String()
	return intent.Intention{Kind: "comment", Value: value}, true
}

// notVouched says, for a doc the agent judged without vouching for it, what
// this run did: fixed in so many places, or nothing found wrong; applied in
// part; and why `checked` stays. Never "fixed" for a patch recording only
// `judged` (workline's first nightly, 2026-10-02).
func notVouched(n int, partly, whole bool) string {
	switch {
	case partly && n > 0:
		return "(fixed in " + nPlaces(n) + ", applied in part: the places refused are left as they were, each said below, and `checked` stays — a person fixes them against the sources, then moves `checked`)"
	case partly:
		return "(nothing applied: every place the fix changed was refused, each said below, and left as it was; `checked` stays and `judged` records when — a person fixes them against the sources, then moves `checked`)"
	}
	what := "nothing found wrong, not vouched for: the agent changed nothing in the doc"
	if n > 0 {
		what = "fixed in " + nPlaces(n) + ", not vouched for: what the agent found wrong is fixed in this run"
	}
	if whole {
		return "(" + what + ", but it could not confirm every sentence against the sources, so `checked` stays and `judged` records when; its note says what — a person reads it, then moves `checked`)"
	}
	return "(" + what + ", but not every source of the doc could be given whole in the task, so `checked` stays and `judged` records when — a person reads it against them, then moves `checked`)"
}

// outcomes says, for the merge request a gardening task opens, what became
// of each doc it judged: fixed and in how many places, or nothing found
// wrong; `checked` moved, or `judged` recorded.
func outcomes(judged map[string]map[string]string, inParts map[string]string, patched, partly map[string]bool, fix *judgedFixes) map[string]string {
	out := map[string]string{}
	fixed := func(p string) string {
		if n := fix.placesFixed(p); n > 0 {
			return "fixed in " + nPlaces(n)
		}
		return ""
	}
	for p := range inParts {
		switch f := fixed(p); {
		case f != "":
			out[p] = "judged in parts, " + f + "; `checked` not moved"
		case judged[p] != nil:
			out[p] = "judged in parts; the fix changed nothing of what they found wrong, for a person; `checked` not moved"
		default:
			out[p] = "judged in parts, nothing found wrong; `checked` not moved"
		}
	}
	for p := range judged {
		if _, ok := inParts[p]; ok {
			continue
		}
		v, ok := patched[p]
		f := fixed(p)
		switch {
		case !ok:
			out[p] = "left as it was"
		case v && f != "":
			out[p] = f + ", `checked` moved"
		case v:
			out[p] = "still true, `checked` moved"
		case f != "":
			out[p] = f + ", `judged` recorded, `checked` not moved"
		default:
			out[p] = "nothing found wrong, `judged` recorded, `checked` not moved"
		}
		if partly[p] {
			out[p] += "; places left for a person, below"
		}
	}
	return out
}

// otherOf is the doc of a pair that is not p.
func otherOf(pair []string, p string) string {
	if pair[0] == p {
		return pair[1]
	}
	return pair[0]
}

// mergeRequest describes, for a run that opens one (ADR-0006), the merge
// request its patches go to: a key per task, so running the task again
// updates it, and a title a commit can carry.
func mergeRequest(runDir string, judged map[string]map[string]string, byAgent bool, withheld []verdict.Finding, outcome map[string]string) map[string]string {
	kind := "suspect"
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "task-kind")); err == nil {
		kind = strings.TrimSpace(string(data))
	}
	if !byAgent { // only what the role regenerates itself
		return map[string]string{"key": "derived", "title": "docs: regenerate the derived blocks",
			"body": "Regenerated by workline's documentalist from the commands the project names in `settings.derive`.\n"}
	}
	var docs []string
	for p := range judged {
		docs = append(docs, p)
	}
	for p := range outcome { // a doc judged in parts with nothing to fix: its header only
		if _, ok := judged[p]; !ok {
			docs = append(docs, p)
		}
	}
	sort.Strings(docs)
	key, title := kind, "docs: bring the docs in line with the code"
	switch kind {
	case "stale":
		title = "docs: read again the docs not confirmed for too long"
	case "propagate":
		title = "docs: bring the product docs up to date for the release"
	case "condense", "split":
		var c condenseTask
		if data, err := os.ReadFile(filepath.Join(runDir, "in", "condense.yaml")); err == nil && yaml.Unmarshal(data, &c) == nil {
			docs, key, title = []string{c.Doc}, kind+" "+c.Doc, "docs: condense "+c.Doc
			if c.Split {
				title = "docs: split " + c.Doc + " into cards"
			}
		}
	case "duplicates":
		var d dedupeTask
		if data, err := os.ReadFile(filepath.Join(runDir, "in", "dedupe.yaml")); err == nil && yaml.Unmarshal(data, &d) == nil {
			docs, key, title = d.Docs, "duplicates "+strings.Join(d.Docs, " "), "docs: keep a repeated passage in one place"
		}
	case "merge-card":
		var m mergeCardTask
		if data, err := os.ReadFile(filepath.Join(runDir, "in", "merge-card.yaml")); err == nil && yaml.Unmarshal(data, &m) == nil {
			docs, key, title = []string{m.Card}, "merge-card "+m.Card, "docs: merge "+m.Card+" into the card it belongs with"
		}
	}
	if len([]rune(title)) > 72 {
		title = "docs: " + kind + ", proposed by the documentalist"
	}
	var body strings.Builder
	when := "when gardening"
	if os.Getenv("WORKLINE_EVENT") == "release" { // a release tool's pull request waits for it (ADR-0017)
		when = "for the release"
	}
	fmt.Fprintf(&body, "Proposed by workline's documentalist, %s (task: %s).\n", when, kind)
	if len(docs) > 0 {
		body.WriteString("\nDocs:\n\n")
		for _, d := range docs {
			if o := outcome[d]; o != "" {
				fmt.Fprintf(&body, "- `%s` — %s\n", d, o)
			} else {
				fmt.Fprintf(&body, "- `%s`\n", d)
			}
		}
	}
	// The places a fix was refused in, left as they were: the person
	// reviewing reads them here (ADR-0014, step 4).
	if len(withheld) > 0 {
		body.WriteString("\nLeft as they were, for a person — the fix refused there, the rest applied, `checked` not moved:\n\n")
		for _, f := range withheld {
			fmt.Fprintf(&body, "- `%s` (%s): %s\n", f.Where, f.Rule, f.Message)
		}
	}
	body.WriteString("\nRunning the same task again updates this merge request; commits added to its branch by hand are overwritten.\n")
	return map[string]string{"key": key, "title": title, "body": body.String()}
}

// trackingIssue is the one issue listing the docs due later — at the release,
// by default — kept in place on the forge when gardening, which sees the main
// branch: a merge request's run would list what is not merged yet. With none
// due, it says so in an issue already open, and opens none.
func trackingIssue(findings []verdict.Finding) (intent.Intention, bool) {
	if os.Getenv("WORKLINE_FORGE") == "" || os.Getenv("WORKLINE_EVENT") != "schedule" {
		return intent.Intention{}, false
	}
	var list strings.Builder
	for _, f := range findings {
		if f.Rule == "pending" {
			why, _, _ := strings.Cut(f.Message, "\n")
			fmt.Fprintf(&list, "- `%s` — %s\n", f.Where, why)
		}
	}
	value := map[string]any{"title": "Docs due at the next release", "sticky": true}
	if list.Len() == 0 {
		value["body"], value["update-only"] = "Nothing is due: every doc that follows another is up to date.", true
	} else {
		value["body"] = "These docs follow docs that changed, and are brought up to date at the moment their edge names — the release, by default — when the documentalist runs at the release (`workline route release`), before the release tool tags; until then, the release waits for them.\n\n" + list.String()
	}
	return intent.Intention{Kind: "issue", Value: value}, true
}

// forAPerson says whether a finding leaves its doc to a person: every note
// that does ends by saying what "a person" does — reads it, judges it,
// fixes the places refused — then moves `checked`.
func forAPerson(f verdict.Finding) bool {
	switch f.Rule {
	case "suspect", "due", "stale":
		return strings.Contains(f.Message, "a person ")
	}
	return false
}

// personIssue keeps, when gardening with a forge, one issue listing the
// docs only a person can clear, each with why and what to do: what the
// machine cannot do is said where the person looks, not left in a job's
// log (principle 14). They stay suspect until cleared; those not judged
// since their sources changed hold the next release (holdTheRelease).
func personIssue(findings []verdict.Finding, repo string) (intent.Intention, bool) {
	if os.Getenv("WORKLINE_FORGE") == "" || os.Getenv("WORKLINE_EVENT") != "schedule" {
		return intent.Intention{}, false
	}
	why := map[string]string{}
	var docs []string
	for _, f := range findings {
		if !forAPerson(f) {
			continue
		}
		doc, _, _ := strings.Cut(f.Where, "#")
		if _, seen := why[doc]; !seen {
			docs = append(docs, doc)
		}
		lines := strings.Split(f.Message, "\n")
		why[doc] = strings.Trim(lines[len(lines)-1], "()")
	}
	sort.Strings(docs)
	value := map[string]any{"title": "Docs waiting for a person", "sticky": true}
	if len(docs) == 0 {
		value["body"], value["update-only"] = "No doc waits for a person: the documentalist judged every one it could.", true
		return intent.Intention{Kind: "issue", Value: value}, true
	}
	head := "HEAD"
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "HEAD").Output(); err == nil {
		head = strings.TrimSpace(string(out))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The documentalist could not clear these docs by itself: they stay suspect until a person does, and those made suspect since the last release, unless judged since, hold the next one. For each one: read it whole against its sources (the ones its header names), fix what is wrong, then set `checked: %s` and `verified: human:<you>` in its header, and commit — on a branch, through a pull request. A doc too large for the agent can also be split or condensed, or its `sources` narrowed to the files it describes, so the agent judges it next time.\n\n", head)
	for _, d := range docs {
		fmt.Fprintf(&b, "- [ ] `%s` — %s\n", d, why[d])
	}
	value["body"] = b.String()
	return intent.Intention{Kind: "issue", Value: value}, true
}
