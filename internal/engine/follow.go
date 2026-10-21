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
//
// With gardening, run when gardening runs, it tends the roles' other merge
// requests too, one a task (ADR-0006): each base moved under is rebuilt
// the same way, or closed, one comment saying why, when nothing of it is
// left to merge — base holds its change already, base changed the lines it
// changed, or it only records when docs were read. One a person committed
// to is left to them, and named only when it no longer merges (ADR-0034,
// amended).
//
// With People, the project's choice, it merges base into each person's
// merge request left in conflict with it only on lines workline writes,
// those settled: a doc's record of when it was checked, a derived block
// (ADR-0041; settle.go).
//
// With Merge, the project's merge policy, it turns the forge's auto-merge
// on for each merge request a person accepted, its head's verdict passed:
// each merges once its own head is green, none brought up to date first
// (ADR-0042, ADR-0051; merge.go). Verdict, a head's verdict the line wrote, is kept
// first, in hand for this run and, where the forge keeps it, for the next.
func Follow(o FollowOptions) *Result {
	repo, forgeSpec, base, gardening := o.Repo, o.Forge, o.Base, o.Gardening
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
	if o.Group != "" {
		return queueGroup(f, o, res)
	}
	if base == "" {
		if base, err = git(repo, nil, "symbolic-ref", "-q", "--short", "HEAD"); err != nil {
			return fail(errors.New("on no branch: give the branch the merge requests go into with --base"))
		}
	}
	local := forge.KeepsBranches(f)
	hand := map[string]kept{}
	m, merges := f.(forge.Merger)
	switch {
	case o.Verdict != "" && o.Merge != "" && merges && !local:
		finding, err := keepVerdict(m, o.Verdict, hand)
		if err != nil {
			return fail(err)
		}
		res.Findings = append(res.Findings, finding)
	case o.Verdict != "" && o.Merge != "":
		res.Findings = append(res.Findings, verdict.Finding{Rule: "verdict-not-kept", Level: "warn",
			Message: "this forge has no auto-merge workline can turn on: the line's verdict is not kept"})
	case o.Verdict != "":
		res.Findings = append(res.Findings, verdict.Finding{Rule: "verdict-not-kept",
			Message: "the project sets no merge policy (`merge:`): the line's verdict is not kept"})
	}
	if o.VerdictOnly {
		res.Summary = "the line's verdict kept, nothing else followed"
		return res
	}
	open, err := f.OpenMergeRequests("workline/")
	if err != nil {
		return fail(err)
	}
	n, rebuilt, closed := 0, 0, 0
	for _, branch := range open {
		role, task, ok := strings.Cut(strings.TrimPrefix(branch, "workline/"), "/")
		if !ok || task != "release" && !gardening {
			continue
		}
		n++
		finding, err := follow(repo, base, branch, role, local, task != "release")
		if err != nil {
			return fail(err)
		}
		switch finding.Rule {
		case "":
			continue
		case "no-longer-applies":
			res.Status = verdict.Human
		case "rebuilt":
			rebuilt++
		case "closed":
			// The forge's own close, its comment saying why; the branch
			// stays, for the next run of the task to propose again from.
			if err := f.CloseMergeRequest(branch, "Closed by workline's "+role+": "+finding.Message+".", forge.Marker("closed")); err != nil {
				return fail(err)
			}
			closed++
		}
		res.Findings = append(res.Findings, finding)
	}
	if gardening {
		res.Summary = fmt.Sprintf("%d merge requests of the roles open, %d rebuilt on %s, %d closed", n, rebuilt, base, closed)
	} else {
		res.Summary = fmt.Sprintf("%d release merge requests open, %d rebuilt on %s", n, rebuilt, base)
	}
	o.Base = base
	if o.People {
		sum, err := people(f, o, local, res)
		if err != nil {
			return fail(err)
		}
		res.Summary += "; " + sum
	}
	if o.Merge != "" {
		sum, err := mergePolicy(f, o, local, res, hand)
		if err != nil {
			return fail(err)
		}
		res.Summary += "; " + sum
	}
	return res
}

