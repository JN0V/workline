// Package sample is the weekly sample of every role's acts (ADR-0046): one
// in ten of a week's acts per role and kind. The docs the documentalist
// vouched for (ADR-0014, step 4) are read whole against their sources at
// the commit `checked` names, by a judge standing apart from the model that
// vouched (ADR-0005); the other roles' acts are drawn and listed for a
// person. Drawing the sample takes no AI, and reads the forge without
// writing to it; the read is the only call, and the engine checks every
// quote the judge gives. What it found goes to the forge, in a job holding
// the forge's token and no AI key (Apply): one tracking issue, a comment per
// week, and a merge request putting back a `checked` found false.
package sample

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/builtin/documentalist"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// OneIn is the share of the docs vouched for that is read: one in ten,
// rounded up, so a week with any reads at least one.
const OneIn = 10

// Options say what to sample and who reads it.
type Options struct {
	Repo     string
	RolesDir string
	Week     string    // the ISO week sampled, 2026-W40; default the last whole week
	Since    string    // read what was vouched for after this commit, up to HEAD, instead of a week
	Judge    string    // an --ai value; default WORKLINE_JUDGE, then the `sample.judge` setting
	Now      time.Time // when the run is, for the default week
	// Forge is read, never written: the product owner's acts, the
	// merge requests merged and their heads, the reviewer's summaries
	// (ADR-0046). nil: those are not drawn, and the sample says so.
	Forge forge.Forge
}

// Result is what a sample found, and what applying it wrote.
type Result struct {
	Status     string            `json:"status"`
	Summary    string            `json:"summary,omitempty"`
	Findings   []verdict.Finding `json:"findings,omitempty"`
	AgentCalls int               `json:"agent-calls"`
	Calls      []agent.Call      `json:"calls,omitempty"`
	Applied    []string          `json:"applied"`
	Refused    []string          `json:"refused"`
	Week       string            `json:"week"`
	From       string            `json:"from,omitempty"` // the window: commits after From, up to To
	To         string            `json:"to,omitempty"`
	Window     string            `json:"window"` // the window, in words
	Vouched    int               `json:"vouched"`
	Reads      []Read            `json:"reads"`
	// Drawn are the other roles' acts drawn, one in ten per role and
	// kind; Counts, each role's kinds of act in the window and how many
	// were drawn, the docs' too; NotDrawn, what could not be drawn, and
	// why (ADR-0046).
	Drawn    []Act    `json:"drawn,omitempty"`
	Counts   []Count  `json:"counts,omitempty"`
	NotDrawn []string `json:"not-drawn,omitempty"`
	// Older are the product owner's acts read without the record
	// ADR-0046 completes, an engine before it having kept none: on their
	// issue as it reads now. Autonomy, the level its acts suggest
	// (ADR-0033).
	Older       []string  `json:"older,omitempty"`
	Autonomy    *Autonomy `json:"autonomy,omitempty"`
	unreachable bool      // the forge did not answer a read
}

// Read is one doc of the sample, and what its reading found.
type Read struct {
	Doc      string            `json:"doc"`
	Commit   string            `json:"commit"`             // the commit that moved its `checked`
	In       string            `json:"in,omitempty"`       // the merge request it came by, read on its head (a squash)
	Checked  map[string]string `json:"checked"`            // what it moved it to
	Previous map[string]string `json:"previous,omitempty"` // what it was before
	By       []string          `json:"vouched-by,omitempty"`
	// Verdict: true, false (a passage proven false), unproven (passages
	// said false, none proven), unearned (its sources do not fit whole), or
	// not-read.
	Verdict      string  `json:"verdict"`
	Why          string  `json:"why,omitempty"`
	Judge        string  `json:"judge,omitempty"`
	Model        string  `json:"model,omitempty"`
	Independence string  `json:"independence,omitempty"`
	Proofs       []Proof `json:"proofs,omitempty"`
	Unproven     []Proof `json:"unproven,omitempty"`
	Panel        *Panel  `json:"panel,omitempty"` // how a false verdict was settled
}

