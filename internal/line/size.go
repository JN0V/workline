package line

// The normal size of an issue, measured per project (ADR-0064, step 2):
// each step's fresh tokens counted in the line's record on its issue
// (Record.Spent), and, each night, the normals derived from the records
// of the issues done, kept in one comment on one issue. Reported only:
// nothing stops or escalates on them yet (ADR-0064, steps 3 to 5).

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/verdict"
)

// SizeTitle is the issue the normals are kept on: one, open, its comment
// written again in place (as the auditor's count, ADR-0046).
const SizeTitle = "workline: the normal size of an issue"

const (
	sizeKey  = "line/size"    // the sticky comment's key
	sizeMark = "size-normals" // the normals' line in it, for the steps after to read
	// sizeWindow: the latest closed issues with the line's record read,
	// a call each; sizePages: the pages of closed issues read at most to
	// find them; sizeMeasured: the issues done of a kind, and the runs of
	// a role, before their own percentile replaces the seed.
	sizeWindow   = 100
	sizePages    = 10
	sizeMeasured = 10
	// issueSeed: an issue's normal until then, the worst measured round
	// (385k a bug) rounded up; doubled at the highest effort.
	issueSeed = 400000
)

// effortHigh is the highest effort level, which doubles a seed, read
// once the project's scheme is in use.
func effortHigh() int { return len(scale.Effort.Names) }

const sizeIntro = `workline's line keeps here the normal size of an issue in this project (ADR-0064): the fresh tokens its runs spent, measured from the line's record on each issue done. Its comment is written again each night, in place. Nothing stops on it yet: it is reported only.

Kept up to date by the line, not by hand.`

// spend adds to spent the fresh tokens of each of steps that called an
// agent, by its role: a run with no call adds none.
func spend(spent map[string][]int, steps []Step) map[string][]int {
	for _, s := range steps {
		if s.Result == nil || len(s.Result.Calls) == 0 {
			continue
		}
		n := 0
		for _, c := range s.Result.Calls {
			n += c.Fresh()
		}
		if n == 0 {
			continue // a call not made, or that said no tokens: nothing to count
		}
		spent = maps.Clone(spent)
		if spent == nil {
			spent = map[string][]int{}
		}
		spent[s.Name] = append(slices.Clone(spent[s.Name]), n)
	}
	return spent
}

// Fresh is the fresh tokens every run counted in the record spent.
func (r Record) Fresh() int {
	n := 0
	for _, runs := range r.Spent {
		for _, t := range runs {
			n += t
		}
	}
	return n
}

// Tokens says a count of tokens short: 950, 10.5k, 400k, 1.6M.
func Tokens(n int) string {
	trim := func(f float64, unit string) string {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0") + unit
	}
	switch {
	case n < 1000:
		return fmt.Sprint(n)
	case n < 1000000:
		return trim(float64(n)/1000, "k")
	}
	return trim(float64(n)/1000000, "M")
}

// Normal is one normal: Tokens, the 85th percentile of N values when
// Measured, else the seed; 0 and not measured, no seed (a role with no
// ai-max-tokens).
type Normal struct {
	Tokens   int  `json:"tokens"`
	Measured bool `json:"measured,omitempty"`
	N        int  `json:"n,omitempty"`
}

func (n Normal) said() string {
	switch {
	case n.Measured:
		return Tokens(n.Tokens) + " (p85)"
	case n.Tokens > 0:
		return Tokens(n.Tokens) + " (seed)"
	}
	return "none (no ai-max-tokens)"
}

// KindNormals are a kind's normals: Done, its issues done with runs
// counted; an issue's, and a run's by role.
type KindNormals struct {
	Kind  string            `json:"kind"`
	Done  int               `json:"done"`
	Issue Normal            `json:"issue"`
	Runs  map[string]Normal `json:"runs"`
}

// Past is an open issue its runs took past its kind's normal: its total,
// or, with Role, one run of that role past the role's normal a run.
type Past struct {
	Issue  int    `json:"issue"`
	Kind   string `json:"kind"`
	Role   string `json:"role,omitempty"`
	Spent  int    `json:"spent"`
	Normal int    `json:"normal"`
}

// Sizes is what `workline size` measured.
type Sizes struct {
	Kinds []KindNormals `json:"kinds"`
	Past  []Past        `json:"past,omitempty"`
}

// SizeOptions: where `workline size` reads, and whether it writes.
type SizeOptions struct {
	Repo, RolesDir, Forge string
	NoApply               bool
}

// noKind names the issues no kind label tells.
const noKind = "(no kind)"

