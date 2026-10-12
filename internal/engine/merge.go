package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// The merge policy (ADR-0042): `workline follow` turns the forge's own
// auto-merge on for the merge requests a person accepted, their head's
// verdict passed, and keeps them up to date with their base, one at a
// time. The forge merges, checking its own rules; no agent is called.

// verdictNotes keeps each head's verdict, a git note on the commit, as
// Gerrit keeps its reviews' (refs/notes/review): whatever event wakes
// `follow` later — a push to the base, a label — it reads there what the
// line said of a head, with the token it holds already.
const verdictNotes = "refs/notes/workline/verdicts"

// HeadVerdict is the line's verdict on a merge request's head, written by
// the judging job (`workline route --verdict`), recorded by the job that
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

// held says whether a verdict blocks only for the release's own fix,
// merged elsewhere first (ADR-0017): a release waiting for its docs.
func (v *HeadVerdict) held() bool {
	n := 0
	for _, f := range v.Findings {
		if f.Level == "" || f.Level == "block" {
			if f.Rule != "fixed-elsewhere" {
				return false
			}
			n++
		}
	}
	return v.Status == verdict.Block && n > 0
}

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

func firstSentence(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}

// recordVerdict keeps the verdict file's head verdict as a note on its
// head, pushed to origin; a push that lost the race to another job's is
// tried again on its notes.
func recordVerdict(repo, file string, local bool) (verdict.Finding, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return verdict.Finding{}, fmt.Errorf("the verdict to record: %w", err)
	}
	var v HeadVerdict
	if err := json.Unmarshal(data, &v); err != nil || v.Head == "" || v.Status == "" {
		return verdict.Finding{}, fmt.Errorf("the verdict to record, %s: no head and status in it (%v)", file, err)
	}
	if _, err := git(repo, nil, "cat-file", "-e", v.Head+"^{commit}"); err != nil && !local {
		if _, err := git(repo, nil, "fetch", "-q", "origin", v.Head); err != nil {
			return verdict.Finding{}, fmt.Errorf("%w: the head judged, %s: %v", forge.ErrUnreachable, short(v.Head), err)
		}
	}
	note, _ := json.Marshal(v)
	for try := 1; ; try++ {
		if !local {
			if err := fetchVerdicts(repo); err != nil {
				return verdict.Finding{}, err
			}
		}
		if _, err := git(repo, nil, "-c", "user.name=workline", "-c", "user.email=workline@localhost",
			"notes", "--ref="+verdictNotes, "add", "-f", "-m", string(note), v.Head); err != nil {
			return verdict.Finding{}, err
		}
		if local {
			break
		}
		// The engine's own record, no person's commits: a machine's hooks
		// have nothing to judge in it.
		_, err := git(repo, nil, "push", "-q", "--no-verify", "origin", verdictNotes+":"+verdictNotes)
		if err == nil {
			break
		}
		if try == 3 {
			return verdict.Finding{}, fmt.Errorf("%w: the verdict's note not pushed: %v", forge.ErrUnreachable, err)
		}
	}
	return verdict.Finding{Rule: "verdict-recorded", Where: short(v.Head),
		Message: fmt.Sprintf("the line's verdict on %s, %s, kept for the merge policy (%s)", short(v.Head), v.Status, verdictNotes)}, nil
}

// fetchVerdicts takes origin's notes of the heads' verdicts, in place of
// the clone's; none there yet is none.
func fetchVerdicts(repo string) error {
	_, err := git(repo, nil, "fetch", "-q", "origin", "+"+verdictNotes+":"+verdictNotes)
	if err != nil && !strings.Contains(err.Error(), "couldn't find remote ref") {
		return fmt.Errorf("%w: %v", forge.ErrUnreachable, err)
	}
	return nil
}

// readVerdict is the verdict noted on head, nil when there is none.
func readVerdict(repo, head string) *HeadVerdict {
	out, err := git(repo, nil, "notes", "--ref="+verdictNotes, "show", head)
	if err != nil {
		return nil
	}
	var v HeadVerdict
	if json.Unmarshal([]byte(out), &v) != nil || v.Status == "" {
		return nil
	}
	return &v
}

// accepted says whether a label is a person's yes, `workline:accepted`
// as GitHub spells it, `workline::accepted` as GitLab's scoped labels.
func accepted(label string) bool {
	return backlog.Accepted(forge.Issue{Labels: []string{label}})
}

