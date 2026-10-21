package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// The merge policy (ADR-0042): `workline follow` turns the forge's own
// auto-merge on for the merge requests a person accepted, their head's
// verdict passed. The forge merges each once its own head is green,
// checking its own rules, none waiting for another nor brought up to date
// first (ADR-0051, cut 2); no agent is called.

// HeadVerdict is the line's verdict on a merge request's head, written by
// the judging job (`workline route --verdict`), kept by the job that
// writes (`workline follow --verdict`).
type HeadVerdict struct {
	Head     string            `json:"head"`
	Status   string            `json:"status"`
	Steps    []StepStatus      `json:"steps,omitempty"`
	Findings []verdict.Finding `json:"findings,omitempty"`
}

// StepStatus is a step of the line, and how it ended.
type StepStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// WriteVerdict writes a head's verdict to file, as JSON.
func WriteVerdict(file string, v HeadVerdict) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, append(data, '\n'), 0o644)
}

// shortReview are the findings saying a step that needed an agent's
// judgement had none, or not whole: no agent, a lens unanswered, the
// tokens spent. A pass with one is no pass for the merge policy: "an agent
// really answered where the project sets one" (ADR-0042).
var shortReview = []string{"not-reviewed", "lens-failed", "review-not-whole"}

// short says why the review of a head that passed falls short, "" when it
// does not.
func (v *HeadVerdict) short() string {
	for _, f := range v.Findings {
		if slices.Contains(shortReview, f.Rule) {
			return fmt.Sprintf("its review falls short (%s): %s", f.Rule, firstSentence(f.Message))
		}
	}
	return ""
}

// waits says why the review of a head that passed is still to come, a
// draft's (verdict.WaitsForReady), "" when it is not.
func (v *HeadVerdict) waits() string {
	for _, f := range v.Findings {
		if f.Rule == verdict.WaitsForReady {
			return "a draft: the rules and gates passed, the AI review waits for it to be marked ready"
		}
	}
	return ""
}

// red says why a head's verdict did not pass.
func (v *HeadVerdict) red() string {
	var rules []string
	for _, f := range v.Findings {
		if (f.Level == "" || f.Level == "block") && !slices.Contains(rules, f.Rule) {
			rules = append(rules, f.Rule)
		}
	}
	var steps []string
	for _, s := range v.Steps {
		if s.Status != verdict.Pass {
			steps = append(steps, s.Name+" "+s.Status)
		}
	}
	why := fmt.Sprintf("the line's verdict on its head is %s", v.Status)
	if len(steps) > 0 {
		why += " (" + strings.Join(steps, ", ") + ")"
	}
	if len(rules) > 0 {
		why += ": " + strings.Join(rules, ", ")
	}
	return why
}

// The classes of a head's verdict, as the policy reads it.
const (
	verdictPass  = "pass"
	verdictShort = "short" // passed, a review short of its agent
	verdictWaits = "waits" // passed, a draft's: the AI review to come once ready
	verdictRed   = "red"
)

// class is the verdict's class for the policy, and why.
func (v *HeadVerdict) class() (string, string) {
	switch {
	case v.Status != verdict.Pass:
		return verdictRed, v.red()
	case v.waits() != "":
		return verdictWaits, v.waits()
	case v.short() != "":
		return verdictShort, v.short()
	}
	return verdictPass, "the line passed, every agent's review whole"
}

func firstSentence(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}

// kept is a head's verdict in hand: its class and why.
type kept struct{ class, why string }

// keepVerdict keeps the verdict the file holds on its head where only the
// token `follow` holds writes it (GitHub: the status workline/verdict, by
// the App; a writer's own status is never read), and has it in hand for
// this run: the pipeline that judged the head reads it there (GitLab).
func keepVerdict(m forge.Merger, file string, hand map[string]kept) (verdict.Finding, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return verdict.Finding{}, fmt.Errorf("the verdict to keep: %w", err)
	}
	var v HeadVerdict
	if err := json.Unmarshal(data, &v); err != nil || v.Head == "" || v.Status == "" {
		return verdict.Finding{}, fmt.Errorf("the verdict to keep, %s: no head and status in it (%v)", file, err)
	}
	class, why := v.class()
	if err := m.KeepVerdict(v.Head, class, why); err != nil {
		return verdict.Finding{}, err
	}
	hand[v.Head] = kept{class, why}
	return verdict.Finding{Rule: "verdict-kept", Where: short(v.Head),
		Message: fmt.Sprintf("the line's verdict on %s: %s, %s", short(v.Head), class, why)}, nil
}

