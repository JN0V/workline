package forge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// The local forge's merge (#1322, ADR-0042 amended): it has no auto-merge
// to turn on, so it completes the mechanism. The merge policy weighs a
// local merge request as on any forge — the same verdict, the same weigh —
// and its last act, turning auto-merge on, is the merge itself: the
// branch's head merged into its base, in this clone, nothing pushed.

// LocalApp is who the local forge says workline is: the author of the
// merge requests from a role's branch (`workline/<role>/<task>`). Nothing
// in the clone tells a role's write from the person's but its branch and
// its commits' trailers, which the merge policy reads (ADR-0042, `normal`).
const LocalApp = "workline"

var _ Merger = &Local{}

// LocalEvent is a label set on a merge request, or taken off, when, and
// the head its branch was on then: a yes covers that head alone.
type LocalEvent struct {
	Label string `yaml:"label"`
	Added bool   `yaml:"added"`
	At    string `yaml:"at"` // RFC 3339
	Head  string `yaml:"head,omitempty"`
}

// app: workline's merge requests are its roles' branches; the commits a
// role makes here carry the clone's own name, so no committer tells them
// apart: "" — their trailers and trees do (engine's notCoded, notOwn).
func (l *Local) app() (string, string, error) { return LocalApp, "", nil }

// author is who opened a merge request of this clone: workline for a
// role's branch, else the person.
func localAuthor(branch string) string {
	if strings.HasPrefix(branch, "workline/") {
		return LocalApp
	}
	return ""
}

// person is the name this clone's commits are made under: the one who
// works in it, every write by command theirs.
func (l *Local) person() string {
	out, err := exec.Command("git", "-C", l.Repo, "var", "GIT_COMMITTER_IDENT").Output()
	if name, _, ok := strings.Cut(string(out), " <"); err == nil && ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return "the person of this clone"
}

