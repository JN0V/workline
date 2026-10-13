package engine

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// A revert demotes (ADR-0042, `normal`; ADR-0026): a revert on the base of
// a merge request workline merged with no person's yes sends its kind back
// to a person's yes; one it cannot tie to a merge request, every kind. What
// holds a demotion is an open issue workline's App opened; it is lifted
// when a person who may push and merge closes it. Where the reading of
// reverts stopped is the App's mark on the base commit it read (a status
// only the App's is read); none found, every kind waits until a person says.

// revertWindow is how many commits of the base's history are looked at for
// the App's mark, and read for reverts when there is none; mergeWindow, how
// many for the merge a revert quoting a title undoes.
const (
	revertWindow = 100
	mergeWindow  = 1000
)

// demotion is a kind sent back to a person's yes: the revert that did it —
// for one lost, the base commit from which nothing is known —, the merge
// request it reverted (0: none found), the issue that says it (0: none
// opened as workline's own yet).
type demotion struct {
	kind, revert string
	request      int
	issue        int
	lost         bool
}

// markText is the App's mark on a base commit: the reverts read up to it,
// and the issues holding a demotion then.
func markText(issues []int) string {
	s := "reverts read; held:"
	for _, id := range issues {
		s += fmt.Sprintf(" #%d", id)
	}
	return s
}

var markIssue = regexp.MustCompile(`#(\d+)`)

