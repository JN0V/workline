package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// Follow keeps a role's release merge requests (fixElsewhere's, ADR-0017)
// on the tip of base, as a release tool keeps its own branch: run when base
// moves, it rebuilds each open one whose branch base moved under, by
// redoing the branch's change on base's new tip, and force-pushes it — the
// merge request stays the same (ADR-0034). A branch already on base's tip
// is left as it is; one a person committed to is never rebuilt over them;
// one whose change no longer applies is left for a person.
func Follow(repo, forgeSpec, base string) *Result {
	res := &Result{Status: verdict.Pass, Applied: []string{}, Refused: []string{}}
	fail := func(err error) *Result {
		res.Status = verdict.Block
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status = verdict.BlockedExternal
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "follow-failed", Message: err.Error()})
		return res
	}
	f, err := forge.Open(forgeSpec, repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return fail(errors.New("follow needs a forge: " + forge.Missing))
	}
	if base == "" {
		if base, err = git(repo, nil, "symbolic-ref", "-q", "--short", "HEAD"); err != nil {
			return fail(errors.New("on no branch: give the branch the merge requests go into with --base"))
		}
	}
	open, err := f.OpenMergeRequests("workline/")
	if err != nil {
		return fail(err)
	}
	local := forge.KeepsBranches(f)
	n := 0
	for _, branch := range open {
		role, task, ok := strings.Cut(strings.TrimPrefix(branch, "workline/"), "/")
		if !ok || task != "release" {
			continue
		}
		n++
		finding, err := follow(repo, base, branch, role, local)
		if err != nil {
			return fail(err)
		}
		if finding.Rule == "" {
			continue
		}
		if finding.Rule == "no-longer-applies" {
			res.Status = verdict.Human
		}
		res.Findings = append(res.Findings, finding)
	}
	rebuilt := 0
	for _, f := range res.Findings {
		if f.Rule == "rebuilt" {
			rebuilt++
		}
	}
	res.Summary = fmt.Sprintf("%d release merge requests open, %d rebuilt on %s", n, rebuilt, base)
	return res
}