// verdictOf is the verdict on head: in hand, else as the forge keeps it;
// "" none.
func verdictOf(m forge.Merger, hand map[string]kept, head string) (kept, error) {
	if k, ok := hand[head]; ok {
		return k, nil
	}
	class, why, err := m.Verdict(head)
	return kept{class, why}, err
}

// accepted says whether a label is a person's yes, `workline:accepted`
// as GitHub spells it, `workline::accepted` as GitLab's scoped labels.
func accepted(label string) bool {
	return backlog.Accepted(forge.Issue{Labels: []string{label}})
}

// theYes is the last time the label was put by someone who may push and
// merge — GitHub's write access, GitLab's Developer —, nil when it is not
// on now by such a hand — put by another, or taken off since —; ignored,
// who put it then.
func theYes(events []forge.LabelEvent) (yes, ignored *forge.LabelEvent) {
	for i := range events {
		e := &events[i]
		switch {
		case !accepted(e.Label):
		case !e.Added:
			yes, ignored = nil, nil
		case e.Writer && !e.Bot && e.Author != "":
			yes, ignored = e, nil
		case yes == nil:
			ignored = e
		}
	}
	return yes, ignored
}

// takenOff is when the label was last taken off, "" when it was not, or
// was put again since.
func takenOff(events []forge.LabelEvent) string {
	at := ""
	for _, e := range events {
		if accepted(e.Label) {
			at = map[bool]string{true: "", false: e.Created}[e.Added]
		}
	}
	return at
}

// later says whether the time a is after b, both RFC 3339 as forges give
// them; compared as written when one does not read.
func later(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339, a)
	tb, errB := time.Parse(time.RFC3339, b)
	if errA != nil || errB != nil {
		return a > b
	}
	return ta.After(tb)
}

// plus is an RFC 3339 time d later; as written when it does not read.
func plus(at string, d time.Duration) string {
	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return at
	}
	return t.Add(d).Format(time.RFC3339Nano)
}

// isAncestor says whether commit a is in b's history.
func isAncestor(repo, a, b string) bool {
	_, err := git(repo, nil, "merge-base", "--is-ancestor", a, b)
	return err == nil
}

// OwnCommit names the role of a commit workline made, "" when it is none.
// Its trailer is not taken on trust — anyone can write one: a merge is
// workline's only when it merges base, or a commit of base's history, and
// its tree is exactly what workline's own merge of its two parents gives
// (OwnMerge); another commit, only when it changes doc pages alone
// (docPage). Else it is judged as a person's.
func OwnCommit(repo, commit, base string) string {
	out, err := git(repo, nil, "log", "-1", "--format=%P%x1f%(trailers:key="+OwnTrailer+",valueonly,separator=%x2c)", commit)
	if err != nil {
		return ""
	}
	parents, role, _ := strings.Cut(out, "\x1f")
	if role = madeBy(role); role == "" {
		return ""
	}
	switch len(strings.Fields(parents)) {
	case 2:
		if !OwnMerge(repo, commit, base) {
			return ""
		}
	case 1:
		files, err := git(repo, nil, "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
		if err != nil || files == "" {
			return ""
		}
		for _, f := range strings.Split(files, "\n") {
			if !docPage(f) {
				return ""
			}
		}
	default:
		return ""
	}
	return role
}

// agentFiles are the files that steer agents, by name: a role's prompt
// or a project's instructions, never a doc page whatever its extension.
var agentFiles = []string{"agents.md", "claude.md", "gemini.md", "copilot-instructions.md"}

// docPage says whether a path is a doc page the roles write: Markdown, not
// under a tool's folder (.workline, .github, .claude, .cursor), not a file
// that steers agents (AGENTS.md, CLAUDE.md…), and in a role's folder only
// a page for people (rolePage), never a facet the engine gives the agent.
func docPage(path string) bool {
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".md") {
		return false
	}
	parts := strings.Split(lower, "/")
	for i, p := range parts[:len(parts)-1] {
		switch p {
		case ".workline", ".github", ".claude", ".cursor":
			return false
		case "roles":
			if !rolePage(parts[i+1:]) {
				return false
			}
		}
	}
	return !slices.Contains(agentFiles, parts[len(parts)-1])
}

