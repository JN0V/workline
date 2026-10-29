package developer

// The bounds at apply (ADR-0054, step 2): post, with no agent, leaves out
// each patch reaching a gate's file, a file that steers agents, a frozen
// test or a file outside the issue's scope, and the agent's labels and
// comments but an objection; roles/developer/docs/bounds.md says why.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/steer"
	"github.com/JN0V/workline/internal/tester"
)

const boundsFile = "bounds.json"

// Bounds are what pre read of the issue and the forge for post to hold the
// patches to.
type Bounds struct {
	Issue int `json:"issue"`
	// Scope: the paths the issue's Scope names and the product owner's
	// sources for it.
	Scope []string `json:"scope"`
	// Frozen: each test the tester froze on an open Draft, by its issue.
	Frozen map[string]int `json:"frozen,omitempty"`
	// Draft is the tester's open Draft for the issue, 0 when none, and
	// Head its branch's commit, where an objection goes on a line.
	Draft int    `json:"draft,omitempty"`
	Head  string `json:"head,omitempty"`
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
	b := Bounds{Issue: is.ID, Scope: scopePaths(is.Body), Frozen: map[string]int{}}
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
			continue
		}
		for file := range r.Files {
			b.Frozen[file] = r.Issue
		}
		if n == is.ID {
			b.Draft = mr.ID
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
// cannot be read is an error: no gate's file is known then.
func gateFiles(repo string) ([]string, error) {
	var cfg struct {
		Gates map[string]struct {
			Checks []struct {
				Run    string   `yaml:"run"`
				Config []string `yaml:"config"`
			} `yaml:"checks"`
		} `yaml:"gates"`
	}
	data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml"))
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
				if st, err := os.Stat(filepath.Join(repo, w)); err == nil && !st.IsDir() {
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

// under says whether p is one of roots or in one of them.
func under(p string, roots []string) bool {
	return slices.ContainsFunc(roots, func(r string) bool {
		r = path.Clean(r)
		return r != "." && (p == r || strings.HasPrefix(p, r+"/"))
	})
}

// besideTest says whether p is a test beside one of the scope's paths, its
// name holding "test" or "spec": in a scope path's folder or under it, or
// in a test folder that sits there. A folder's own folder is not its
// neighbourhood: a Scope `src` lets in no test at the root.
func besideTest(p string, scope []string, isDir func(string) bool) bool {
	name := strings.ToLower(path.Base(p))
	if !strings.Contains(name, "test") && !strings.Contains(name, "spec") {
		return false
	}
	dirs := []string{path.Dir(p)}
	parts := strings.Split(path.Dir(p), "/")
	for i, part := range parts {
		if part == "test" || part == "tests" || part == "spec" || part == "__tests__" {
			dirs = append(dirs, path.Join(append([]string{"."}, parts[:i]...)...))
		}
	}
	for _, d := range dirs {
		for _, s := range scope {
			if d == path.Dir(s) && !isDir(s) || under(d, []string{s}) {
				return true
			}
		}
	}
	return false
}

// checker says, for a file a patch touches, the rule that refuses it, ""
// when it may be touched.
type checker struct {
	b     Bounds
	gates []string
	roles string // the role folders, when in the repository
	repo  string
	steer func(string) bool
	// unread is why the gates or the steering files could not be read:
	// every file is refused then, rather than one let past.
	unread error
}

func newChecker(repo string, b Bounds) checker {
	c := checker{b: b, repo: repo, steer: func(string) bool { return false }}
	c.gates, c.unread = gateFiles(repo)
	if s, err := steer.Files(repo, "HEAD"); err != nil {
		c.unread = errors.Join(c.unread, fmt.Errorf("the files that steer agents: %w", err))
	} else {
		c.steer = s
	}
	if d := os.Getenv("WORKLINE_ROLES_DIR"); d != "" {
		if rel, err := filepath.Rel(repo, d); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			c.roles = filepath.ToSlash(rel)
		}
	}
	return c
}

func (c checker) rule(p string) string {
	p = path.Clean(filepath.ToSlash(p))
	switch {
	case p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p):
		return "outside-scope" // out of the repository
	case under(p, ciFiles) || under(p, c.gates):
		return "gate-file"
	case c.unread != nil:
		return "bounds-unread"
	case c.steer(p) || c.roles != "" && under(p, []string{c.roles}) || c.roleFolder(p):
		return "steering-file"
	case c.b.Frozen[p] > 0:
		return "acceptance-test-edit"
	case !under(p, c.b.Scope) && !besideTest(p, c.b.Scope, c.isDir):
		return "outside-scope"
	}
	return ""
}

// isDir says whether a Scope's path is a folder: one in the repository,
// or, not there yet, a name with no extension.
func (c checker) isDir(p string) bool {
	if st, err := os.Stat(filepath.Join(c.repo, filepath.FromSlash(p))); err == nil {
		return st.IsDir()
	}
	return path.Ext(p) == ""
}

// roleFolder says whether p is in a role's folder of the repository: a
// `roles/<name>/` holding a role.yaml, a role's facets the engine gives
// its agent, and its pages beside them.
func (c checker) roleFolder(p string) bool {
	parts := strings.Split(p, "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] != "roles" {
			continue
		}
		if c.repo == "" {
			return true
		}
		if _, err := os.Stat(filepath.Join(c.repo, filepath.FromSlash(path.Join(parts[:i+2]...)), "role.yaml")); err == nil {
			return true
		}
	}
	return false
}