// theYes is the last time the label was put by someone who can write
// (ADR-0038's who counts), nil when it is not on now by such a hand —
// put by another, or taken off since —; ignored, who put it then.
func theYes(events []forge.LabelEvent) (yes, ignored *forge.LabelEvent) {
	for i := range events {
		e := &events[i]
		switch {
		case !accepted(e.Label):
		case !e.Added:
			yes, ignored = nil, nil
		case e.Insider && !e.Bot && e.Author != "":
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

// followsOwn says whether a commit is `follow`'s own update: a merge
// carrying workline's trailer, the base merged in (ADR-0041, ADR-0042).
func followsOwn(repo, commit string) bool {
	out, err := git(repo, nil, "log", "-1", "--format=%P%x1f%(trailers:key="+OwnTrailer+",valueonly)", commit)
	if err != nil {
		return false
	}
	parents, by, _ := strings.Cut(out, "\x1f")
	return len(strings.Fields(parents)) == 2 && strings.TrimSpace(by) != ""
}

// pushedAfter says what was pushed to a merge request after the yes given
// at yes, "" nothing but `follow`'s own updates: the yes covers the head
// it was given on, as approvals reset on a push (ADR-0042). The head it
// was given on is the last pushed before it, as the forge says; a forge
// that does not say leaves the commits' own dates, past base.
func pushedAfter(repo string, st forge.MergeState, yes, baseTip, tip string) (string, error) {
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
		if _, err := git(repo, nil, "cat-file", "-e", covered+"^{commit}"); err != nil || !isAncestor(repo, covered, tip) {
			return fmt.Sprintf("its branch was rewritten (%s, over %s)", short(tip), short(covered)), nil
		}
		// The base's commits a merge of it brings are no push to judge.
		out, err := git(repo, nil, "rev-list", "--reverse", covered+".."+tip, "^"+baseTip)
		if err != nil {
			return "", err
		}
		for _, c := range strings.Fields(out) {
			if !followsOwn(repo, c) {
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
		if ok && later(at, yes) && !followsOwn(repo, c) {
			return commitSaid(repo, c) + " was committed", nil
		}
	}
	return "", nil
}

// commitSaid names a commit by its short id and subject.
func commitSaid(repo, c string) string {
	subject, _ := git(repo, nil, "log", "-1", "--format=%s", c)
	return fmt.Sprintf("%s (%q)", short(c), subject)
}

// releaseFix says whether a branch is a role's fix for a release, opened
// apart from the release tool's pull request (ADR-0017): first in line,
// before the release (ADR-0042).
func releaseFix(branch string) bool {
	role, task, ok := strings.Cut(strings.TrimPrefix(branch, "workline/"), "/")
	return strings.HasPrefix(branch, "workline/") && ok && role != "" && task == "release"
}

// inLine is a merge request the merge policy keeps up to date.
type inLine struct {
	mr     forge.MergeRequest
	where  string
	at     string // when the yes was given: the label put, or auto-merge turned on by a person
	first  bool   // the release's own fix: before the rest
	behind bool
}

// mergePolicy runs the project's merge policy on the merge requests into
// o.Base, says what it did of each, and sums it up for the run's summary.
func mergePolicy(f forge.Forge, o FollowOptions, local bool, res *Result) (string, error) {
	sum := "merge policy (" + o.Merge + ")"
	m, ok := f.(forge.Merger)
	if !ok || local {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-unsupported", Level: "warn",
			Message: "this forge has no auto-merge workline can turn on: nothing turned on, no merge request brought up to date"})
		return sum + ": the forge has no auto-merge", nil
	}
	if why := forge.MergeToken(f); why != "" {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-token", Level: "warn",
			Message: why + ": nothing turned on, no merge request brought up to date"})
		return sum + ": nothing turned on, the token's merges would start no pipeline", nil
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
	open, err := f.ListMergeRequests()
	if err != nil {
		return "", err
	}
	if !local {
		if err := fetchVerdicts(o.Repo); err != nil {
			return "", err
		}
	}
	fix := ""
	for _, mr := range open {
		if mr.Base == o.Base && releaseFix(mr.Branch) {
			fix = forge.MergeRef(f, mr.ID)
		}
	}
	var line []*inLine
	acts := map[string]int{}
	for _, mr := range open {
		if mr.Base != o.Base || !slices.ContainsFunc(mr.Labels, accepted) && !mr.AutoMerge {
			continue
		}
		before := len(res.Findings)
		e, err := weigh(f, m, o, local, mr, fix, res)
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
	sort.SliceStable(line, func(i, j int) bool {
		a, b := line[i], line[j]
		switch {
		case a.first != b.first:
			return a.first
		case a.at != b.at:
			return a.at != "" && (b.at == "" || later(b.at, a.at))
		}
		return a.mr.ID < b.mr.ID
	})
	var order []string
	for _, e := range line {
		order = append(order, e.where)
	}
	if len(order) == 0 {
		sum += fmt.Sprintf(": none in line into %s", o.Base)
	} else {
		sum += fmt.Sprintf(": in line into %s, %s", o.Base, strings.Join(order, ", "))
	}
	if n := acts["auto-merge-on"]; n > 0 {
		sum += fmt.Sprintf("; auto-merge turned on for %d", n)
	}
	if n := acts["merged-at-once"]; n > 0 {
		sum += fmt.Sprintf("; %d merged at once", n)
	}
	for _, e := range line {
		if !e.behind {
			sum += fmt.Sprintf("; first in line %s, up to date: it waits for its checks", e.where)
			return sum, nil
		}
		if !e.mr.Here {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "fork-behind", Level: "warn", Where: e.where,
				Message: fmt.Sprintf("behind %s, from a fork: never brought up to date, workline may not write to a fork; its author updates it", o.Base)})
			continue
		}
		finding, err := bringUp(f, o, local, e.mr)
		if err != nil {
			return "", err
		}
		finding.Where = e.where
		res.Findings = append(res.Findings, finding)
		if finding.Rule == "updated" {
			return sum + fmt.Sprintf("; first in line %s, brought up to date", e.where), nil
		}
	}
	return sum, nil
}