// rolePage says whether a path in a role's folder, `roles/<name>/…`, is
// one of its pages for people, never loaded for an agent (the role
// contract's layout): its README.md, a page under its docs/. Its facets —
// persona, knowledge, instruction, policy, lenses, kinds, skills — are not.
func rolePage(rest []string) bool {
	return len(rest) == 2 && rest[1] == "readme.md" || len(rest) > 2 && rest[1] == "docs"
}

// OwnMerge says whether a merge commit is one `follow` makes: its second
// parent base, or in base's history — the base merged in, never another
// branch —, merged into its first, workline's own lines settled
// (ADR-0041), and nothing else: its tree exactly the one that merge gives.
func OwnMerge(repo, commit, base string) bool {
	out, err := git(repo, nil, "rev-list", "--parents", "-n", "1", commit)
	parts := strings.Fields(out)
	if err != nil || len(parts) != 3 || base == "" || !isAncestor(repo, parts[2], base) {
		return false
	}
	tree, _, theirs, err := settleTree(repo, parts[1], parts[2], func(string) bool { return true })
	if err != nil || len(theirs) > 0 {
		return false
	}
	own, err := git(repo, nil, "rev-parse", commit+"^{tree}")
	return err == nil && own == tree
}

// ownRecord says whether a commit changes, against its one parent, only
// lines workline writes in the docs (ADR-0041): a header's checked,
// judged, judged-in-parts or verified, and derived blocks reading what
// their commands give at the commit (regenerated). A doc's text, another
// header key, a file added, removed, made executable or no doc's: not.
func ownRecord(repo, commit string, isDoc func(string) bool, regenerated func(commit string, docs []string) bool) bool {
	out, err := git(repo, nil, "rev-list", "--parents", "-n", "1", commit)
	parent := strings.Fields(out)
	if err != nil || len(parent) != 2 || isDoc == nil {
		return false
	}
	// Each change, `:mode mode id id status` then its path.
	raw, err := git(repo, nil, "diff", "--raw", "--no-renames", "--no-abbrev", "-z", parent[1], commit)
	parts := strings.Split(strings.TrimSuffix(raw, "\x00"), "\x00")
	if err != nil || raw == "" || len(parts)%2 != 0 {
		return false
	}
	var blocks []string
	for i := 0; i < len(parts); i += 2 {
		f, file := strings.Fields(parts[i]), parts[i+1]
		if len(f) != 5 || f[4] != "M" || f[0] != ":"+f[1] || !regular(f[1]) || !isDoc(file) {
			return false
		}
		was, ok1 := blob(repo, parent[1], file)
		now, ok2 := blob(repo, commit, file)
		switch {
		case !ok1 || !ok2:
			return false
		case unworked(was) == unworked(now):
		case underived(unworked(was)) == underived(unworked(now)):
			blocks = append(blocks, file)
		default:
			return false
		}
	}
	return len(blocks) == 0 || regenerated != nil && regenerated(commit, blocks)
}

// ownRebuild says whether tip is covered rebuilt on the base by `follow`
// (ADR-0034): one commit on a commit of base's history, its tree exactly
// the change of covered redone on it.
func ownRebuild(repo, covered, tip, base string) bool {
	out, err := git(repo, nil, "rev-list", "--parents", "-n", "1", tip)
	parts := strings.Fields(out)
	if err != nil || len(parts) != 2 || !isAncestor(repo, parts[1], base) {
		return false
	}
	tree, clashes, err := redo(repo, parts[1], covered)
	if err != nil || len(clashes) > 0 {
		return false
	}
	own, err := git(repo, nil, "rev-parse", tip+"^{tree}")
	return err == nil && own == tree
}