// demotedKind reads a demotion issue's title: the kind it holds, kindAll
// for every kind; ok false when it is no demotion's.
func demotedKind(title string) (string, bool) {
	rest, ok := strings.CutPrefix(title, "workline: its ")
	if !ok {
		return "", false
	}
	if strings.HasPrefix(rest, "merge requests wait for a person's yes again") {
		return kindAll, true
	}
	kind, after, _ := strings.Cut(rest, " ")
	if slices.Contains(RoutineKinds, kind) && strings.HasPrefix(after, "merge requests wait for a person's yes again") {
		return kind, true
	}
	return "", false
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

// logged is a commit of the base's history and its message.
type logged struct{ sha, msg string }

// commits are tip's history, every parent, newest first, with their
// messages, a commit before its parents whatever their dates: the n last,
// or those past from when it is given.
func commits(repo, tip, from string, n int) ([]logged, error) {
	args := []string{"log", "--topo-order", "--format=%H%x1f%B%x1e"}
	if from != "" {
		args = append(args, from+".."+tip)
	} else {
		args = append(args, "-n", strconv.Itoa(n), tip)
	}
	out, err := git(repo, nil, args...)
	if err != nil {
		return nil, err
	}
	var all []logged
	for _, rec := range strings.Split(out, "\x1e") {
		if sha, msg, ok := strings.Cut(strings.TrimSpace(rec), "\x1f"); ok {
			all = append(all, logged{sha, msg})
		}
	}
	return all, nil
}

// revertOf is the merge request commit r reverts, 0 when it names none
// found, and whether r is a revert at all. Named in its message — git's
// words naming the commit reverted, GitLab's or GitHub's naming the merge
// request —, else found by the title it quotes: GitHub's own revert says
// only `Revert "<title>"`. Among the merges with that title in older, the
// commits older than r, newest first, the one r undoes exactly — the same
// change, inverted — is taken, else the newest.
func revertOf(repo string, r logged, older func() []logged) (int, bool) {
	subject, _, _ := strings.Cut(r.msg, "\n")
	if !revertSubject.MatchString(subject) && !strings.Contains(r.msg, "This reverts ") && !revertsPull.MatchString(r.msg) {
		return 0, false
	}
	for _, re := range []*regexp.Regexp{revertsRequest, revertsPull} {
		if m := re.FindStringSubmatch(r.msg); m != nil {
			n, _ := strconv.Atoi(m[1])
			return n, true
		}
	}
	if m := revertsCommit.FindStringSubmatch(r.msg); m != nil {
		if was, err := git(repo, nil, "log", "-1", "--format=%B", m[1]); err == nil {
			if n, _ := requestOf(was); n > 0 {
				return n, true
			}
		}
	}
	m := revertSubject.FindStringSubmatch(subject)
	if m == nil {
		return 0, true
	}
	if n, _ := requestOf(m[1]); n > 0 {
		return n, true // `Revert "<title> (#N)"`
	}
	undone, first := "", 0
	for _, c := range older() {
		n, title := requestOf(c.msg)
		if n == 0 || title != m[1] {
			continue
		}
		if undone == "" {
			undone = patchID(repo, r.sha, r.sha+"^")
		}
		if undone != "" && patchID(repo, c.sha+"^1", c.sha) == undone {
			return n, true
		}
		if first == 0 {
			first = n
		}
	}
	return first, true
}

// patchID is the id git gives the change from a to b, whatever its
// context lines; "" when it cannot be told.
func patchID(repo, a, b string) string {
	diff, err := git(repo, nil, "diff", a, b)
	if err != nil || diff == "" {
		return ""
	}
	out, err := gitIn(repo, diff+"\n", "patch-id", "--stable")
	if err != nil {
		return ""
	}
	id, _, _ := strings.Cut(out, " ")
	return id
}

// requestOf is the merge request a merge's message names, 0 none, and its
// title: GitHub's squash, `<title> (#N)`; GitHub's merge, `Merge pull
// request #N …` and the title below; GitLab's merge, the title below and
// `See merge request …!N`.
func requestOf(msg string) (int, string) {
	subject, body, _ := strings.Cut(msg, "\n")
	subject = strings.TrimSpace(subject)
	below := ""
	for _, l := range strings.Split(body, "\n") {
		if below = strings.TrimSpace(l); below != "" {
			break
		}
	}
	if m := squashSubject.FindStringSubmatchIndex(subject); m != nil {
		n, _ := strconv.Atoi(subject[m[2]:m[3]])
		return n, strings.TrimSpace(subject[:m[0]])
	}
	for _, m := range [][]string{mergeSubject.FindStringSubmatch(subject), mergeRequestOf.FindStringSubmatch(msg)} {
		if m != nil {
			n, _ := strconv.Atoi(m[1])
			return n, below
		}
	}
	return 0, ""
}

// remoteTip is base's tip on origin, fetched with its history.
func remoteTip(repo, base string) (string, error) {
	if _, err := git(repo, nil, "fetch", "-q", "origin", "refs/heads/"+base); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return git(repo, nil, "rev-parse", "FETCH_HEAD")
}

// demotedTitle is the issue that says a revert sent a kind — every kind,
// kindAll — back to a person's yes; its revert named, so nobody opens it
// before.
func demotedTitle(kind, revert string) string {
	if kind == kindAll {
		return "workline: its merge requests wait for a person's yes again (revert " + short(revert) + ")"
	}
	return "workline: its " + kind + " merge requests wait for a person's yes again (revert " + short(revert) + ")"
}

// lostTitle is the issue that says which reverts were read is not known
// from a base commit on.
func lostTitle(tip string) string {
	return "workline: its merge requests wait for a person's yes again (no record of reverts read before " + short(tip) + ")"
}

// demotions reads the reverts on the base past the App's mark, opens an
// issue for each demotion, and returns each kind held (kindAll: every
// kind) with the issue holding it. What holds: each open demotion issue the
// App opened, and one closed by anyone but a person who may push and
// merge. Then it marks the base's tip as read, with the issues held.
func demotions(f forge.Forge, o FollowOptions, baseTip, app string, res *Result) (map[string]string, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return map[string]string{kindAll: "no backlog on this forge"}, nil
	}
	chain, err := git(o.Repo, nil, "rev-list", "--first-parent", "-n", strconv.Itoa(revertWindow), baseTip)
	if err != nil {
		return nil, err
	}
	from, mark := "", ""
	for _, c := range strings.Fields(chain) {
		text, ok, err := forge.Marked(f, c)
		switch {
		case errors.Is(err, forge.ErrNoMark):
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unmarked", Level: "warn",
				Message: "at `normal`, this forge keeps no mark only workline's App writes: which reverts were read cannot be known, so nothing merges with no person's yes"})
			return map[string]string{kindAll: "no mark on this forge"}, nil
		case err != nil && errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
			return nil, err
		case err != nil:
			c = "" // its mark could not be read: as none
		}
		if ok && c != "" {
			from, mark = c, text
			break
		}
	}
	var fresh []demotion
	if from == "" {
		fresh = append(fresh, demotion{kind: kindAll, revert: baseTip, lost: true})
	}
	// Every parent: a revert merged with a merge commit is on its second.
	history, err := commits(o.Repo, baseTip, from, revertWindow)
	if err != nil {
		return nil, err
	}
	var deeper []logged // what a revert may name, read once if needed
	var deepErr error
	for i, r := range history {
		older := func() []logged {
			if deeper == nil && deepErr == nil {
				deeper, deepErr = commits(o.Repo, baseTip, "", mergeWindow)
			}
			at := slices.IndexFunc(deeper, func(c logged) bool { return c.sha == r.sha })
			if at < 0 {
				return history[i+1:]
			}
			return deeper[at+1:]
		}
		n, revert := revertOf(o.Repo, r, older)
		if deepErr != nil {
			return nil, fmt.Errorf("the base's history, for what a revert undoes: %w", deepErr)
		}
		if !revert {
			continue
		}
		kind := kindAll // a revert of nothing found: every kind, failing closed
		if n > 0 {
			notes, err := b.Notes(forge.Target{Kind: "merge-request", ID: n})
			switch {
			case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
				return nil, err
			case err != nil:
				notes = nil // no merge request of that number here
			}
			alone, ok := ownNote(notes, "merge-alone=", app)
			switch {
			case ok && slices.Contains(RoutineKinds, alone):
				kind = alone
			case err == nil:
				continue // a merge request merged with a person's yes, or never by workline
			}
		}
		if kind == kindAll {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "revert-unread", Level: "warn", Where: short(r.sha),
				Message: fmt.Sprintf("`%s` reverts something workline cannot tie to a merge request: every kind of its own merge requests waits for a person's yes, until a person says", firstSentence(r.msg))})
		}
		fresh = append(fresh, demotion{kind: kind, revert: r.sha, request: n})
	}
	demoted := map[string]string{}
	var issues []int
	unrecorded := false
	for _, d := range fresh {
		if err := demote(f, &d, app, res); err != nil {
			return nil, err
		}
		if d.issue == 0 {
			unrecorded = true
			if _, ok := demoted[d.kind]; !ok {
				demoted[d.kind] = "its issue not opened yet"
			}
			continue
		}
		issues = append(issues, d.issue)
	}
	// What the mark held, and every open demotion issue the App opened:
	// whatever any record says, those hold.
	for _, m := range markIssue.FindAllStringSubmatch(mark, -1) {
		id, _ := strconv.Atoi(m[1])
		issues = append(issues, id)
	}
	open, err := b.Issues()
	if err != nil {
		return nil, err
	}
	for _, is := range open {
		if _, ok := demotedKind(is.Title); ok && is.Author == app && strings.Contains(is.Body, "<!-- workline:merge-demoted=") {
			issues = append(issues, is.ID)
		}
	}
	slices.Sort(issues)
	issues = slices.Compact(issues)
	var kept []int
	for _, id := range issues {
		kind, hold, err := holds(f, id, app, res)
		if err != nil {
			return nil, err
		}
		if !hold {
			continue
		}
		kept = append(kept, id)
		if _, ok := demoted[kind]; !ok {
			demoted[kind] = fmt.Sprintf("#%d", id)
		}
	}
	// A demotion not yet in an issue of the App's: the mark stays where it
	// was, and the revert is read again at the next run.
	if !unrecorded {
		if err := forge.Mark(f, baseTip, markText(kept)); err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return nil, err
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unmarked", Level: "warn",
				Message: fmt.Sprintf("the base's tip, %s, could not be marked as read (%s): the next run reads again from the last mark: %s", short(baseTip), forge.ReadStatus, forge.Said(err))})
		}
	}
	return demoted, nil
}

