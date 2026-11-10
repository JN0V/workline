package engine

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/release"
	"github.com/JN0V/workline/internal/verdict"
)

// A milestone's release candidate (ADR-0060, §4): at `merge: {autonomy:
// enterprising}`, a feature merges by the policy, in a milestone named by
// a version; once all its issues are closed, `follow` cuts
// `release/vX.Y` from the base and asks the release tool for
// `vX.Y.0-rc.1` there, by a commit `Release-As:` (release-please's word),
// and opens the candidate's issue in the milestone. The release tool tags
// the candidate and publishes it as a pre-release; workline tags nothing
// (ADR-0017).
//
// A person tries it, then says yes with the label `workline:accepted` on
// that issue: `follow` asks the release tool for `vX.Y.0` on the branch,
// once the branch holds nothing past the candidate tried but the versions
// the tool bumps, and closes the issue and the milestone. Closed by a
// person instead, the candidate is refused: features wait for a person's
// yes on each pull request again, in an issue as a revert's demotion,
// until a person closes it. An issue of the milestone merged into the
// base after the cut is copied onto the branch, and the next candidate,
// `-rc.N+1`, asked for.

// CandidateLabel marks the issue a release candidate is tried on.
const CandidateLabel = "workline:candidate"

// candidateSays is the label's description in the forge's list.
const candidateSays = "A release candidate to try: workline:accepted ships its version; closing it refuses it"

// kindFeature is the demotion a refused candidate makes: the developer's
// features wait for a person's yes on each pull request again.
const kindFeature = "feature"

// minor is a milestone's version, its major and minor: the candidate's
// branch and versions are told from it.
type minor struct{ major, minor int }

func (v minor) title() string   { return fmt.Sprintf("v%d.%d", v.major, v.minor) }
func (v minor) branch() string  { return fmt.Sprintf("release/v%d.%d", v.major, v.minor) }
func (v minor) version() string { return fmt.Sprintf("%d.%d.0", v.major, v.minor) }
func (v minor) rc(n int) string { return fmt.Sprintf("%s-rc.%d", v.version(), n) }
func (v minor) less(w minor) bool {
	return v.major < w.major || v.major == w.major && v.minor < w.minor
}

var (
	versionTitle = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.0)?$`)
	versionTag   = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-rc\.(\d+))?$`)
	branchName   = regexp.MustCompile(`^release/v(\d+)\.(\d+)$`)
	releaseAsRC  = regexp.MustCompile(`(?m)^Release-As: \d+\.\d+\.\d+-rc\.(\d+)\s*$`)
)

// minorOf reads a version's major and minor from a regexp's first two
// groups.
func minorOf(m []string) minor {
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	return minor{a, b}
}

// milestoneVersion reads a milestone's title as a version, `v1.4`, `1.4`
// or `v1.4.0`; ok false for a title that names none.
func milestoneVersion(title string) (minor, bool) {
	m := versionTitle.FindStringSubmatch(strings.TrimSpace(title))
	if m == nil {
		return minor{}, false
	}
	return minorOf(m), true
}

// remote is what origin holds of the releases: each version tag and each
// candidate's branch, with its commit.
type remote struct {
	tags     map[string]string
	branches map[string]string
}

