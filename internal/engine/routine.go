package engine

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// The merge policy's `normal` level (ADR-0042, amended): workline's own
// routine merge requests merge with no person's yes — every commit
// workline's own, checked by its tree, the line passed on the head with
// every agent's review whole. Each such merge is recorded on the merge
// request; a revert of one sends its kind back to a person's yes
// (ADR-0026), in an issue a person closes to restore it.

// MergeNormal is the merge policy's level where workline's own routine
// merge requests need no person's yes (role.MergeLevels).
const MergeNormal = "normal"

// The kinds of workline's routine merge requests, told by what their change
// is, never by their branch's name nor a trailer: the higher wins.
const (
	kindRecords = "records" // only the lines recording when docs were read
	kindDerived = "derived" // derived blocks regenerated, and records
	kindDocs    = "docs"    // doc pages a role wrote
)

// RoutineKinds are the kinds, from the narrowest.
var RoutineKinds = []string{kindRecords, kindDerived, kindDocs}

// kindSays says what a kind's change is, for a person.
var kindSays = map[string]string{
	kindRecords: "it only records when docs were read",
	kindDerived: "it regenerates derived blocks",
	kindDocs:    "it changes doc pages a role wrote",
}

// revertWindow is how many commits of the base's history, from its tip,
// are read for reverts: each is read once, its demotion kept in an issue.
const revertWindow = 100

// routineCandidate says whether a merge request may be workline's own
// routine one, at `normal`: from a role's branch of this repository
// (`workline/<role>/<task>`), opened by workline — a bot, or the user
// `follow`'s token writes as —, not a release tool's. Its commits, its
// kind and its head's verdict are weighed after (weighRoutine).
func routineCandidate(f forge.Forge, o FollowOptions, mr forge.MergeRequest) bool {
	role, task, ok := strings.Cut(strings.TrimPrefix(mr.Branch, "workline/"), "/")
	switch {
	case o.Merge != MergeNormal, !mr.Here, !strings.HasPrefix(mr.Branch, "workline/"), !ok, role == "", task == "":
		return false
	case o.Release != nil && o.Release(mr.Branch):
		return false
	}
	return mr.Bot || mr.Author != "" && mr.Author == forge.Self(f)
}

// notOwn names the first commit of the branch, past base, that is not
// workline's own as OwnCommit tells it — by its tree, never its trailer
// alone —; "" when every one is.
func notOwn(repo, baseTip, tip string) (string, error) {
	out, err := git(repo, nil, "rev-list", "--reverse", baseTip+".."+tip)
	if err != nil {
		return "", err
	}
	for _, c := range strings.Fields(out) {
		if OwnCommit(repo, c, baseTip) == "" {
			return commitSaid(repo, c), nil
		}
	}
	return "", nil
}

// routineKind is the kind of the branch's change since it left base:
// records when it changes only the lines recording when docs were read,
// derived when it changes derived blocks besides, docs else.
func routineKind(repo, baseTip, tip string) (string, error) {
	from, err := git(repo, nil, "merge-base", baseTip, tip)
	if err != nil {
		return "", err
	}
	changes, err := git(repo, nil, "diff", "--name-status", "--no-renames", "-z", from, tip)
	if err != nil {
		return "", err
	}
	kind := kindRecords
	parts := strings.Split(strings.TrimSuffix(changes, "\x00"), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] != "M" {
			return kindDocs, nil
		}
		was, _ := blob(repo, from, parts[i+1])
		now, _ := blob(repo, tip, parts[i+1])
		switch {
		case unrecorded(was) == unrecorded(now):
		case underived(unrecorded(was)) == underived(unrecorded(now)):
			kind = kindDerived
		default:
			return kindDocs, nil
		}
	}
	return kind, nil
}