// follow rebuilds one role's branch on base's tip, and says what it did;
// nothing when the branch is on base's tip already.
func follow(repo, base, branch, role string, local bool) (verdict.Finding, error) {
	baseTip, tip, err := tips(repo, base, branch, local)
	if err != nil || tip == "" {
		return verdict.Finding{}, err
	}
	if _, err := git(repo, nil, "merge-base", "--is-ancestor", baseTip, tip); err == nil {
		return verdict.Finding{}, nil // on base's tip already
	}
	// The forge lists the branches, not where they go: a branch that did not
	// leave from base's history goes elsewhere, a maintenance branch, and is
	// never put on base.
	if first, _ := git(repo, nil, "rev-list", "--reverse", "--grep=^"+OwnTrailer+": "+regexp.QuoteMeta(role)+"$", baseTip+".."+tip); first != "" {
		first, _, _ = strings.Cut(first, "\n")
		if _, err := git(repo, nil, "merge-base", "--is-ancestor", first+"^", baseTip); err != nil {
			return verdict.Finding{Rule: "not-rebuilt", Level: "warn", Where: branch,
				Message: fmt.Sprintf("it did not leave from %s, so its merge request goes into another branch: not rebuilt on %s", base, base)}, nil
		}
	}
	if who := personsCommit(repo, baseTip, tip, role); who != "" {
		return verdict.Finding{Rule: "not-rebuilt", Level: "warn", Where: branch,
			Message: fmt.Sprintf("%s moved under it, but %s: not rebuilt over it; rebase it by hand, or close it and the next release run proposes the fix again", base, who)}, nil
	}
	tree, clashes, err := redo(repo, baseTip, tip)
	if err != nil {
		return verdict.Finding{}, err
	}
	if len(clashes) > 0 {
		return verdict.Finding{Rule: "no-longer-applies", Where: branch,
			Message: fmt.Sprintf("its change no longer applies on %s, which changed the same lines of %s: left as it is, for a person to rebase or close", base, strings.Join(clashes, ", "))}, nil
	}
	same, _ := git(repo, nil, "rev-parse", baseTip+"^{tree}")
	if tree == same {
		return verdict.Finding{Rule: "already-on-base", Level: "warn", Where: branch,
			Message: fmt.Sprintf("%s holds its change already: nothing left to merge; close it", base)}, nil
	}
	commit, err := recommit(repo, tip, tree, baseTip)
	if err != nil {
		return verdict.Finding{}, err
	}
	move := []string{"push", "-q", "--force-with-lease=refs/heads/" + branch + ":" + tip, "origin", commit + ":refs/heads/" + branch}
	if local {
		move = []string{"update-ref", "refs/heads/" + branch, commit, tip}
	}
	if _, err := git(repo, nil, move...); err != nil {
		if local {
			return verdict.Finding{}, err
		}
		return verdict.Finding{}, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return verdict.Finding{Rule: "rebuilt", Where: branch,
		Message: fmt.Sprintf("rebuilt on %s (%s): the same change, its merge request kept", base, short(baseTip))}, nil
}

// tips are base's and branch's commits, from origin, or from the clone for
// a forge kept there; tip is "" when the branch is gone.
func tips(repo, base, branch string, local bool) (baseTip, tip string, err error) {
	if local {
		if baseTip, err = git(repo, nil, "rev-parse", "--verify", "refs/heads/"+base); err != nil {
			return "", "", fmt.Errorf("the base %s is not in this clone", base)
		}
		tip, _ = git(repo, nil, "rev-parse", "--verify", "-q", "refs/heads/"+branch)
		return baseTip, tip, nil
	}
	ref := func(b string) (string, error) {
		if _, err := git(repo, nil, "fetch", "-q", "origin", "refs/heads/"+b); err != nil {
			return "", err
		}
		return git(repo, nil, "rev-parse", "FETCH_HEAD")
	}
	if baseTip, err = ref(base); err != nil {
		return "", "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	if out, err := git(repo, nil, "ls-remote", "origin", "refs/heads/"+branch); err != nil {
		return "", "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	} else if out == "" {
		return baseTip, "", nil
	}
	if tip, err = ref(branch); err != nil {
		return "", "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return baseTip, tip, nil
}

// personsCommit names the first commit of the branch, past base, the role
// did not make: one without its trailer, or a merge. "" when all are its.
func personsCommit(repo, baseTip, tip, role string) string {
	out, err := git(repo, nil, "log", "--format=%h%x1f%p%x1f%s%x1f%(trailers:key="+OwnTrailer+",valueonly,separator=%x2c)%x1e", baseTip+".."+tip)
	if err != nil {
		return "its commits could not be read: " + err.Error()
	}
	for _, rec := range strings.Split(out, "\x1e") {
		fields := strings.Split(strings.TrimSpace(rec), "\x1f")
		if len(fields) < 4 {
			continue
		}
		if strings.Contains(fields[1], " ") || strings.TrimSpace(fields[3]) != role {
			return fmt.Sprintf("a person committed to it (%s %q)", fields[0], fields[2])
		}
	}
	return ""
}

// redo puts the branch's change, since it left base, on base's tip, file by
// file, and returns the tree: a doc's front matter merged key by key, main's
// value kept where both changed one, so a `checked` moved on main beside a
// `judged` the branch wrote is no conflict; the rest merged as git does.
// clashes are the files where the change no longer applies.
func redo(repo, baseTip, tip string) (tree string, clashes []string, err error) {
	from, err := git(repo, nil, "merge-base", baseTip, tip)
	if err != nil {
		return "", nil, err
	}
	changes, err := git(repo, nil, "diff", "--name-status", "--no-renames", "-z", from, tip)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "workline-follow-")
	if err != nil {
		return "", nil, err
	}
	defer os.RemoveAll(tmp)
	index := filepath.Join(tmp, "index")
	idx := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+index)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := idx("read-tree", baseTip); err != nil {
		return "", nil, err
	}
	parts := strings.Split(strings.TrimSuffix(changes, "\x00"), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		status, path := parts[i], parts[i+1]
		old, hadOld := blob(repo, from, path)
		ours, hasOurs := blob(repo, baseTip, path)
		theirs, hasTheirs := blob(repo, tip, path)
		switch {
		case status == "D":
			if hasOurs && ours != old {
				clashes = append(clashes, path)
				continue
			}
			if _, err := idx("update-index", "--force-remove", "--", path); err != nil {
				return "", nil, err
			}
			continue
		case !hadOld && hasOurs && ours != theirs, // added on both sides, differently
			hadOld && !hasOurs: // changed on the branch, gone from base
			clashes = append(clashes, path)
			continue
		}
		merged, ok := theirs, true
		if hasOurs && hadOld {
			merged, ok = mergeDoc(tmp, old, ours, theirs)
		}
		if !ok || !hasTheirs {
			clashes = append(clashes, path)
			continue
		}
		id, err := gitIn(repo, merged, "hash-object", "-w", "--stdin")
		if err != nil {
			return "", nil, err
		}
		mode, _ := git(repo, nil, "ls-tree", "--format=%(objectmode)", tip, "--", path)
		if mode == "" {
			mode = "100644"
		}
		if _, err := idx("update-index", "--add", "--cacheinfo", mode+","+id+","+path); err != nil {
			return "", nil, err
		}
	}
	if len(clashes) > 0 {
		return "", clashes, nil
	}
	tree, err = idx("write-tree")
	return tree, nil, err
}