// pushedAfter says what was pushed to a merge request after the yes given
// at yes, "" nothing but workline's own: the yes covers the head it was
// given on, as approvals reset on a push (ADR-0042). The head it was given
// on is the last pushed before it, as the forge says; a forge that does
// not say leaves the commits' own dates, past base. workline's own is
// checked by its tree, never taken on its trailer: `follow`'s update
// (OwnMerge, ownRebuild), or a commit changing only the lines workline
// writes in the docs (ownRecord), as a merge of the base settles them.
func pushedAfter(o FollowOptions, branch string, st forge.MergeState, yes, baseTip, tip string) (string, error) {
	repo := o.Repo
	own := func(c string) bool {
		if ownTrailer(repo, c) == "" {
			return false // a person's commit is a person's
		}
		return OwnMerge(repo, c, baseTip) || ownRecord(repo, c, o.IsDoc, func(at string, docs []string) bool { return regenerates(o, at, docs) })
	}
	covered := ""
	for _, p := range st.Pushes {
		if !later(p.At, yes) {
			covered = p.Head
		}
	}
	// Labelled as it was opened: GitLab dates its first version a moment
	// after the label put with it.
	if covered == "" && len(st.Pushes) > 0 && !later(st.Pushes[0].At, plus(yes, time.Minute)) {
		covered = st.Pushes[0].Head
	}
	if covered != "" {
		if covered == tip {
			return "", nil
		}
		if _, err := git(repo, nil, "cat-file", "-e", covered+"^{commit}"); err != nil {
			git(repo, nil, "fetch", "-q", "origin", covered) // gone from the branch, maybe still on the forge
		}
		if !isAncestor(repo, covered, tip) {
			if strings.HasPrefix(branch, "workline/") && ownRebuild(repo, covered, tip, baseTip) {
				return "", nil
			}
			return fmt.Sprintf("its branch was rewritten (%s, over %s)", short(tip), short(covered)), nil
		}
		// The base's commits a merge of it brings are no push to judge.
		out, err := git(repo, nil, "rev-list", "--reverse", covered+".."+tip, "^"+baseTip)
		if err != nil {
			return "", err
		}
		for _, c := range strings.Fields(out) {
			if !own(c) {
				return commitSaid(repo, c) + " was pushed", nil
			}
		}
		return "", nil
	}
	out, err := git(repo, nil, "log", "--reverse", "--format=%H%x1f%cI", baseTip+".."+tip)
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(out, "\n") {
		c, at, ok := strings.Cut(l, "\x1f")
		if ok && later(at, yes) && !own(c) {
			return commitSaid(repo, c) + " was committed", nil
		}
	}
	return "", nil
}

// ownTrailer is the role a commit's trailer names, "".
func ownTrailer(repo, c string) string {
	by, _ := git(repo, nil, "log", "-1", "--format=%(trailers:key="+OwnTrailer+",valueonly,separator=%x2c)", c)
	return madeBy(by)
}

// commitSaid names a commit by its short id and subject.
func commitSaid(repo, c string) string {
	subject, _ := git(repo, nil, "log", "-1", "--format=%s", c)
	return fmt.Sprintf("%s (%q)", short(c), subject)
}

// inLine is a merge request accepted into the base: it merges once its
// own head is green, or waits for its verdict.
type inLine struct {
	mr     forge.MergeRequest
	where  string
	behind bool
}

