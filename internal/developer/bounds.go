package developer

// The bounds at apply (ADR-0054, step 2): post, with no agent, leaves out
// each patch reaching a gate's file, a file that steers agents, a frozen
// test or a path out of the repository, and the agent's labels and
// comments but an objection; roles/developer/docs/bounds.md says why. The
// issue's Scope is a hint, not a fence (ADR-0065, change 2): a file
// beyond it is applied and listed on the pull request.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/steer"
	"github.com/JN0V/workline/internal/testdef"
	"github.com/JN0V/workline/internal/tester"
	"github.com/JN0V/workline/internal/work"
)

const boundsFile = "bounds.json"

// NoteMax is the most bytes of the one note the developer may post on its
// own issue.
const NoteMax = 4000

// noteOf is the body of the note bounded kept among the proposals after the
// claim: a comment on the issue, with no merge request nor objection.
func noteOf(kept []intent.Intention, issue int) string {
	for _, in := range kept {
		m, ok := in.Value.(map[string]any)
		if !ok || in.Kind != "comment" || m["merge-request"] != nil || fmt.Sprint(m["issue"]) != strconv.Itoa(issue) {
			continue
		}
		if body, ok := m["body"].(string); ok {
			return body
		}
	}
	return ""
}

// Bounds are what pre read of the issue and the forge for post to hold the
// patches to.
type Bounds struct {
	Issue int `json:"issue"`
	// Scope: the paths the issue's Scope names and the product owner's
	// sources for it.
	Scope []string `json:"scope"`
	// Kind: the issue's kind, as its labels say it; its commits' type.
	Kind string `json:"kind,omitempty"`
	// Scenarios: the names of the issue's Validation scenarios, the one
	// an objection names among them (#913).
	Scenarios []string `json:"scenarios,omitempty"`
	// Covers: each scenario's tests, as the tester's agent declared them
	// on its Draft's record: an objection naming a test is on the
	// scenario it covers (ADR-0065, change 4).
	Covers map[string][]string `json:"covers,omitempty"`
	// Frozen: each test the tester froze on an open Draft, by its issue;
	// a file two Drafts froze, the issue's own Draft's.
	Frozen map[string]int `json:"frozen,omitempty"`
	// Blobs: each frozen test's blob, as the tester's record names it.
	Blobs map[string]string `json:"blobs,omitempty"`
	// Unread: the issue's tester Drafts whose record could not be read,
	// their frozen tests unknown.
	Unread []int `json:"unread,omitempty"`
	// Draft is the tester's open Draft for the issue, 0 when none, and
	// Head its branch's commit, where an objection goes on a line.
	Draft int    `json:"draft,omitempty"`
	Head  string `json:"head,omitempty"`
	// RedUnproven: why the judge did not settle the Draft's red proof
	// (ADR-0065), as its record says; its pull request says it too.
	RedUnproven string `json:"red-unproven,omitempty"`
	// Proof: the kind of proof the tester recorded in place of a Draft
	// (ADR-0059, decision 4), when it did.
	Proof *tester.Proof `json:"proof,omitempty"`
	// TestPaths, TestHomes: the tester's test-paths and test-homes, the
	// project's one definition of a test file (ADR-0065): a new test file
	// whose code lives beside the Scope is not beyond it (ADR-0063, §9).
	// Unset, or its settings unread, the generic ones (testdef).
	TestPaths []string       `json:"test-paths,omitempty"`
	TestHomes []testdef.Home `json:"test-homes,omitempty"`
	// HomesIgnored: why the project's test-homes were ignored, the
	// generic ones read in their place; said on the bounds' verdict.
	HomesIgnored string `json:"homes-ignored,omitempty"`
	// Beside: the new test files an earlier step or round wrote beside
	// the Scope, as its record names them: on its branch now, still
	// beside it, whatever the commit the rules are read at holds (a
	// review round in CI runs on the pull request's head).
	Beside []string `json:"beside,omitempty"`
}

// Refusal is a file a patch reached and post refused, and why: kept in the
// record the claim's comment holds, for the auditor (ADR-0046).
type Refusal struct {
	File string `json:"file"`
	Rule string `json:"rule"`
	// Detail is why the bounds could not be read, for bounds-unread.
	Detail string `json:"detail,omitempty"`
}