// why is what a refusal says to a person.
func why(r Refusal) string {
	switch r.Rule {
	case "bounds-unread":
		return "the engine could not read what bounds the developer (" + r.Detail + "): every patch refused rather than one let past; a person fixes it and runs it again"
	case "gate-file":
		return "a gate's definition or the CI's: only a person changes it (docs/spec/gates.md, protection); a person must do it"
	case "steering-file":
		return "a file that steers agents or a role's folder: never the developer's, whatever the scope; a person must do it"
	case "acceptance-test-edit":
		return "an acceptance test the tester froze: read-only to the developer, which objects to it on its line instead; the tester keeps or rewrites it (ADR-0055)"
	}
	return "outside the issue's scope, the product owner's sources for it and the tests beside them: anything found there belongs in an issue (principle 8)"
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

// body is the objection's first comment on the test's line, in the form
// the tester reads (roles/tester/README.md, frozen tests and objections).
func (o objection) body() string {
	return fmt.Sprintf("%s\n- test: %s\n- scenario: %s\n- kind: %s\n- why: %s", tester.ObjectionWord, o.Test, o.Scenario, o.Kind, o.Why)
}

// bounded holds the agent's proposals to the bounds: the claim pre made,
// the patches within them, the objections on the tester's Draft; the rest
// refused, each said.
func bounded(repo string, b Bounds, claim, proposed []intent.Intention) (kept []intent.Intention, refused []Refusal, said []string) {
	c := newChecker(repo, b)
	var objections []forge.LineComment
	isClaim := func(i intent.Intention) bool {
		return slices.ContainsFunc(claim, func(f intent.Intention) bool { return f.Kind == i.Kind && same(f.Value, i.Value) })
	}
	for _, i := range proposed {
		switch {
		case isClaim(i):
		case i.Kind == "patch":
			var bad []Refusal
			for _, f := range touched(i.Value) {
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
				said = append(said, "a comment: the developer's one comment is an objection to a frozen test; the claim's is the engine's")
			case b.Draft == 0 || b.Frozen[o.Test] != b.Issue:
				said = append(said, fmt.Sprintf("an objection to `%s`: not a test the tester froze for #%d on an open Draft", o.Test, b.Issue))
			case b.Head == "":
				said = append(said, fmt.Sprintf("an objection to `%s`: the commit of the tester's Draft !%d could not be read, so no line to put it on", o.Test, b.Draft))
			case o.Line < 1 || strings.TrimSpace(o.Scenario) == "" || strings.TrimSpace(o.Kind) == "" || strings.TrimSpace(o.Why) == "":
				said = append(said, fmt.Sprintf("an objection to `%s`: it names the test's line, the scenario, the kind and why", o.Test))
			default:
				objections = append(objections, forge.LineComment{Path: o.Test, Line: o.Line, Key: "objection-" + o.Test, Body: o.body()})
			}
		default:
			said = append(said, fmt.Sprintf("a %s: the developer never labels, accepts, merges, closes nor moves an issue (ADR-0054)", i.Kind))
		}
	}
	kept = append(slices.Clone(claim), kept...)
	if len(objections) > 0 {
		kept = append(kept, intent.Intention{Kind: "comment", Value: map[string]any{
			"merge-request": b.Draft, "sticky": "objection",
			"body":   fmt.Sprintf("**The developer** objects to %d of the tester's tests of #%d, on their lines: the tester keeps or rewrites each (ADR-0055).", len(objections), b.Issue),
			"review": map[string]any{"head": b.Head, "body": "", "comments": objections}}})
	}
	return kept, refused, said
}

// touched lists the files a patch touches: its `file`, the `+++` lines
// of its diff, and both sides of each `diff --git` header and `rename`
// line, so that a rename or a deletion is held to the bounds too.
func touched(v any) []string {
	files := intent.PatchFiles(v)
	diff, _ := v.(string)
	for _, l := range strings.Split(diff, "\n") {
		if rest, ok := strings.CutPrefix(l, "diff --git a/"); ok {
			if a, b, ok := strings.Cut(rest, " b/"); ok {
				files = append(files, a, b)
			}
		}
		for _, w := range []string{"rename from ", "rename to ", "copy from ", "copy to "} {
			if p, ok := strings.CutPrefix(l, w); ok {
				files = append(files, strings.TrimSpace(p))
			}
		}
	}
	for i, f := range files {
		files[i] = path.Clean(f)
	}
	slices.Sort(files)
	return slices.Compact(files)
}

// same says whether two proposals' values are the same, as YAML reads them.
func same(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
