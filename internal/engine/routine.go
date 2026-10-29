package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/steer"
)

// The merge policy's `normal` level (ADR-0042, amended): workline's own
// routine merge requests — as the forge says it: opened and pushed by the
// App, its commits and pages workline's — merge with no person's yes, the
// line passed on the head; a kind of role.MergeAlone (`reader`) only once
// the project lets it. Each is recorded on it; a revert of one sends its
// kind back to a person's yes (demote.go).

// MergeNormal is the merge policy's level where workline's own routine
// merge requests need no person's yes (role.MergeLevels).
const MergeNormal = "normal"

// The kinds of workline's routine merge requests, told by what their change
// is, never by their branch's name nor a trailer: the higher wins.
const (
	kindRecords = "records" // only the lines recording when docs were read
	kindDerived = "derived" // derived blocks as their commands give them, and records
	kindDocs    = "docs"    // doc pages a role wrote
	kindReader  = "reader"  // pages an agent read for their reader, recorded read (`reader:`), reworded
	kindAll     = "all"     // every kind: a revert that names nothing workline can find
	kindCode    = "code"    // the developer's code for an issue no feature (code.go)
)

// RoutineKinds are the kinds, from the narrowest.
var RoutineKinds = []string{kindRecords, kindDerived, kindDocs, kindReader}

// mergeKinds are the kinds recorded on a merge with no person's yes, a
// revert demoting each: the routine ones, and the developer's code.
var mergeKinds = append(slices.Clone(RoutineKinds), kindCode)

// kindSays says what a kind's change is, for a person.
var kindSays = map[string]string{
	kindRecords: "it only records when docs were read",
	kindDerived: "it regenerates derived blocks, as their commands give them",
	kindDocs:    "it changes doc pages a role wrote",
	kindReader:  "it rewords pages an agent read for their reader",
}

// normal is what `normal` reads once a run: the App workline's token writes
// as and the name its jobs commit with ("" a person's token: nothing is
// workline's own), the pages agents are told to read, the kinds of
// role.MergeAlone the project lets merge alone, and the kinds waiting for
// a person's yes again, each with the issue that keeps it.
type normal struct {
	app, committer string
	steers         func(path string) bool
	alone          []string
	demoted        map[string]string
}

// asked says whether a kind waits for a person's yes until the project
// lets it merge alone (`merge.alone`), and has not.
func (n *normal) asked(kind string) bool {
	return slices.Contains(role.MergeAlone, kind) && !slices.Contains(n.alone, kind)
}

// routineCandidate says whether a merge request may be workline's own
// routine one, at `normal`: from a role's branch of this repository
// (`workline/<role>/<task>`), opened by workline's App, not a release
// tool's. Who pushed, its commits, pages, kind and verdict are weighed
// after (weighRoutine).
func routineCandidate(o FollowOptions, n *normal, mr forge.MergeRequest) bool {
	role, task, ok := strings.Cut(strings.TrimPrefix(mr.Branch, "workline/"), "/")
	switch {
	case n == nil, n.app == "", !mr.Here, !strings.HasPrefix(mr.Branch, "workline/"), !ok, role == "", task == "":
		return false
	case o.Release != nil && o.Release(mr.Branch):
		return false
	}
	return mr.Author == n.app
}

// notOwn names the first commit of the branch, past base, that is not
// workline's own: not committed under the name workline's jobs commit
// with, or not workline's own by its tree (OwnCommit); "" when every one
// is.
func notOwn(repo, baseTip, tip, committer string) (string, error) {
	out, err := git(repo, nil, "log", "--reverse", "--format=%H%x1f%cn", baseTip+".."+tip)
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(out, "\n") {
		c, name, ok := strings.Cut(l, "\x1f")
		if !ok {
			continue
		}
		if name != committer || OwnCommit(repo, c, baseTip) == "" {
			return commitSaid(repo, c) + ", committed by " + name, nil
		}
	}
	return "", nil
}

// steering is, at tip, what says whether a page steers agents
// (internal/steer).
var steering = steer.Files

// notAllowed names the first file the branch changes since it left base
// that workline may not change with no person's yes: one that is no doc of
// the documentalist's (its `docs` setting), not a doc page, or one agents
// are told to read; "" when there is none.
func notAllowed(repo, baseTip, tip string, isDoc, steers func(string) bool) (string, error) {
	from, err := git(repo, nil, "merge-base", baseTip, tip)
	if err != nil {
		return "", err
	}
	files, err := git(repo, nil, "diff", "--name-only", "--no-renames", "-z", from, tip)
	if err != nil {
		return "", err
	}
	for _, p := range strings.Split(strings.TrimSuffix(files, "\x00"), "\x00") {
		switch {
		case p == "":
		case isDoc == nil || !isDoc(p) || !docPage(p):
			return p + ", no doc of the documentalist's", nil
		case steers(p):
			return p + ", which agents are told to read", nil
		}
	}
	return "", nil
}