// follow rebuilds one role's branch on base's tip, and says what it did;
// nothing when the branch is on base's tip already. A gardening task's
// branch with nothing left to merge is said `closed`, for Follow to close.
func follow(repo, base, branch, role string, local, gardening bool) (verdict.Finding, error) {
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
		if !gardening {
			return verdict.Finding{Rule: "not-rebuilt", Level: "warn", Where: branch,
				Message: fmt.Sprintf("%s moved under it, but %s: not rebuilt over it; rebase it by hand, or close it and the next release run proposes the fix again", base, who)}, nil
		}
		// Theirs now: said only when it no longer merges, as the forge
		// shows it — git's own merge, in memory — or when that could not
		// be told, never as a conflict it may not be.
		clash, err := conflicts(repo, baseTip, tip)
		switch {
		case err != nil:
			return verdict.Finding{Rule: "conflict-unread", Level: "warn", Where: branch,
				Message: fmt.Sprintf("%s: left to them, neither rebuilt nor closed; whether it is in conflict with %s could not be told: %v", who, base, err)}, nil
		case clash:
			return verdict.Finding{Rule: "not-rebuilt", Level: "warn", Where: branch,
				Message: fmt.Sprintf("in conflict with %s, and %s: left to them, to rebase or close", base, who)}, nil
		}
		return verdict.Finding{}, nil
	}
	tree, clashes, err := redo(repo, baseTip, tip)
	if err != nil {
		return verdict.Finding{}, err
	}
	if len(clashes) > 0 {
		if gardening {
			return verdict.Finding{Rule: "closed", Where: branch,
				Message: fmt.Sprintf("%s changed the same lines of %s since, so its change no longer applies; the next gardening judges the docs again on %s and proposes what is still needed", base, strings.Join(clashes, ", "), base)}, nil
		}
		return verdict.Finding{Rule: "no-longer-applies", Where: branch,
			Message: fmt.Sprintf("its change no longer applies on %s, which changed the same lines of %s: left as it is, for a person to rebase or close", base, strings.Join(clashes, ", "))}, nil
	}
	same, _ := git(repo, nil, "rev-parse", baseTip+"^{tree}")
	if tree == same {
		if gardening {
			return verdict.Finding{Rule: "closed", Where: branch,
				Message: fmt.Sprintf("%s holds its change already: nothing is left to merge", base)}, nil
		}
		return verdict.Finding{Rule: "already-on-base", Level: "warn", Where: branch,
			Message: fmt.Sprintf("%s holds its change already: nothing left to merge; close it", base)}, nil
	}
	if gardening {
		// A record only: closed once base changed the docs it records —
		// read again or vouched for there, it says nothing more. Base
		// moving elsewhere leaves it true: rebuilt, so its docs are not
		// judged again, an agent asked for the same record (#303).
		docs, err := recordOnly(repo, baseTip, tree)
		if err != nil {
			return verdict.Finding{}, err
		}
		if len(docs) > 0 {
			from, err := git(repo, nil, "merge-base", baseTip, tip)
			if err != nil {
				return verdict.Finding{}, err
			}
			moved, err := git(repo, nil, append([]string{"diff", "--name-only", from, baseTip, "--"}, docs...)...)
			if err != nil {
				return verdict.Finding{}, err
			}
			if moved != "" {
				return verdict.Finding{Rule: "closed", Where: branch,
					Message: fmt.Sprintf("it only records when docs were read (`judged`), and %s changed %s since; the next gardening judges again those still due", base, strings.Join(strings.Split(moved, "\n"), ", "))}, nil
			}
		}
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

// conflicts says whether merging tip into baseTip meets a conflict, as the
// forge shows it: git's own merge, in memory (git merge-tree, 2.38). A
// variable, for a test to make git fail.
var conflicts = func(repo, baseTip, tip string) (bool, error) {
	out, err := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "--no-messages", baseTip, tip).CombinedOutput()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return false, nil
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		return true, nil
	}
	return false, fmt.Errorf("git merge-tree: %v: %s", err, strings.TrimSpace(string(out)))
}

// recordKey is a header line recording when a doc was read, and by whom:
// never what the doc says (ADR-0013).
var recordKey = regexp.MustCompile(`^(judged|judged-in-parts|verified):`)

// recordOnly gives the docs, when going from baseTip to tree changes
// nothing but the header lines recording when they were read — what
// merging a merge request holding only `judged` would bring; none when it
// changes anything else.
func recordOnly(repo, baseTip, tree string) ([]string, error) {
	changes, err := git(repo, nil, "diff", "--name-status", "--no-renames", "-z", baseTip, tree)
	if err != nil {
		return nil, err
	}
	var docs []string
	parts := strings.Split(strings.TrimSuffix(changes, "\x00"), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] != "M" {
			return nil, nil
		}
		was, _ := blob(repo, baseTip, parts[i+1])
		now, _ := blob(repo, tree, parts[i+1])
		if unrecorded(was) != unrecorded(now) {
			return nil, nil
		}
		docs = append(docs, parts[i+1])
	}
	return docs, nil
}

// unrecorded is a doc without the lines of its header that record when it
// was read: a front matter between `---` lines, or a header between
// `<!-- workline` and `-->` where a front matter would show.
func unrecorded(doc string) string { return unheaded(doc, recordKey.MatchString) }

// unworked is a doc without the lines of its header workline writes
// (workKeys, ADR-0041): those recording when it was read, and `checked`.
func unworked(doc string) string {
	return unheaded(doc, func(line string) bool {
		k := frontKey.FindStringSubmatch(line)
		return k != nil && workKeys[k[1]]
	})
}