// Proof is a passage said false, and the source's words saying otherwise.
type Proof struct {
	Passage string `json:"passage"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Quote   string `json:"quote,omitempty"`
	Name    string `json:"name,omitempty"`
	Why     string `json:"why,omitempty"`
}

// candidate is a `checked` the documentalist moved.
type candidate struct {
	doc, commit       string
	checked, previous map[string]string
	by                []string
	in                string // the merge request it came by, read on its head
}

// Draw picks the week's sample: each doc of it read, the other roles'
// acts drawn for a person.
func Draw(o Options) *Result {
	res := &Result{Status: verdict.Pass, Applied: []string{}, Refused: []string{}, Reads: []Read{}}
	if err := draw(o, res); err != nil {
		res.Status, res.Summary = verdict.Block, err.Error()
		res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
	}
	return res
}

func draw(o Options, res *Result) error {
	settings, err := roleSettings(o)
	if err != nil {
		return err
	}
	week, from, to, err := Week(o.Week, o.Now)
	if err != nil {
		return err
	}
	res.Week = week
	base, tip := "", ""
	// The days the product owner's acts are drawn from, [first, end), and
	// since when the merge requests merged are listed.
	since, first, end := from, from.Format(time.DateOnly), to.Format(time.DateOnly)
	if o.Since != "" {
		if base, err = git(o.Repo, "rev-parse", "--verify", o.Since+"^{commit}"); err != nil {
			return fmt.Errorf("--since %s: not a commit", o.Since)
		}
		if tip, err = git(o.Repo, "rev-parse", "HEAD"); err != nil {
			return err
		}
		res.Window = fmt.Sprintf("after %.7s", base)
		var sec int64
		at, _ := git(o.Repo, "log", "-1", "--format=%ct", base)
		fmt.Sscan(at, &sec)
		since = time.Unix(sec, 0).UTC()
		first, end = since.Format(time.DateOnly), ""
	} else {
		if base, tip, err = window(o.Repo, from, to); err != nil {
			return err
		}
		res.Window = fmt.Sprintf("reaching %s from %s to %s", branchName(o.Repo), from.Format("2006-01-02"), to.Add(-time.Second).Format("2006-01-02"))
	}
	res.From, res.To = base, tip
	commits, err := revList(o.Repo, base, tip, "")
	if err != nil {
		return err
	}
	after := ""
	if a := string(settings.Sample.After); a != "" {
		if after, err = git(o.Repo, "rev-parse", "--verify", a+"^{commit}"); err != nil {
			return fmt.Errorf("sample.after %s: not a commit of this clone", a)
		}
		res.Window += fmt.Sprintf(", after %s", a)
	}
	// The merge requests merged in the window, their heads fetched: a
	// `checked` naming a commit a squash left off the branch is read
	// there, and the trailers a squash dropped (ADR-0046). A day early: a
	// merge's moment and its commit's may differ.
	h := newHeads()
	if o.Forge != nil {
		if h, err = mergedIn(o.Repo, o.Forge, since.AddDate(0, 0, -1), base, tip, after); err != nil {
			res.NotDrawn = append(res.NotDrawn, fmt.Sprintf("the merge requests merged in the window: the forge did not answer: %v", err))
			res.unreachable = true
		}
	}
	docCommits, err := revList(o.Repo, base, tip, after)
	if err != nil {
		return err
	}
	cands, err := vouched(o.Repo, settings, append(docCommits, h.own...), h)
	if err != nil {
		return err
	}
	res.Vouched = len(cands)
	var picked []candidate
	if len(cands) == 0 {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-empty", Level: "warn",
			Message: fmt.Sprintf("no doc vouched for by the documentalist in the commits %s: nothing to read", res.Window)})
	} else {
		picked = Pick(week, cands)
		res.Counts = append(res.Counts, Count{Role: docsRole, Kind: "checked", Acts: len(cands), Drawn: len(picked)})
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-window", Level: "warn",
			Message: fmt.Sprintf("%s vouched for by the documentalist in the commits %s; %d read, one in ten", plural(len(cands), "doc"), res.Window, len(picked))})
	}
	// The other roles' acts (ADR-0046).
	acts := actsOfTheWindow(o, res, h, commits, first, end)
	drawn, counts := drawEach(week, acts)
	res.Drawn, res.Counts = drawn, append(res.Counts, counts...)
	// Each read, each role's first before any second, until the week's
	// tokens are spent.
	spec := o.Judge
	if spec == "" {
		spec = os.Getenv("WORKLINE_JUDGE")
	}
	if spec == "" {
		spec = settings.Sample.Judge
	}
	rd := &reader{o: o, settings: settings, spec: spec, res: res, cap: settings.Sample.TokensCap()}
	if len(picked) > 0 || len(drawn) > 0 {
		if rd.auditor, err = role.Load(o.RolesDir, "auditor"); err != nil {
			return err
		}
	}
	res.Reads = make([]Read, len(picked))
	queue := map[string][]func(){}
	for i := range picked {
		queue[docsRole] = append(queue[docsRole], func() { res.Reads[i] = rd.readOne(picked[i]) })
	}
	for i := range res.Drawn {
		a := &res.Drawn[i]
		queue[a.Role] = append(queue[a.Role], func() { rd.readAct(a) })
	}
	for round, more := 0, true; more; round++ {
		more = false
		for _, r := range roleOrder {
			if round < len(queue[r]) {
				queue[r][round]()
				more = true
			}
		}
	}
	for _, r := range res.Reads {
		res.Findings = append(res.Findings, findings(r)...)
	}
	for _, c := range counts {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-acts", Level: "warn",
			Message: fmt.Sprintf("%s, %s, by %s %s; %d drawn, one in ten", plural(c.Acts, "act"), kindWords(c), roleWords(c.Role), actsWindow(c.Role, res.Window, first, end), c.Drawn)})
	}
	for _, a := range res.Drawn {
		res.Findings = append(res.Findings, actFinding(a))
		if a.older && a.Verdict != "not-read" {
			res.Older = append(res.Older, a.Ref)
		}
	}
	if len(res.Older) > 0 {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-record-older", Level: "warn", Message: olderWords(res.Older)})
	}
	res.Findings = append(res.Findings, autonomyFinding(res.Autonomy)...)
	for _, why := range res.NotDrawn {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "sample-not-drawn", Level: "warn", Message: "not drawn: " + why})
	}
	res.Status = verdict.Pass
	tally, acted := map[string]int{}, map[string]int{}
	weigh := func(v, why string) {
		switch {
		case v == "not-read" && strings.Contains(why, "could not answer"):
			if res.Status == verdict.Pass {
				res.Status = verdict.BlockedExternal
			}
		case v != "true":
			res.Status = verdict.Human
		}
	}
	for _, r := range res.Reads {
		tally[r.Verdict]++
		weigh(r.Verdict, r.Why)
	}
	for _, a := range res.Drawn {
		acted[a.Verdict]++
		weigh(a.Verdict, a.Why)
	}
	if res.unreachable && res.Status == verdict.Pass {
		res.Status = verdict.BlockedExternal
	}
	res.Summary = fmt.Sprintf("%s: no doc vouched for", week)
	if len(cands) > 0 {
		var parts []string
		for _, v := range []string{"true", "false", "unearned", "unproven", "not-read"} {
			if tally[v] > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", tally[v], v))
			}
		}
		res.Summary = fmt.Sprintf("%s: %d of %d docs vouched for read: %s", week, len(res.Reads), len(cands), strings.Join(parts, ", "))
	}
	if len(drawn) > 0 {
		roles := map[string]bool{}
		for _, a := range drawn {
			roles[a.Role] = true
		}
		var parts []string
		for _, v := range []string{"true", "false", "unproven", "not-read"} {
			if acted[v] > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", acted[v], v))
			}
		}
		res.Summary += fmt.Sprintf("; %s of %s read: %s", plural(len(drawn), "act"), plural(len(roles), "other role"), strings.Join(parts, ", "))
	}
	return nil
}

// revList is the commits of base..tip, newest first, merges left out,
// none an ancestor of after; none with no tip.
func revList(repo, base, tip, after string) ([]string, error) {
	if tip == "" || tip == base {
		return nil, nil
	}
	args := []string{"rev-list", "--no-merges", tip}
	for _, not := range []string{base, after} {
		if not != "" {
			args = append(args, "^"+not)
		}
	}
	list, err := git(repo, args...)
	return strings.Fields(list), err
}

// actsWindow says where a role's acts were drawn from.
func actsWindow(role, window, first, end string) string {
	switch {
	case role == reviewRole:
		return "on the merge requests merged in the commits " + window
	case role == commitRole:
		return "in the commits " + window
	case end == "":
		return "since " + first
	}
	last, _ := time.Parse(time.DateOnly, end)
	return fmt.Sprintf("from %s to %s", first, last.AddDate(0, 0, -1).Format(time.DateOnly))
}

// roleSettings are the documentalist's settings, the project's over the role's.
func roleSettings(o Options) (documentalist.Settings, error) {
	r, err := role.Load(o.RolesDir, "documentalist")
	if err != nil {
		return documentalist.Settings{}, err
	}
	cfg, err := role.LoadProjectConfig(o.Repo)
	if err != nil {
		return documentalist.Settings{}, err
	}
	return documentalist.SettingsFrom(r.MergedSettings(cfg))
}

// Week reads an ISO week (2026-W40) and the instants it starts and ends, in
// UTC; with none, the last whole week before now.
func Week(week string, now time.Time) (string, time.Time, time.Time, error) {
	if week == "" {
		y, w := now.UTC().AddDate(0, 0, -7).ISOWeek()
		week = fmt.Sprintf("%d-W%02d", y, w)
	}
	var y, w int
	if _, err := fmt.Sscanf(week, "%d-W%d", &y, &w); err != nil || w < 1 || w > 53 {
		return "", time.Time{}, time.Time{}, fmt.Errorf("week %q: expected an ISO week, like 2026-W40", week)
	}
	jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.UTC)
	monday := jan4.AddDate(0, 0, -((int(jan4.Weekday())+6)%7)+7*(w-1))
	if gy, gw := monday.ISOWeek(); gy != y || gw != w {
		return "", time.Time{}, time.Time{}, fmt.Errorf("week %q: %d has no week %d", week, y, w)
	}
	return fmt.Sprintf("%d-W%02d", y, w), monday, monday.AddDate(0, 0, 7), nil
}

// window is the range of the commits that reached the branch in [from, to),
// by the dates of its first-parent history: base..tip, base empty from the
// start; tip empty when nothing reached it before to.
func window(repo string, from, to time.Time) (base, tip string, err error) {
	out, err := git(repo, "log", "--first-parent", "--format=%H %ct", "HEAD")
	if err != nil {
		return "", "", err
	}
	for _, l := range strings.Split(out, "\n") {
		sha, ts, ok := strings.Cut(l, " ")
		if !ok {
			continue
		}
		var sec int64
		fmt.Sscan(ts, &sec)
		at := time.Unix(sec, 0)
		if tip == "" && at.Before(to) {
			tip = sha
		}
		if at.Before(from) {
			return sha, tip, nil
		}
	}
	return "", tip, nil
}

// vouched lists the `checked` the documentalist moved in the commits
// given — the window's, newest first, then the heads' of the merge
// requests merged in it —, one per doc and value: a commit carrying its
// Workline-Role (its own, or squashed into another's message), or setting
// `verified: agent:documentalist` on the doc. A squash drops its commits'
// trailers, or mixes them, in the commit a merge request became: the
// commit of its head that moved the same `checked` says whose the move
// was, and who vouched (ADR-0046); the act stays the squash's, its earlier
// `checked` the one the branch had.
func vouched(repo string, s documentalist.Settings, commits []string, h *heads) ([]candidate, error) {
	var out []candidate
	seen := map[string]bool{}
	for _, c := range commits {
		bot, by, err := trailers(repo, c)
		if err != nil {
			return nil, err
		}
		files, err := git(repo, "diff-tree", "--root", "--no-commit-id", "-r", "--name-only", c)
		if err != nil {
			return nil, err
		}
		for _, f := range strings.Split(files, "\n") {
			if f == "" || !s.IsDoc(f) {
				continue
			}
			m, ok := moved(repo, c, f)
			if !ok {
				continue
			}
			key := f + "\x00" + fmt.Sprint(m.checked)
			if seen[key] {
				continue // a later move to the same commit stands for it
			}
			isBot, who, mine := bot, by, bot || m.verified
			if heads := h.of[c]; len(heads) > 0 {
				// A squash's message and header mix its commits' acts: the
				// head's commit that moved this `checked` says whose it is.
				if isBot, who, err = movedOnHead(repo, heads, f, m.checked); err != nil {
					return nil, err
				}
				mine = isBot
			}
			if !mine {
				continue
			}
			seen[key] = true
			out = append(out, candidate{doc: f, commit: c, checked: m.checked, previous: m.previous, by: who, in: h.in[c]})
		}
	}
	return out, nil
}

// trailers say whether a commit's message carries the documentalist's
// Workline-Role, and the models its Workline-Model names.
func trailers(repo, c string) (bool, []string, error) {
	msg, err := git(repo, "log", "-1", "--format=%B", c)
	if err != nil {
		return false, nil, err
	}
	bot, by := false, []string(nil)
	for _, l := range strings.Split(msg, "\n") {
		l = strings.TrimSpace(l)
		if v, ok := strings.CutPrefix(l, "Workline-Role:"); ok && strings.TrimSpace(v) == "documentalist" {
			bot = true
		}
		if v, ok := strings.CutPrefix(l, "Workline-Model:"); ok && strings.TrimSpace(v) != "" {
			by = append(by, strings.TrimSpace(v))
		}
	}
	return bot, by, nil
}

// move is a doc's `checked` a commit moved.
type move struct {
	checked, previous map[string]string
	verified          bool // `verified: agent:documentalist` newly set
}

// moved is the move of f's `checked` in commit c, if c moved it.
func moved(repo, c, f string) (move, bool) {
	after, err := git(repo, "show", c+":"+f)
	if err != nil {
		return move{}, false // removed
	}
	d, _ := documentalist.ParseDoc(f, []byte(after))
	if d == nil || len(d.Checked) == 0 {
		return move{}, false
	}
	before, _ := git(repo, "show", c+"^:"+f)
	prev, _ := documentalist.ParseDoc(f, []byte(before))
	var previous map[string]string
	if prev != nil {
		previous = prev.Checked
	}
	if fmt.Sprint(previous) == fmt.Sprint(d.Checked) {
		return move{}, false
	}
	return move{checked: d.Checked, previous: previous, verified: verifiedByAgent(after) && !verifiedByAgent(before)}, true
}

// movedOnHead reads, on a squashed merge request's head, the newest commit
// that moved f's `checked` to the value given: whether it was the
// documentalist's — its Workline-Role, or `verified: agent:documentalist`
// newly set — and the models it names. None, or a person's: not the
// documentalist's.
func movedOnHead(repo string, heads []string, f string, checked map[string]string) (bool, []string, error) {
	for _, c := range heads {
		m, ok := moved(repo, c, f)
		if !ok || fmt.Sprint(m.checked) != fmt.Sprint(checked) {
			continue
		}
		bot, by, err := trailers(repo, c)
		if err != nil || !(bot || m.verified) {
			return false, nil, err
		}
		return true, by, nil
	}
	return false, nil, nil
}

// verifiedByAgent: the doc's header says the documentalist vouched for it.
func verifiedByAgent(content string) bool {
	meta, _ := documentalist.Header(content)
	for _, l := range strings.Split(meta, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "verified:"); ok && strings.TrimSpace(v) == "agent:documentalist" {
			return true
		}
	}
	return false
}

// Pick draws one in OneIn of the candidates, rounded up, the same each time
// for the same week: each is ranked by a digest of the week, the doc and the
// commit, and the first are taken.
func Pick(week string, cands []candidate) []candidate {
	keys := make([]string, len(cands))
	for i, c := range cands {
		keys[i] = c.doc + "\x00" + c.commit
	}
	var picked []candidate
	for _, i := range pick(week, keys) {
		picked = append(picked, cands[i])
	}
	sort.Slice(picked, func(i, j int) bool { return picked[i].doc < picked[j].doc })
	return picked
}

// pick draws one in OneIn of the keys, rounded up, ranked by a digest of the
// week and each key: the places of those drawn, in the keys' order.
func pick(week string, keys []string) []int {
	n := (len(keys) + OneIn - 1) / OneIn
	rank := func(i int) string {
		h := sha256.Sum256([]byte(week + "\x00" + keys[i]))
		return hex.EncodeToString(h[:])
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	// Equal keys rank equal: the first in the keys' order first, so a
	// rerun draws the same.
	sort.SliceStable(order, func(a, b int) bool { return rank(order[a]) < rank(order[b]) })
	picked := order[:n]
	sort.Ints(picked)
	return picked
}

// readOne has a doc read, whole, against its sources at the commit its
// `checked` names, and checks every quote of the answer.
func (rd *reader) readOne(c candidate) Read {
	repo, s, auditor, spec := rd.o.Repo, rd.settings, rd.auditor, rd.spec
	r := Read{Doc: c.doc, Commit: short(repo, c.commit), In: c.in, Checked: c.checked, Previous: c.previous, By: c.by}
	content, err := git(repo, "show", c.commit+":"+c.doc)
	d, _ := documentalist.ParseDoc(c.doc, []byte(content))
	if err != nil || d == nil {
		r.Verdict, r.Why = "not-read", "the doc cannot be read as it was vouched for, for a person"
		return r
	}
	evidence, fits, err := documentalist.SourcesAt(repo, d, s.WholeCap())
	switch {
	case err != nil:
		r.Verdict, r.Why = "not-read", err.Error()+missingWords(rd.o.Forge, err)+"; for a person"
		return r
	case !fits:
		r.Verdict = "unearned"
		r.Why = fmt.Sprintf("its sources at the commit `checked` names take more than the %d characters a doc's sources may take to be read whole (whole-chars): that `checked` could not have been earned (ADR-0014, step 0)", s.WholeCap())
		return r
	}
	if spec == "" || spec == "none" {
		r.Verdict, r.Why = "not-read", "no judge is set (--judge, WORKLINE_JUDGE, or the `sample.judge` setting): for a person to read against its sources"
		return r
	}
	question, err := rd.facet("checked")
	if err != nil {
		r.Verdict, r.Why = "not-read", err.Error()+"; for a person"
		return r
	}
	voucher, models := "", []string(nil)
	for _, b := range c.by {
		if voucher == "" {
			voucher = b
		}
		_, m, _ := strings.Cut(b, ":")
		models = append(models, m)
	}
	// Never the model that vouched: set to it, the judge is the other one
	// (ADR-0005, the best independence available).
	if alias, ok := strings.CutPrefix(spec, "claude:"); ok && judge.Provider(voucher) == "claude" {
		alias, _, _ = strings.Cut(alias, "@")
		for _, m := range models {
			if alias != "" && strings.Contains(m, alias) {
				spec = judge.Pick("claude", m)
				r.Why = fmt.Sprintf("the judge set is the model that vouched (%s): read by %s instead", voucher, spec)
			}
		}
	}
	if why := rd.over(); why != "" {
		r.Verdict, r.Why = "not-read", why
		return r
	}
	r.Judge = spec
	answer, call, err := ask(spec, auditor, "## The question\n\n"+question+"\n\n"+task(c, d, content, evidence))
	rd.called(call)
	r.Model = call.Model
	r.Independence = "unknown (no Workline-Model on the commit that vouched)"
	level := ""
	if voucher != "" {
		level = judge.Independence(spec, call.Model, voucher, models)
		r.Independence = fmt.Sprintf("%s (%s → %s)", level, strings.Join(c.by, ", "), orUnknown(call.Agent, call.Model))
	}
	switch {
	case errors.Is(err, agent.ErrUnavailable):
		r.Verdict, r.Why = "not-read", "the judge could not answer: "+err.Error()+"; for a person"
		return r
	case err != nil:
		r.Verdict, r.Why = "not-read", "the judge's answer could not be read: "+err.Error()+"; for a person"
		return r
	case level == "context":
		r.Verdict, r.Why = "not-read", fmt.Sprintf("the judge answered with the model that vouched (%s): no independent read; for a person", call.Model)
		return r
	}
	if floor := s.Sample.AtLeast; floor != "" {
		if _, ok := judge.Levels[floor]; !ok {
			r.Verdict, r.Why = "not-read", fmt.Sprintf("sample.at-least %q is not provider, model or context", floor)
			return r
		}
		if judge.Levels[level] < judge.Levels[floor] {
			r.Verdict, r.Why = "not-read", fmt.Sprintf("the judge stands below sample.at-least (%s): %s; for a person", floor, r.Independence)
			return r
		}
	}
	check(repo, d, content, answer, &r)
	if r.Verdict == "false" {
		r.Panel = rd.settleByPanel(c.doc+"\x00"+c.commit, "checked", true, r.Proofs, task(c, d, content, evidence), &r.Verdict, &r.Why)
	}
	return r
}

// missingWords say, for a `checked` naming a commit the clone does not
// have, where else the sample looked for it, or how it would: a squash
// leaves it on its merge request's head (ADR-0046).
func missingWords(f forge.Forge, err error) string {
	if !errors.Is(err, documentalist.ErrNotInClone) {
		return ""
	}
	if f == nil {
		return ": a squash may have left it off the branch; with a forge (`--forge`, or the project's `forge`), the sample fetches it from the head of its merge request"
	}
	if _, ok := f.(forge.Merged); ok && !forge.KeepsBranches(f) {
		return ", nor the heads of the merge requests merged in the window"
	}
	return ""
}

// claimed is a claim of the auditor's answer: the catalogue's shape.
type claimed struct {
	Quote  string `yaml:"quote"`
	Status string `yaml:"status"`
	Name   string `yaml:"name"`
	Why    string `yaml:"why"`
	Source *struct {
		Path  string `yaml:"path"`
		Quote string `yaml:"quote"`
	} `yaml:"source"`
}

// check reads the answer's claims against the doc and its sources: a
// passage the doc holds, and a source's words found outside a comment at the
// commit `checked` names, or a name the code no longer had then.
func check(repo string, d *documentalist.Doc, content string, answer []intent.Intention, r *Read) {
	notes, claims := 0, 0
	for _, in := range answer {
		switch in.Kind {
		case "note":
			notes++
			continue
		case "claim":
		default:
			continue
		}
		claims++
		data, _ := yaml.Marshal(in.Value)
		var c claimed
		yaml.Unmarshal(data, &c)
		p := Proof{Passage: strings.Join(strings.Fields(c.Quote), " "), Why: c.Why}
		why := ""
		switch {
		case p.Passage == "":
			why = "the claim quotes no passage of the doc"
		case !strings.Contains(strings.Join(strings.Fields(content), " "), p.Passage):
			why = fmt.Sprintf("%q is not in the doc", p.Passage)
		case c.Status == "contradicted" && (c.Source == nil || c.Source.Path == "" || strings.TrimSpace(c.Source.Quote) == ""):
			why = "a `contradicted` claim gives its source's `path` and `quote`"
		case c.Status == "contradicted":
			p.Path, p.Quote = c.Source.Path, strings.Join(strings.Fields(c.Source.Quote), " ")
			src, at, ok := documentalist.SourceFileAt(repo, d, c.Source.Path)
			if !ok {
				why = fmt.Sprintf("%s is not a file under the doc's sources (%s) at the commit `checked` names", c.Source.Path, strings.Join(d.Sources, ", "))
				break
			}
			found, code, line := documentalist.QuoteIn(c.Source.Path, src, c.Source.Quote)
			p.Line = line
			switch {
			case !found:
				why = fmt.Sprintf("%q is not in %s at %.7s", p.Quote, c.Source.Path, at)
			case !code:
				why = fmt.Sprintf("%q is a comment (%s:%d), not the code: a comment is not evidence", p.Quote, c.Source.Path, line)
			}
		case c.Status == "gone":
			p.Name = strings.Trim(c.Name, "`() ")
			if p.Name == "" || !strings.Contains(p.Passage, p.Name) {
				why = "a `gone` claim names, in `name`, a name of the passage the code no longer has"
				break
			}
			rev := d.Checked[""]
			if rev == "" {
				why = "a `gone` claim is checked against this repository's code only"
				break
			}
			if in, err := documentalist.InCodeAt(repo, rev, p.Name); err != nil || in {
				why = fmt.Sprintf("`%s` is in the code at %s", p.Name, rev)
			}
		default:
			why = fmt.Sprintf("status %q proves nothing: a claim is `contradicted`, with a source, or `gone`, with a name", c.Status)
		}
		if why != "" {
			p.Why = why
			r.Unproven = append(r.Unproven, p)
		} else {
			r.Proofs = append(r.Proofs, p)
		}
	}
	switch {
	case len(r.Proofs) > 0:
		r.Verdict = "false"
	case claims > 0:
		r.Verdict = "unproven"
	case notes == 0:
		r.Verdict, r.Why = "not-read", "the judge answered nothing; for a person"
	default:
		r.Verdict = "true"
	}
}

