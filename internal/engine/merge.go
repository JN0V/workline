package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/developer"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/steer"
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

// StepStatus is a step of the line, and how it ended; Findings, those of
// its findings that block, when it did not pass.
type StepStatus struct {
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Findings []verdict.Finding `json:"findings,omitempty"`
}

// Blocking is the findings of fs that block: no level, or block.
func Blocking(fs []verdict.Finding) []verdict.Finding {
	var out []verdict.Finding
	for _, f := range fs {
		if f.Level == "" || f.Level == "block" {
			out = append(out, f)
		}
	}
	return out
}

// verdictMarker marks the comment saying the line's verdict on a merge
// request's head, edited at each head.
const verdictMarker = "line-verdict"

// Said at most, a step's findings and a finding's message, in runes.
const (
	saidFindings = 10
	saidMessage  = 600
)

// SayVerdict says the line's verdict on its head on merge request id,
// where a person reads it (#1327): blocked, each step that blocked, its
// findings, each saying what lifts it, and what a person can do. Kept in
// one comment edited at each head; a pass is said there only once a block
// was, so a merge request the line passed carries no comment of it.
func SayVerdict(f forge.Forge, id int, v HeadVerdict) error {
	t := forge.Target{Kind: "merge-request", ID: id}
	marker := forge.Marker(verdictMarker)
	if v.Status == verdict.Pass {
		return f.Sticky(t, fmt.Sprintf("**The line passed** on its head (`%s`): no step blocks it.", short(v.Head)), marker, false)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**The line blocks this head** (`%s`): it does not merge, a person's yes neither, until the line passes on a new head.\n", short(v.Head))
	// Each step that did not pass, its findings: a passing step's
	// findings with no level (a doc suspect) block nothing.
	said := false
	for _, s := range v.Steps {
		if s.Status == verdict.Pass {
			continue
		}
		said = true
		fmt.Fprintf(&b, "\n- **%s**: %s", s.Name, s.Status)
		sayFindings(&b, s.Findings)
	}
	if !said {
		fmt.Fprintf(&b, "\n- %s", v.red())
	}
	fmt.Fprintf(&b, "\n\n**What a person can do**: mend what each finding says in a new commit, then run the line again on it "+
		"(`workline route merge-request --target merge-request:%d`); a finding that does not fit is answered where its message says "+
		"(an objection on the tester's line, a reply to the reviewer).", id)
	return f.Sticky(t, b.String(), marker, true)
}

// sayFindings adds fs to b, a bullet each, saidFindings at most.
func sayFindings(b *strings.Builder, fs []verdict.Finding) {
	for i, fd := range fs {
		if i == saidFindings {
			fmt.Fprintf(b, "\n  - and %d more, in the line's output", len(fs)-i)
			return
		}
		where := ""
		if fd.Where != "" {
			where = " on `" + fd.Where + "`"
		}
		fmt.Fprintf(b, "\n  - `%s`%s: %s", fd.Rule, where, clipSaid(fd.Message))
	}
}

// clipSaid is s on one line, cut to saidMessage runes.
func clipSaid(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > saidMessage {
		return string(r[:saidMessage]) + "…"
	}
	return s
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
// does not: the first of shortReview it holds, so a review not whole is
// named alone only when no agent was missing nor failed.
func (v *HeadVerdict) short() string {
	for _, rule := range shortReview {
		for _, f := range v.Findings {
			if f.Rule == rule {
				return fmt.Sprintf("%s(%s): %s", shortSaid, f.Rule, firstSentence(f.Message))
			}
		}
	}
	return ""
}

// shortRules are the rules of shortReview the verdict holds, in that
// order: kept with the verdict, so a person's reading takes a review not
// whole only when it is the one short rule (#1107).
func (v *HeadVerdict) shortRules() []string {
	var rules []string
	for _, rule := range shortReview {
		if slices.ContainsFunc(v.Findings, func(f verdict.Finding) bool { return f.Rule == rule }) {
			rules = append(rules, rule)
		}
	}
	return rules
}

// shortSaid opens why a review falls short.
const shortSaid = "its review falls short "

// keptWhy is why as the forge keeps it, the short rules first between
// brackets, "[lens-failed,review-not-whole] its review falls short…":
// first, so a description cut to the forge's length still holds them.
// readRules reads them back; none from a verdict an earlier workline
// kept, whose text named its first short rule only.
func keptWhy(rules []string, why string) string {
	if len(rules) == 0 {
		return why
	}
	return "[" + strings.Join(rules, ",") + "] " + why
}

func readRules(kept string) (rules []string, why string) {
	if !strings.HasPrefix(kept, "[") {
		return nil, kept
	}
	list, why, ok := strings.Cut(kept[1:], "] ")
	if !ok || list == "" {
		return nil, kept
	}
	return strings.Split(list, ","), why
}

// notWhole is the review too large for the reviewer's parts, or cut by
// its tokens: the one short review a person's own reading takes (#1107).
const notWhole = "review-not-whole"

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

// kept is a head's verdict in hand: its class and why, and the rules
// that made its review short, none when unknown.
type kept struct {
	class, why string
	rules      []string
}

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
	return keepHead(m, v, hand)
}

// KeepVerdict keeps the line's verdict on its head where m keeps it: the
// local forge's line keeps its own as it runs, no file between (#1322).
func KeepVerdict(m forge.Merger, v HeadVerdict) (verdict.Finding, error) {
	return keepHead(m, v, map[string]kept{})
}

func keepHead(m forge.Merger, v HeadVerdict, hand map[string]kept) (verdict.Finding, error) {
	class, why := v.class()
	var rules []string
	if class == verdictShort {
		rules = v.shortRules()
	}
	if err := m.KeepVerdict(v.Head, class, keptWhy(rules, why)); err != nil {
		return verdict.Finding{}, err
	}
	hand[v.Head] = kept{class, why, rules}
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
	rules, why := readRules(why)
	return kept{class, why, rules}, err
}

// accepted says whether a label is a person's yes, `workline:accepted`
// as GitHub spells it, `workline::accepted` as GitLab's scoped labels;
// `workline:accepted-whole` is one too.
func accepted(label string) bool {
	return plainAccepted(label) || acceptedWhole(label)
}

// The label of a person's yes who also read the change whole themselves:
// it takes a review not whole (#1107). Apart from `workline:accepted`, so
// a saved filter putting that one in bulk never takes a review short.
const (
	labelAcceptedWhole  = "workline:accepted-whole"
	scopedAcceptedWhole = "workline::accepted-whole"
	acceptedWholeSays   = "A person's yes to merge, who read the pull request whole: it takes a review not whole"
)

// acceptedWholeLabel is the label as the forge's list holds it: scoped on
// GitLab, as `workline::accepted`, whose place it takes there.
func acceptedWholeLabel(f any) string {
	if forge.ScopedLabels(f) {
		return scopedAcceptedWhole
	}
	return labelAcceptedWhole
}

// plainAccepted is `workline:accepted` alone, either spelling.
func plainAccepted(label string) bool {
	return backlog.Accepted(forge.Issue{Labels: []string{label}})
}

func acceptedWhole(label string) bool {
	return label == labelAcceptedWhole || label == scopedAcceptedWhole
}

// theYes is the last time one of the labels is told by `is` was put by
// someone who may push and merge — GitHub's write access, GitLab's
// Developer —, of those on now; nil when none is on now by such a hand —
// put by another, or taken off since —; ignored, who put one then. Each
// label goes by its own events: one taken off leaves another on.
func theYes(events []forge.LabelEvent, is func(string) bool) (yes, ignored *forge.LabelEvent) {
	on, by := map[string]int{}, map[string]int{}
	for i, e := range events {
		switch {
		case !is(e.Label):
		case !e.Added:
			delete(on, e.Label)
			delete(by, e.Label)
		case e.Writer && !e.Bot && e.Author != "":
			on[e.Label] = i
			delete(by, e.Label)
		default:
			if _, ok := on[e.Label]; !ok {
				by[e.Label] = i
			}
		}
	}
	last := func(at map[string]int) *forge.LabelEvent {
		i := -1
		for _, j := range at {
			i = max(i, j)
		}
		if i < 0 {
			return nil
		}
		return &events[i]
	}
	if yes = last(on); yes != nil {
		return yes, nil
	}
	return nil, last(by)
}

// takenOff is when a yes label was last taken off, and which, "" when
// none was, or one was put again since.
func takenOff(events []forge.LabelEvent) (at, label string) {
	for _, e := range events {
		if accepted(e.Label) {
			at, label = map[bool]string{true: "", false: e.Created}[e.Added], e.Label
		}
	}
	return at, label
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
	return !steer.Agent(lower)
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

// pushedAfterAt says what was pushed to a merge request after the yes given
// at yes, "" nothing but workline's own: the yes covers the head it was
// given on, as approvals reset on a push (ADR-0042). The head it was given
// on is the last pushed before it, as the forge says; a forge that does
// not say leaves the commits' own dates, past base. workline's own is
// checked by its tree, never taken on its trailer: `follow`'s update
// (OwnMerge, ownRebuild), or a commit changing only the lines workline
// writes in the docs (ownRecord), as a merge of the base settles them.
// It gives the commit that was pushed too ("" when none is one commit).
func pushedAfterAt(o FollowOptions, branch string, st forge.MergeState, yes, baseTip, tip string) (string, string, error) {
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
			return "", "", nil
		}
		if _, err := git(repo, nil, "cat-file", "-e", covered+"^{commit}"); err != nil {
			git(repo, nil, "fetch", "-q", "origin", covered) // gone from the branch, maybe still on the forge
		}
		if !isAncestor(repo, covered, tip) {
			if strings.HasPrefix(branch, "workline/") && ownRebuild(repo, covered, tip, baseTip) {
				return "", "", nil
			}
			return fmt.Sprintf("its branch was rewritten (%s, over %s)", short(tip), short(covered)), "", nil
		}
		// The base's commits a merge of it brings are no push to judge.
		out, err := git(repo, nil, "rev-list", "--reverse", covered+".."+tip, "^"+baseTip)
		if err != nil {
			return "", "", err
		}
		for _, c := range strings.Fields(out) {
			if !own(c) {
				return commitSaid(repo, c) + " was pushed", c, nil
			}
		}
		return "", "", nil
	}
	out, err := git(repo, nil, "log", "--reverse", "--format=%H%x1f%cI", baseTip+".."+tip)
	if err != nil {
		return "", "", err
	}
	for _, l := range strings.Split(out, "\n") {
		c, at, ok := strings.Cut(l, "\x1f")
		if ok && later(at, yes) && !own(c) {
			return commitSaid(repo, c) + " was committed", c, nil
		}
	}
	return "", "", nil
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
	// up brings it up to date with the base when it conflicts, or, clean
	// asked, when it merges clean (upToDate): acted, not merged now.
	up func(clean bool, due string) (bool, error)
}

