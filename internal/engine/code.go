package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// The developer's code at `normal` and `enterprising` (ADR-0054, step 7;
// ADR-0060): its pull request,
// workline's own as the forge says it, merges with no person's yes when
// the issue it closes is a fix, every gate passed, no finding of the
// reviewer's open and its bug reproduced, the tester's test red on the
// base (ADR-0060); a feature or a task waits for a person's yes, but at
// `enterprising`, where it goes into a milestone whose release candidate
// a person tries (candidate.go). The kind is the issue's labels', never
// the agent's say (roles/developer/docs/merge.md).

// codeRoles are the roles whose commits a developer's branch holds: the
// tester's acceptance tests, then the developer's code and fixes.
var codeRoles = []string{"tester", "developer"}

// weighCode weighs a developer's pull request no person said yes to, past
// who opened and pushed it: its commits, the pages it changes, the issue
// it closes and the reviewer's findings, then as any of workline's own.
func weighCode(f forge.Forge, m forge.Merger, o FollowOptions, w weighed, n *normal, hand map[string]kept, waits func(level, why string) (*inLine, error)) (*inLine, error) {
	who, err := notCoded(o.Repo, w.baseTip, w.tip, n.committer)
	if err != nil {
		return nil, err
	}
	if who != "" {
		return waits("", "from the developer's branch, but "+who+" is not workline's own")
	}
	page, err := steersAny(o.Repo, w.baseTip, w.tip, n.steers)
	if err != nil {
		return nil, err
	}
	if page != "" {
		return waits("", "the developer's, but it changes "+page+", which agents are told to read")
	}
	d, err := developer.ReadMerge(f, w.mr)
	if err != nil {
		return nil, err
	}
	switch {
	case d.Why != "":
		return waits("", "from the developer's branch, but "+d.Why)
	case d.Kind == "":
		return waits("", fmt.Sprintf("%s: whether it is a feature, a label on #%d says (`bug`, `enhancement`, `task`)", d.Says(), d.Issue))
	case d.WaitsForAYes() && o.Merge != MergeEnterprising:
		// Said once on it, where a person reviewing looks, the decision's
		// record for the auditor (ADR-0046).
		t := forge.Target{Kind: "merge-request", ID: w.mr.ID}
		body := fmt.Sprintf("**Waiting for a person's yes**: this is %s. The project's merge policy is `%s`: the developer's fix of a reproduced bug merges once every gate passes, a feature or a task waits for a person's yes.\n\n- To accept this one, put the label `%s`.\n- A bug? Change the label on #%d.",
			d.Says(), o.Merge, backlog.AcceptedLabel(f), d.Issue)
		if err := f.Comment(t, body, forge.Marker("merge-waits="+d.Kind)); err != nil {
			return nil, err
		}
		return waits("", d.Says())
	case d.WaitsForAYes() && n.demoted[kindFeature] != "":
		// At `enterprising`, a candidate refused: features and tasks wait
		// for a person's yes again until a person lifts it (ADR-0060).
		t := forge.Target{Kind: "merge-request", ID: w.mr.ID}
		body := fmt.Sprintf("**Waiting for a person's yes**: this is %s. The project's merge policy is `enterprising`, but a release candidate was refused: features and tasks wait for a person's yes on each pull request until a person closes %s.\n\n- To accept this one, put the label `%s`.",
			d.Says(), n.demoted[kindFeature], backlog.AcceptedLabel(f))
		if err := f.Comment(t, body, forge.Marker("merge-waits="+d.Kind)); err != nil {
			return nil, err
		}
		return waits("warn", fmt.Sprintf("%s, but features and tasks wait for a person's yes since a release candidate was refused (%s)", d.Says(), n.demoted[kindFeature]))
	case len(d.Open) > 0:
		return waits("warn", fmt.Sprintf("%s, but the reviewer's findings are open: %s", d.Says(), strings.Join(d.Open, "; ")))
	}
	if d.WaitsForAYes() && d.Milestone == "" {
		// At `enterprising`, a feature or a task goes into a milestone,
		// whose release candidate a person tries (ADR-0060): the product
		// owner puts a feature there at ready; one readied before, here.
		title, err := featureMilestone(f, o.Repo, d.Issue)
		if err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return nil, err
			}
			return waits("warn", fmt.Sprintf("%s, but #%d could not be put in a milestone, its release candidate's: %s", d.Says(), d.Issue, forge.Said(err)))
		}
		w.say("milestone-set", "", fmt.Sprintf("#%d, a %s, put in milestone %s: its release candidate is tried before the version ships", d.Issue, d.Kind, title))
	}
	if d.Kind == developer.KindFix {
		why, err := unreproduced(o.Repo, w.baseTip, w.tip, d)
		if err != nil {
			return nil, err
		}
		if why != "" {
			// Said once on it, as a feature's wait is: a fix with no test
			// that failed before proves nothing (ADR-0060).
			t := forge.Target{Kind: "merge-request", ID: w.mr.ID}
			body := fmt.Sprintf("**Waiting for a person's yes**: this is %s, but %s. The project's merge policy is `%s`: a bug's fix merges once a test that failed on the base proves the bug and every gate passes.\n\n- To accept this one, put the label `%s`.\n- Not a bug? Change the label on #%d.",
				d.Says(), why, o.Merge, backlog.AcceptedLabel(f), d.Issue)
			if err := f.Comment(t, body, forge.Marker("merge-waits="+developer.Unreproduced)); err != nil {
				return nil, err
			}
			return waits("", d.Says()+", but "+why)
		}
	}
	return admit(f, m, o, w, n, hand, waits, kindCode, d.Says())
}