// weigh reads one merge request into the base, accepted or its auto-merge
// on, does what the policy says of it — auto-merge on or off, a yes taken
// back — and says it; it returns it when it is in line.
func weigh(f forge.Forge, m forge.Merger, o FollowOptions, local bool, mr forge.MergeRequest, fix string, res *Result) (*inLine, error) {
	where := forge.MergeRef(f, mr.ID)
	say := func(rule, level, msg string) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: rule, Level: level, Where: where, Message: msg})
	}
	st, err := m.MergeState(mr)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) {
			return nil, err
		}
		say("merge-state-unread", "warn", "its state could not be read, left as it is: "+err.Error())
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
	e := &inLine{mr: mr, where: where, first: releaseFix(mr.Branch), behind: !isAncestor(o.Repo, baseTip, tip)}
	off := func() error {
		if !st.AutoMerge {
			return nil
		}
		_, err := m.AutoMerge(mr, tip, false)
		return err
	}
	if !slices.ContainsFunc(mr.Labels, accepted) {
		// Auto-merge on, no yes. Turned on while the label was on, the
		// label taken off since: it goes off, the veto by the label. By a
		// person: it joins the line. By another bot (a dependency bot's
		// own auto-merge): left to it.
		if taken := takenOff(st.Labels); taken != "" && st.AutoMergeAt != "" && !later(st.AutoMergeAt, taken) {
			if err := off(); err != nil {
				return nil, err
			}
			say("auto-merge-off", "", "the label `workline:accepted` was taken off since auto-merge was turned on: auto-merge turned off")
			return nil, nil
		}
		if st.AutoMergeBot || st.Draft {
			return nil, nil
		}
		e.at = st.AutoMergeAt
		return e, nil
	}
	yes, ignored := theYes(st.Labels)
	if yes == nil {
		who := "a hand the forge does not name"
		if ignored != nil && ignored.Author != "" {
			who = "@" + ignored.Author
		}
		say("label-ignored", "warn", fmt.Sprintf("labelled `workline:accepted` by %s, who cannot write to the repository: no yes, nothing turned on", who))
		return nil, nil
	}
	by := "@" + yes.Author
	if st.Draft {
		// The veto by Draft: auto-merge on goes off, and stays off once
		// it is ready until the label is put again.
		if st.AutoMerge {
			if err := off(); err != nil {
				return nil, err
			}
			say("draft", "", fmt.Sprintf("accepted by %s, then marked a draft: auto-merge turned off, until the label is put again", by))
			return nil, nil
		}
		say("draft", "", fmt.Sprintf("accepted by %s, but a draft: nothing turned on until it is marked ready", by))
		return nil, nil
	}
	pushed, err := pushedAfter(o.Repo, st, yes.Created, baseTip, tip)
	if err != nil {
		return nil, err
	}
	if pushed != "" {
		if err := off(); err != nil {
			return nil, err
		}
		var labels []string
		for _, l := range mr.Labels {
			if accepted(l) {
				labels = append(labels, l)
			}
		}
		t := forge.Target{Kind: "merge-request", ID: mr.ID}
		if err := f.Label(t, nil, labels); err != nil {
			return nil, err
		}
		body := fmt.Sprintf("**workline took `%s` off**: %s after %s accepted this merge request, on %s. A yes covers the head it was given on, so its auto-merge is off. Put the label again to accept the head as it is now (%s).",
			strings.Join(labels, "`, `"), pushed, by, yes.Created, short(tip))
		if err := f.Comment(t, body, forge.Marker("merge-reset="+tip)); err != nil {
			return nil, err
		}
		say("accepted-reset", "warn", fmt.Sprintf("%s after %s accepted it: label and auto-merge taken off, said in a comment", pushed, by))
		return nil, nil
	}
	if !st.AutoMerge && st.AutoMergeAt != "" && later(st.AutoMergeAt, yes.Created) {
		say("vetoed", "", fmt.Sprintf("auto-merge was turned on after %s's yes, and off since: left off until the label is put again", by))
		return nil, nil
	}
	e.at = yes.Created
	v := readVerdict(o.Repo, tip)
	switch {
	case v == nil:
		if !st.AutoMerge {
			say("verdict-missing", "", fmt.Sprintf("accepted by %s; no verdict of the line on its head (%s) yet: auto-merge waits for it", by, short(tip)))
		}
		return e, nil
	case v.held() && fix != "":
		say("waits-for-docs", "", fmt.Sprintf("accepted by %s; the release waits for its docs, %s, to merge first", by, fix))
		return nil, nil
	case v.held() && e.behind:
		return e, nil // its docs merged since: brought up to date, judged again
	case v.Status != verdict.Pass || v.short() != "":
		rule, why, then := "verdict-red", v.red(), "auto-merge stays off"
		if v.Status == verdict.Pass {
			rule, why = "review-short", v.short()
		}
		if st.AutoMerge && st.AutoMergeBot && !later(yes.Created, st.AutoMergeAt) {
			// On since the yes, by workline's token: the yes no longer
			// holds for this head.
			if err := off(); err != nil {
				return nil, err
			}
			then = "auto-merge turned off"
		}
		say(rule, "warn", fmt.Sprintf("accepted by %s, but %s; %s, out of line", by, why, then))
		return nil, nil
	case st.Checks == "failure":
		say("checks-red", "warn", fmt.Sprintf("accepted by %s, but a check failed on its head (%s): out of line", by, short(tip)))
		return nil, nil
	case st.AutoMerge:
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

// bringUp brings the first in line up to date with the base: a role's own
// branch rebuilt on it (ADR-0034), another's merged into as the forge's
// Update branch does, workline's own lines settled (ADR-0041). It says
// `updated`, or why it was left, out of line.
func bringUp(f forge.Forge, o FollowOptions, local bool, mr forge.MergeRequest) (verdict.Finding, error) {
	var finding verdict.Finding
	var err error
	if role, task, ok := strings.Cut(strings.TrimPrefix(mr.Branch, "workline/"), "/"); strings.HasPrefix(mr.Branch, "workline/") && ok {
		if finding, err = follow(o.Repo, o.Base, mr.Branch, role, local, task != "release"); err != nil {
			return finding, err
		}
		switch finding.Rule {
		case "rebuilt":
			finding.Rule = "updated"
		case "closed":
			if err := f.CloseMergeRequest(mr.Branch, "Closed by workline's "+role+": "+finding.Message+".", forge.Marker("closed")); err != nil {
				return finding, err
			}
		}
	} else if finding, err = settle(o.Repo, o.Base, mr, local, o.IsDoc, true); err != nil {
		return finding, err
	} else if finding.Rule == "merged" {
		finding.Rule = "updated"
	}
	switch finding.Rule {
	case "updated":
		finding.Message = "first in line, brought up to date: " + finding.Message + "; it merges once its checks pass again"
	case "":
		finding = verdict.Finding{Rule: "updated", Message: "first in line, up to date already"}
	default:
		finding.Rule, finding.Level = "update-left", "warn"
		finding.Message += "; out of line: the next one is brought up to date"
	}
	return finding, nil
}