// readBounds reads, on the forge, what bounds the issue's patches.
func readBounds(f forge.Forge, bl forge.Backlog, repo string, is forge.Issue) (Bounds, error) {
	files := repoFiles(repo)
	// The Scope's paths and globs, nothing guessed from a name (ADR-0065,
	// change 2): a hint, a file beyond it listed on the pull request.
	b := Bounds{Issue: is.ID, Scope: inRepo(scopePaths(is.Body, files), files), Frozen: map[string]int{}, Blobs: map[string]string{}}
	for _, sc := range work.Scenarios(work.ByMeaning(is.Body)[work.Validation].Text) {
		b.Scenarios = append(b.Scenarios, sc.Name)
	}
	// The project's one definition, as every role reads it (ADR-0065).
	tests, err := role.TestDef(os.Getenv("WORKLINE_ROLES_DIR"), repo)
	b.TestPaths, b.TestHomes = tests.Paths, tests.Homes
	if errors.Is(err, testdef.ErrHomes) {
		b.HomesIgnored = err.Error()
	}
	comments, err := bl.Comments(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil {
		return b, err
	}
	if st, found, err := backlog.ReadState(comments, "product-owner"); found && err == nil {
		for _, s := range st.Sources {
			if s, ok := scopeEntry(s); ok && !slices.Contains(b.Scope, s) {
				b.Scope = append(b.Scope, s)
			}
		}
	}
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return b, err
	}
	for _, mr := range mrs {
		n := testerIssue(mr.Branch)
		if n == 0 {
			continue
		}
		r, ok, err := tester.ReadRecord(f, mr.ID)
		if err != nil {
			return b, err
		}
		if !ok {
			if n == is.ID {
				b.Unread = append(b.Unread, mr.ID)
			}
			continue
		}
		for file, blob := range r.Files {
			if r.Issue != is.ID && b.Frozen[file] == is.ID {
				continue // a file two Drafts share: the issue's own Draft's blob
			}
			b.Frozen[file], b.Blobs[file] = r.Issue, blob
		}
		if n == is.ID {
			b.Draft, b.RedUnproven, b.Covers = mr.ID, r.RedUnproven, r.Covers
			for _, ref := range []string{"refs/remotes/origin/" + mr.Branch, "refs/heads/" + mr.Branch} {
				if h, err := gitOut(repo, "rev-parse", "-q", "--verify", ref+"^{commit}"); err == nil && h != "" {
					b.Head = h
					break
				}
			}
		}
	}
	return b, nil
}

// testerIssue is the issue a tester's branch is for, 0 when not one.
func testerIssue(branch string) int {
	var n int
	if _, err := fmt.Sscanf(strings.TrimPrefix(branch, tester.TestsBranch), "%d", &n); err != nil || !strings.HasPrefix(branch, tester.TestsBranch) {
		return 0
	}
	return n
}

// ciFiles are the files and folders a CI runs from: GitHub's workflows and
// actions, GitLab's pipeline, workline's own settings and gates.
var ciFiles = []string{".workline", ".github/workflows", ".github/actions", ".gitlab-ci.yml", ".gitlab/ci"}

// gateFiles are the files a gate's checks run or read their config from,
// as .workline/config.yaml declares them: a word of a check's command
// that is a file of the repository, and its `config`. A config that
// cannot be read is an error: no gate's file is known then. At rev, a
// commit, they are read there, never in the checkout; "" reads the
// checkout.
func gateFiles(repo, rev string) ([]string, error) {
	var cfg struct {
		Gates map[string]struct {
			Checks []struct {
				Run    string   `yaml:"run"`
				Config []string `yaml:"config"`
			} `yaml:"checks"`
		} `yaml:"gates"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
	if rev != "" {
		data, err = readAt(repo, rev, ".workline/config.yaml")
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf(".workline/config.yaml: %w", err)
	}
	var out []string
	for _, g := range cfg.Gates {
		for _, c := range g.Checks {
			for _, f := range c.Config {
				if f, ok := repoPath(f); ok {
					out = append(out, f)
				}
			}
			for _, w := range strings.Fields(c.Run) {
				w = strings.Trim(w, `"'`)
				if kindAt(repo, rev, w) == "file" {
					if f, ok := repoPath(w); ok {
						out = append(out, f)
					}
				}
			}
		}
	}
	return out, nil
}