// task is what the judge is given: the doc whole as it was vouched for, and
// its sources whole at the commit its `checked` names.
func task(c candidate, d *documentalist.Doc, content string, evidence []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## The doc: %s, as vouched for in %.7s (`checked`: %s)\n\n", c.doc, c.commit, checkedWords(c.checked))
	fmt.Fprintf(&b, "```markdown\n%s\n```\n\n## Its sources (%s), whole\n\n", strings.TrimRight(content, "\n"), strings.Join(d.Sources, ", "))
	for _, e := range evidence {
		b.WriteString(e + "\n\n")
	}
	return b.String()
}

// ask puts the task to the judge as the auditor, in a run folder of its own.
func ask(spec string, auditor *role.Role, taskText string) ([]intent.Intention, agent.Call, error) {
	ag, err := agent.Parse(spec)
	if err != nil {
		return nil, agent.Call{}, err
	}
	if ag == nil {
		return nil, agent.Call{}, errors.New("no agent to judge with")
	}
	dir, err := os.MkdirTemp("", "workline-sample-")
	if err != nil {
		return nil, agent.Call{}, err
	}
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "in"), 0o755)
	os.MkdirAll(filepath.Join(dir, "out"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "in", "task.md"), []byte(taskText), 0o644); err != nil {
		return nil, agent.Call{}, err
	}
	call, err := ag.Propose(agent.Request{RunDir: dir, Repo: dir, Role: auditor})
	call.Task = "sample"
	if err != nil {
		return nil, call, err
	}
	in, err := intent.Read(filepath.Join(dir, "out", "intentions.yaml"))
	return in, call, err
}