// routineKind is the kind of the branch's change since it left base:
// reader when it records a page read for its reader — a `reader:` line of
// a doc's header written —, whatever else it changes; records when it
// changes only the lines recording when docs were read; derived when it
// changes derived blocks besides, and regenerated says each reads what its
// command gives on tip; docs else.
func routineKind(repo, baseTip, tip string, regenerated func(docs []string) bool) (string, error) {
	from, err := git(repo, nil, "merge-base", baseTip, tip)
	if err != nil {
		return "", err
	}
	changes, err := git(repo, nil, "diff", "--name-status", "--no-renames", "-z", from, tip)
	if err != nil {
		return "", err
	}
	var blocks []string
	parts := strings.Split(strings.TrimSuffix(changes, "\x00"), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] == "D" {
			continue
		}
		was, _ := blob(repo, from, parts[i+1])
		if now, ok := blob(repo, tip, parts[i+1]); ok && readMark(now) != "" && readMark(now) != readMark(was) {
			return kindReader, nil
		}
	}
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] != "M" {
			return kindDocs, nil
		}
		was, ok1 := blob(repo, from, parts[i+1])
		now, ok2 := blob(repo, tip, parts[i+1])
		if !ok1 || !ok2 {
			return "", fmt.Errorf("%s, changed on the branch, could not be read at %s and %s", parts[i+1], short(from), short(tip))
		}
		switch {
		case unrecorded(was) == unrecorded(now):
		case underived(unrecorded(was)) == underived(unrecorded(now)):
			blocks = append(blocks, parts[i+1])
		default:
			return kindDocs, nil
		}
	}
	switch {
	case len(blocks) == 0:
		return kindRecords, nil
	case regenerated != nil && regenerated(blocks):
		return kindDerived, nil
	}
	return kindDocs, nil // derived blocks not as their commands give them
}

// readerKey is the header line the documentalist's reader task writes on
// each page it had an agent read: the hash of the body read.
var readerKey = regexp.MustCompile(`^reader:\s*\S+\s*$`)

// readMark is a doc's `reader:` line, its header read as unrecorded reads
// it; "" when it holds none, or has no header closed.
func readMark(doc string) string {
	lines := strings.Split(doc, "\n")
	end := ""
	switch {
	case len(lines) > 0 && lines[0] == "---":
		end = "---"
	case len(lines) > 0 && strings.TrimSpace(lines[0]) == "<!-- workline":
		end = "-->"
	default:
		return ""
	}
	mark := ""
	for _, l := range lines[1:] {
		l = strings.TrimSpace(l)
		switch {
		case l == end:
			return mark
		case readerKey.MatchString(l):
			mark = l
		}
	}
	return ""
}

// underived is a doc with its derived blocks emptied, as the documentalist
// reads them: a marker with no end of its own is no block, and the text
// after it stays the doc's.
func underived(doc string) string {
	var b strings.Builder
	last := 0
	for _, blk := range derivedBlocks(doc) {
		b.WriteString(doc[last:blk.start])
		fmt.Fprintf(&b, "<!-- workline:derive %s --><!-- workline:end -->", blk.name)
		last = blk.end
	}
	b.WriteString(doc[last:])
	return b.String()
}

// regenerates says whether docs' derived blocks, at tip, read what their
// commands give on tip's tree (o.Derive, the documentalist's `derive`
// settings, run in a worktree of tip); false when it cannot be told.
func regenerates(o FollowOptions, tip string, docs []string) bool {
	if o.Derive == nil {
		return false
	}
	tmp, err := os.MkdirTemp("", "workline-derive-")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, "tree")
	if _, err := git(o.Repo, nil, "worktree", "add", "-q", "--detach", dir, tip); err != nil {
		return false
	}
	defer git(o.Repo, nil, "worktree", "remove", "--force", dir)
	contents := map[string]string{}
	for _, d := range docs {
		text, ok := blob(o.Repo, tip, d)
		if !ok {
			return false
		}
		contents[d] = text
	}
	return o.Derive(dir, contents)
}

// weighed is a merge request as weigh read it, and how it acts on it.
type weighed struct {
	mr           forge.MergeRequest
	st           forge.MergeState
	baseTip, tip string
	e            *inLine
	say          func(rule, level, msg string)
	takeBack     func(why string) error
}