// repoPath is p cleaned, its slashes forward, when it names a file or a
// folder inside the repository other than its root.
func repoPath(p string) (string, bool) {
	p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
	if p == "." || p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p) || filepath.IsAbs(p) {
		return "", false
	}
	return p, true
}

// scopeEntry is a Scope's or a source's path, when it bounds less than
// the whole repository: "." or "./" would put every file in bounds.
func scopeEntry(p string) (string, bool) {
	return repoPath(p)
}

// inRepo is scope, each path a file of files names in its place: `parts.go`
// the one file of that name (work.InRepo); a folder, or a name several
// files have, left as written.
func inRepo(scope, files []string) []string {
	out := slices.Clone(scope)
	for i, p := range out {
		if f := work.InRepo(p, files); f != "" {
			out[i] = f
		}
	}
	return out
}

// under says whether p is one of roots or in one of them, or matches one
// that is a pattern, as the tester reads a Scope (work.Files, #801).
func under(p string, roots []string) bool {
	return slices.ContainsFunc(roots, func(r string) bool {
		r = path.Clean(r)
		if work.Pattern(r) && pathglob.Match(r, p) {
			return true
		}
		return r != "." && (p == r || strings.HasPrefix(p, r+"/"))
	})
}

// tests is the project's one definition of a test file (ADR-0065).
func (b Bounds) tests() testdef.Def { return testdef.From(b.TestPaths, b.TestHomes) }

// besideTest says whether p is a test (the project's definition,
// ADR-0065) in a scope file's folder or under a scope folder, new or not:
// `foo_test.go` beside `foo.go`. A test elsewhere is beside the Scope only
// when new (newTestBeside): an existing one, a fixture a frozen test may
// read, is beyond it. A folder's own folder is not its neighbourhood: a
// Scope `src` holds no test at the root.
func besideTest(p string, scope []string, isDir func(string) bool, tests testdef.Def) bool {
	if !tests.Test(p) {
		return false
	}
	d := path.Dir(p)
	return slices.ContainsFunc(scope, func(s string) bool { return d == path.Dir(s) && !isDir(s) || under(d, []string{s}) })
}

// checker says, for a file a patch touches, the rule that refuses it, ""
// when it may be touched.
type checker struct {
	b     Bounds
	at    string // the commit the rules are read at; "" the checkout
	gates []string
	repo  string
	// steer says whether a path never merges alone: CODEOWNERS's list,
	// else the files that steer agents (internal/steer).
	steer func(string) bool
	// unread is why the gates or the steering files could not be read:
	// every file is refused then, rather than one let past.
	unread error
}

func newChecker(repo string, b Bounds) checker { return newCheckerAt(repo, b, "") }

// Guarded says, for a file, the rule that keeps a change to it from
// being trusted to its own run — "gate-file" (CI, workline's config, a
// gate's script or config), "steering-file" (what CODEOWNERS lists, an
// agent file), or
// "bounds-unread" when those could not be read —, the rules read at the
// commit at; "" for any other file.
func Guarded(repo, at string) func(string) string {
	c := newCheckerAt(repo, Bounds{}, at)
	return func(p string) string {
		switch r := c.rule(p); r {
		case "gate-file", "steering-file", "bounds-unread":
			return r
		}
		return ""
	}
}

// newCheckerAt reads the rules at the commit at: the gates, the files
// that never merge alone, a Scope's folders; "" reads them in the
// checkout, the steering files at HEAD.
func newCheckerAt(repo string, b Bounds, at string) checker {
	c := checker{b: b, repo: repo, at: at, steer: func(string) bool { return false }}
	c.gates, c.unread = gateFiles(repo, at)
	tip := at
	if tip == "" {
		tip = "HEAD"
	}
	if s, err := steer.Files(repo, tip); err != nil {
		c.unread = errors.Join(c.unread, fmt.Errorf("the files that steer agents: %w", err))
	} else {
		c.steer = func(p string) bool { return s(p) != "" }
	}
	return c
}