// sized is an issue read with the line's record on it.
type sized struct {
	is  forge.Issue
	rec Record
}

// Size measures the project's normals from the line's records on its
// issues done, says the open issues past theirs, and keeps both in one
// comment on the issue SizeTitle names, unless NoApply. A project with no
// line of states, or no record read, gets nothing written.
func Size(o SizeOptions) (*engine.Result, *Sizes) {
	res := &engine.Result{Status: verdict.Pass}
	fail := func(err error) (*engine.Result, *Sizes) {
		res.Status, res.Summary = verdict.Block, err.Error()
		if errors.Is(err, forge.ErrUnreachable) {
			res.Status = verdict.BlockedExternal
		}
		res.Findings = append(res.Findings, verdict.Finding{Rule: "engine-error", Message: err.Error()})
		return res, nil
	}
	cfg, err := routing.Load(o.Repo)
	if err != nil {
		return fail(err)
	}
	if len(cfg.States) == 0 {
		res.Summary = "no line of states here (routing.states): no record to measure, nothing written"
		return res, nil
	}
	f, _, err := openForge(engine.Options{Repo: o.Repo, Forge: o.Forge})
	if err != nil {
		return fail(err)
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return fail(errors.New("the forge lists no issues: no record to measure"))
	}
	recordOf := func(is forge.Issue) (Record, bool, error) {
		comments, err := b.Comments(forge.Target{Kind: "issue", ID: is.ID})
		if err != nil {
			return Record{}, false, err
		}
		rec, had := recordIn(comments)
		return rec, had, nil
	}
	doneRead, pages, err := doneRecords(closedPages(b), recordOf, func(is forge.Issue) (bool, error) { return doneByMR(f, b, is) })
	if err != nil {
		return fail(err)
	}
	if pages >= sizePages {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "size-window-short", Level: "info",
			Message: fmt.Sprintf("%d pages of closed issues read, fewer than %d with the line's record among them: the normals stand on those found", pages, sizeWindow)})
	}
	openIssues, err := b.Issues()
	if err != nil {
		return fail(err)
	}
	var open []forge.Issue
	for _, is := range openIssues {
		if _, ok := cfg.StateOf(is.Labels); ok && !backlog.Tracking(is, "") {
			open = append(open, is)
		}
	}
	var openRead []sized
	for _, is := range open {
		rec, had, err := recordOf(is)
		if err != nil {
			return fail(err)
		}
		if had && rec.Fresh() > 0 {
			openRead = append(openRead, sized{is, rec})
		}
	}
	if len(doneRead) == 0 && len(openRead) == 0 {
		res.Summary = "no issue holds the line's count of tokens yet: nothing to measure, nothing written"
		return res, nil
	}
	if note := effortScheme(o.Repo, b, open); note != nil {
		res.Findings = append(res.Findings, *note)
	}
	high := effortHigh()

	roles := map[string]bool{}
	for _, s := range cfg.States {
		if s.Role != "" {
			roles[s.Role] = true
		}
	}
	for _, r := range append(slices.Clone(doneRead), openRead...) {
		for name := range r.rec.Spent {
			roles[name] = true
		}
	}
	seeds := runSeeds(o, slices.Sorted(maps.Keys(roles)))

	kinds := slices.Clone(backlog.IssueKinds)
	kindOf := func(is forge.Issue) string {
		if k, _ := backlog.KindOf(is.Labels); k != "" {
			return k
		}
		return noKind
	}
	for _, r := range append(slices.Clone(doneRead), openRead...) {
		if k := kindOf(r.is); !slices.Contains(kinds, k) {
			kinds = append(kinds, k)
		}
	}
	sizes := &Sizes{}
	byKind := map[string]KindNormals{}
	for _, k := range kinds {
		var totals []int
		runs := map[string][]int{}
		for _, r := range doneRead {
			if kindOf(r.is) != k {
				continue
			}
			totals = append(totals, r.rec.Fresh())
			for name, t := range r.rec.Spent {
				runs[name] = append(runs[name], t...)
			}
		}
		kn := KindNormals{Kind: k, Done: len(totals), Issue: Normal{Tokens: issueSeed}, Runs: map[string]Normal{}}
		if len(totals) >= sizeMeasured {
			kn.Issue = Normal{Tokens: p85(totals), Measured: true, N: len(totals)}
		}
		for name := range roles {
			kn.Runs[name] = Normal{Tokens: seeds[name]}
			if len(totals) >= sizeMeasured && len(runs[name]) >= sizeMeasured {
				kn.Runs[name] = Normal{Tokens: p85(runs[name]), Measured: true, N: len(runs[name])}
			}
		}
		byKind[k] = kn
		sizes.Kinds = append(sizes.Kinds, kn)
	}
	// effort scales the seed alone, an issue's and a run's (ADR-0064, §1)
	limit := func(n Normal, is forge.Issue) int {
		if !n.Measured && backlog.Effort(is) == high {
			return 2 * n.Tokens
		}
		return n.Tokens
	}
	for _, r := range openRead {
		k := kindOf(r.is)
		var found []Past
		if l := limit(byKind[k].Issue, r.is); r.rec.Fresh() > l {
			found = append(found, Past{Issue: r.is.ID, Kind: k, Spent: r.rec.Fresh(), Normal: l})
		}
		for _, name := range slices.Sorted(maps.Keys(r.rec.Spent)) {
			l := limit(byKind[k].Runs[name], r.is)
			if top := slices.Max(r.rec.Spent[name]); l > 0 && top > l {
				found = append(found, Past{Issue: r.is.ID, Kind: k, Role: name, Spent: top, Normal: l})
			}
		}
		for _, p := range found {
			res.Findings = append(res.Findings, verdict.Finding{Rule: "size-past-normal", Level: "info", Where: fmt.Sprintf("#%d", p.Issue),
				Message: p.said() + "; reported only, nothing stops on it yet"})
		}
		sizes.Past = append(sizes.Past, found...)
	}
	sort.SliceStable(sizes.Past, func(i, j int) bool { return sizes.Past[i].Issue < sizes.Past[j].Issue })

	res.Summary = fmt.Sprintf("normals of %d issues done", len(doneRead))
	past := ""
	if n := len(sizes.Past); n > 0 {
		past = fmt.Sprintf("; %d past %s normal on open issues", n, plural(n, "its", "their"))
	}
	if o.NoApply {
		res.Summary += past
		return res, sizes
	}
	id, err := f.KeepIssue(SizeTitle, sizeIntro, true)
	if err != nil {
		return fail(err)
	}
	if err := f.Sticky(forge.Target{Kind: "issue", ID: id}, backlog.Inert(sizeComment(sizes, slices.Sorted(maps.Keys(roles)))), forge.Marker("sticky="+sizeKey), true); err != nil {
		return fail(err)
	}
	res.Applied = append(res.Applied, "comment")
	res.Summary += fmt.Sprintf(", kept on issue #%d", id) + past
	return res, sizes
}