// holds says whether an issue holds a demotion, and of which kind: one the
// App opened, open; or closed by anyone but a person who may push and merge
// — a bot, someone with no write access, a hand the forge does not name —,
// said. Closed by such a person, it is lifted, said.
func holds(f forge.Forge, id int, app string, res *Result) (string, bool, error) {
	where := fmt.Sprintf("#%d", id)
	is, err := f.Issue(id)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return "", false, err
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "restore-unread", Level: "warn", Where: where,
			Message: "a demotion's issue could not be read: every kind of workline's own merge requests waits for a person's yes, " + forge.Said(err)})
		return kindAll, true, nil
	}
	kind, ok := demotedKind(is.Title)
	if !ok || is.Author != app {
		return "", false, nil // never one of the App's demotions
	}
	which := "its " + kind + " merge requests"
	if kind == kindAll {
		which = "every kind of its own merge requests"
	}
	if !is.Closed {
		return kind, true, nil
	}
	who, ok, err := forge.Closing(f, id)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return "", false, err
		}
		ok = false
	}
	if ok && who.Writer && !who.Bot && who.Author != "" && who.Author != app {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-restored", Where: where,
			Message: fmt.Sprintf("closed by @%s: %s merge with no person's yes again", who.Author, which)})
		return kind, false, nil
	}
	by := "a hand the forge does not name"
	if ok && who.Author != "" {
		by = "@" + who.Author
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "restore-ignored", Level: "warn", Where: where,
		Message: fmt.Sprintf("closed by %s, who may not lift it — a bot, or no write access —: %s still wait for a person's yes; reopen it, and a person who may push and merge closes it", by, which)})
	return kind, true, nil
}