func (c checker) rule(p string) string {
	p = path.Clean(filepath.ToSlash(p))
	switch {
	case p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p):
		return "outside-repo"
	case under(p, ciFiles) || under(p, c.gates):
		return "gate-file"
	case c.unread != nil:
		return "bounds-unread"
	case c.steer(p) || steer.Agent(p):
		return "steering-file"
	case c.b.Frozen[p] > 0:
		return "acceptance-test-edit"
	}
	return ""
}

// beyond says whether p, a file the rules let in, lies beyond the issue's
// Scope (ADR-0065, change 2): not in it, nor a test beside it. Applied
// all the same, and listed on the pull request for the reviewer's
// `intent` lens to judge.
func (c checker) beyond(p string) bool {
	p = path.Clean(filepath.ToSlash(p))
	return !under(p, c.b.Scope) && !besideTest(p, c.b.Scope, c.isDir, c.b.tests()) && !c.newTestBeside(p)
}

// patchFiles are the files kept patches touch, each once, sorted, those
// keep says.
func patchFiles(patches []intent.Intention, keep func(string) bool) []string {
	var out []string
	for _, p := range patches {
		files, _ := touched(p.Value)
		for _, f := range files {
			f = path.Clean(filepath.ToSlash(f))
			if keep(f) && !slices.Contains(out, f) {
				out = append(out, f)
			}
		}
	}
	slices.Sort(out)
	return out
}

// besideKept are the new test files patches wrote beside the Scope, not
// in it nor beside it before (ADR-0063, §9): said on the pull request,
// folded into the Scope after.
func besideKept(repo string, b Bounds, patches []intent.Intention) []string {
	c := newChecker(repo, b)
	return patchFiles(patches, func(f string) bool {
		return c.rule(f) == "" && !under(f, b.Scope) && !besideTest(f, b.Scope, c.isDir, b.tests()) && c.newTestBeside(f)
	})
}

// beyondKept are the files patches wrote beyond the Scope (ADR-0065,
// change 2), each with its agent's one-line reason, or none given.
func beyondKept(repo string, b Bounds, patches []intent.Intention, why map[string]string) []Beyond {
	c := newChecker(repo, b)
	var out []Beyond
	for _, f := range patchFiles(patches, func(f string) bool { return c.rule(f) == "" && c.beyond(f) }) {
		out = append(out, Beyond{File: f, Why: why[f]})
	}
	return out
}

// Beyond is a file a change wrote beyond its issue's Scope, and the
// one-line reason its agent gave, "" when it gave none.
type Beyond struct {
	File string `json:"file"`
	Why  string `json:"why,omitempty"`
}

// BeyondHeading heads the list of files beyond the Scope, on the pull
// request and in a round's commit: the reviewer's `intent` lens reads
// it there.
var BeyondHeading = "Beyond the " + work.Name(work.Scope)

// beyondLines are the files beyond the Scope, one bullet each, its
// reason or that none was given.
func beyondLines(l []Beyond) string {
	var b strings.Builder
	for _, x := range l {
		why := x.Why
		if why == "" {
			why = "no reason given by its agent"
		}
		fmt.Fprintf(&b, "- `%s`: %s\n", x.File, inertText(strings.Join(strings.Fields(why), " ")))
	}
	return b.String()
}

// BeyondLines are a record's files beyond the Scope, as its pull request
// lists them: what the auditor samples (ADR-0046); "" for none.
func BeyondLines(l []Beyond) string { return beyondLines(l) }

// unionBeyond is a with each file of b it lacks, a later reason kept
// where the earlier gave none.
func unionBeyond(a, b []Beyond) []Beyond {
	out := slices.Clone(a)
	for _, x := range b {
		i := slices.IndexFunc(out, func(y Beyond) bool { return y.File == x.File })
		switch {
		case i < 0:
			out = append(out, x)
		case out[i].Why == "":
			out[i].Why = x.Why
		}
	}
	return out
}