// mergePolicy runs the project's merge policy on the merge requests into
// o.Base, says what it did of each, and sums it up for the run's summary.
func mergePolicy(f forge.Forge, o FollowOptions, local bool, res *Result, hand map[string]kept) (string, error) {
	sum := "merge policy (" + o.Merge + ")"
	m, ok := f.(forge.Merger)
	if !ok || local {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unsupported", Level: "warn",
			Message: "this forge has no auto-merge workline can turn on: nothing turned on"})
		return sum + ": the forge has no auto-merge", nil
	}
	if why := forge.MergeToken(f); why != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-token", Level: "warn",
			Message: why + ": nothing turned on"})
		return sum + ": nothing turned on, the token's merges would start no pipeline", nil
	}
	// A yes covers the head it was given on: the forge must hold it there
	// (GitHub: a required status a new head lacks), not a race with the
	// next run.
	why, err := m.Guarded(o.Base)
	switch {
	case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
		return "", err
	case err != nil:
		why = "whether a push after a yes is held could not be read: " + forge.Said(err)
	}
	if why != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unguarded", Level: "warn",
			Message: why + ": nothing turned on"})
		return sum + ": nothing turned on, a push after a yes would not be held", nil
	}
	// The label in the forge's list, for a person to put, in bulk, from a
	// saved filter (ADR-0038).
	if b, ok := f.(forge.Backlog); ok {
		err := b.EnsureLabel(backlog.AcceptedLabel(f), "0e8a16", backlog.AcceptedSays)
		switch {
		case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
			return "", err
		case err != nil:
			res.Findings = append(res.Findings, verdict.Finding{Rule: "label-not-made", Level: "warn",
				Message: fmt.Sprintf("the label %s could not be made in the forge's list, for a person to put: %v", backlog.AcceptedLabel(f), err)})
		}
	}
	// At `normal`: the App workline's own work comes from, the pages agents
	// read, the kinds the project lets merge alone besides the routine
	// ones, those waiting for a person's yes again since a revert — said
	// where the level is set.
	var n *normal
	if o.Merge == MergeNormal {
		tip, err := remoteTip(o.Repo, o.Base)
		if err != nil {
			return "", err
		}
		n = &normal{alone: o.Alone}
		if n.app, n.committer, err = forge.App(f); err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return "", err
			}
			n.app = ""
		}
		if n.app == "" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-app-unknown", Level: "warn",
				Message: "at `normal`, the token is no App's — a person's, or the forge does not say whose —: nothing is taken for workline's own work, every merge request waits for a person's yes, as at `cautious`"})
		} else {
			if n.steers, err = steering(o.Repo, tip); err != nil {
				return "", err
			}
			if n.demoted, err = demotions(f, o, tip, n.app, res); err != nil {
				return "", err
			}
		}
		sum = "merge policy (normal; " + alone(n) + ")"
	}
	open, err := f.ListMergeRequests()
	if err != nil {
		return "", err
	}
	var line []*inLine
	acts := map[string]int{}
	for _, mr := range open {
		switch {
		case mr.Base != o.Base:
			continue
		case !slices.ContainsFunc(mr.Labels, accepted) && !mr.AutoMerge && !routineCandidate(o, n, mr):
			if o.Merge == MergeNormal && o.Release != nil && o.Release(mr.Branch) {
				res.Findings = append(res.Findings, verdict.Finding{Rule: "waits-for-a-yes", Where: forge.MergeRef(f, mr.ID),
					Message: "the release tool's: merging it is the decision to release, never made with no person's yes; the label `workline:accepted`"})
			}
			continue
		}
		before := len(res.Findings)
		e, err := weigh(f, m, o, local, mr, res, hand, n)
		if err != nil {
			return "", err
		}
		for _, x := range res.Findings[before:] {
			acts[x.Rule]++
		}
		if e != nil {
			line = append(line, e)
		}
	}
	var names []string
	for _, e := range line {
		names = append(names, e.where)
	}
	if len(names) == 0 {
		sum += fmt.Sprintf(": none accepted into %s", o.Base)
	} else {
		sum += fmt.Sprintf(": accepted into %s, %s: each merges once its own head is green", o.Base, strings.Join(names, ", "))
	}
	if n := acts["auto-merge-on"]; n > 0 {
		sum += fmt.Sprintf("; auto-merge turned on for %d", n)
	}
	if n := acts["merged-at-once"]; n > 0 {
		sum += fmt.Sprintf("; %d merged at once", n)
	}
	// No one waits for another to merge, none is brought up to date to
	// merge (ADR-0051, cut 2).
	return sum, nil
}

