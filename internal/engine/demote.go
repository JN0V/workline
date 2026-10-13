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
// to a person's yes; a revert it cannot tie to a merge request, every kind.
// Each is read once: the engine's record on the forge, the ref recordRef,
// keeps the last base commit read and each demotion with the issue that
// says it. A demotion is lifted when a person who may push and merge closes
// that issue — never a bot, nor someone else.

// recordRef is the engine's record of the merge policy on the forge.
const recordRef = "refs/workline/merge-policy"

// revertWindow is how many commits of the base's history are read for
// reverts when no record says where the last run stopped; mergeWindow, how
// many for the merge a revert quoting a title undoes.
const (
	revertWindow = 100
	mergeWindow  = 1000
)

// demotion is a kind sent back to a person's yes: the revert that did it,
// the merge request it reverted (0: none found), the issue that says it (0:
// none could be opened as workline's own yet).
type demotion struct {
	kind, revert string
	request      int
	issue        int
}

// record is what the engine keeps of the merge policy: the last base commit
// read for reverts, and the demotions in force.
type record struct {
	read     string
	demoted  []demotion
	was, sha string // the record's commit as read, "" none
}

// readRecord reads the record from origin; an empty one when there is none.
func readRecord(repo string) (record, error) {
	out, err := git(repo, nil, "ls-remote", "origin", recordRef)
	if err != nil {
		return record{}, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	sha, _, _ := strings.Cut(out, "\t")
	if sha = strings.TrimSpace(sha); sha == "" {
		return record{}, nil
	}
	if _, err := git(repo, nil, "fetch", "-q", "origin", recordRef); err != nil {
		return record{}, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	msg, err := git(repo, nil, "log", "-1", "--format=%B", sha)
	if err != nil {
		return record{}, err
	}
	r := record{was: sha}
	for _, l := range strings.Split(msg, "\n") {
		f := strings.Fields(l)
		switch {
		case len(f) == 2 && f[0] == "read":
			r.read = f[1]
		case len(f) == 5 && f[0] == "demoted":
			n, _ := strconv.Atoi(f[3])
			i, _ := strconv.Atoi(f[4])
			r.demoted = append(r.demoted, demotion{kind: f[1], revert: f[2], request: n, issue: i})
		}
	}
	return r, nil
}

// text is the record as its commit's message.
func (r record) text() string {
	var b strings.Builder
	b.WriteString("workline: the merge policy's record (ADR-0042)\n\nThe last base commit read for reverts, and each kind of workline's own\nmerge requests sent back to a person's yes: kind, revert, merge request\nreverted, issue.\n\n")
	fmt.Fprintf(&b, "read %s\n", r.read)
	for _, d := range r.demoted {
		fmt.Fprintf(&b, "demoted %s %s %d %d\n", d.kind, d.revert, d.request, d.issue)
	}
	return b.String()
}

// writeRecord pushes the record to origin when it changed, its last
// version as lease: a record moved meanwhile is no record overwritten.
func writeRecord(repo string, r record) error {
	tree, err := gitIn(repo, "", "mktree")
	if err != nil {
		return err
	}
	args := []string{"-c", "user.name=workline", "-c", "user.email=workline@noreply.invalid", "commit-tree", tree, "-m", r.text()}
	if r.was != "" {
		args = append(args, "-p", r.was)
	}
	commit, err := git(repo, nil, args...)
	if err != nil {
		return err
	}
	if _, err := git(repo, nil, "push", "-q", "--force-with-lease="+recordRef+":"+r.was, "origin", commit+":"+recordRef); err != nil {
		return fmt.Errorf("%w: the merge policy's record (%s) not written: %v", forge.ErrUnreachable, recordRef, err)
	}
	return nil
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

// demotions reads the base's reverts past the record, demotes their kinds,
// lifts those a person may lift, writes the record back, and returns each
// kind demoted (kindAll: every kind) with the issue that says it.
func demotions(f forge.Forge, o FollowOptions, baseTip, app string, res *Result) (map[string]string, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, nil
	}
	rec, err := readRecord(o.Repo)
	if err != nil {
		return nil, err
	}
	was := rec.text()
	// Past the commit the record read; the base rewritten since, past what
	// it shares with it; no record yet, the last revertWindow, said.
	from := ""
	switch {
	case rec.read != "" && isAncestor(o.Repo, rec.read, baseTip):
		from = rec.read
	case rec.read != "":
		from, _ = git(o.Repo, nil, "merge-base", rec.read, baseTip)
	}
	if from == "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-record-started",
			Message: fmt.Sprintf("no record of the merge policy to read from (%s): the base's last %d commits read for reverts, none before", recordRef, revertWindow)})
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
		rec.demoted = append(rec.demoted, demotion{kind: kind, revert: r.sha, request: n})
	}
	rec.read = baseTip
	demoted := map[string]string{}
	var kept []demotion
	for _, d := range rec.demoted {
		lifted, err := demote(f, b, &d, app, res)
		if err != nil {
			return nil, err
		}
		if lifted {
			continue
		}
		kept = append(kept, d)
		if _, ok := demoted[d.kind]; !ok {
			demoted[d.kind] = map[bool]string{true: fmt.Sprintf("#%d", d.issue), false: "its issue not opened yet"}[d.issue > 0]
		}
	}
	rec.demoted = kept
	if rec.text() != was {
		if err := writeRecord(o.Repo, rec); err != nil {
			return nil, err
		}
	}
	return demoted, nil
}