// remoteReleases lists origin's version tags and candidates' branches.
func remoteReleases(repo string) (remote, error) {
	r := remote{tags: map[string]string{}, branches: map[string]string{}}
	out, err := git(repo, nil, "ls-remote", "origin", "refs/tags/v*", "refs/heads/release/v*")
	if err != nil {
		return r, fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	for _, l := range strings.Split(out, "\n") {
		sha, ref, ok := strings.Cut(l, "\t")
		if !ok {
			continue
		}
		if tag, ok := strings.CutPrefix(ref, "refs/tags/"); ok {
			if name, peeled := strings.CutSuffix(tag, "^{}"); peeled {
				r.tags[name] = sha // the commit an annotated tag names
			} else if _, seen := r.tags[tag]; !seen {
				r.tags[tag] = sha
			}
			continue
		}
		r.branches[strings.TrimPrefix(ref, "refs/heads/")] = sha
	}
	return r, nil
}

// lastRC is the highest `-rc.N` tagged for a version, and its tag; 0
// when none is.
func (r remote) lastRC(v minor) (int, string) {
	n, tag := 0, ""
	for t := range r.tags {
		m := versionTag.FindStringSubmatch(t)
		if m == nil || m[4] == "" || minorOf(m) != v || m[3] != "0" {
			continue
		}
		if k, _ := strconv.Atoi(m[4]); k > n {
			n, tag = k, t
		}
	}
	return n, tag
}

// released is the highest version origin's tags name, prereleases left
// out: a milestone at or below it is out already.
func (r remote) released() minor {
	var top minor
	for t := range r.tags {
		if m := versionTag.FindStringSubmatch(t); m != nil && m[4] == "" && top.less(minorOf(m)) {
			top = minorOf(m)
		}
	}
	return top
}

// highest is the highest version origin's tags, its candidates' branches
// and the open milestones name.
func (r remote) highest(ms []forge.Milestone) minor {
	var top minor
	for t := range r.tags {
		if m := versionTag.FindStringSubmatch(t); m != nil && top.less(minorOf(m)) {
			top = minorOf(m)
		}
	}
	for b := range r.branches {
		if m := branchName.FindStringSubmatch(b); m != nil && top.less(minorOf(m)) {
			top = minorOf(m)
		}
	}
	for _, m := range ms {
		if v, ok := milestoneVersion(m.Title); ok && top.less(v) {
			top = v
		}
	}
	return top
}

// NextMilestone is the milestone a feature goes into (ADR-0060, for the
// maintainer, point 2): the open milestone of the lowest version whose
// candidate is not cut yet, past the last version released; none, the
// next minor version past every one
// origin and the milestones name, opened by SetMilestone. Its title, and
// whether it is opened.
func NextMilestone(b forge.Backlog, repo string) (string, bool, error) {
	ms, err := b.Milestones()
	if err != nil {
		return "", false, err
	}
	r, err := remoteReleases(repo)
	if err != nil {
		return "", false, err
	}
	best, title, released := minor{}, "", r.released()
	for _, m := range ms {
		v, ok := milestoneVersion(m.Title)
		if _, cut := r.branches[v.branch()]; !ok || cut || !released.less(v) {
			continue // cut already, or out: an old milestone left open
		}
		if title == "" || v.less(best) {
			best, title = v, m.Title
		}
	}
	if title != "" {
		return title, false, nil
	}
	top := r.highest(ms)
	return minor{top.major, top.minor + 1}.title(), true, nil
}

// releaseAs commits on parent, its tree unchanged, the word that asks
// the release tool for version: release-please's `Release-As:` footer.
func releaseAs(repo, parent, version string) (string, error) {
	msg := fmt.Sprintf("chore: release %s\n\nRelease-As: %s\n", version, version)
	return git(repo, strings.NewReader(msg), "commit-tree", parent+"^{tree}", "-p", parent)
}

// push sets origin's branch to commit, a fast-forward.
func push(repo, commit, branch string) error {
	if _, err := git(repo, nil, "push", "-q", "origin", commit+":refs/heads/"+branch); err != nil {
		return fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return nil
}

// candidateMark is the marker of a candidate's issue: the version it was
// opened for.
func candidateMark(version string) string { return forge.Marker("candidate=v" + version) }

var candidateOf = regexp.MustCompile(`<!-- workline:candidate=v\d+\.\d+\.\d+-rc\.(\d+) -->`)

// rcOf is the `-rc.N` a candidate's issue was opened for, 0 when its body
// names none.
func rcOf(is forge.Issue) int {
	m := candidateOf.FindStringSubmatch(is.Body)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// candidateRun is what one run of the candidates reads once.
type candidateRun struct {
	f        forge.Forge
	b        forge.Backlog
	mi       forge.Milestoned
	o        FollowOptions
	app      string
	tip      string // the base's tip on origin
	r        remote
	res      *Result
	accepted string // the label of a person's yes
}

func (c *candidateRun) say(rule, level, where, msg string) {
	c.res.Findings = append(c.res.Findings, verdict.Finding{Rule: rule, Level: level, Where: where, Message: msg})
}

// candidates tends the milestones' release candidates at `enterprising`,
// before the merge policy weighs anything: a refusal's demotion is then
// read by it. It sums up what it did.
func candidates(f forge.Forge, o FollowOptions, local bool, res *Result) (string, error) {
	b, ok := f.(forge.Backlog)
	mi, ok2 := f.(forge.Milestoned)
	if !ok || !ok2 || local {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "candidate-unsupported", Level: "warn",
			Message: "at `enterprising`, this forge does not list a milestone's issues: no release candidate is cut, features merge alone with none to try"})
		return "candidates: none, the forge lists no milestone's issues", nil
	}
	app, _, err := forge.App(f)
	if err != nil && errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
		return "", err
	}
	if app == "" {
		return "candidates: none, the token is no App's", nil
	}
	c := &candidateRun{f: f, b: b, mi: mi, o: o, app: app, res: res, accepted: backlog.AcceptedLabel(f)}
	if c.tip, err = remoteTip(o.Repo, o.Base); err != nil {
		return "", err
	}
	if c.r, err = remoteReleases(o.Repo); err != nil {
		return "", err
	}
	ms, err := b.Milestones()
	if err != nil {
		return "", err
	}
	var said []string
	for _, m := range ms {
		v, ok := milestoneVersion(m.Title)
		if !ok {
			continue // a person's milestone, named by no version: no candidate
		}
		s, err := c.milestone(m.Title, v)
		if err != nil {
			return "", err
		}
		if s != "" {
			said = append(said, s)
		}
	}
	if len(said) == 0 {
		return "candidates: none to cut or ship", nil
	}
	return "candidates: " + strings.Join(said, "; "), nil
}