// unheaded is a doc without the lines of its header drop names, the header
// read as unrecorded says; a doc with no header, or one never closed, as
// it is.
func unheaded(doc string, drop func(line string) bool) string {
	lines := strings.Split(doc, "\n")
	end := ""
	switch {
	case len(lines) > 0 && lines[0] == "---":
		end = "---"
	case len(lines) > 0 && strings.TrimSpace(lines[0]) == "<!-- workline":
		end = "-->"
	default:
		return doc
	}
	out := []string{lines[0]}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == end {
			return strings.Join(append(out, lines[i:]...), "\n")
		}
		if !drop(lines[i]) {
			out = append(out, lines[i])
		}
	}
	return doc
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
		if strings.Contains(fields[1], " ") || madeBy(fields[3]) != role {
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
	o, u, t := front(old), front(ours), front(theirs)
	if !o.ok || !u.ok || !t.ok || !o.same(u) || !o.same(t) {
		return mergeFile(tmp, old, ours, theirs)
	}
	head, ok, _ := mergeFront(o.head, u.head, t.head, nil)
	if !ok {
		return mergeFile(tmp, old, ours, theirs)
	}
	body, ok := mergeFile(tmp, o.body, u.body, t.body)
	if !ok {
		return "", false
	}
	return u.open + head + u.close + body, true
}

// header is a doc's header, split: its first line and its last, the lines
// between, and the body after it.
type header struct {
	open, close string
	head        []string
	body        string
	ok          bool // false when the doc has none
}

// same says whether two headers open and close alike.
func (h header) same(o header) bool { return h.open == o.open && h.close == o.close }

// front splits a doc into its header and its body: a front matter between
// `---` lines, or, where a front matter would show (a README), a header
// between `<!-- workline` and `-->`, as the documentalist reads it.
func front(s string) header {
	lines := strings.SplitAfter(s, "\n")
	end := ""
	switch first := strings.TrimSuffix(lines[0], "\n"); {
	case first == "---":
		end = "---"
	case strings.TrimSpace(first) == "<!-- workline":
		end = "-->"
	default:
		return header{body: s}
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == end && strings.HasSuffix(lines[i], "\n") {
			return header{open: lines[0], close: lines[i], head: lines[1:i], body: strings.Join(lines[i+1:], ""), ok: true}
		}
	}
	return header{body: s}
}

// mergeFront merges headers whose every line sets one key: a key the
// branch changed takes the branch's line, one only base changed base's, one
// both changed base's — the later judgement; a key the branch added goes
// after the key it followed. read is false for a header it cannot read
// line by line. With settles, base's is taken only for the keys it names
// when both changed one: another both changed is a clash, ok false.
func mergeFront(old, ours, theirs []string, settles func(key string) bool) (head string, ok, read bool) {
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
	o, oOrder, ok1 := keyed(old)
	u, uOrder, ok2 := keyed(ours)
	t, tOrder, ok3 := keyed(theirs)
	if !ok1 || !ok2 || !ok3 {
		return "", false, false
	}
	for _, k := range slices.Concat(oOrder, uOrder, tOrder) {
		ov, inOld := o[k]
		uv, inOurs := u[k]
		tv, inTheirs := t[k]
		baseMoved := inOld != inOurs || uv != ov
		branchMoved := inOld != inTheirs || tv != ov
		if settles != nil && baseMoved && branchMoved && (inOurs != inTheirs || uv != tv) && !settles(k) {
			return "", false, true
		}
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
	return out.String(), true, true
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
	sign, err := signing(repo)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", slices.Concat([]string{"-C", repo, "commit-tree"}, sign, []string{tree, "-p", parent, "-F", "-"})...)
	cmd.Stdin = strings.NewReader(msg + "\n")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+a[0], "GIT_AUTHOR_EMAIL="+a[1], "GIT_AUTHOR_DATE="+a[2])
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("git commit-tree%s: %s", signed(sign), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// signing is how commit-tree signs a commit workline pushes: `-S` when git
// is set to sign commits (`commit.gpgsign`), with the key and format git
// is given (`user.signingkey`, `gpg.format`), as `git commit` signs; none
// when it is not. commit-tree reads none of it itself, and a forge that
// refuses unsigned commits refused its push (#549). A setting git cannot
// read fails, as it fails `git commit`.
func signing(repo string) ([]string, error) {
	out, err := exec.Command("git", "-C", repo, "config", "--bool", "--get", "commit.gpgsign").Output()
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		return nil, nil // not set
	case errors.As(err, &ee):
		return nil, fmt.Errorf("git config commit.gpgsign: %s", strings.TrimSpace(string(ee.Stderr)))
	case err != nil:
		return nil, err
	case strings.TrimSpace(string(out)) == "true":
		return []string{"-S"}, nil
	}
	return nil, nil
}

// signed says, in a failed commit's error, that it was signing as git is
// set to: a key missing, unreadable or refused is told apart from the rest.
func signed(sign []string) string {
	if len(sign) == 0 {
		return ""
	}
	return " (signing, as commit.gpgsign asks)"
}

func short(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