// blob is a file's content at a commit, and whether it is there.
func blob(repo, commit, path string) (string, bool) {
	out, err := exec.Command("git", "-C", repo, "cat-file", "blob", commit+":"+path).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func gitIn(repo, stdin string, args ...string) (string, error) {
	return git(repo, strings.NewReader(stdin), args...)
}

// frontKey is a front matter line that sets a key on one line.
var frontKey = regexp.MustCompile(`^([A-Za-z0-9_-]+):`)

// mergeDoc merges a file's three versions — old, where the branch left
// base; ours, base's tip; theirs, the branch's — and says whether it could.
// A doc's front matter is merged key by key, each key set on its own line;
// the body, and a file with no such front matter, as `git merge-file`.
func mergeDoc(tmp, old, ours, theirs string) (string, bool) {
	oh, ob, ok1 := front(old)
	uh, ub, ok2 := front(ours)
	th, tb, ok3 := front(theirs)
	if !ok1 || !ok2 || !ok3 {
		return mergeFile(tmp, old, ours, theirs)
	}
	head, ok := mergeFront(oh, uh, th)
	if !ok {
		return mergeFile(tmp, old, ours, theirs)
	}
	body, ok := mergeFile(tmp, ob, ub, tb)
	if !ok {
		return "", false
	}
	return "---\n" + head + "---\n" + body, true
}

// front splits a doc into its front matter's lines and its body; ok is
// false when it has none.
func front(s string) (head []string, body string, ok bool) {
	if !strings.HasPrefix(s, "---\n") {
		return nil, s, false
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return nil, s, false
	}
	inner := s[4 : 4+end+1]
	return strings.SplitAfter(strings.TrimSuffix(inner, "\n"), "\n"), s[4+end+5:], true
}

// mergeFront merges front matters whose every line sets one key: a key the
// branch changed takes the branch's line, one only base changed base's, one
// both changed base's — the later judgement; a key the branch added goes
// after the key it followed. ok is false for a front matter it cannot read
// line by line.
func mergeFront(old, ours, theirs []string) (string, bool) {
	keyed := func(lines []string) (map[string]string, []string, bool) {
		m, order := map[string]string{}, []string{}
		for _, l := range lines {
			if !strings.HasSuffix(l, "\n") {
				l += "\n"
			}
			k := frontKey.FindStringSubmatch(l)
			if k == nil {
				return nil, nil, false
			}
			if _, dup := m[k[1]]; dup {
				return nil, nil, false
			}
			m[k[1]], order = l, append(order, k[1])
		}
		return m, order, true
	}
	o, _, ok1 := keyed(old)
	u, uOrder, ok2 := keyed(ours)
	t, tOrder, ok3 := keyed(theirs)
	if !ok1 || !ok2 || !ok3 {
		return "", false
	}
	line := map[string]string{} // each key's merged line
	var keys []string
	for _, k := range uOrder {
		ov, inOld := o[k]
		tv, inTheirs := t[k]
		switch {
		case inOld && !inTheirs && u[k] == ov:
			continue // the branch took it out, base left it
		case inOld && inTheirs && tv != ov && u[k] == ov:
			line[k] = tv
		default:
			line[k] = u[k]
		}
		keys = append(keys, k)
	}
	for i, k := range tOrder {
		_, inOurs := u[k]
		if _, inOld := o[k]; inOurs || inOld {
			continue // base's, or base took it out
		}
		at := 0 // after the key it follows on the branch, else first
		for j := i - 1; j >= 0 && at == 0; j-- {
			if p := slices.Index(keys, tOrder[j]); p >= 0 {
				at = p + 1
			}
		}
		keys, line[k] = slices.Insert(keys, at, k), t[k]
	}
	var out strings.Builder
	for _, k := range keys {
		out.WriteString(line[k])
	}
	return out.String(), true
}

// mergeFile merges three texts as git does, and says whether it could.
func mergeFile(tmp, old, ours, theirs string) (string, bool) {
	paths := []string{filepath.Join(tmp, "ours"), filepath.Join(tmp, "old"), filepath.Join(tmp, "theirs")}
	for i, s := range []string{ours, old, theirs} {
		if err := os.WriteFile(paths[i], []byte(s), 0o644); err != nil {
			return "", false
		}
	}
	out, err := exec.Command("git", append([]string{"merge-file", "-p"}, paths...)...).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// recommit commits tree on parent with the message and author of the
// branch's tip: the same change, by the same hand, on base's new tip.
func recommit(repo, tip, tree, parent string) (string, error) {
	who, err := git(repo, nil, "log", "-1", "--format=%an%x1f%ae%x1f%aI", tip)
	if err != nil {
		return "", err
	}
	msg, err := git(repo, nil, "log", "-1", "--format=%B", tip)
	if err != nil {
		return "", err
	}
	a := strings.Split(who, "\x1f")
	cmd := exec.Command("git", "-C", repo, "commit-tree", tree, "-p", parent, "-F", "-")
	cmd.Stdin = strings.NewReader(msg + "\n")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+a[0], "GIT_AUTHOR_EMAIL="+a[1], "GIT_AUTHOR_DATE="+a[2])
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("git commit-tree: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func short(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
