package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// FollowOptions are what `workline follow` is given.
type FollowOptions struct {
	Repo, Forge, Base string
	// Gardening: the roles' gardening merge requests too (ADR-0034, amended).
	Gardening bool
	// IsDoc says whether a file is one of the docs, whose header workline
	// writes; nil, none is.
	IsDoc func(path string) bool
	// Release says whether a branch is a release tool's: its merge request
	// is a bot's, rebuilt by the tool.
	Release func(branch string) bool
	// Derive says whether the docs' derived blocks, in the tree at dir,
	// read what the documentalist's `derive` commands give there: a merge
	// request regenerating them is `derived` at `normal` only then; nil,
	// never.
	Derive func(dir string, docs map[string]string) bool
	// Merge is the project's merge policy, its level (`merge: {autonomy:
	// cautious}`); "" none: no merge request's auto-merge is touched
	// (ADR-0042, merge.go).
	Merge string
	// Alone are the kinds that wait for a person's yes at `normal` the
	// project lets merge with none (`merge: {alone: [reader]}`,
	// role.MergeAlone).
	Alone []string
	// Verdict is a file holding the line's verdict on a merge request's
	// head (`workline route --verdict`), kept first, for the merge policy;
	// with VerdictOnly, nothing else is done: a job of its own keeps it,
	// which no newer run cancels (GitHub's concurrency).
	Verdict     string
	VerdictOnly bool
	// Group is a merge queue's group, its branch as GitHub names it
	// (gh-readonly-queue/<base>/pr-<n>-<head>), GroupHead its commit: the
	// yes the base requires is set on that commit as the head queued holds
	// it, and nothing else is followed (ADR-0050, queue.go).
	Group, GroupHead string
}

// workKeys are the keys of a doc's header workline writes, recording when
// it was checked against its sources and how (ADR-0013, ADR-0014): every
// other key is the person's, `sources` first.
var workKeys = map[string]bool{"checked": true, "judged": true, "judged-in-parts": true, "verified": true}

// derived is a block regenerated from the code on the default branch
// (ADR-0027), as the documentalist reads it.
var derived = regexp.MustCompile(`(?s)<!-- workline:derive (\S+) -->(.*?)<!-- workline:end -->`)

// headTips are base's tip and the merge request's head: its branch here,
// or, for one from a fork, the ref the forge keeps its head under.
func headTips(repo, base string, mr forge.MergeRequest, local bool) (baseTip, tip string, err error) {
	if mr.Here {
		return tips(repo, base, mr.Branch, local)
	}
	fetch := func(ref string) (string, error) {
		if _, err := git(repo, nil, "fetch", "-q", "origin", ref); err != nil {
			return "", err
		}
		return git(repo, nil, "rev-parse", "FETCH_HEAD")
	}
	if baseTip, err = fetch("refs/heads/" + base); err != nil {
		return "", "", err
	}
	tip, err = fetch(mr.Head)
	return baseTip, tip, err
}

// settleTree merges baseTip into tip as git does, in memory (git
// merge-tree, 2.38), and settles each file in conflict whose every
// conflict is in a line workline writes: settled are those, theirs the
// files in conflict otherwise; tree is the merge's when theirs is empty.
func settleTree(repo, tip, baseTip string, isDoc func(string) bool) (tree string, settled, theirs []string, err error) {
	cmd := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "-z", "--no-messages", tip, baseTip)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	out, err := cmd.Output()
	var ee *exec.ExitError
	if err != nil && !(errors.As(err, &ee) && ee.ExitCode() == 1) {
		return "", nil, nil, fmt.Errorf("git merge-tree: %v: %s", err, strings.TrimSpace(errOut.String()))
	}
	parts := strings.Split(string(out), "\x00")
	tree = parts[0]
	// Each file in conflict, its versions by stage: 1 where they left each
	// other, 2 the branch's, 3 base's.
	type version struct{ mode, id string }
	stages := map[string]*[4]version{}
	var files []string
	for _, p := range parts[1:] {
		info, file, ok := strings.Cut(p, "\t")
		f := strings.Fields(info)
		if !ok || len(f) != 3 {
			continue
		}
		n, _ := strconv.Atoi(f[2])
		if n < 1 || n > 3 {
			continue
		}
		if stages[file] == nil {
			stages[file] = &[4]version{}
			files = append(files, file)
		}
		stages[file][n] = version{f[0], f[1]}
	}
	if err != nil && len(files) == 0 {
		return "", nil, nil, errors.New("git merge-tree met a conflict it named no file of")
	}
	tmp, err := os.MkdirTemp("", "workline-settle-")
	if err != nil {
		return "", nil, nil, err
	}
	defer os.RemoveAll(tmp)
	idx := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := idx("read-tree", tree); err != nil {
		return "", nil, nil, err
	}
	for _, file := range files {
		v := stages[file]
		if isDoc == nil || !isDoc(file) || v[2].mode != v[3].mode || !regular(v[1].mode) || !regular(v[2].mode) || !regular(v[3].mode) {
			theirs = append(theirs, file) // a submodule, a link, a file gone on a side: the person's
			continue
		}
		var text [4]string
		for n := 1; n <= 3; n++ {
			var ok bool
			if text[n], ok = object(repo, v[n].id); !ok {
				return "", nil, nil, fmt.Errorf("%s: its version %s could not be read", file, v[n].id)
			}
		}
		merged, ok, err := settleDoc(repo, file, text[1], text[2], text[3])
		if err != nil {
			return "", nil, nil, err
		}
		if !ok {
			theirs = append(theirs, file)
			continue
		}
		id, err := gitIn(repo, merged, "hash-object", "-w", "--stdin")
		if err != nil {
			return "", nil, nil, err
		}
		if _, err := idx("update-index", "--cacheinfo", v[2].mode+","+id+","+file); err != nil {
			return "", nil, nil, err
		}
		settled = append(settled, file)
	}
	if len(theirs) > 0 {
		return "", settled, theirs, nil
	}
	tree, err = idx("write-tree")
	return tree, settled, nil, err
}