// milestone tends one milestone's candidate, and says what it did, "" for
// nothing.
func (c *candidateRun) milestone(title string, v minor) (string, error) {
	issues, err := c.mi.MilestoneIssues(title)
	if err != nil {
		return "", err
	}
	var cands, others []forge.Issue
	for _, is := range issues {
		if slices.Contains(is.Labels, CandidateLabel) {
			cands = append(cands, is)
		} else {
			others = append(others, is)
		}
	}
	branch := v.branch()
	if _, cut := c.r.branches[branch]; !cut {
		if len(others) == 0 || slices.ContainsFunc(others, func(is forge.Issue) bool { return !is.Closed }) {
			return "", nil
		}
		if last := c.r.released(); !last.less(v) {
			// An old milestone left open: its version, or a later one, is
			// out already; a candidate of it would ship main as it is now.
			c.say("candidate-version-past", "warn", title, fmt.Sprintf("its issues are closed, but v%d.%d.0 is released already: no candidate cut; close the milestone", last.major, last.minor))
			return "", nil
		}
		head, err := releaseAs(c.o.Repo, c.tip, v.rc(1))
		if err != nil {
			return "", err
		}
		if err := push(c.o.Repo, head, branch); err != nil {
			return "", err
		}
		id, err := c.open(title, v, 1)
		if err != nil {
			return "", err
		}
		c.say("candidate-cut", "", title, fmt.Sprintf("its %d issues closed: %s cut from %s (%s), the release tool asked for v%s; tried on #%d", len(others), branch, c.o.Base, short(c.tip), v.rc(1), id))
		return fmt.Sprintf("%s cut, v%s asked for", branch, v.rc(1)), nil
	}
	btip, err := fetchBranch(c.o.Repo, branch)
	if err != nil {
		return "", err
	}
	copied, btip, err := c.copy(v, others, btip)
	if err != nil {
		return "", err
	}
	asked := askedRC(c.o.Repo, c.tip, btip)
	sort.Slice(cands, func(i, j int) bool { return cands[i].ID < cands[j].ID })
	var open, last *forge.Issue
	for i := range cands {
		last = &cands[i]
		if !cands[i].Closed {
			open = &cands[i]
		}
	}
	switch {
	case open != nil:
		if len(copied) > 0 {
			t := forge.Target{Kind: "issue", ID: open.ID}
			body := fmt.Sprintf("**v%s asked for**: %s copied onto `%s`. Try it once the release tool has tagged it; a yes given before it covers what it replaces only.", v.rc(asked), strings.Join(copied, ", "), branch)
			if err := c.f.Comment(t, body, forge.Marker(fmt.Sprintf("candidate-rc=%d", asked))); err != nil {
				return "", err
			}
		}
		return c.yes(title, v, *open, btip)
	case last != nil && rcOf(*last) >= asked:
		return c.refused(v, *last)
	}
	// None open for the last candidate asked for: refused, then fixed; or
	// its issue lost.
	id, err := c.open(title, v, asked)
	if err != nil {
		return "", err
	}
	c.say("candidate-cut", "", title, fmt.Sprintf("v%s asked for on %s; tried on #%d", v.rc(asked), branch, id))
	return fmt.Sprintf("v%s on %s, tried on #%d", v.rc(asked), branch, id), nil
}