// newTestBeside says whether p is a new test file beside the Scope
// (ADR-0063, §9): a test by the project's definition (ADR-0065), not in
// the commit the rules are read at, and its code's folder (Def.Home) a
// Scope path's folder, in one, or holding one. A test the definition
// misses is beyond the Scope, listed. One named like a class of the main
// code is the reviewer's to judge (its `tests` lens).
func (c checker) newTestBeside(p string) bool {
	tests := c.b.tests()
	if !tests.Test(p) || kindAt(c.repo, c.at, p) != "" && !slices.Contains(c.b.Beside, p) {
		return false
	}
	return testBeside(tests.Home(p), c.b.Scope, c.isDir)
}

// testBeside says whether a test file's home, its code's folder
// (Def.Home), is a Scope path's folder, is in one, or holds one. The root
// is a home only to a Scope file at the root: a Scope `src/x.c` lets in
// no test of a `test/` at the root.
func testBeside(home string, scope []string, isDir func(string) bool) bool {
	for _, s := range scope {
		s = path.Clean(s)
		if work.Pattern(s) {
			continue
		}
		dir := s
		if !isDir(s) {
			dir = path.Dir(s)
		}
		switch {
		case dir == home:
			return true
		case home != "." && strings.HasPrefix(dir, home+"/"):
			return true // Storage/test/… beside Storage/include/…
		case dir != "." && strings.HasPrefix(home, dir+"/"):
			return true
		}
	}
	return false
}

// isDir says whether a Scope's path is a folder: one in the repository,
// or, not there yet, a name with no extension.
func (c checker) isDir(p string) bool {
	if k := kindAt(c.repo, c.at, p); k != "" {
		return k == "dir"
	}
	return path.Ext(p) == ""
}

// kindAt says what p is at rev, a commit, or in the checkout when rev is
// "": "file", "dir", or "" when it is not there.
func kindAt(repo, rev, p string) string {
	if rev == "" {
		st, err := os.Stat(filepath.Join(repo, filepath.FromSlash(p)))
		switch {
		case err != nil:
			return ""
		case st.IsDir():
			return "dir"
		}
		return "file"
	}
	switch t, _ := gitOut(repo, "cat-file", "-t", rev+":"+path.Clean(filepath.ToSlash(p))); t {
	case "tree":
		return "dir"
	case "blob":
		return "file"
	}
	return ""
}

// readAt reads p at rev, os.ErrNotExist when it is not there.
func readAt(repo, rev, p string) ([]byte, error) {
	if kindAt(repo, rev, p) != "file" {
		return nil, os.ErrNotExist
	}
	return exec.Command("git", "-C", repo, "show", rev+":"+p).Output()
}

// why is what a refusal says to a person.
func why(r Refusal) string {
	switch r.Rule {
	case "patch-unread":
		return "a patch naming no file, or a diff header whose files cannot be told apart: nothing the bounds can hold, so refused rather than let past"
	case "bounds-unread":
		return "the engine could not read what bounds the developer (" + r.Detail + "): every patch refused rather than one let past; a person fixes it and runs it again"
	case "gate-file":
		return "a gate's definition or the CI's: only a person changes it (docs/spec/gates.md, protection); a person must do it"
	case "steering-file":
		return "a file that steers agents, or one CODEOWNERS lists: never the developer's, whatever the scope; a person must do it"
	case "symlink-out":
		return "a link leading out of the worktree: nothing outside the repository is the developer's, so refused and reverted"
	case "not-text":
		return "not a text file (a link, a binary, a submodule): the developer writes text files, so refused and reverted; a person must do it"
	case conflictRule:
		return "a file not in conflict: a round merging the base in settles its conflicts' markers and changes nothing else (ADR-0067)"
	case "acceptance-test-edit":
		return "an acceptance test the tester froze: read-only to the developer, which objects to it on its line instead; the tester keeps or rewrites it (ADR-0055)"
	}
	return "a path out of the repository: nothing outside it is the developer's"
}

// objection is the agent's word against a frozen test, as it proposes it.
type objection struct {
	Test     string `yaml:"test" json:"test"`
	Line     int    `yaml:"line" json:"line"`
	Scenario string `yaml:"scenario" json:"scenario"`
	Kind     string `yaml:"kind" json:"kind"`
	Why      string `yaml:"why" json:"why"`
}