// demote keeps one demotion: its issue opened as workline's own when it
// has none — one of that title by anyone else is never taken for it —, the
// merge request reverted told; lifted, true, when a person who may push
// and merge closed the issue. Closed by anyone else, it holds, said.
func demote(f forge.Forge, b forge.Backlog, d *demotion, app string, res *Result) (bool, error) {
	where := short(d.revert)
	if d.request > 0 {
		where = forge.MergeRef(f, d.request)
	}
	which := "its " + d.kind + " merge requests"
	if d.kind == kindAll {
		which = "every kind of its own merge requests"
	}
	if d.issue == 0 {
		body := fmt.Sprintf("**A revert, %s, sends %s back to a person's yes**: the label `workline:accepted` on each, as at `cautious`.", short(d.revert), which)
		if d.request > 0 {
			body = fmt.Sprintf("**%s merged with no person's yes, and was reverted** by %s. %s — %s — now wait for a person's yes, the label `workline:accepted`, on each, as at `cautious`.", where, short(d.revert), strings.ToUpper(which[:1])+which[1:], kindSays[d.kind])
		} else {
			body += " The revert names no merge request workline can find, so it may have undone one merged with no person's yes."
		}
		body += "\n\n**To let workline merge them alone again, close this issue** — a person who may push and merge. Leave it open while you are not sure."
		id, err := f.OpenIssue(demotedTitle(d.kind, d.revert), body, forge.Marker("merge-demoted="+d.revert))
		if err == nil {
			var is *forge.Issue
			if is, err = f.Issue(id); err == nil && (app == "" || is.Author != app) {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "demotion-unrecorded", Level: "warn", Where: where,
					Message: fmt.Sprintf("reverted by %s: %s wait for a person's yes, but #%d, of that title, is @%s's, not workline's: it is not taken for it", short(d.revert), which, id, is.Author)})
				return false, nil
			}
		}
		if err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return false, err
			}
			res.Findings = append(res.Findings, verdict.Finding{Rule: "demotion-unrecorded", Level: "warn", Where: where,
				Message: fmt.Sprintf("reverted by %s: %s wait for a person's yes, but the issue that says it could not be opened: %s", short(d.revert), which, forge.Said(err))})
			return false, nil
		}
		d.issue = id
		if d.request > 0 {
			if err := f.Comment(forge.Target{Kind: "merge-request", ID: d.request}, fmt.Sprintf("**Reverted** by %s: workline's %s merge requests wait for a person's yes again, until a person closes #%d.", short(d.revert), d.kind, id),
				forge.Marker("merge-demoted="+d.revert)); err != nil {
				return false, err
			}
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-demoted", Level: "warn", Where: where,
			Message: fmt.Sprintf("reverted by %s: %s wait for a person's yes again, until a person who may push and merge closes #%d", short(d.revert), which, id)})
		return false, nil
	}
	is, err := f.Issue(d.issue)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return false, err
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "restore-unread", Level: "warn", Where: fmt.Sprintf("#%d", d.issue),
			Message: fmt.Sprintf("%s wait for a person's yes since %s; the issue that says it could not be read: %s", which, short(d.revert), forge.Said(err))})
		return false, nil
	}
	if !is.Closed {
		return false, nil
	}
	who, ok, err := forge.Closing(f, d.issue)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return false, err
		}
		ok = false
	}
	if ok && who.Writer && !who.Bot && who.Author != "" && who.Author != app {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-restored", Where: fmt.Sprintf("#%d", d.issue),
			Message: fmt.Sprintf("closed by @%s: %s merge with no person's yes again, as before %s", who.Author, which, short(d.revert))})
		return true, nil
	}
	by := "a hand the forge does not name"
	if ok && who.Author != "" {
		by = "@" + who.Author
	}
	res.Findings = append(res.Findings, verdict.Finding{Rule: "restore-ignored", Level: "warn", Where: fmt.Sprintf("#%d", d.issue),
		Message: fmt.Sprintf("closed by %s, who may not lift it — a bot, or no write access —: %s still wait for a person's yes; reopen it, and a person who may push and merge closes it", by, which)})
	return false, nil
}