// tip is a branch's commit here, "" when it is gone.
func (l *Local) tip(branch string) string {
	out, err := exec.Command("git", "-C", l.Repo, "rev-parse", "-q", "--verify", "refs/heads/"+branch+"^{commit}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// labelled records on a merge request each label set or taken off, with
// the head its branch is on: the history a forge keeps of its labels.
func (l *Local) labelled(it *LocalItem, add, remove []string) {
	at, head := time.Now().UTC().Format(time.RFC3339), l.tip(it.Branch)
	for _, a := range add {
		if !slices.Contains(it.Labels, a) {
			it.Events = append(it.Events, LocalEvent{Label: a, Added: true, At: at, Head: head})
		}
	}
	for _, r := range remove {
		if slices.Contains(it.Labels, r) {
			it.Events = append(it.Events, LocalEvent{Label: r, At: at, Head: head})
		}
	}
}

// MergeState reads a merge request of this clone: its branch's tip, its
// labels' history, each yes by the person of the clone, who may merge
// here, and the head each was given on. A label on with no history — put
// by hand in its file — is dated nowhere: a yes covering no head. Nothing
// is pushed here, no check runs: who wrote its commits is told by them.
func (l *Local) MergeState(mr MergeRequest) (MergeState, error) {
	it, err := l.Item("merge-request", mr.ID)
	if err != nil {
		return MergeState{}, err
	}
	person := l.person()
	st := MergeState{Head: l.tip(it.Branch), Draft: it.Draft, Mergeable: true}
	for _, e := range it.Events {
		st.Labels = append(st.Labels, LabelEvent{Label: e.Label, Added: e.Added, Writer: true, Note: Note{Author: person, Insider: true, Created: e.At}})
		if e.Added && e.Head != "" {
			st.Pushes = append(st.Pushes, Push{Head: e.Head, At: e.At})
		}
	}
	for _, label := range it.Labels {
		if !slices.ContainsFunc(it.Events, func(e LocalEvent) bool { return e.Label == label }) {
			st.Labels = append(st.Labels, LabelEvent{Label: label, Added: true, Writer: true, Note: Note{Author: person, Insider: true}})
		}
	}
	by := person
	if a := localAuthor(it.Branch); a != "" {
		by = a
	}
	st.Pushers = []string{by}
	return st, nil
}

// Guarded: the merge is made here, of the head weighed, in the run that
// weighed it: a commit after the yes is never merged with it.
func (l *Local) Guarded(base string) (string, error) { return "", nil }

// Accept: nothing to hold; the merge takes the head the yes covers.
func (l *Local) Accept(head string, ok bool, why string) error { return nil }

// localVerdict is the line's verdict kept on a head, in the clone.
type localVerdict struct {
	Class string `json:"class"`
	Why   string `json:"why"`
}

var fullSHA = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

func (l *Local) verdictPath(head string) (string, error) {
	if !fullSHA.MatchString(head) {
		return "", fmt.Errorf("a verdict is kept on a commit's full id, not %q", head)
	}
	d, err := l.dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "verdicts", head+".json"), nil
}

// KeepVerdict keeps the line's verdict on head in the clone,
// .git/workline/verdicts/<head>.json: a new head has none until the line
// runs on it.
func (l *Local) KeepVerdict(head, class, why string) error {
	if err := l.writable(); err != nil {
		return err
	}
	p, err := l.verdictPath(head)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(localVerdict{Class: class, Why: why})
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

// Verdict reads back the verdict kept on head; "" none.
func (l *Local) Verdict(head string) (string, string, error) {
	p, err := l.verdictPath(head)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var v localVerdict
	if err := json.Unmarshal(data, &v); err != nil {
		return "", "", fmt.Errorf("%s: %v", p, err)
	}
	return v.Class, v.Why, nil
}

// mark keeps what the merge policy read on a base commit, in the clone.
func (l *Local) mark(commit, text string) error {
	if err := l.writable(); err != nil {
		return err
	}
	d, err := l.dir()
	if err != nil {
		return err
	}
	p := filepath.Join(d, "marks", commit)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(text+"\n"), 0o644)
}

func (l *Local) marked(commit string) (string, bool, error) {
	d, err := l.dir()
	if err != nil {
		return "", false, err
	}
	data, err := os.ReadFile(filepath.Join(d, "marks", commit))
	if err != nil {
		return "", false, nil
	}
	return strings.TrimSpace(string(data)), true, nil
}

// closing: the local forge keeps no history; an issue closed here was
// closed by the person of the clone, who may merge here — the engine
// never closes a demotion's issue.
func (l *Local) closing(id int) (LabelEvent, bool, error) {
	it, err := l.Item("issue", id)
	if err != nil || it.State == "open" {
		return LabelEvent{}, false, err
	}
	return LabelEvent{Writer: true, Note: Note{Author: l.person(), Insider: true}}, true, nil
}

// AutoMerge on merges the merge request now, as a forge's auto-merge
// would once its rules are met: head, the one weighed, merged into its
// base with a merge commit naming it (`See merge request !<n>`, read back
// for a revert). Refused, nothing merged, when its branch moved since,
// when it conflicts with its base, or when the base is checked out with
// uncommitted changes; a worktree is never left mid-merge. Off: nothing
// was turned on.
func (l *Local) AutoMerge(mr MergeRequest, head string, on bool) (bool, error) {
	if !on {
		return false, nil
	}
	if err := l.writable(); err != nil {
		return false, fmt.Errorf("%w: %v", errRefused, err)
	}
	it, err := l.Item("merge-request", mr.ID)
	if err != nil {
		return false, err
	}
	refuse := func(format string, a ...any) (bool, error) {
		return false, fmt.Errorf("%w: "+format, append([]any{errRefused}, a...)...)
	}
	switch {
	case it.Draft:
		return refuse("a draft is never merged")
	case it.Branch == "" || it.Base == "":
		return refuse("it names no branch or no base")
	case l.tip(it.Branch) != head:
		return refuse("its branch %s moved since %s was weighed", it.Branch, shortID(head))
	}
	base := l.tip(it.Base)
	if base == "" {
		return refuse("its base %s is not in this clone", it.Base)
	}
	git := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return strings.TrimSpace(string(out)), fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := git(l.Repo, "merge-base", "--is-ancestor", head, base); err == nil {
		return true, nil // its base holds it already
	}
	out, err := exec.Command("git", "-C", l.Repo, "merge-tree", "--write-tree", "--name-only", "--no-messages", base, head).Output()
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee) && ee.ExitCode() == 1:
		// The tree, then the files that conflict, one a line.
		c := &Conflict{Base: it.Base}
		_, files, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
		for f := range strings.Lines(files) {
			if f = strings.TrimSpace(f); f != "" {
				c.Files = append(c.Files, f)
			}
		}
		return false, fmt.Errorf("%w: %w", errRefused, c)
	case err != nil:
		return false, fmt.Errorf("git merge-tree (2.38 or later): %v", err)
	}
	tree, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	wt, err := checkedOut(l.Repo, it.Base)
	if err != nil {
		return false, err
	}
	if wt != "" {
		dirty, err := git(wt, "status", "--porcelain", "--untracked-files=no")
		if err != nil {
			return false, err
		}
		if dirty != "" {
			return refuse("%s is checked out at %s with uncommitted changes: commit or stash them, and the next `workline follow` merges it", it.Base, wt)
		}
	}
	msg := fmt.Sprintf("Merge branch '%s' into '%s'\n\n%s\n\nSee merge request !%d\n", it.Branch, it.Base, it.Title, it.ID)
	args := []string{"commit-tree", tree, "-p", base, "-p", head, "-F", "-"}
	if signed, _ := git(l.Repo, "config", "--bool", "--get", "commit.gpgsign"); signed == "true" {
		args = append([]string{"commit-tree", "-S"}, args[1:]...)
	}
	cmd := exec.Command("git", append([]string{"-C", l.Repo}, args...)...)
	cmd.Stdin = strings.NewReader(msg)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git commit-tree: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	commit := strings.TrimSpace(string(out))
	if wt != "" {
		// The base checked out moves with its files, a fast-forward to the
		// merge made: git refuses it whole, or makes it whole.
		if _, err := git(wt, "merge", "--ff-only", "-q", commit); err != nil {
			return refuse("%s, checked out at %s, could not move to the merge: %v", it.Base, wt, err)
		}
		return true, nil
	}
	if _, err := git(l.Repo, "update-ref", "refs/heads/"+it.Base, commit, base); err != nil {
		return refuse("%s moved while it merged: %v", it.Base, err)
	}
	return true, nil
}

// checkedOut is the worktree of the clone where branch is checked out,
// "" none.
func checkedOut(repo, branch string) (string, error) {
	out, err := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %v", err)
	}
	dir := ""
	for _, line := range strings.Split(string(out), "\n") {
		if d, ok := strings.CutPrefix(line, "worktree "); ok {
			dir = d
		}
		if line == "branch refs/heads/"+branch {
			return dir, nil
		}
	}
	return "", nil
}

func shortID(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}
