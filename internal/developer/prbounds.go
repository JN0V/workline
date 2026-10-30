package developer

// The bounds on the pull request (ADR-0058, step 1): on the merge-request
// event, no agent, the developer's pull request is held to the issue's
// bounds on its diff against its base, whoever pushed its commits. The
// rules are read at the base's tip, never at the head: a pull request
// cannot loosen its own check. A file refused blocks, named with its
// rule; the verdict is kept on the pull request, for the auditor
// (ADR-0046). A person's branch is not held: it is reviewed as theirs.

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

// prBoundsFile keeps, between pre and post, the verdict pre reached.
const prBoundsFile = "pr-bounds.json"

// BoundsKey is the sticky key of the bounds' record on the pull request.
const BoundsKey = "bounds"

// prBounds is the record of one check, hidden in its comment.
type prBounds struct {
	Act     string    `json:"act"` // "bounds"
	Issue   int       `json:"issue"`
	Base    string    `json:"base"` // the base's tip the rules were read at
	Head    string    `json:"head"`
	Files   int       `json:"files"`
	Refused []Refusal `json:"refused"`
}

func (r prBounds) body() string {
	var b strings.Builder
	fmt.Fprintf(&b, "**The developer's bounds** on this pull request, read at the base's %s: %d file%s changed since it, ", shortSHA(r.Base), r.Files, plural(r.Files))
	if len(r.Refused) == 0 {
		b.WriteString("each within the bounds of #" + fmt.Sprint(r.Issue) + ".\n")
	} else {
		fmt.Fprintf(&b, "%d refused: its next round or a person reverts them ([the bounds](https://github.com/JN0V/workline/blob/main/roles/developer/docs/bounds.md)).\n\n| File | Rule |\n|---|---|\n", len(r.Refused))
		for _, x := range r.Refused {
			fmt.Fprintf(&b, "| `%s` | %s |\n", x.File, x.Rule)
		}
	}
	data, _ := json.Marshal(r)
	fmt.Fprintf(&b, "\n<!-- %s %s -->", ActMark, data)
	return b.String()
}

// boundsPre is the check, the developer on a merge request.
func boundsPre(runDir, repo string, id int) int {
	where := fmt.Sprintf("merge request %d", id)
	if id <= 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no merge request to hold to the bounds",
			Findings: []verdict.Finding{{Rule: "no-merge-request", Message: "the bounds check runs on a pull request: --target merge-request:<n>"}}})
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge: the bounds were not checked",
			Findings: []verdict.Finding{{Rule: "no-forge", Where: where, Message: "the bounds check reads the pull request's branch and its issue on a forge; " + forge.Missing}}})
	}
	bl, ok := f.(forge.Backlog)
	if !ok {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the forge lists no comments: the bounds were not checked",
			Findings: []verdict.Finding{{Rule: "no-backlog", Where: where, Message: "the issue's " + work.Name(work.Scope) + " and the tester's record are read on the forge, which this one does not give"}}})
	}
	mr, err := f.MergeRequest(id)
	if err != nil {
		return external(err)
	}
	n := developerIssue(mr.Branch)
	if n == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "not a developer's pull request: no bounds to hold",
			Findings: []verdict.Finding{{Rule: "not-a-developers-branch", Level: "info", Where: where,
				Message: "it comes from " + mr.Branch + ": a person's pull request, reviewed as one; the bounds hold the developer's own, " + Branch + "<n>"}}})
	}
	base, head, err := prRange(runDir, repo, mr)
	if err != nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the pull request's commits cannot be read: the bounds were not checked",
			Findings: []verdict.Finding{{Rule: "bounds-unread", Where: where, Message: err.Error()}}})
	}
	is, err := f.Issue(n)
	if err != nil {
		return external(err)
	}
	b, err := readBounds(f, bl, repo, *is)
	if err != nil {
		return external(err)
	}
	// The tests its own record names, frozen at the commit it started
	// from, when the tester's Draft is closed.
	if a, ok, err := ReadRecord(f, id); err != nil {
		return external(err)
	} else if ok && a.From != "" {
		for _, t := range a.Acceptance {
			if _, frozen := b.Frozen[t]; frozen {
				continue
			}
			if blob, err := gitOut(repo, "rev-parse", "-q", "--verify", a.From+":"+t); err == nil && blob != "" {
				b.Frozen[t], b.Blobs[t] = n, blob
			}
		}
	}
	files, refused, err := prRefusals(repo, b, base, head)
	if err != nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the pull request's diff cannot be read: the bounds were not checked",
			Findings: []verdict.Finding{{Rule: "bounds-unread", Where: where, Message: err.Error()}}})
	}
	rec := prBounds{Act: "bounds", Issue: n, Base: base, Head: head, Files: files, Refused: refused}
	if rec.Refused == nil {
		rec.Refused = []Refusal{}
	}
	v := verdict.Verdict{Status: verdict.Pass, Final: true,
		Summary: fmt.Sprintf("the bounds of #%d hold: %d file%s changed, none refused", n, files, plural(files))}
	if len(refused) > 0 {
		v.Status = verdict.Block
		v.Summary = fmt.Sprintf("the bounds of #%d refuse %d of %d file%s changed", n, len(refused), files, plural(files))
	}
	for _, r := range refused {
		v.Findings = append(v.Findings, verdict.Finding{Rule: r.Rule, Where: r.File,
			Message: "changed on the developer's branch, which its bounds refuse, whoever pushed it: " + why(r)})
	}
	if code := writeIntents(runDir, "fallback.yaml", []intent.Intention{{Kind: "comment",
		Value: map[string]any{"sticky": BoundsKey, "body": rec.body()}}}); code != 0 {
		return code
	}
	if err := writeIn(runDir, prBoundsFile, v); err != nil {
		return fail(err)
	}
	return 0 // no question: post gives the verdict, the engine keeps the record
}

