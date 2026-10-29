package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
)

// The developer's code at `normal` (ADR-0054, step 7): its pull request,
// workline's own as the forge says it, merges with no person's yes when
// the issue it closes is a fix or a task, every gate passed and no
// finding of the reviewer's open; a feature waits for a person's yes. The
// kind is the issue's labels', never the agent's say
// (roles/developer/docs/merge.md).

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
	case d.Kind == developer.KindFeature:
		// Said once on it, where a person reviewing looks, the decision's
		// record for the auditor (ADR-0046).
		t := forge.Target{Kind: "merge-request", ID: w.mr.ID}
		body := fmt.Sprintf("**Waiting for a person's yes**: this is %s. The project's merge policy is `normal`: the developer's fix or task merges once every gate passes, a feature waits for a person's yes.\n\n- To accept this one, put the label `%s`.\n- Not a feature? Change the label on #%d.",
			d.Says(), backlog.AcceptedLabel(f), d.Issue)
		if err := f.Comment(t, body, forge.Marker("merge-waits="+developer.KindFeature)); err != nil {
			return nil, err
		}
		return waits("", d.Says())
	case len(d.Open) > 0:
		return waits("warn", fmt.Sprintf("%s, but the reviewer's findings are open: %s", d.Says(), strings.Join(d.Open, "; ")))
	}
	return admit(f, m, o, w, n, hand, waits, kindCode, d.Says())
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