// readObjection reads a comment's `objection` field, if it has one.
func readObjection(v any) (objection, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return objection{}, false
	}
	raw, ok := m["objection"]
	if !ok {
		return objection{}, false
	}
	data, _ := json.Marshal(raw)
	var o objection
	if json.Unmarshal(data, &o) != nil {
		return objection{}, true
	}
	return o, true
}

// key is the objection's on its line, one per test and line (#899): one
// still open there is not posted again; another line of the same test
// opens its own comment.
func (o objection) key() string { return fmt.Sprintf("objection-%s:%d", o.Test, o.Line) }

// body is the objection's first comment on the test's line, in the form
// the tester reads (roles/tester/README.md, frozen tests and objections).
func (o objection) body() string {
	return fmt.Sprintf("%s\n- test: %s\n- scenario: %s\n- kind: %s\n- why: %s", tester.ObjectionWord, o.Test, o.Scenario, o.Kind, o.Why)
}

// scenarioOf is the issue's scenario an objection names: the one its name
// spells, letters and digits alike; else the one the tester declared the
// named test covers (`CartTest.addsItem`), whatever the test's name
// (ADR-0065, change 4). "" when none, or two; the name as it is when the
// issue's are unknown. No guess from a test's name: an objection on the
// wrong scenario is worse than one said and not posted.
func scenarioOf(name string, scenarios []string, covers map[string][]string) string {
	if len(scenarios) == 0 {
		return name
	}
	var found []string
	for _, sc := range scenarios {
		if c := compact(sc); c != "" && compact(name) == c {
			found = append(found, sc)
		}
	}
	switch len(found) {
	case 1:
		return found[0]
	case 0:
		if sc := tester.CoveredBy(covers, name); slices.Contains(scenarios, sc) {
			return sc
		}
	}
	return ""
}