// findings say each read, at its doc.
func findings(r Read) []verdict.Finding {
	at := fmt.Sprintf("vouched for in %s at `checked: %s`", r.where(), checkedWords(r.Checked))
	who := ""
	if r.Judge != "" {
		who = fmt.Sprintf("; read by %s, independence: %s", r.Judge, r.Independence)
	}
	switch r.Verdict {
	case "true":
		return []verdict.Finding{{Rule: "vouched-true", Where: r.Doc, Level: "warn", Message: "nothing false found, read whole against its sources " + at + who}}
	case "false":
		var lines []string
		for _, p := range r.Proofs {
			lines = append(lines, p.String())
		}
		return []verdict.Finding{{Rule: "vouched-false", Where: r.Doc, Message: "a false `checked`, " + at + ": " + strings.Join(lines, "; ") + who}}
	case "unproven":
		var out []verdict.Finding
		for _, p := range r.Unproven {
			out = append(out, verdict.Finding{Rule: "vouched-unproven", Where: r.Doc,
				Message: fmt.Sprintf("said false, not proven: %q — %s; for a person (%s%s)", p.Passage, p.Why, at, who)})
		}
		return out
	case "unearned":
		return []verdict.Finding{{Rule: "vouched-unearned", Where: r.Doc, Message: r.Why + "; " + at}}
	}
	return []verdict.Finding{{Rule: "sample-not-read", Where: r.Doc, Message: r.Why + "; " + at}}
}