// weigh reads one merge request into the base, accepted or its auto-merge
// on, does what the policy says of it — auto-merge on or off, the yes held
// on its head (Accept) or taken back — and says it; it returns it when it
// is accepted.
func weigh(f forge.Forge, m forge.Merger, o FollowOptions, local bool, mr forge.MergeRequest, res *Result, hand map[string]kept, n *normal) (*inLine, error) {
	where := forge.MergeRef(f, mr.ID)
	say := func(rule, level, msg string) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: level, Where: where, Message: msg})
	}
	st, err := m.MergeState(mr)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return nil, err
		}
		say("merge-state-unread", "warn", "its state could not be read, left as it is: "+forge.Said(err))
		return nil, nil
	}
	baseTip, tip, err := headTips(o.Repo, o.Base, mr, local)
	switch {
	case err != nil && mr.Here:
		return nil, err
	case err != nil:
		say("merge-state-unread", "warn", fmt.Sprintf("from a fork, its head could not be read, left as it is: %v", err))
		return nil, nil
	case tip == "":
		return nil, nil // its branch is gone
	}
	e := &inLine{mr: mr, where: where, behind: !isAncestor(o.Repo, baseTip, tip)}
	off := func() error {
		if !st.AutoMerge {
			return nil
		}
		_, err := m.AutoMerge(mr, tip, false)
		return err
	}
	// takeBack turns auto-merge off and marks the head not accepted.
	takeBack := func(why string) error {
		if err := off(); err != nil {
			return err
		}
		return m.Accept(tip, false, why)
	}
	// At `normal`, workline's own routine merge request, no person's yes
	// given: weighed by what it is (routine.go).
	routine := routineCandidate(o, n, mr)
	w := weighed{mr: mr, st: st, baseTip: baseTip, tip: tip, e: e, say: say, takeBack: takeBack}
	labelled := slices.ContainsFunc(mr.Labels, accepted)
	yes, ignored := theYes(st.Labels)
	switch {
	case !labelled:
		// Auto-merge on, no label. Turned on while the label was on, the
		// label taken off since: it goes off, the veto by the label. By a
		// person: their own yes, weighed as the label's. By another bot (a
		// dependency bot's own auto-merge): left to it.
		if taken := takenOff(st.Labels); taken != "" && st.AutoMergeAt != "" && !later(st.AutoMergeAt, taken) {
			if err := takeBack("the label was taken off"); err != nil {
				return nil, err
			}
			say("auto-merge-off", "", "the label `workline:accepted` was taken off since auto-merge was turned on: auto-merge turned off")
			return nil, nil
		}
		if st.AutoMergeBot || st.AutoMergeBy == "" {
			if routine {
				return weighRoutine(f, m, o, w, n, hand)
			}
			return nil, nil
		}
		yes = &forge.LabelEvent{Writer: true, Note: forge.Note{Author: st.AutoMergeBy, Created: st.AutoMergeAt}}
	case yes == nil:
		who := "a hand the forge does not name"
		if ignored != nil && ignored.Author != "" {
			who = "@" + ignored.Author
		}
		if routine {
			say("label-ignored", "warn", fmt.Sprintf("labelled `workline:accepted` by %s, who may not push nor merge here: no yes, weighed as workline's own", who))
			return weighRoutine(f, m, o, w, n, hand)
		}
		say("label-ignored", "warn", fmt.Sprintf("labelled `workline:accepted` by %s, who may not push nor merge here: no yes, nothing turned on", who))
		return nil, nil
	}
	by := "@" + yes.Author
	if st.Draft {
		// The veto by Draft: auto-merge on goes off, and stays off once
		// it is ready until the label is put again.
		if st.AutoMerge {
			if err := takeBack("marked a draft"); err != nil {
				return nil, err
			}
			say("draft", "", fmt.Sprintf("accepted by %s, then marked a draft: auto-merge turned off, until the label is put again", by))
			return nil, nil
		}
		say("draft", "", fmt.Sprintf("accepted by %s, but a draft: nothing turned on until it is marked ready", by))
		return nil, nil
	}
	pushed, err := pushedAfter(o, mr.Branch, st, yes.Created, baseTip, tip)
	if err != nil {
		return nil, err
	}
	if pushed != "" {
		if err := takeBack(pushed + " after the yes: put the label again"); err != nil {
			return nil, err
		}
		var labels []string
		for _, l := range mr.Labels {
			if accepted(l) {
				labels = append(labels, l)
			}
		}
		t := forge.Target{Kind: "merge-request", ID: mr.ID}
		if len(labels) > 0 {
			if err := f.Label(t, nil, labels); err != nil {
				return nil, err
			}
		}
		took, again := "`"+strings.Join(labels, "`, `")+"`", "Put the label again"
		if len(labels) == 0 { // the person's own auto-merge, their yes
			took, again = "auto-merge", "Turn auto-merge on again"
		}
		body := fmt.Sprintf("**workline took %s off**: %s after %s accepted this merge request, on %s. A yes covers the head it was given on, so its auto-merge is off. %s to accept the head as it is now (%s).",
			took, pushed, by, yes.Created, again, short(tip))
		if err := f.Comment(t, body, forge.Marker("merge-reset="+tip)); err != nil {
			return nil, err
		}
		say("accepted-reset", "warn", fmt.Sprintf("%s after %s accepted it: the yes and auto-merge taken off, said in a comment", pushed, by))
		return nil, nil
	}
	if !st.AutoMerge && st.AutoMergeAt != "" && later(st.AutoMergeAt, yes.Created) {
		if err := m.Accept(tip, false, "auto-merge turned off since: put the label again"); err != nil {
			return nil, err
		}
		say("vetoed", "", fmt.Sprintf("auto-merge was turned on after %s's yes, and off since: left off until the label is put again", by))
		return nil, nil
	}
	if st.ChecksUnread != "" {
		say("checks-unread", "warn", "its head's checks could not be read, only the line's verdict counts: "+st.ChecksUnread)
	}
	if st.Checks == "failure" {
		say("checks-red", "warn", fmt.Sprintf("accepted by %s, but a check failed on its head (%s): auto-merge left as it is", by, short(tip)))
		return nil, nil
	}
	v, err := verdictOf(m, hand, tip)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return nil, err
		}
		say("verdict-unread", "warn", fmt.Sprintf("accepted by %s; its head's verdict could not be read: %s", by, forge.Said(err)))
		return nil, nil
	}
	ours := st.AutoMerge && st.AutoMergeBot && !later(yes.Created, st.AutoMergeAt)
	switch v.class {
	case "":
		say("verdict-missing", "", fmt.Sprintf("accepted by %s; no verdict of the line on its head (%s) yet: it waits for it", by, short(tip)))
		return e, nil
	case verdictWaits: // marked ready, the run on ready not done yet
		say("review-to-come", "", fmt.Sprintf("accepted by %s; its head's (%s) verdict is the draft's, %s: it waits for the run on ready", by, short(tip), v.why))
		return e, nil
	case verdictRed, verdictShort:
		rule := map[string]string{verdictRed: "verdict-red", verdictShort: "review-short"}[v.class]
		then := "auto-merge stays off"
		switch {
		case ours:
			if err := off(); err != nil {
				return nil, err
			}
			then = "auto-merge turned off"
		case st.AutoMerge:
			then = by + "'s auto-merge left on, held by the yes not given"
		}
		if err := m.Accept(tip, false, v.why); err != nil {
			return nil, err
		}
		say(rule, "warn", fmt.Sprintf("accepted by %s, but %s; %s", by, v.why, then))
		return nil, nil
	}
	// Its verdict passed: the yes held on this head, then auto-merge on.
	if err := m.Accept(tip, true, "accepted by "+by+", its head's verdict passed"); err != nil {
		return nil, err
	}
	if st.AutoMerge {
		return e, nil
	}
	merged, err := m.AutoMerge(mr, tip, true)
	switch {
	case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
		return nil, err
	case err != nil:
		say("auto-merge-refused", "warn", fmt.Sprintf("accepted by %s, its head's verdict passed, but the forge refused to turn auto-merge on: %s", by, forge.Said(err)))
		return nil, nil
	case merged:
		say("merged-at-once", "", fmt.Sprintf("accepted by %s, its head's verdict passed: merged at once, the forge's rules met already", by))
		return nil, nil
	}
	say("auto-merge-on", "", fmt.Sprintf("accepted by %s, its head's verdict passed: auto-merge turned on, the forge merges it once its checks pass", by))
	return e, nil
}