// fetchBranch fetches origin's branch and returns its tip.
func fetchBranch(repo, branch string) (string, error) {
	if _, err := git(repo, nil, "fetch", "-q", "--tags", "origin", "refs/heads/"+branch); err != nil {
		return "", fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return git(repo, nil, "rev-parse", "FETCH_HEAD")
}

// askedRC is the last `-rc.N` the branch asks the release tool for past
// the base, 1 when it names none.
func askedRC(repo, baseTip, btip string) int {
	out, _ := git(repo, nil, "log", "--format=%B", baseTip+".."+btip)
	n := 1
	for _, m := range releaseAsRC.FindAllStringSubmatch(out, -1) {
		if k, _ := strconv.Atoi(m[1]); k > n {
			n = k
		}
	}
	return n
}

// open opens the candidate's issue for `-rc.n`, in its milestone.
func (c *candidateRun) open(title string, v minor, n int) (int, error) {
	if err := c.b.EnsureLabel(CandidateLabel, "fbca04", candidateSays); err != nil {
		return 0, err
	}
	if err := c.b.EnsureLabel(c.accepted, "0e8a16", backlog.AcceptedSays); err != nil {
		return 0, err
	}
	branch := v.branch()
	body := fmt.Sprintf("**A release candidate of milestone %s**: its issues are merged. workline cut `%s` from `%s` and asked the release tool for `v%s` there; the tool tags it and publishes it as a pre-release.\n\n"+
		"- **Try it**: the pre-release `v%s`, once tagged.\n"+
		"- **Ship it**: put the label `%s` on this issue. workline asks the release tool for `v%s` from `%s`, the candidate as you tried it.\n"+
		"- **Refuse it**: close this issue. Features then wait for a person's yes on each pull request again, until you close the issue that says it. Put what is wrong in issues of milestone %s: once merged, workline copies them onto `%s` and asks for the next candidate.",
		title, branch, c.o.Base, v.rc(n), v.rc(n), c.accepted, v.version(), branch, title, branch)
	id, err := c.f.OpenIssue(fmt.Sprintf("Release candidate v%s: try it, then say yes", v.rc(n)), body, candidateMark(v.rc(n)))
	if err != nil {
		return 0, err
	}
	t := forge.Target{Kind: "issue", ID: id}
	if err := c.f.Label(t, []string{CandidateLabel}, nil); err != nil {
		return 0, err
	}
	return id, c.b.SetMilestone(id, title)
}

// copy copies onto the branch each issue of the milestone merged into the
// base since the cut, as `git cherry-pick -x` does, in the order merged,
// then asks the release tool for the next candidate. It says the merge
// requests copied, and the branch's tip.
func (c *candidateRun) copy(v minor, issues []forge.Issue, btip string) ([]string, string, error) {
	merged, ok := c.f.(forge.Merged)
	if !ok {
		return nil, btip, nil
	}
	from, err := git(c.o.Repo, nil, "merge-base", c.tip, btip)
	if err != nil {
		return nil, btip, err
	}
	cut, err := git(c.o.Repo, nil, "rev-list", "--reverse", "--first-parent", from+".."+btip)
	if err != nil || cut == "" {
		return nil, btip, err
	}
	first, _, _ := strings.Cut(cut, "\n")
	at, err := git(c.o.Repo, nil, "log", "-1", "--format=%cI", first)
	if err != nil {
		return nil, btip, err
	}
	closers := map[int]int{} // merge request -> the issue it closed
	for _, is := range issues {
		if !is.Closed {
			continue
		}
		cs, err := c.b.Closers(is.ID)
		if err != nil {
			return nil, btip, err
		}
		for _, x := range cs {
			if n, err := strconv.Atoi(strings.TrimLeft(x.Ref, "#!")); err == nil && x.Kind == "pull-request" {
				closers[n] = is.ID
			}
		}
	}
	if len(closers) == 0 {
		return nil, btip, nil
	}
	since, err := parseTime(at)
	if err != nil {
		return nil, btip, err
	}
	all, err := merged.MergedSince(since)
	if err != nil {
		return nil, btip, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].MergedAt < all[j].MergedAt })
	logged, _ := git(c.o.Repo, nil, "log", "--format=%B", from+".."+btip)
	head, branch := btip, v.branch()
	var copied []string
	for _, mr := range all {
		issue, ok := closers[mr.ID]
		if !ok || mr.Base != c.o.Base || mr.MergedAs == "" || isAncestor(c.o.Repo, mr.MergedAs, btip) ||
			strings.Contains(logged, "(cherry picked from commit "+mr.MergedAs) {
			continue
		}
		next, err := cherryPick(c.o.Repo, head, mr.MergedAs)
		if err != nil {
			c.say("candidate-copy-failed", "warn", forge.MergeRef(c.f, mr.ID), fmt.Sprintf("it closed #%d, of milestone %s, but could not be copied onto %s: %v; a person copies it there", issue, v.title(), branch, err))
			break
		}
		head = next
		copied = append(copied, fmt.Sprintf("%s (#%d)", forge.MergeRef(c.f, mr.ID), issue))
	}
	if len(copied) == 0 {
		return nil, btip, nil
	}
	n, _ := c.r.lastRC(v)
	n = max(n, askedRC(c.o.Repo, c.tip, btip)) + 1
	if head, err = releaseAs(c.o.Repo, head, v.rc(n)); err != nil {
		return nil, btip, err
	}
	if err := push(c.o.Repo, head, branch); err != nil {
		return nil, btip, err
	}
	c.say("candidate-copied", "", branch, fmt.Sprintf("%s copied from %s, the release tool asked for v%s", strings.Join(copied, ", "), c.o.Base, v.rc(n)))
	return copied, head, nil
}