// closedPages pages b's closed issues, the last updated first where the
// forge lists them so, else all of them read once, the highest number
// first.
func closedPages(b forge.Backlog) func(int) ([]forge.Issue, bool, error) {
	if p, ok := b.(forge.ClosedPager); ok {
		return p.ClosedPage
	}
	return func(n int) ([]forge.Issue, bool, error) {
		if n > 1 {
			return nil, false, nil
		}
		all, err := b.AllIssues()
		var closed []forge.Issue
		for _, is := range all {
			if is.Closed {
				closed = append(closed, is)
			}
		}
		sort.Slice(closed, func(i, j int) bool { return closed[i].ID > closed[j].ID })
		return closed, false, err
	}
}

// doneRecords reads, the last closed first, the closed issues the line
// touched — those holding its record —, until sizeWindow of them or
// sizePages pages; of those, it keeps the ones done (done) whose runs
// were counted. It says the pages read.
func doneRecords(page func(int) ([]forge.Issue, bool, error), recordOf func(forge.Issue) (Record, bool, error),
	done func(forge.Issue) (bool, error)) ([]sized, int, error) {
	var out []sized
	touched, n := 0, 0
	for more := true; more && touched < sizeWindow && n < sizePages; {
		n++
		var list []forge.Issue
		var err error
		if list, more, err = page(n); err != nil {
			return nil, n, err
		}
		for _, is := range list {
			if touched >= sizeWindow {
				break
			}
			if backlog.Tracking(is, "") || is.Reason == "not_planned" || is.Reason == "duplicate" {
				continue
			}
			rec, had, err := recordOf(is)
			if err != nil {
				return nil, n, err
			}
			if !had {
				continue // the line never touched it
			}
			touched++
			if rec.Fresh() == 0 {
				continue
			}
			ok, err := done(is)
			if err != nil {
				return nil, n, err
			}
			if ok {
				out = append(out, sized{is, rec})
			}
		}
	}
	return out, n, nil
}