// weighRoutine weighs one of workline's own merge requests no person said
// yes to, at `normal`, and does what the policy says of it: auto-merge on
// and the yes held on its head with no person's yes, the merge recorded on
// it; or why it waits for a person's. It returns it when it is accepted.
func weighRoutine(f forge.Forge, m forge.Merger, o FollowOptions, w weighed, n *normal, hand map[string]kept) (*inLine, error) {
	mr, st, tip := w.mr, w.st, w.tip
	ours := st.AutoMerge && st.AutoMergeBot
	waits := func(level, why string) (*inLine, error) {
		if ours {
			if err := w.takeBack(why + ": put the label to accept it"); err != nil {
				return nil, err
			}
		}
		w.say("waits-for-a-yes", level, why+": it waits for a person's yes, the label `workline:accepted`")
		return nil, nil
	}
	if st.Draft {
		if ours {
			if err := w.takeBack("marked a draft"); err != nil {
				return nil, err
			}
			w.say("draft", "", "workline's own, marked a draft: auto-merge turned off, until a person labels it `workline:accepted`")
			return nil, nil
		}
		w.say("draft", "", "workline's own, but a draft: nothing turned on")
		return nil, nil
	}
	// Who pushed, as the forge says: a commit says anything of itself. None
	// named is not known: never taken for the App's.
	if len(st.Pushers) == 0 {
		return waits("", "opened by workline, but who pushed to its branch the forge does not say")
	}
	for _, p := range st.Pushers {
		if p != n.app {
			who := "a hand the forge does not name"
			if p != "" {
				who = "@" + p
			}
			return waits("", fmt.Sprintf("opened by workline, but %s pushed to its branch", who))
		}
	}
	if strings.HasPrefix(mr.Branch, developer.Branch) {
		return weighCode(f, m, o, w, n, hand, waits)
	}
	who, err := notOwn(o.Repo, w.baseTip, tip, n.committer)
	if err != nil {
		return nil, err
	}
	if who != "" {
		return waits("", fmt.Sprintf("from workline's branch, but %s is not workline's own", who))
	}
	page, err := notAllowed(o.Repo, w.baseTip, tip, o.IsDoc, n.steers)
	if err != nil {
		return nil, err
	}
	if page != "" {
		return waits("", "workline's own, but it changes "+page)
	}
	kind, err := routineKind(o.Repo, w.baseTip, tip, func(docs []string) bool { return regenerates(o, tip, docs) })
	if err != nil {
		return nil, err
	}
	if n.asked(kind) {
		// Said once on it, where a person reviewing looks: how to accept
		// it, and how to let its kind merge alone.
		t := forge.Target{Kind: "merge-request", ID: mr.ID}
		body := fmt.Sprintf("**Waiting for a person's yes**: this is workline's own %s merge request — %s. The project's merge policy is `normal`, but this kind merges with no person's yes only once the project lets it.\n\n- To accept this one, put the label `%s`.\n- To let workline's %s merge requests merge alone, add the kind to the policy in `.workline/config.yaml`: `merge: {autonomy: normal, alone: [%s]}`.",
			kind, kindSays[kind], backlog.AcceptedLabel(f), kind, kind)
		if err := f.Comment(t, body, forge.Marker("merge-waits="+kind)); err != nil {
			return nil, err
		}
		return waits("", fmt.Sprintf("workline's own %s (%s), which merges with no person's yes only once the project lets it, `merge: {autonomy: normal, alone: [%s]}`", kind, kindSays[kind], kind))
	}
	return admit(f, m, o, w, n, hand, waits, kind, kindSays[kind])
}