// where is the commit that vouched, and the merge request it came by
// when a squash left its commits on its head.
func (r Read) where() string {
	if r.In != "" {
		return r.Commit + " (" + r.In + ")"
	}
	return r.Commit
}

// String is a proof as the findings and the issue say it.
func (p Proof) String() string {
	if p.Name != "" {
		return fmt.Sprintf("%q: `%s` is not in the code", p.Passage, p.Name)
	}
	at := p.Path
	if p.Line > 0 {
		at += fmt.Sprintf(":%d", p.Line)
	}
	if p.Passage == "" {
		return fmt.Sprintf("%s says %q", at, p.Quote)
	}
	return fmt.Sprintf("%q: %s says %q", p.Passage, at, p.Quote)
}

func checkedWords(c map[string]string) string {
	if len(c) == 1 && c[""] != "" {
		return c[""]
	}
	var parts []string
	for k, v := range c {
		if k == "" {
			k = "this repository"
		}
		parts = append(parts, k+": "+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func orUnknown(agentName, model string) string {
	if model == "" {
		return agentName
	}
	return agentName + ":" + model
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func short(repo, sha string) string {
	if s, err := git(repo, "rev-parse", "--short", sha); err == nil {
		return s
	}
	return sha
}

func branchName(repo string) string {
	if b, err := git(repo, "symbolic-ref", "--short", "HEAD"); err == nil {
		return b
	}
	return "HEAD"
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.quotePath=off"}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(exit.Stderr)))
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