// parseTime reads a commit's date, RFC 3339.
func parseTime(at string) (time.Time, error) { return time.Parse(time.RFC3339, at) }

// cherryPick applies commit's change on head, as `git cherry-pick -x`,
// with no working tree: the new commit, or why it does not apply.
func cherryPick(repo, head, commit string) (string, error) {
	tree, err := git(repo, nil, "merge-tree", "--write-tree", "--merge-base="+commit+"^", head, commit)
	if err != nil {
		return "", errors.New("its change clashes with the branch")
	}
	tree, _, _ = strings.Cut(tree, "\n")
	msg, err := git(repo, nil, "log", "-1", "--format=%B", commit)
	if err != nil {
		return "", err
	}
	msg += "\n\n(cherry picked from commit " + commit + ")\n"
	return git(repo, strings.NewReader(msg), "commit-tree", tree, "-p", head)
}

// yes reads a person's yes on the open candidate, and ships the version
// on it: the release tool asked for vX.Y.0 on the branch, the issue and
// the milestone closed.
func (c *candidateRun) yes(title string, v minor, is forge.Issue, btip string) (string, error) {
	where := fmt.Sprintf("#%d", is.ID)
	t := forge.Target{Kind: "issue", ID: is.ID}
	if !slices.Contains(is.Labels, c.accepted) {
		return fmt.Sprintf("v%s waits for a person's yes on #%d", v.version(), is.ID), nil
	}
	events, err := c.b.LabelEvents(is.ID)
	if err != nil {
		return "", err
	}
	var yes *forge.LabelEvent
	for i := range events {
		if events[i].Label == c.accepted && events[i].Added {
			yes = &events[i]
		}
	}
	switch {
	case yes == nil:
		c.say("candidate-waits", "warn", where, "labelled `"+c.accepted+"` by a hand the forge does not name: no yes")
		return "", nil
	case yes.Bot || yes.Author == c.app || !yes.Writer || yes.Author == "":
		c.say("candidate-waits", "warn", where, fmt.Sprintf("labelled `%s` by @%s, who may not push nor merge here: no yes", c.accepted, yes.Author))
		return "", nil
	}
	by := "@" + yes.Author
	n, tag := c.r.lastRC(v)
	if n == 0 {
		c.say("candidate-waits", "", where, fmt.Sprintf("%s's yes, but the release tool has tagged no v%s-rc yet: it waits for the tag", by, v.version()))
		return "", nil
	}
	tagged, err := git(c.o.Repo, nil, "rev-parse", tag+"^{commit}")
	if err != nil {
		return "", err
	}
	at, _ := git(c.o.Repo, nil, "log", "-1", "--format=%cI", tagged)
	// What ships is what was tried: nothing past the candidate but the
	// versions the tool bumps and its changelog; the yes given since.
	untried, err := untried(c.o.Repo, tagged, btip)
	if err != nil {
		return "", err
	}
	again := ""
	switch {
	case later(at, yes.Created):
		again = fmt.Sprintf("%s was tagged after %s's yes", tag, by)
	case untried != "":
		again = fmt.Sprintf("`%s` holds %s past %s, not tried yet: the next candidate carries it", v.branch(), untried, tag)
	}
	if again != "" {
		if err := c.f.Label(t, nil, []string{c.accepted}); err != nil {
			return "", err
		}
		body := fmt.Sprintf("**workline took `%s` off**: %s. Try the newest candidate once tagged, then put the label again.", c.accepted, again)
		if err := c.f.Comment(t, body, forge.Marker("candidate-again="+short(btip))); err != nil {
			return "", err
		}
		c.say("candidate-yes-reset", "warn", where, again+": the yes taken off, said on it")
		return "", nil
	}
	head, err := releaseAs(c.o.Repo, btip, v.version())
	if err != nil {
		return "", err
	}
	if err := push(c.o.Repo, head, v.branch()); err != nil {
		return "", err
	}
	body := fmt.Sprintf("**Shipping v%s**: %s's yes to `%s`. workline asked the release tool for `v%s` on `%s`, that candidate as tried and its versions; the tool tags it and publishes it.", v.version(), by, tag, v.version(), v.branch())
	if err := c.f.Comment(t, body, forge.Marker("candidate-shipped=v"+v.version())); err != nil {
		return "", err
	}
	// The milestone first: a run that stops between the two finds the
	// issue still open, and closes it then.
	if err := c.mi.CloseMilestone(title); err != nil {
		return "", err
	}
	if err := c.b.Close(is.ID, 0); err != nil {
		return "", err
	}
	c.say("candidate-shipped", "", where, fmt.Sprintf("%s's yes to %s: the release tool asked for v%s on %s; #%d and milestone %s closed", by, tag, v.version(), v.branch(), is.ID, title))
	return fmt.Sprintf("v%s shipped on %s's yes", v.version(), by), nil
}