// underived is a doc with its derived blocks emptied.
func underived(doc string) string {
	return derived.ReplaceAllString(doc, "<!-- workline:derive $1 --><!-- workline:end -->")
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
// it; or why it waits for a person's. It returns it when it is in line.
func weighRoutine(f forge.Forge, m forge.Merger, o FollowOptions, w weighed, demoted map[string]string, hand map[string]kept) (*inLine, error) {
	mr, st, tip, e := w.mr, w.st, w.tip, w.e
	ours := st.AutoMerge && st.AutoMergeBot
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
	who, err := notOwn(o.Repo, w.baseTip, tip)
	if err != nil {
		return nil, err
	}
	if who != "" {
		if ours {
			if err := w.takeBack(who + " is not workline's own: put the label to accept it"); err != nil {
				return nil, err
			}
		}
		w.say("waits-for-a-yes", "", fmt.Sprintf("from workline's branch, but %s is not workline's own: it waits for a person's yes, the label `workline:accepted`", who))
		return nil, nil
	}
	kind, err := routineKind(o.Repo, w.baseTip, tip)
	if err != nil {
		return nil, err
	}
	if issue, ok := demoted[kind]; ok {
		if ours {
			if err := w.takeBack(kind + " waits for a person's yes since a revert: put the label to accept it"); err != nil {
				return nil, err
			}
		}
		w.say("waits-for-a-yes", "warn", fmt.Sprintf("workline's own %s (%s), but its %s merge requests wait for a person's yes since one was reverted (%s): the label `workline:accepted`", kind, kindSays[kind], kind, issue))
		return nil, nil
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
		w.say("checks-red", "warn", fmt.Sprintf("workline's own %s, but a check failed on its head (%s): out of line, the next one comes up", kind, short(tip)))
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
		w.say("verdict-missing", "", fmt.Sprintf("workline's own %s; no verdict of the line on its head (%s) yet: it waits for it, the next one comes up meanwhile", kind, short(tip)))
		return e, nil
	case verdictHeld, verdictRed, verdictShort:
		// Held by the yes not given on this head (the status Accepted;
		// GitLab, the head's pipeline red): auto-merge left as it is, for
		// the next head the line passes.
		rule := map[string]string{verdictHeld: "verdict-red", verdictRed: "verdict-red", verdictShort: "review-short"}[v.class]
		if err := m.Accept(tip, false, v.why); err != nil {
			return nil, err
		}
		w.say(rule, "warn", fmt.Sprintf("workline's own %s, but %s: no yes on its head, out of line", kind, v.why))
		return nil, nil
	}
	why := fmt.Sprintf("workline's own %s — %s —, every commit workline's own, checked by its tree; its head's verdict passed, every agent's review whole", kind, kindSays[kind])
	if err := m.Accept(tip, true, "with no person's yes: "+why); err != nil {
		return nil, err
	}
	if st.AutoMerge {
		e.ready = true
		return e, nil
	}
	// The record, where a person sees it and the engine finds it again
	// when a revert names this merge request.
	t := forge.Target{Kind: "merge-request", ID: mr.ID}
	body := fmt.Sprintf("**Merging with no person's yes**: the project's merge policy is `normal`, and this is workline's own %s: %s. Every commit is workline's own, checked by its tree, and the line passed on its head (%s), every agent's review whole. To hold it, mark it a draft or turn auto-merge off. Reverted, workline's %s merge requests wait for a person's yes again.",
		kind, kindSays[kind], short(tip), kind)
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
	e.ready = true
	return e, nil
}

// The ways a revert names what it reverts: git's own message, GitLab's
// revert of a merge request, GitHub's revert pull request; and how a
// merge on the base names its merge request: GitHub's squash or merge
// subject, GitLab's merge commit.
var (
	revertsCommit  = regexp.MustCompile(`(?m)^This reverts commit ([0-9a-f]{7,40})\b`)
	revertsRequest = regexp.MustCompile(`(?m)This reverts merge request !(\d+)\b`)
	revertsPull    = regexp.MustCompile(`(?m)^Reverts (?:[\w.-]+/[\w.-]+)?#(\d+)\b`)
	revertSubject  = regexp.MustCompile(`^Revert "(.*)"`)
	squashSubject  = regexp.MustCompile(`\((?:#|!)(\d+)\)\s*$`)
	mergeSubject   = regexp.MustCompile(`^Merge pull request #(\d+)\b`)
	mergeRequestOf = regexp.MustCompile(`(?m)^See merge request \S*!(\d+)\b`)
)

// revertOf is the merge request a commit's message says it reverts, 0 when
// it is no revert or names none.
func revertOf(repo, msg string) int {
	subject, _, _ := strings.Cut(msg, "\n")
	if !revertSubject.MatchString(subject) && !strings.Contains(msg, "This reverts ") && !revertsPull.MatchString(msg) {
		return 0
	}
	for _, re := range []*regexp.Regexp{revertsRequest, revertsPull} {
		if m := re.FindStringSubmatch(msg); m != nil {
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	if m := revertsCommit.FindStringSubmatch(msg); m != nil {
		if was, err := git(repo, nil, "log", "-1", "--format=%B", m[1]); err == nil {
			if n := requestOf(was); n > 0 {
				return n
			}
		}
	}
	if m := revertSubject.FindStringSubmatch(subject); m != nil {
		return requestOf(m[1])
	}
	return 0
}

// requestOf is the merge request a merge's message names, 0 none.
func requestOf(msg string) int {
	subject, _, _ := strings.Cut(msg, "\n")
	for _, m := range [][]string{squashSubject.FindStringSubmatch(strings.TrimSpace(subject)), mergeSubject.FindStringSubmatch(subject), mergeRequestOf.FindStringSubmatch(msg)} {
		if m != nil {
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	return 0
}

// remoteTip is base's tip on origin, fetched with its history.
func remoteTip(repo, base string) (string, error) {
	if _, err := git(repo, nil, "fetch", "-q", "origin", "refs/heads/"+base); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return git(repo, nil, "rev-parse", "FETCH_HEAD")
}

// demotedTitle is the issue that keeps a kind waiting for a person's yes.
func demotedTitle(kind string) string {
	return "workline: its " + kind + " merge requests wait for a person's yes again"
}

// ownNote finds, among a merge request's comments, the engine's one whose
// marker starts with key — written by a bot, or by no one the forge names
// (the simulated forge's own) —: the marker's value after key, and whether
// there is one. A person's comment ending with a marker is theirs.
func ownNote(notes []forge.Note, key string) (string, bool) {
	for _, n := range notes {
		last, engine := backlog.EngineMarker(n.Body)
		if !engine || !n.Bot && n.Author != "" {
			continue
		}
		if v, ok := strings.CutPrefix(last, "<!-- workline:"+key); ok {
			return strings.TrimSpace(strings.TrimSuffix(v, "-->")), true
		}
	}
	return "", false
}

// demotions reads which kinds of workline's routine merge requests wait for
// a person's yes again. A revert on base of one merged with no person's yes
// — the base's last revertWindow commits read, each revert once — demotes
// its kind: an issue opened, kept open until a person closes it, and the
// merge request told. It returns each kind demoted and the issue that
// keeps it.
func demotions(f forge.Forge, o FollowOptions, baseTip string, res *Result) (map[string]string, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, nil
	}
	out, err := git(o.Repo, nil, "log", "--first-parent", "-n", strconv.Itoa(revertWindow), "--format=%H%x1f%B%x1e", baseTip)
	if err != nil {
		return nil, err
	}
	demoted := map[string]string{}
	for _, rec := range strings.Split(out, "\x1e") {
		sha, msg, ok := strings.Cut(strings.TrimSpace(rec), "\x1f")
		if !ok {
			continue
		}
		n := revertOf(o.Repo, msg)
		if n == 0 {
			continue
		}
		t := forge.Target{Kind: "merge-request", ID: n}
		notes, err := b.Notes(t)
		switch {
		case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
			return nil, err
		case err != nil:
			continue // no merge request of that number here
		}
		kind, alone := ownNote(notes, "merge-alone=")
		if !alone || !slices.Contains(RoutineKinds, kind) {
			continue
		}
		if _, read := ownNote(notes, "merge-demoted="+sha); read {
			continue // read already: its issue opened, closed since or not
		}
		where, subject := forge.MergeRef(f, n), firstSentence(msg)
		body := fmt.Sprintf("**%s merged with no person's yes, and was reverted** by %s (%q). Its kind, %s — %s —, now waits for a person's yes, the label `workline:accepted`, on each merge request, as at `cautious`.\n\n**To let workline merge its %s merge requests alone again, close this issue.** Leave it open while you are not sure.",
			where, short(sha), subject, kind, kindSays[kind], kind)
		id, err := f.OpenIssue(demotedTitle(kind), body, forge.Marker("merge-demoted="+sha))
		if err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return nil, err
			}
			// Held in this run all the same; read again at the next.
			demoted[kind] = where + " reverted"
			res.Findings = append(res.Findings, verdict.Finding{Rule: "demotion-unrecorded", Level: "warn", Where: where,
				Message: fmt.Sprintf("merged with no person's yes, reverted by %s: workline's %s merge requests wait for a person's yes in this run, but the issue that keeps it could not be opened: %s", short(sha), kind, forge.Said(err))})
			continue
		}
		issue := fmt.Sprintf("#%d", id)
		if err := f.Comment(t, fmt.Sprintf("**Reverted** by %s: workline's %s merge requests wait for a person's yes again, until a person closes %s.", short(sha), kind, issue),
			forge.Marker("merge-demoted="+sha)); err != nil {
			return nil, err
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-demoted", Level: "warn", Where: where,
			Message: fmt.Sprintf("merged with no person's yes, reverted by %s (%q): workline's %s merge requests wait for a person's yes again, until a person closes %s", short(sha), subject, kind, issue)})
	}
	open, err := b.Issues()
	if err != nil {
		return nil, err
	}
	for _, is := range open {
		for _, kind := range RoutineKinds {
			if is.Title == demotedTitle(kind) {
				demoted[kind] = fmt.Sprintf("#%d", is.ID)
			}
		}
	}
	return demoted, nil
}

// alone sums up, for the run's summary, what merges with no person's yes
// at `normal`, and what waits again since a revert.
func alone(demoted map[string]string) string {
	var free, held []string
	for _, kind := range RoutineKinds {
		if issue, ok := demoted[kind]; ok {
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
	return s
}