// admit weighs workline's own merge request of kind, says what its change
// is, past what tells its kind: held since a revert of its kind, vetoed,
// red, or its yes set on its head with no person's, recorded on it, and
// auto-merge on.
func admit(f forge.Forge, m forge.Merger, o FollowOptions, w weighed, n *normal, hand map[string]kept, waits func(level, why string) (*inLine, error), kind, says string) (*inLine, error) {
	mr, st, tip, e := w.mr, w.st, w.tip, w.e
	issue, held := n.demoted[kind]
	if all, ok := n.demoted[kindAll]; ok && !held {
		issue, held = all, true
	}
	if held {
		return waits("warn", fmt.Sprintf("workline's own %s (%s), but its %s merge requests wait for a person's yes since a revert (%s)", kind, says, kind, issue))
	}
	if !st.AutoMerge && st.AutoMergeAt != "" {
		// The veto by turning auto-merge off: it stays off until a person's
		// yes, as at `cautious`.
		if err := m.Accept(tip, false, "auto-merge turned off since it was turned on: put the label to accept it"); err != nil {
			return nil, err
		}
		w.say("vetoed", "", fmt.Sprintf("workline's own %s, its auto-merge turned off since it was turned on: left off until a person labels it `workline:accepted`", kind))
		return nil, nil
	}
	if st.ChecksUnread != "" {
		w.say("checks-unread", "warn", "its head's checks could not be read, only the line's verdict counts: "+st.ChecksUnread)
	}
	if st.Checks == "failure" {
		w.say("checks-red", "warn", fmt.Sprintf("workline's own %s, but a check failed on its head (%s): auto-merge left as it is", kind, short(tip)))
		return nil, nil
	}
	v, err := verdictOf(m, hand, tip)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return nil, err
		}
		w.say("verdict-unread", "warn", fmt.Sprintf("workline's own %s; its head's verdict could not be read: %s", kind, forge.Said(err)))
		return nil, nil
	}
	switch v.class {
	case "":
		w.say("verdict-missing", "", fmt.Sprintf("workline's own %s; no verdict of the line on its head (%s) yet: it waits for it", kind, short(tip)))
		return e, nil
	case verdictWaits: // marked ready, the run on ready not done yet
		w.say("review-to-come", "", fmt.Sprintf("workline's own %s; its head's (%s) verdict is the draft's, %s: it waits for the run on ready", kind, short(tip), v.why))
		return e, nil
	case verdictRed, verdictShort:
		// Held by the yes not given on this head (the status Accepted;
		// GitLab, the head's pipeline red): auto-merge left as it is, for
		// the next head the line passes.
		rule := map[string]string{verdictRed: "verdict-red", verdictShort: "review-short"}[v.class]
		if err := m.Accept(tip, false, v.why); err != nil {
			return nil, err
		}
		w.say(rule, "warn", fmt.Sprintf("workline's own %s, but %s: no yes on its head", kind, v.why))
		return nil, nil
	}
	why := fmt.Sprintf("workline's own %s — %s —, opened and pushed by its App, every commit its own; its head's verdict passed, every agent's review whole", kind, says)
	if err := m.Accept(tip, true, "with no person's yes: "+why); err != nil {
		return nil, err
	}
	if st.AutoMerge {
		return e, nil
	}
	// The record, where a person sees it and the engine finds it again
	// when a revert names this merge request.
	t := forge.Target{Kind: "merge-request", ID: mr.ID}
	body := fmt.Sprintf("**Merging with no person's yes**: the project's merge policy is `normal`, and this is workline's own %s: %s. Opened and pushed by workline's App, every commit its own, and the line passed on its head (%s), every agent's review whole. To hold it, mark it a draft or turn auto-merge off. Reverted, workline's %s merge requests wait for a person's yes again.",
		kind, says, short(tip), kind)
	if err := f.Comment(t, body, forge.Marker("merge-alone="+kind)); err != nil {
		return nil, err
	}
	merged, err := m.AutoMerge(mr, tip, true)
	switch {
	case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
		return nil, err
	case err != nil:
		w.say("auto-merge-refused", "warn", fmt.Sprintf("workline's own %s, its head's verdict passed, but the forge refused to turn auto-merge on: %s", kind, forge.Said(err)))
		return nil, nil
	case merged:
		w.say("merged-at-once", "", fmt.Sprintf("with no person's yes, %s: merged at once, the forge's rules met already", why))
		return nil, nil
	}
	w.say("auto-merge-on", "", fmt.Sprintf("with no person's yes, %s: auto-merge turned on, the forge merges it once its checks pass", why))
	return e, nil
}

// ownNote finds, among a merge request's comments, the one the App wrote
// whose marker starts with key: the marker's value after key, and whether
// there is one. Anyone else's comment ending with a marker is theirs.
func ownNote(notes []forge.Note, key, app string) (string, bool) {
	for _, n := range notes {
		last, engine := backlog.EngineMarker(n.Body)
		if !engine || app == "" || n.Author != app {
			continue
		}
		if v, ok := strings.CutPrefix(last, "<!-- workline:"+key); ok {
			return strings.TrimSpace(strings.TrimSuffix(v, "-->")), true
		}
	}
	return "", false
}

// alone sums up, for the run's summary, what merges with no person's yes
// at `normal`, what waits again since a revert, and what waits until the
// project lets it merge alone.
func alone(n *normal) string {
	if n.app == "" {
		return "with no person's yes: none, the token is no App's"
	}
	var free, held, asked []string
	for _, kind := range mergeKinds {
		if n.asked(kind) {
			asked = append(asked, kind)
			continue
		}
		issue, ok := n.demoted[kind]
		if all, every := n.demoted[kindAll]; every && !ok {
			issue, ok = all, true
		}
		if ok {
			held = append(held, kind+" ("+issue+")")
		} else {
			free = append(free, kind)
		}
	}
	s := "with no person's yes: workline's own merge requests, "
	if len(free) == 0 {
		s += "none"
	} else {
		s += strings.Join(free, ", ")
	}
	if len(held) > 0 {
		s += "; waiting for a person's yes again since a revert: " + strings.Join(held, ", ")
	}
	if len(asked) > 0 {
		s += "; only with a person's yes, until `merge.alone` lists them: " + strings.Join(asked, ", ")
	}
	return s
}