// compact is a name's letters and digits, lower case, as the tester
// matches an objection's scenario to the issue's.
func compact(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// bounded holds the agent's proposals to the bounds: the claim pre made,
// the patches within them, the objections on the tester's Draft; the rest
// refused, each said.
func bounded(repo string, b Bounds, claim, proposed []intent.Intention) (kept []intent.Intention, refused []Refusal, said []string) {
	c := newChecker(repo, b)
	var objections []forge.LineComment
	var note *intent.Intention
	noted := false
	isClaim := func(i intent.Intention) bool {
		return slices.ContainsFunc(claim, func(f intent.Intention) bool { return f.Kind == i.Kind && same(f.Value, i.Value) })
	}
	for _, i := range proposed {
		switch {
		case isClaim(i):
		case i.Kind == "patch":
			var bad []Refusal
			files, read := touched(i.Value)
			if !read || len(files) == 0 { // a patch the bounds cannot hold
				ref := Refusal{File: "(a patch)", Rule: "patch-unread"}
				if c.unread != nil {
					ref.Rule, ref.Detail = "bounds-unread", c.unread.Error()
				}
				bad = append(bad, ref)
			}
			for _, f := range files {
				if r := c.rule(f); r != "" {
					ref := Refusal{File: path.Clean(filepath.ToSlash(f)), Rule: r}
					if r == "bounds-unread" {
						ref.Detail = c.unread.Error()
					}
					bad = append(bad, ref)
				}
			}
			if len(bad) == 0 {
				kept = append(kept, i)
			}
			refused = append(refused, bad...)
		case i.Kind == "comment":
			o, ok := readObjection(i.Value)
			switch {
			case !ok:
				said = append(said, "a comment: the developer's one comment is an objection to a frozen test; the claim's is the engine's"+objectionHint(b))
			case b.Draft == 0 || b.Frozen[o.Test] != b.Issue:
				said = append(said, fmt.Sprintf("an objection to `%s`: not a test the tester froze for #%d on an open Draft", o.Test, b.Issue))
			case b.Head == "":
				said = append(said, fmt.Sprintf("an objection to `%s`: the commit of the tester's Draft !%d could not be read, so no line to put it on", o.Test, b.Draft))
			case o.Line < 1 || strings.TrimSpace(o.Scenario) == "" || strings.TrimSpace(o.Kind) == "" || strings.TrimSpace(o.Why) == "":
				said = append(said, fmt.Sprintf("an objection to `%s`: it names the test's line, the scenario, the kind and why", o.Test))
			case scenarioOf(o.Scenario, b.Scenarios, b.Covers) == "" && len(tester.Covering(b.Covers, o.Scenario)) > 1:
				said = append(said, fmt.Sprintf("an objection to `%s`: the test %q covers %s, as the tester declared it: name the one scenario it objects for, so the tester reads it: not posted",
					o.Test, o.Scenario, strings.Join(tester.Covering(b.Covers, o.Scenario), " and ")))
			case scenarioOf(o.Scenario, b.Scenarios, b.Covers) == "":
				said = append(said, fmt.Sprintf("an objection to `%s`: its scenario %q is none of #%d's (%s), so the tester would not read it: not posted",
					o.Test, o.Scenario, b.Issue, strings.Join(b.Scenarios, ", ")))
			default:
				o.Scenario = scenarioOf(o.Scenario, b.Scenarios, b.Covers)
				objections = append(objections, forge.LineComment{Path: o.Test, Line: o.Line, Key: o.key(), Body: o.body()})
			}
		case i.Kind == "note":
			on, body := proposedNote(i.Value, b.Issue)
			switch {
			case on != strconv.Itoa(b.Issue):
				said = append(said, fmt.Sprintf("a note on #%s: the developer's one note is on its own issue, #%d (not-the-developers)", on, b.Issue)+objectionHint(b))
			case strings.TrimSpace(body) == "":
				said = append(said, "a note with no body: nothing to post")
			case noted:
				said = append(said, fmt.Sprintf("a note on #%d: the developer posts one note only, the first is kept (not-the-developers)", b.Issue))
			default:
				noted = true
				body = inertText(body) // posted as the developer: it forges no heading and pings no one
				if len(body) > NoteMax {
					cut := NoteMax
					for cut > 0 && !utf8.RuneStart(body[cut]) {
						cut--
					}
					body = body[:cut]
					said = append(said, fmt.Sprintf("a note on #%d: cut to its limit of %d bytes", b.Issue, NoteMax))
				}
				note = &intent.Intention{Kind: "comment", Value: map[string]any{"issue": b.Issue, "sticky": "note", "body": body}}
			}
		default:
			said = append(said, fmt.Sprintf("a %s: the developer never labels, accepts, merges, closes nor moves an issue (ADR-0054)", i.Kind))
		}
	}
	kept = append(slices.Clone(claim), kept...)
	if note != nil {
		kept = append(kept, *note)
	}
	if len(objections) > 0 {
		kept = append(kept, intent.Intention{Kind: "comment", Value: map[string]any{
			"merge-request": b.Draft, "sticky": "objection",
			"body": fmt.Sprintf("**The developer** objects to the tester's tests of #%d, on their lines: the tester keeps or rewrites each (ADR-0055).\n\n%s\n%s\n%s",
				b.Issue, forge.ReviewHere, forge.Unplaced(objections, "the engine that applied this run posts no comment on a line").Section(), forge.ReviewEnd),
			"review": map[string]any{"head": b.Head, "body": "", "comments": objections}}})
	}
	return kept, refused, said
}

// proposedNote reads a note's issue and body: a map {issue, body}, its
// body given as text too, or plain text, as the intentions' schema shows
// a note (#902). A note naming no issue is on the developer's own.
func proposedNote(v any, own int) (issue, body string) {
	if text, ok := v.(string); ok {
		return strconv.Itoa(own), text
	}
	m, _ := v.(map[string]any)
	body, _ = m["body"].(string)
	if body == "" {
		body, _ = m["text"].(string)
	}
	if n, ok := m["issue"]; ok && n != nil {
		return fmt.Sprint(n), body
	}
	return strconv.Itoa(own), body
}

// touched lists the files a patch touches: its `file`, the `---` and
// `+++` lines of its diff, both sides of each `diff --git` header and of
// each `rename` or `copy` line, so that a rename or a deletion is held to
// the bounds too. A path git quotes is unquoted. read is false when a
// header names files it cannot tell apart (a name holding " b/") and no
// other line of its section names them: the patch is refused then.
func touched(v any) (files []string, read bool) {
	diff, isDiff := v.(string)
	if !isDiff {
		return intent.PatchFiles(v), true
	}
	read = true
	named := true // the current section's files are known
	lines := strings.Split(diff, "\n")
	for i, l := range lines {
		if rest, ok := strings.CutPrefix(l, "diff --git "); ok {
			read = read && named
			a, b, ok := gitHeader(rest)
			named = ok
			if ok {
				files = append(files, a, b)
			}
			continue
		}
		// A `---` line is a file's only before its `+++` and a hunk: a
		// removed line "-- x" or an added "++ y" names no file.
		if strings.HasPrefix(l, "--- ") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "+++ ") &&
			(i+2 == len(lines) || strings.HasPrefix(lines[i+2], "@@")) {
			for _, h := range []string{l, lines[i+1]} {
				p, _, _ := strings.Cut(h[4:], "\t")
				if p = unquote(strings.TrimSpace(p)); p != "/dev/null" {
					files = append(files, stripSide(p))
					named = true
				}
			}
		}
		for _, w := range []string{"rename from ", "rename to ", "copy from ", "copy to "} {
			if p, ok := strings.CutPrefix(l, w); ok {
				files = append(files, unquote(strings.TrimSpace(p)))
				named = true
			}
		}
	}
	read = read && named
	for i, f := range files {
		files[i] = path.Clean(f)
	}
	slices.Sort(files)
	return slices.Compact(files), read
}