// boundsPost gives the verdict pre reached.
func boundsPost(runDir string) int {
	var v verdict.Verdict
	if err := readIn(runDir, prBoundsFile, &v); err != nil {
		return fail(err)
	}
	if code := writeVerdict(runDir, v); code != 0 || v.Status == verdict.Pass {
		return code
	}
	return 1 // a block, as post's exit says it (docs/spec/role-contract.md)
}

// prRange is the commit the rules are read at, the base's tip, and the
// pull request's head: the range given (--input range=base..head), else
// the branches as fetched. Neither read is an error, never a pass.
func prRange(runDir, repo string, mr forge.MergeRequest) (base, head string, err error) {
	commit := func(refs ...string) string {
		for _, r := range refs {
			if r == "" || r == "refs/heads/" || r == "refs/remotes/origin/" {
				continue
			}
			if h, err := gitOut(repo, "rev-parse", "-q", "--verify", r+"^{commit}"); err == nil && h != "" {
				return h
			}
		}
		return ""
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "input", "range")); err == nil {
		if a, z, ok := strings.Cut(strings.TrimSpace(string(data)), ".."); ok {
			base, head = commit(a), commit(strings.TrimPrefix(z, "."))
		}
	}
	if head == "" {
		head = commit("refs/remotes/origin/"+mr.Branch, "refs/heads/"+mr.Branch, mr.Head)
	}
	if base == "" && mr.Base != "" {
		base = commit("refs/remotes/origin/"+mr.Base, "refs/heads/"+mr.Base)
	}
	switch {
	case head == "":
		return "", "", fmt.Errorf("the head of %s is not in this clone: fetch it, or give --input range=<base>..<head>", mr.Branch)
	case base == "":
		return "", "", fmt.Errorf("the base %q of the pull request is not in this clone: fetch it, or give --input range=<base>..<head>", mr.Base)
	}
	return base, head, nil
}

// prRefusals holds each file the head changed since its merge base with
// base to the bounds, the rules read at base: how many changed, and the
// ones refused. A frozen test as the tester wrote it is not the
// developer's change.
func prRefusals(repo string, b Bounds, base, head string) (files int, refused []Refusal, err error) {
	from, err := gitOut(repo, "merge-base", base, head)
	if err != nil {
		return 0, nil, fmt.Errorf("no merge base between %s and %s", shortSHA(base), shortSHA(head))
	}
	raw, err := gitOut(repo, "-c", "core.quotePath=false", "diff", "--raw", "--no-abbrev", "--no-renames", "--no-ext-diff", "-z", from, head)
	if err != nil {
		return 0, nil, fmt.Errorf("git diff %s..%s: %w", shortSHA(from), shortSHA(head), err)
	}
	c := newCheckerAt(repo, b, base)
	fields := strings.Split(raw, "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		meta, name := strings.Fields(strings.TrimPrefix(fields[i], ":")), fields[i+1]
		if len(meta) < 5 || name == "" {
			return files, refused, errors.New("git diff --raw: a line it cannot read")
		}
		files++
		mode, blob := meta[1], meta[3]
		if b.Frozen[name] > 0 && blob == b.Blobs[name] {
			continue // the tester's, as it froze it
		}
		rule := c.rule(name)
		if mode == "120000" {
			if to, err := gitOut(repo, "cat-file", "blob", blob); err != nil || linkOut(name, to) {
				rule = "symlink-out"
			}
		}
		if rule != "" {
			r := Refusal{File: name, Rule: rule}
			if rule == "bounds-unread" {
				r.Detail = c.unread.Error()
			}
			refused = append(refused, r)
		}
	}
	// A test frozen for its issue that the head lacks or changed, though
	// the diff does not show it: added by the tester's Draft, then
	// deleted on the branch, it is in neither end of the range.
	for _, t := range slices.Sorted(maps.Keys(b.Frozen)) {
		if b.Frozen[t] != b.Issue || slices.ContainsFunc(refused, func(r Refusal) bool { return r.File == t }) {
			continue
		}
		if blob, _ := gitOut(repo, "rev-parse", "-q", "--verify", head+":"+t); blob != b.Blobs[t] {
			refused = append(refused, Refusal{File: t, Rule: "acceptance-test-edit"})
		}
	}
	return files, refused, nil
}

// linkOut says a link at name pointing to to leads out of the tree.
func linkOut(name, to string) bool {
	if path.IsAbs(filepath.ToSlash(to)) || filepath.IsAbs(to) {
		return true
	}
	p := path.Join(path.Dir(name), filepath.ToSlash(to))
	return p == ".." || strings.HasPrefix(p, "../")
}