// untried names the files the branch changes past the candidate tried
// that are more than the release tool's bumps and changelog; "" when none.
func untried(repo, tagged, btip string) (string, error) {
	files, err := git(repo, nil, "diff", "--name-only", "--no-renames", tagged, btip)
	if err != nil || files == "" {
		return "", err
	}
	var more []string
	for _, p := range strings.Split(files, "\n") {
		if strings.HasPrefix(strings.ToUpper(p[strings.LastIndex(p, "/")+1:]), "CHANGELOG") {
			continue
		}
		diff, err := git(repo, nil, "diff", "--no-renames", tagged, btip, "--", p)
		if err != nil {
			return "", err
		}
		now := func() (string, error) { return git(repo, nil, "show", btip+":"+p) }
		if !release.OnlyBumps(p, diff, now) {
			more = append(more, "`"+p+"`")
		}
	}
	return strings.Join(more, ", "), nil
}

// refused reads a candidate's issue closed by someone else than workline:
// a person who may push and merge refuses it, once, and features wait for
// a person's yes again, in a demotion's issue (demote.go).
func (c *candidateRun) refused(v minor, is forge.Issue) (string, error) {
	where := fmt.Sprintf("#%d", is.ID)
	notes, err := c.b.Notes(forge.Target{Kind: "issue", ID: is.ID})
	if err != nil && !errors.Is(err, forge.ErrAuthorsUnknown) {
		return "", err
	}
	if _, shipped := ownNote(notes, "candidate-shipped=", c.app, false); shipped {
		return "", nil
	}
	if _, done := ownNote(notes, "candidate-refused=", c.app, false); done {
		return "", nil
	}
	who, ok, err := forge.Closing(c.f, is.ID)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return "", err
		}
		ok = false
	}
	if !ok || !who.Writer || who.Bot || who.Author == "" || who.Author == c.app {
		by := "a hand the forge does not name"
		if ok && who.Author != "" {
			by = "@" + who.Author
		}
		c.say("candidate-refusal-ignored", "warn", where, fmt.Sprintf("closed by %s, who may not push nor merge here: not taken for a refusal; reopen it, and a person who may push and merge closes it", by))
		return "", nil
	}
	rc := v.rc(rcOf(is))
	title := "workline: its " + kindFeature + " merge requests wait for a person's yes again (candidate v" + rc + " refused)"
	body := fmt.Sprintf("**@%s refused the release candidate `v%s`** (#%d). The merge policy is `enterprising`: features merged with no person's yes, the candidate their check. Until this issue is closed, each feature's pull request waits for a person's yes again, the label `%s`, as at `normal`.\n\n"+
		"**To let features merge alone again, close this issue** — a person who may push and merge; a bot's close never lifts it.", who.Author, rc, is.ID, c.accepted)
	if _, err := c.f.OpenIssue(title, body, forge.Marker("merge-demoted=candidate-v"+rc)); err != nil {
		return "", err
	}
	note := fmt.Sprintf("**Refused by @%s**: features wait for a person's yes on each pull request again, said in an issue of its own. Put what is wrong in issues of milestone %s: once merged, workline copies them onto `%s` and asks for the next candidate.", who.Author, v.title(), v.branch())
	if err := c.f.Comment(forge.Target{Kind: "issue", ID: is.ID}, note, forge.Marker("candidate-refused=v"+rc)); err != nil {
		return "", err
	}
	c.say("candidate-refused", "warn", where, fmt.Sprintf("closed by @%s: v%s refused, features wait for a person's yes again", who.Author, rc))
	return fmt.Sprintf("v%s refused by @%s", rc, who.Author), nil
}