// doneByMR says whether a closed issue was done: closed by a merge
// request, as the forge links it. The local forge links none: there,
// closed as completed.
func doneByMR(f forge.Forge, b forge.Backlog, is forge.Issue) (bool, error) {
	if _, local := f.(*forge.Local); local {
		return is.Reason == "" || is.Reason == "completed", nil
	}
	closers, err := b.Closers(is.ID)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(closers, func(c forge.Closer) bool { return c.Kind == "pull-request" }), nil
}

func (p Past) said() string {
	if p.Role != "" {
		return fmt.Sprintf("#%d (%s): a %s run of %s, past its normal of %s", p.Issue, p.Kind, p.Role, Tokens(p.Spent), Tokens(p.Normal))
	}
	return fmt.Sprintf("#%d (%s): %s, past its normal of %s", p.Issue, p.Kind, Tokens(p.Spent), Tokens(p.Normal))
}

// p85 is the 85th percentile of values, by nearest rank.
func p85(values []int) int {
	s := slices.Sorted(slices.Values(values))
	return s[max(0, int(math.Ceil(0.85*float64(len(s))))-1)]
}

// runSeeds is each role's seed a run: its ai-max-tokens, the project's
// over the role's own; 0 for a role with none or not found.
func runSeeds(o SizeOptions, roles []string) map[string]int {
	seeds := map[string]int{}
	pcfg, err := role.LoadProjectConfig(o.Repo)
	if err != nil || o.RolesDir == "" {
		return seeds
	}
	for _, name := range roles {
		r, err := role.Load(o.RolesDir, name)
		if err != nil {
			continue
		}
		switch v := r.MergedSettings(pcfg)["ai-max-tokens"].(type) {
		case int:
			seeds[name] = v
		case float64:
			seeds[name] = int(v)
		}
	}
	return seeds
}

// effortScheme reads where the project keeps effort, as the product owner
// does (ADR-0040): a setting or a family that does not read is said, and
// no issue's effort read.
func effortScheme(repo string, b forge.Backlog, open []forge.Issue) *verdict.Finding {
	said := func(err error) *verdict.Finding {
		return &verdict.Finding{Rule: "size-effort-unread", Level: "info",
			Message: "the issues' effort is not read, a seed not doubled at effort " + scale.Effort.Name(effortHigh()) + ": " + err.Error()}
	}
	c, err := role.LoadProjectConfig(repo)
	if err != nil {
		return said(err)
	}
	var cfg backlog.Config
	settings := c.Roles[backlog.Keeper].Settings
	if cfg.Priority, err = scale.ReadSetting(scale.Priority, settings[scale.Priority.Axis]); err != nil {
		return said(err)
	}
	if cfg.Effort, err = scale.ReadSetting(scale.Effort, settings[scale.Effort.Axis]); err != nil {
		return said(err)
	}
	scheme, _, err := backlog.Where(b, cfg, open)
	if err != nil {
		return said(err)
	}
	scale.Use(scheme)
	return nil
}

// sizeComment is the comment the normals are kept in: a table a person
// reads, the open issues past theirs, then the normals as the engine
// reads them back.
func sizeComment(s *Sizes, roles []string) string {
	var b strings.Builder
	b.WriteString("**workline's line** measured the normal size of an issue here (ADR-0064): the fresh tokens its runs spent, ")
	fmt.Fprintf(&b, "the 85th percentile (p85) of the last %d issues done, per kind; a seed until %d of a kind are done. ", sizeWindow, sizeMeasured)
	b.WriteString("Reported only: nothing stops on it yet.\n\n| Kind | Done | An issue |")
	for _, r := range roles {
		fmt.Fprintf(&b, " %s, a run |", r)
	}
	b.WriteString("\n|---|---|---|" + strings.Repeat("---|", len(roles)) + "\n")
	for _, k := range s.Kinds {
		fmt.Fprintf(&b, "| %s | %d | %s |", k.Kind, k.Done, k.Issue.said())
		for _, r := range roles {
			fmt.Fprintf(&b, " %s |", k.Runs[r].said())
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\nA seed: %s an issue, a run its role's `ai-max-tokens`; doubled for an issue of effort %s.\n\n", Tokens(issueSeed), scale.Effort.Name(effortHigh()))
	b.WriteString("**Open issues past their normal**:")
	if len(s.Past) == 0 {
		b.WriteString(" none.\n")
	} else {
		b.WriteString("\n\n")
		for _, p := range s.Past {
			b.WriteString("- " + p.said() + "\n")
		}
	}
	data, _ := json.Marshal(s)
	b.WriteString("\n<!-- workline:" + sizeMark + " " + string(data) + " -->")
	return b.String()
}