// unreproduced says why a fix's bug is not shown reproduced: its code
// starts from no tester's Draft, or its branch holds no commit of the
// tester's past base; "" when it is.
func unreproduced(repo, baseTip, tip string, d developer.Merge) (string, error) {
	if !d.Reproduced {
		return "its code starts from no tester's Draft: no test red on the base shows the bug", nil
	}
	out, err := git(repo, nil, "log", "--format=%H", baseTip+".."+tip)
	if err != nil {
		return "", err
	}
	for _, c := range strings.Fields(out) {
		if slices.ContainsFunc(strings.Split(ownTrailer(repo, c), ","), func(r string) bool { return strings.TrimSpace(r) == "tester" }) {
			return "", nil
		}
	}
	return "its branch holds no commit of the tester's: no test red on the base shows the bug", nil
}

// notCoded names the first commit of the branch, past base, that is not
// the developer's nor the tester's: not committed under the name
// workline's jobs commit with, or with no trailer naming one of them; ""
// when every one is.
func notCoded(repo, baseTip, tip, committer string) (string, error) {
	out, err := git(repo, nil, "log", "--reverse", "--format=%H%x1f%cn", baseTip+".."+tip)
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(out, "\n") {
		c, name, ok := strings.Cut(l, "\x1f")
		if !ok {
			continue
		}
		by := ownTrailer(repo, c)
		if name != committer || by == "" || slices.ContainsFunc(strings.Split(by, ","), func(r string) bool { return !slices.Contains(codeRoles, strings.TrimSpace(r)) }) {
			return commitSaid(repo, c) + ", committed by " + name, nil
		}
	}
	return "", nil
}

// steersAny names the first file the branch changes since it left base
// that agents are told to read; "" when there is none.
func steersAny(repo, baseTip, tip string, steers func(string) bool) (string, error) {
	from, err := git(repo, nil, "merge-base", baseTip, tip)
	if err != nil {
		return "", err
	}
	files, err := git(repo, nil, "diff", "--name-only", "--no-renames", "-z", from, tip)
	if err != nil {
		return "", err
	}
	for _, p := range strings.Split(strings.TrimSuffix(files, "\x00"), "\x00") {
		if p != "" && steers(p) {
			return p, nil
		}
	}
	return "", nil
}

// featureMilestone puts a feature's issue in the next open milestone,
// opened when there is none (NextMilestone), and says which.
func featureMilestone(f forge.Forge, repo string, issue int) (string, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return "", errors.New("this forge keeps no milestones")
	}
	title, _, err := NextMilestone(b, repo)
	if err != nil {
		return "", err
	}
	return title, b.SetMilestone(issue, title)
}