// gitHeader reads the two paths of a `diff --git` header, quoted or not;
// ok is false when they cannot be told apart.
func gitHeader(rest string) (a, b string, ok bool) {
	if strings.HasPrefix(rest, `"`) {
		q, err := strconv.QuotedPrefix(rest)
		if err != nil {
			return "", "", false
		}
		a, _ = strconv.Unquote(q)
		b = unquote(strings.TrimPrefix(rest[len(q):], " "))
		return stripSide(a), stripSide(b), strings.HasPrefix(b, "b/")
	}
	if i := strings.LastIndex(rest, ` "b/`); i > 0 && strings.HasSuffix(rest, `"`) {
		return stripSide(rest[:i]), stripSide(unquote(rest[i+1:])), true
	}
	// Unquoted: split at the " b/" that leaves the same name on both
	// sides; a rename's names differ, its rename lines name them.
	var cuts []int
	for i := 0; ; {
		j := strings.Index(rest[i:], " b/")
		if j < 0 {
			break
		}
		cuts = append(cuts, i+j)
		i += j + 1
	}
	for _, c := range cuts {
		if x, y := stripSide(rest[:c]), stripSide(rest[c+1:]); x == y {
			return x, y, true
		}
	}
	if len(cuts) == 1 {
		return stripSide(rest[:cuts[0]]), stripSide(rest[cuts[0]+1:]), true
	}
	return "", "", false
}

// unquote is a path as git writes it, C-quoted when it holds a special
// character, read back.
func unquote(p string) string {
	if strings.HasPrefix(p, `"`) {
		if u, err := strconv.Unquote(p); err == nil {
			return u
		}
	}
	return p
}

// stripSide drops the a/ or b/ git puts before a path.
func stripSide(p string) string {
	if s, ok := strings.CutPrefix(p, "a/"); ok {
		return s
	}
	if s, ok := strings.CutPrefix(p, "b/"); ok {
		return s
	}
	return p
}

// same says whether two proposals' values are the same, as YAML reads them.
func same(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// objectionHint tells, when the tester froze tests for the issue, that a test
// that cannot pass is objected to in the objections file, not in a text: a
// note or a comment reaches no tester.
func objectionHint(b Bounds) string {
	for _, n := range b.Frozen {
		if n == b.Issue {
			return ". If it says an acceptance test cannot pass, an objection was due: write it in `" + ObjectionsFile +
				"` (test, line, scenario, kind `contradicts` when the test goes against a documented rule), so that the tester keeps or rewrites the test; a text reaches no tester"
		}
	}
	return ""
}