// demote opens a demotion's issue as workline's own — one of that title by
// anyone else is never taken for it, and the demotion stays unrecorded —,
// and tells the merge request reverted.
func demote(f forge.Forge, d *demotion, app string, res *Result) error {
	where := short(d.revert)
	if d.request > 0 {
		where = forge.MergeRef(f, d.request)
	}
	which := "its " + d.kind + " merge requests"
	if d.kind == kindAll {
		which = "every kind of its own merge requests"
	}
	title := demotedTitle(d.kind, d.revert)
	body := fmt.Sprintf("**A revert, %s, sends %s back to a person's yes**: the label `workline:accepted` on each, as at `cautious`. The revert names no merge request workline can find, so it may have undone one merged with no person's yes.", short(d.revert), which)
	switch {
	case d.lost:
		title = lostTitle(d.revert)
		body = fmt.Sprintf("**Workline cannot tell which reverts it read before %s**: no mark of its App's (the status `%s`) on the base's last %d commits — the first run at `normal`, a mark lost, or `follow` not run for long. A revert there may have undone a merge request merged with no person's yes, so every kind of its own merge requests waits for a person's yes, the label `workline:accepted`, as at `cautious`.", short(d.revert), forge.ReadStatus, revertWindow)
	case d.request > 0:
		body = fmt.Sprintf("**%s merged with no person's yes, and was reverted** by %s. %s — %s — now wait for a person's yes, the label `workline:accepted`, on each, as at `cautious`.", where, short(d.revert), strings.ToUpper(which[:1])+which[1:], kindSays[d.kind])
	}
	body += "\n\n**To let workline merge them alone again, close this issue** — a person who may push and merge; a bot's close never lifts it. Leave it open while you are not sure."
	id, err := f.OpenIssue(title, body, forge.Marker("merge-demoted="+d.revert))
	if err == nil {
		var is *forge.Issue
		if is, err = f.Issue(id); err == nil && (app == "" || is.Author != app) {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "demotion-unrecorded", Level: "warn", Where: where,
				Message: fmt.Sprintf("%s: %s wait for a person's yes, but #%d, of that title, is @%s's, not workline's: it is not taken for it, and this is read again next run", short(d.revert), which, id, is.Author)})
			return nil
		}
	}
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return err
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "demotion-unrecorded", Level: "warn", Where: where,
			Message: fmt.Sprintf("%s: %s wait for a person's yes, but the issue that says it could not be opened, read again next run: %s", short(d.revert), which, forge.Said(err))})
		return nil
	}
	d.issue = id
	if d.request > 0 {
		if err := f.Comment(forge.Target{Kind: "merge-request", ID: d.request}, fmt.Sprintf("**Reverted** by %s: workline's %s merge requests wait for a person's yes again, until a person closes #%d.", short(d.revert), d.kind, id),
			forge.Marker("merge-demoted="+d.revert)); err != nil {
			return err
		}
	}
	rule, said := "merge-demoted", fmt.Sprintf("reverted by %s: %s wait for a person's yes again, until a person who may push and merge closes #%d", short(d.revert), which, id)
	if d.lost {
		rule, said = "merge-record-lost", fmt.Sprintf("no mark of workline's App on the base's last %d commits: which reverts were read is not known, so %s wait for a person's yes, until a person who may push and merge closes #%d", revertWindow, which, id)
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: "warn", Where: where, Message: said})
	return nil
}