// settleDoc merges a doc's three versions — where the branch left base,
// the branch's, base's — with workline's lines made alike first: the
// header merged key by key, base's value where both changed one of
// workKeys, any other both changed a clash; each derived block base's.
// The rest is merged by git's own merge, so a conflict there is the
// person's, as git shows it them; ok false then.
func settleDoc(repo, file, old, branch, onBase string) (string, bool, error) {
	o, b, u := front(old), front(branch), front(onBase)
	if o.ok && b.ok && u.ok && o.same(b) && o.same(u) {
		head, ok, read := mergeFront(o.head, u.head, b.head, func(k string) bool { return workKeys[k] })
		switch {
		case read && !ok:
			return "", false, nil
		case read:
			h := u.open + head + u.close
			old, branch, onBase = h+o.body, h+b.body, h+u.body
		}
	}
	return mergeBlobs(repo, "doc"+path.Ext(file), asBase(old, onBase), asBase(branch, onBase), onBase)
}

// asBase is a doc with each derived block base's, where base has it.
func asBase(doc, base string) string {
	blocks := map[string]string{}
	for _, b := range derivedBlocks(base) {
		if _, seen := blocks[b.name]; !seen {
			blocks[b.name] = base[b.start:b.end]
		}
	}
	var out strings.Builder
	at := 0
	for _, b := range derivedBlocks(doc) {
		theirs, ok := blocks[b.name]
		if !ok {
			continue
		}
		out.WriteString(doc[at:b.start] + theirs)
		at = b.end
	}
	return out.String() + doc[at:]
}

// derivedBlock is a derived block's name, and where it is in a doc.
type derivedBlock struct {
	name       string
	start, end int
}

// derivedBlocks finds a doc's derived blocks, as `derived` reads them,
// each closed before the next opens: a marker with no end of its own is
// no block, left as it is, never swallowing the text after it up to
// another block's end (the documentalist's `derive-unclosed`).
func derivedBlocks(doc string) []derivedBlock {
	const open, end = "<!-- workline:derive ", "<!-- workline:end -->"
	var all []derivedBlock
	for at := 0; ; {
		s := strings.Index(doc[at:], open)
		if s < 0 {
			return all
		}
		s += at
		m := derived.FindStringSubmatchIndex(doc[s:])
		next := strings.Index(doc[s+len(open):], open)
		closes := strings.Index(doc[s+len(open):], end)
		if m == nil || m[0] != 0 || closes < 0 || next >= 0 && next < closes {
			at = s + len(open) // unclosed: left as it is
			continue
		}
		all = append(all, derivedBlock{name: doc[s+m[2] : s+m[3]], start: s, end: s + m[1]})
		at = s + m[1]
	}
}

// mergeBlobs merges three versions of a file by git's own merge, the one
// the forge runs (git merge-tree, on commits of that file alone): ok false
// on a conflict.
func mergeBlobs(repo, name, old, ours, theirs string) (string, bool, error) {
	commit := func(content string, parents ...string) (string, error) {
		id, err := gitIn(repo, content, "hash-object", "-w", "--stdin")
		if err != nil {
			return "", err
		}
		tree, err := gitIn(repo, "100644 blob "+id+"\t"+name+"\n", "mktree")
		if err != nil {
			return "", err
		}
		// Never pushed, never kept: never signed, whatever git is set to
		// (commit-tree signs only when told, with -S).
		args := []string{"-c", "user.name=workline", "-c", "user.email=workline@localhost", "commit-tree", tree, "-m", name}
		for _, p := range parents {
			args = append(args, "-p", p)
		}
		return git(repo, nil, args...)
	}
	from, err := commit(old)
	if err != nil {
		return "", false, err
	}
	a, err := commit(ours, from)
	if err != nil {
		return "", false, err
	}
	b, err := commit(theirs, from)
	if err != nil {
		return "", false, err
	}
	out, err := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "--no-messages", a, b).Output()
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("git merge-tree: %v", err)
	}
	tree, _, _ := strings.Cut(string(out), "\n")
	merged, ok := object(repo, strings.TrimSpace(tree)+":"+name)
	if !ok {
		return "", false, fmt.Errorf("git merge-tree: %s not in the merge", name)
	}
	return merged, true, nil
}

// regular says whether a tree entry's mode is a file's, executable or not:
// never a submodule (160000), a link (120000) or a side that has none.
func regular(mode string) bool { return mode == "100644" || mode == "100755" }

// object is a blob's content, by its id or tree:path, and whether it is
// there.
func object(repo, id string) (string, bool) {
	out, err := exec.Command("git", "-C", repo, "cat-file", "blob", id).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