// upToDate is e.up, none on a request weighed without it.
func (e *inLine) upToDate(clean bool, due string) (bool, error) {
	if e == nil || e.up == nil {
		return false, nil
	}
	return e.up(clean, due)
}

// mergePolicy runs the project's merge policy on the merge requests into
// o.Base, says what it did of each, and sums it up for the run's summary.
func mergePolicy(f forge.Forge, o FollowOptions, local bool, res *Result, hand map[string]kept) (string, error) {
	sum := "merge policy (" + o.Merge + ")"
	m, ok := f.(forge.Merger)
	if !ok {
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
	// The labels in the forge's list, for a person to put: the yes, in
	// bulk, from a saved filter (ADR-0038); the yes of who read it whole,
	// one by one (#1107).
	if b, ok := f.(forge.Backlog); ok {
		for _, l := range [][3]string{
			{backlog.AcceptedLabel(f), "0e8a16", backlog.AcceptedSays},
			{acceptedWholeLabel(f), "0b5d1e", acceptedWholeSays},
			{backlog.LabelSizeLarge, backlog.SizeColor, backlog.TierSays(backlog.TierLarge)},
			{backlog.LabelSizeXLarge, backlog.SizeColor, backlog.TierSays(backlog.TierXLarge)},
		} {
			err := b.EnsureLabel(l[0], l[1], l[2])
			switch {
			case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
				return "", err
			case err != nil:
				res.Findings = append(res.Findings, verdict.Finding{Rule: "label-not-made", Level: "warn",
					Message: fmt.Sprintf("the label %s could not be made in the forge's list, for a person to put: %v", l[0], err)})
			}
		}
	}
	// At `normal`: the App workline's own work comes from, the pages agents
	// read, the kinds the project lets merge alone besides the routine
	// ones, those waiting for a person's yes again since a revert — said
	// where the level is set.
	var n *normal
	if byPolicy(o.Merge) {
		tip, err := remoteTip(o.Repo, o.Base)
		if local {
			tip, err = git(o.Repo, nil, "rev-parse", "--verify", "refs/heads/"+o.Base)
		}
		if err != nil {
			return "", err
		}
		n = &normal{alone: o.Alone, local: local}
		if n.app, n.committer, err = forge.App(f); err != nil {
			if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
				return "", err
			}
			n.app = ""
		}
		if n.app == "" {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "merge-app-unknown", Level: "warn",
				Message: "at `" + o.Merge + "`, the token is no App's — a person's, or the forge does not say whose —: nothing is taken for workline's own work, every merge request waits for a person's yes, as at `cautious`"})
		} else {
			if n.steers, err = steering(o.Repo, tip); err != nil {
				return "", err
			}
			if n.demoted, err = demotions(f, o, tip, n.app, local, res); err != nil {
				return "", err
			}
		}
		sum = "merge policy (" + o.Merge + "; " + alone(n) + ")"
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
			switch {
			case byPolicy(o.Merge) && o.Release != nil && o.Release(mr.Branch):
				res.Findings = append(res.Findings, verdict.Finding{Rule: "waits-for-a-yes", Where: forge.MergeRef(f, mr.ID),
					Message: "the release tool's: merging it is the decision to release, never made with no person's yes; the label `workline:accepted`"})
			case !mr.Here:
				res.Findings = append(res.Findings, verdict.Finding{Rule: "waits-for-a-yes", Where: forge.MergeRef(f, mr.ID),
					Message: "from a fork: never merged without a person's yes; the label `workline:accepted`"})
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
	e.up = func(clean bool, due string) (bool, error) {
		return upToDate(f, o, local, mr, baseTip, tip, say, clean, due)
	}
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
	yes, ignored := theYes(st.Labels, accepted)
	// The yes of a person who also read the change whole themselves, and
	// the plain one: each covers the head it was given on.
	whole, wholeIgnored := theYes(st.Labels, acceptedWhole)
	plain, _ := theYes(st.Labels, plainAccepted)
	switch {
	case !labelled:
		// Auto-merge on, no label. Turned on while the label was on, the
		// label taken off since: it goes off, the veto by the label. By a
		// person: their own yes, weighed as the label's. By another bot (a
		// dependency bot's own auto-merge): left to it.
		if taken, label := takenOff(st.Labels); taken != "" && st.AutoMergeAt != "" && !later(st.AutoMergeAt, taken) {
			if err := takeBack("the label was taken off"); err != nil {
				return nil, err
			}
			say("auto-merge-off", "", fmt.Sprintf("the label `%s` was taken off since auto-merge was turned on: auto-merge turned off", label))
			return nil, nil
		}
		if st.AutoMergeBot || st.AutoMergeBy == "" {
			if routine {
				return weighRoutine(f, m, o, w, n, hand)
			}
			return nil, nil
		}
		yes = &forge.LabelEvent{Writer: true, Note: forge.Note{Author: st.AutoMergeBy, Created: st.AutoMergeAt}}
		plain, whole = yes, nil
	case yes == nil:
		label, who := mr.Labels[slices.IndexFunc(mr.Labels, accepted)], "a hand the forge does not name"
		if ignored != nil {
			label = ignored.Label
			if ignored.Author != "" {
				who = "@" + ignored.Author
			}
		}
		if routine {
			say("label-ignored", "warn", fmt.Sprintf("labelled `%s` by %s, who may not push nor merge here: no yes, weighed as workline's own", label, who))
			return weighRoutine(f, m, o, w, n, hand)
		}
		say("label-ignored", "warn", fmt.Sprintf("labelled `%s` by %s, who may not push nor merge here: no yes, nothing turned on", label, who))
		return nil, nil
	}
	if whole == nil && wholeIgnored != nil {
		who := "a hand the forge does not name"
		if wholeIgnored.Author != "" {
			who = "@" + wholeIgnored.Author
		}
		say("label-ignored", "warn", fmt.Sprintf("labelled `%s` by %s, who may not push nor merge here: no reading, only %s's yes counts", wholeIgnored.Label, who, "@"+yes.Author))
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
	// Each yes covers the head it was given on, weighed on its own: a
	// stale one is taken off alone while another covers the head (#1107).
	var stale []*forge.LabelEvent
	pushed, pushedC := "", ""
	for _, g := range []*forge.LabelEvent{plain, whole} {
		if g == nil {
			continue
		}
		p, pc, err := pushedAfterAt(o, mr.Branch, st, g.Created, baseTip, tip)
		if err != nil {
			return nil, err
		}
		if p != "" {
			stale, pushed, pushedC = append(stale, g), p, pc
		}
	}
	t := forge.Target{Kind: "merge-request", ID: mr.ID}
	if len(stale) == 1 && plain != nil && whole != nil {
		old, fresh := stale[0], whole
		if old == whole {
			fresh, whole = plain, nil
		}
		var labels []string
		for _, l := range mr.Labels {
			if accepted(l) && acceptedWhole(l) == acceptedWhole(old.Label) {
				labels = append(labels, l)
			}
		}
		if err := f.Label(t, nil, labels); err != nil {
			return nil, err
		}
		body := fmt.Sprintf("**workline took `%s` off**: %s after @%s put it, on %s. A yes covers the head it was given on. `%s`, put by @%s since, covers the head as it is now (%s).",
			old.Label, pushed, old.Author, old.Created, fresh.Label, fresh.Author, short(tip))
		if err := f.Comment(t, body, forge.Marker("merge-reset="+tip+"/"+old.Label)); err != nil {
			return nil, err
		}
		say("accepted-reset", "warn", fmt.Sprintf("%s after @%s put `%s`: that label taken off, said in a comment; `%s`, put since, covers its head", pushed, old.Author, old.Label, fresh.Label))
		yes, by = fresh, "@"+fresh.Author
	} else if len(stale) > 0 {
		if err := takeBack(pushed + " after the yes: put the label again"); err != nil {
			return nil, err
		}
		var labels []string
		for _, l := range mr.Labels {
			if accepted(l) {
				labels = append(labels, l)
			}
		}
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
		body += settlementDiff(o.Repo, pushedC)
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
	if !mr.Here {
		// A fork's run is the fork's to write: no verdict of it is kept,
		// none read. The person's yes on its head, and its checks all
		// green — its own judge's among them —, are what it goes by (#1043).
		// Its run is the workflow its head holds: one that changes what
		// those checks run, or what steers agents, could turn them green by
		// itself, so the line never arms it; a person merges it by hand.
		if files := forkGuarded(o.Repo, baseTip, tip); files != "" {
			why := "from a fork, it changes what its own checks run or what steers agents (" + files + "): the line never merges it, a person may by hand"
			if st.AutoMerge && st.AutoMergeBot {
				if err := off(); err != nil {
					return nil, err
				}
			}
			if err := m.Accept(tip, false, why); err != nil {
				return nil, err
			}
			say("fork-gate-files", "warn", fmt.Sprintf("accepted by %s, but %s", by, why))
			return nil, nil
		}
		if st.Checks != "success" {
			say("fork-checks-pending", "", fmt.Sprintf("accepted by %s; from a fork, it waits for every check on its head (%s) to pass: no verdict of a fork's run is kept", by, short(tip)))
			return e, nil
		}
		say("fork-yes", "", fmt.Sprintf("from a fork: the line kept no verdict of its run and read no review of it; %s's yes on its head carries it, its checks all green", by))
		return armed(f, m, mr, tip, st, e, say, by, "its checks passed")
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
	case "held":
		// A verdict kept before ADR-0051, a release held for its docs: the
		// release no longer waits for them, so it reads as a pass, below.
	case "":
		say("verdict-missing", "", fmt.Sprintf("accepted by %s; no verdict of the line on its head (%s) yet: it waits for it", by, short(tip)))
		return e, nil
	case verdictWaits: // marked ready, the run on ready not done yet
		say("review-to-come", "", fmt.Sprintf("accepted by %s; its head's (%s) verdict is the draft's, %s: it waits for the run on ready", by, short(tip), v.why))
		return e, nil
	case verdictRed, verdictShort:
		notRead := v.class == verdictShort && slices.Equal(v.rules, []string{notWhole})
		if notRead && whole != nil {
			// A person read it whole: their reading takes the review the
			// reviewer could not make whole, never a check (#1107).
			read := fmt.Sprintf("read whole by @%s (%s)", whole.Author, notWhole)
			say("review-read-whole", "warn", fmt.Sprintf("its review is not whole, the change too large for the reviewer's parts or its tokens; `%s`: %s, whose reading stands for the reviewer's", whole.Label, read))
			return armed(f, m, mr, tip, st, e, say, by, read)
		}
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
		switch {
		case notRead:
			then += fmt.Sprintf("; a person who read it whole puts `%s`", acceptedWholeLabel(f))
		case whole != nil && v.class == verdictShort && v.rules == nil:
			then += fmt.Sprintf("; `%s` takes a review whose one short rule is %s, which this verdict, kept by an earlier workline, does not tell: run the line again on its head", whole.Label, notWhole)
		}
		if err := m.Accept(tip, false, v.why); err != nil {
			return nil, err
		}
		say(rule, "warn", fmt.Sprintf("accepted by %s, but %s; %s", by, v.why, then))
		return nil, nil
	}
	return armed(f, m, mr, tip, st, e, say, by, "its head's verdict passed")
}

// forkGuarded names the files a fork's change touches, past base, that
// its own run cannot be trusted with — a gate's or CI's, one that steers
// agents —, the rules read at base, at most three then a count; "" none.
// A change that cannot be read names itself.
func forkGuarded(repo, base, tip string) string {
	out, err := git(repo, nil, "diff", "--name-only", "--no-renames", base+"..."+tip)
	if err != nil {
		return "its change could not be read"
	}
	rule := developer.Guarded(repo, base)
	var hit []string
	for _, p := range strings.Fields(out) {
		if rule(p) != "" {
			hit = append(hit, p)
		}
	}
	if len(hit) > 3 {
		return fmt.Sprintf("%s and %d more", strings.Join(hit[:3], ", "), len(hit)-3)
	}
	return strings.Join(hit, ", ")
}

// armed holds the yes on a head that passed (passed says how), then turns
// auto-merge on, or merges it now when the forge would.
func armed(f forge.Forge, m forge.Merger, mr forge.MergeRequest, tip string, st forge.MergeState, e *inLine, say func(rule, level, msg string), by, passed string) (*inLine, error) {
	if err := m.Accept(tip, true, "accepted by "+by+", "+passed); err != nil {
		return nil, err
	}
	if st.AutoMerge {
		return stalled(f, m, mr, tip, st, e, say, fmt.Sprintf("accepted by %s, %s", by, passed))
	}
	if acted, err := e.upToDate(false, "a person's yes on it"); acted || err != nil {
		return nil, err // in conflict with the base: a merge round, not a merge (ADR-0067)
	}
	merged, err := m.AutoMerge(mr, tip, true)
	switch {
	case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
		return nil, err
	case err != nil:
		say("auto-merge-refused", "warn", fmt.Sprintf("accepted by %s, %s, but the forge refused to turn auto-merge on: %s", by, passed, forge.Said(err)))
		sayRefused(f, mr, tip, "a person's yes on it", err, say)
		return nil, nil
	case merged:
		say("merged-at-once", "", fmt.Sprintf("accepted by %s, %s: merged at once, the forge's rules met already", by, passed))
		sayUnrefused(f, mr, tip, true, say)
		return nil, nil
	}
	say("auto-merge-on", "", fmt.Sprintf("accepted by %s, %s: auto-merge turned on, the forge merges it once its checks pass", by, passed))
	sayUnrefused(f, mr, tip, false, say)
	// Its checks green, the forge not merging it at once: the run that
	// arms it says what holds it, not a later one that may never come (#1417).
	return e, heldByConversations(m, mr, tip, st, say, fmt.Sprintf("accepted by %s, %s", by, passed))
}

// stalled weighs a merge request the policy merges whose auto-merge is on
// already: left to the forge while its rules are not met, merged now once
// the forge says it would merge it. GitHub's auto-merge acts on an event —
// a check ending, a push — and never again once it missed one (#993);
// GitLab's merges once the head's pipeline succeeds, the same now. A merge
// the forge refuses leaves auto-merge on, for the next run.
func stalled(f forge.Forge, m forge.Merger, mr forge.MergeRequest, tip string, st forge.MergeState, e *inLine, say func(rule, level, msg string), who string) (*inLine, error) {
	if !st.Mergeable {
		// Behind its base and held by the forge, its checks green: the
		// base merged in, by the engine or a merge round (ADR-0067).
		if acted, err := e.upToDate(st.Behind && st.Checks == "success", "its auto-merge on since "+st.AutoMergeAt); acted || err != nil {
			return e, err
		}
		return e, heldByConversations(m, mr, tip, st, say, who)
	}
	merged, err := m.AutoMerge(mr, tip, true)
	switch {
	case errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err):
		return nil, err
	case err != nil:
		say("merge-refused", "warn", fmt.Sprintf("%s; auto-merge on since %s, the forge's rules met, but it refused to merge it now: %s; auto-merge left on", who, st.AutoMergeAt, forge.Said(err)))
		sayRefused(f, mr, tip, "its auto-merge on since "+st.AutoMergeAt, err, say)
		return e, nil
	case merged:
		say("merged-at-once", "", fmt.Sprintf("%s; auto-merge on since %s, but the forge had not merged it, its rules met: merged now", who, st.AutoMergeAt))
		sayUnrefused(f, mr, tip, true, say)
		return nil, nil
	}
	return e, nil
}

// heldByConversations says what holds a merge request the forge does not
// merge, its auto-merge on and its checks green, when conversations on its
// lines are not resolved and its base requires each resolved (#1039):
// each one's place and link, in the run's findings and once a head on the
// merge request itself, where a person looks. The reviewer resolves its
// own once the code answers them; a person's, or one a person spoke in,
// stays theirs. A base whose rule cannot be read is said as one that may
// require them; a forge that lists no conversation says nothing.
func heldByConversations(m forge.Merger, mr forge.MergeRequest, tip string, st forge.MergeState, say func(rule, level, msg string), who string) error {
	th, ok := m.(forge.Threaded)
	if !ok || st.Draft || st.Checks != "success" {
		return nil
	}
	base := mr.Base
	if base == "" {
		base = "the base"
	}
	rule := fmt.Sprintf("if %s requires every conversation resolved", base)
	if r, ok := m.(forge.ConversationRules); ok {
		c, err := r.Conversations(mr.Base)
		switch {
		case err == nil && c.Required:
			rule = fmt.Sprintf("%s requires every conversation resolved", base)
		case err == nil && c.Unread == "":
			return nil // they hold nothing
		}
	}
	threads, err := th.Threads(mr.ID)
	if err != nil {
		if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
			return err
		}
		say("conversations-unread", "warn", fmt.Sprintf("%s; auto-merge on, its checks green, but the forge does not merge it, and its conversations on lines could not be read: %s", who, forge.Said(err)))
		return nil
	}
	var open []string
	for _, t := range threads {
		if t.Resolved {
			continue
		}
		place := fmt.Sprintf("`%s:%d`", t.Path, t.Line)
		if t.URL != "" {
			place = fmt.Sprintf("[%s:%d](%s)", t.Path, t.Line, t.URL)
		}
		if len(t.Comments) > 0 && t.Comments[0].Author != "" {
			place += " by " + t.Comments[0].Author
		}
		open = append(open, place)
	}
	if len(open) == 0 {
		return nil
	}
	noun, verb := "conversations", "hold"
	if len(open) == 1 {
		noun, verb = "conversation", "holds"
	}
	held := fmt.Sprintf("%d %s on its lines not resolved", len(open), noun)
	say("held-by-conversations", "warn", fmt.Sprintf("%s; auto-merge on, its checks green, but the forge does not merge it: %s %s it, %s: %s", who, held, verb, rule, strings.Join(open, ", ")))
	f, ok := m.(forge.Forge)
	if !ok {
		return nil
	}
	body := fmt.Sprintf("**Held by %s**: its checks are green and auto-merge is on, but %s. The reviewer resolves its own once the code answers them; a person resolves the rest, or answers in them:\n\n- %s",
		held, rule, strings.Join(open, "\n- "))
	err = f.Comment(forge.Target{Kind: "merge-request", ID: mr.ID}, body, forge.Marker(fmt.Sprintf("held-by-conversations=%s/%d", short(tip), len(open))))
	if errors.Is(err, forge.ErrUnreachable) && !forge.Refused(err) {
		return err
	}
	if err != nil {
		say("held-not-said", "warn", "could not say on it what holds it: "+forge.Said(err))
	}
	return nil
}

// settlementDiff is what a merge commit's author wrote to settle its
// conflicts, as a comment's text: the lines the commit's tree has over the
// tree git's own merge of its two parents gives, file by file; "" when the
// commit is no merge, or git's merge is the commit's tree.
func settlementDiff(repo, commit string) string {
	if commit == "" {
		return ""
	}
	out, err := git(repo, nil, "rev-list", "--parents", "-n", "1", commit)
	parts := strings.Fields(out)
	if err != nil || len(parts) != 3 {
		return ""
	}
	// git's own merge, its conflicts written as markers in the files.
	cmd := exec.Command("git", "-C", repo, "merge-tree", "--write-tree", "-z", "--no-messages", parts[1], parts[2])
	raw, err := cmd.Output()
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return ""
	}
	tree, _, _ := strings.Cut(string(raw), "\x00")
	if tree == "" {
		return ""
	}
	diff, err := git(repo, nil, "diff", "-U0", "--no-color", "--no-renames", "--no-ext-diff", tree, commit)
	if err != nil || diff == "" {
		return ""
	}
	const max = 200
	var b strings.Builder
	n := 0
	header, open := false, false
	file := ""
	closeFence := func() {
		if open {
			b.WriteString("~~~\n")
			open = false
		}
	}
	for _, l := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			closeFence()
			header, file = true, ""
		case header && strings.HasPrefix(l, "+++ "):
			if f := strings.TrimPrefix(l, "+++ "); f != "/dev/null" {
				file = strings.TrimPrefix(f, "b/")
			}
		case strings.HasPrefix(l, "@@"):
			header = false
		case !header && strings.HasPrefix(l, "+"):
			if n++; n > max {
				continue
			}
			if !open {
				fmt.Fprintf(&b, "\n`%s`:\n\n~~~\n", file)
				open = true
			}
			b.WriteString(l[1:] + "\n")
		}
	}
	closeFence()
	if n == 0 {
		return ""
	}
	if n > max {
		fmt.Fprintf(&b, "\n(%d more lines)\n", n-max)
	}
	return "\n\nWhat the agent wrote to settle the conflict, over git's own merge:\n" + b.String()
}
