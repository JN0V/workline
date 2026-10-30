package engine

// How the engine applies a role's acts on the backlog (ADR-0018 and after):
// an issue announced obsolete or kept open, an issue opened through the one
// way, an order, a rename, a split, a link set or taken off, a refine; the
// run's plan of those acts, and what it leaves on each issue.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/board"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/scale"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// announce tells an issue it looks obsolete (ADR-0024): the label first —
// a comment without it reads as kept open, never the other way round —,
// then the comment to its reporter, with the block read back at the next
// runs.
func (a *applier) announce(c backlog.Proposal) error {
	b := a.forge.(forge.Backlog)
	is, err := a.forge.Issue(c.Issue)
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	if err := b.EnsureLabel(backlog.LabelObsolete, "cfd3d7", "Announced obsolete: closed after a delay unless someone writes or takes this label off"); err != nil {
		return err
	}
	if err := a.forge.Label(t, []string{backlog.LabelObsolete}, nil); err != nil {
		return err
	}
	days := backlog.Settings(a.settings)["close-obsolete"].Delay()
	ann := backlog.Announcement{Quote: *c.Quote, Why: c.Why, Commit: head,
		Announced: time.Now().Format("2006-01-02"), By: a.model}
	return a.forge.Comment(t, backlog.Inert(backlog.AnnouncementComment(is.Author, c, a.role, ann, days)), backlog.AnnounceMarker(a.role))
}

// keepOpen settles an issue's announcement as obsolete: kept open, its
// label taken off, its evidence recorded in the state so it is not
// announced again for it, and the issue told why when no person kept it.
func keepOpen(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	t := forge.Target{Kind: "issue", ID: c.Issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	key := backlog.ObsoleteKey(*c.Quote)
	if !slices.Contains(st.Kept, key) {
		st.Kept = append(st.Kept, key)
	}
	if err := f.Label(t, nil, []string{backlog.LabelObsolete}); err != nil {
		return err
	}
	if err := f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false); err != nil {
		return err
	}
	if !c.Say {
		return nil
	}
	return f.Comment(t, backlog.Inert(backlog.KeptComment(c.Why)), backlog.KeptMarker(role))
}

// openings is the run's one way to open an issue (backlog.Openings), made
// on the first one; the role's issues-max caps its findings opened.
func (a *applier) openings() (*backlog.Openings, error) {
	if a.opens == nil {
		max := 3
		switch n := a.settings["issues-max"].(type) {
		case int:
			max = n
		case float64:
			max = int(n)
		}
		o, err := backlog.NewOpenings(a.forge, max)
		if err != nil {
			return nil, err
		}
		a.opens = o
	}
	return a.opens, nil
}

// openIssue opens an issue a role found outside its task, once, through
// the one way every role opens one (ADR-0018). Its subject is keyed by the
// engine, never by the agent: the line of code it quotes (`at`), found
// again here, or else its title.
func (a *applier) openIssue(title, body string, m map[string]any) error {
	o, err := a.openings()
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	found, _ := m["found"].(string) // where the role found it, said in the line the issue ends with
	op := backlog.Opening{Role: a.role, Title: title, Body: body, Key: backlog.TitleKey(title), Sources: strs(m["sources"]), Commit: head, Triage: true, Found: found}
	if at, ok := m["at"].(map[string]any); ok {
		path, _ := at["path"].(string)
		text, _ := at["text"].(string)
		if _, _, original, found := backlog.Locate(a.repo, path, text); found {
			line, _, _ := strings.Cut(original, "\n")
			op.Key = backlog.CodeKey(path, line)
			if !slices.Contains(op.Sources, path) {
				op.Sources = append(op.Sources, path)
			}
		} else {
			// Keyed by its title instead: another role finding the same
			// line would not meet it, so it is said.
			a.findings = append(a.findings, verdict.Finding{Rule: "issue-quote-not-found", Level: "warn", Where: path,
				Message: fmt.Sprintf("%q: the line it quotes is not found in %s; keyed by its title, a duplicate on that line not seen", title, path)})
		}
	}
	// The form a role's key had before every role shared one: its name first.
	op.Also = []string{"issue=" + a.role + "/" + strings.TrimPrefix(op.Key, "issue=")}
	outcome, id, err := o.Open(op)
	if err != nil {
		return err
	}
	switch outcome {
	case backlog.Capped:
		a.capped++
	case backlog.Settled, backlog.FoundAgain:
		a.findings = append(a.findings, verdict.Finding{Rule: "issue-closed", Level: "warn", Where: fmt.Sprintf("#%d", id),
			Message: fmt.Sprintf("%q: #%d, closed, holds this subject; not opened again (%s)", title, id, map[string]string{
				backlog.Settled: "closed as not planned or as a duplicate: nothing written", backlog.FoundAgain: "said once on it that it was found again"}[outcome])})
	}
	return nil
}

// where is where the project keeps each axis, detected once, at the first
// act that writes one, as the plan was decided (ADR-0040).
func (a *applier) where() (scale.Scheme, error) {
	if a.scheme != nil {
		return *a.scheme, nil
	}
	cfg, err := backlog.ReadConfig(a.settings)
	if err != nil {
		return scale.Before, err
	}
	s := scale.Before
	if cfg.Detects() {
		b := a.forge.(forge.Backlog)
		open, err := b.Issues()
		if err != nil {
			return s, err
		}
		if s, _, err = backlog.Where(b, cfg, open); err != nil {
			return s, err
		}
	}
	a.scheme = &s
	return s, nil
}

// splitEffort splits an order carrying an effort in two acts, each with
// its own mode, cap and undo (ADR-0040): the order, if it sets a priority,
// and an effort. The agent proposes one intention, `order: {issue,
// priority, effort, why}`; an effort said by name reads as its level.
func splitEffort(in []intent.Intention) []intent.Intention {
	var out []intent.Intention
	for _, i := range in {
		m, ok := i.Value.(map[string]any)
		e, has := m["effort"]
		if i.Kind != "order" || !ok || !has || e == nil {
			out = append(out, i)
			continue
		}
		level := 0 // none read: refused as no level (effort-level)
		switch v := e.(type) {
		case int:
			level = v
		case float64:
			if v == float64(int(v)) {
				level = int(v) // 1.9 is no level, never cut to 1
			}
		case string:
			if l, ok := scale.Effort.Word(v); ok {
				level = l
			} else {
				level, _ = strconv.Atoi(v)
			}
		}
		eff := map[string]any{"issue": m["issue"], "effort": level, "why": m["why"]}
		for _, k := range []string{"ticked", "own", "reason"} {
			if v, ok := m[k]; ok {
				eff[k] = v
			}
		}
		if m["priority"] != nil {
			order := map[string]any{}
			for k, v := range m {
				if k != "effort" {
					order[k] = v
				}
			}
			out = append(out, intent.Intention{Kind: "order", Value: order})
		}
		out = append(out, intent.Intention{Kind: "effort", Value: eff})
	}
	return out
}

// level sets an issue's priority (order) or effort (effort) where the
// project keeps it, one label of the family, its others taken off, and
// records it in the issue's state (docs/spec/backlog-acts/order.md):
// workline's own label created when missing, the forge's words all at
// once and ranked, a project's labels used as they are. Read again
// first: a value changed since the plan, or no longer placeable, gets
// nothing written, said (ADR-0040).
func (a *applier) level(c backlog.Proposal) error {
	where, err := a.where()
	if err != nil {
		return err
	}
	src, l := where.Priority, c.Priority
	if c.Do == "effort" {
		src, l = where.Effort, c.Effort
	}
	f := a.forge
	b := f.(forge.Backlog)
	is, err := f.Issue(c.Issue)
	if err != nil {
		return err
	}
	axis := src.Family.Scale.Axis
	rule, why := backlog.Placeable(src, *is, l)
	if now := src.Read(backlog.On(*is)); rule == "" && !(len(now) == 0 && c.Had == 0) && !(len(now) == 1 && now[0] == c.Had) {
		rule, why = axis+"-changed", fmt.Sprintf("its %s changed since the run read it: a person's, kept", axis)
	}
	if rule != "" {
		a.findings = append(a.findings, verdict.Finding{Rule: rule, Level: "warn", Where: fmt.Sprintf("#%d", c.Issue),
			Message: why + " (found when applying)"})
		return nil
	}
	label, _ := src.Family.Label(l)
	t := forge.Target{Kind: "issue", ID: c.Issue}
	switch {
	case src.Weight:
		// GitLab's weight: 1, 3 or 5, read back (ADR-0040).
		if err := forge.SetWeight(f, c.Issue, scale.WeightOf(l)); err != nil {
			return err
		}
	case src.InField():
		// The forge's own field, by the option's name (ADR-0040).
		if err := forge.SetField(f, c.Issue, forge.Field{ID: src.Field.ID, Name: src.Field.Name}, label.Name); err != nil {
			return err
		}
	default:
		create := []scale.Label{label}
		if src.Words {
			create = src.Family.Labels
		}
		for _, x := range create {
			if x.Color == "" || a.ensured[x.Name] {
				continue
			}
			if err := forge.EnsureRankedLabel(b, x.Name, x.Color, x.Describe(strings.ReplaceAll(a.role, "-", " ")), x.Rank); err != nil {
				return err
			}
			if a.ensured == nil {
				a.ensured = map[string]bool{}
			}
			a.ensured[x.Name] = true
		}
		if err := f.Label(t, []string{label.Name}, src.Family.Others(l)); err != nil {
			return err
		}
	}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, a.role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	if src.Hidden() {
		l = backlog.HiddenLevel // set by the role, its value never shown on a public issue
	}
	if c.Do == "effort" {
		st.Effort = l
	} else {
		st.Priority = l
	}
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// rename sets an issue's title and records it in its state: a title other
// than the one recorded is a person's (ADR-0022).
func rename(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	title := strings.TrimSpace(c.Title)
	if err := b.SetTitle(c.Issue, title); err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	st.Title = title
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false)
}

// split opens a need's children through the one way, each with its four
// sections, links them to it — a sub-issue where the forge has them, a task
// list in its body elsewhere — and records them in its state, last: a run
// stopped half-way, resumed, finds the children it opened (ADR-0022).
func (a *applier) split(c backlog.Proposal) error {
	b := a.forge.(forge.Backlog)
	o, err := a.openings()
	if err != nil {
		return err
	}
	head, err := git(a.repo, nil, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	if err := b.EnsureLabel(backlog.AcceptedLabel(a.forge), "0e8a16", backlog.AcceptedSays); err != nil {
		return err
	}
	var ids, listed []int
	closed := map[int]bool{} // children closed already: left as they are
	for _, ch := range c.Into {
		body, wrote := backlog.ChildBody(c.Issue, ch, a.role)
		outcome, id, err := o.Open(backlog.Opening{Role: a.role, Key: backlog.SplitKey(c.Issue, ch.Title), Title: strings.TrimSpace(ch.Title),
			Body: body, From: fmt.Sprintf(" from #%d", c.Issue), Sources: ch.Sources, Commit: head, Wrote: wrote})
		if err != nil {
			return err
		}
		if outcome == backlog.Settled {
			a.findings = append(a.findings, verdict.Finding{Rule: "issue-closed", Level: "warn", Where: fmt.Sprintf("#%d", id),
				Message: fmt.Sprintf("%q, a part of #%d: #%d, closed, holds it; not opened again", ch.Title, c.Issue, id)})
			ids = append(ids, id) // still one of its children: the issue is not split again
			closed[id] = true
			continue
		}
		ids = append(ids, id)
		t := forge.Target{Kind: "issue", ID: id}
		if err := a.forge.Label(t, []string{backlog.LabelToRefine, backlog.LabelDraft}, nil); err != nil {
			return err
		}
		native, err := b.AddSubIssue(c.Issue, id)
		if err != nil {
			return err
		}
		if !native {
			listed = append(listed, id)
		}
	}
	// What a child waits on among its siblings (ADR-0028): opened, never
	// read yet, so its state's digest is left; the parent's state keeps
	// them, the role's own links.
	after := map[int][]int{}
	for i, ch := range c.Into {
		var blockers []int
		for _, k := range ch.After {
			blockers = append(blockers, ids[k-1])
		}
		if len(blockers) > 0 && !closed[ids[i]] {
			if err := a.depend(ids[i], blockers, false); err != nil {
				return err
			}
			after[ids[i]] = blockers
		}
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	body := ""
	if len(listed) > 0 {
		is, err := a.forge.Issue(c.Issue)
		if err != nil {
			return err
		}
		if body = backlog.ListChildren(is.Body, listed); body != is.Body {
			if err := b.SetBody(c.Issue, body); err != nil {
				return err
			}
		}
	}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, a.role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	st.Split = ids
	if len(after) > 0 {
		st.After = after
	}
	if body != "" {
		st.Keep(body) // the engine's own change: not read again for it
	}
	return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// undepend takes off links the role set (ADR-0028): those the check found
// in the forge's own relation, and the rest from the engine's line in the
// body, whose new digest the issue's state gets.
func (a *applier) undepend(c backlog.Proposal) error {
	b := a.forge.(forge.Backlog)
	for _, bl := range c.Native {
		if err := b.RemoveBlocker(c.Issue, bl); err != nil {
			return err
		}
	}
	is, err := a.forge.Issue(c.Issue)
	if err != nil {
		return err
	}
	body := backlog.WithoutBlockers(is.Body, c.BlockedBy)
	if body == is.Body {
		return nil
	}
	if err := b.SetBody(c.Issue, body); err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, found, err := backlog.ReadState(comments, a.role)
	if err != nil || !found {
		return err // no state yet: the body is read as it is at its first
	}
	st.Keep(body)
	return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// depend records what an issue waits on (ADR-0028): the forge's own
// relation where it has one, else the engine's line in its body, rewritten
// with them; with digest, the issue's state gets the body's digest, so the
// engine's own change is not read as a person's.
func (a *applier) depend(id int, blockers []int, digest bool) error {
	b := a.forge.(forge.Backlog)
	var inBody []int
	for _, bl := range blockers {
		native, err := b.AddBlocker(id, bl)
		if err != nil {
			return err
		}
		if !native {
			inBody = append(inBody, bl)
		}
	}
	if len(inBody) == 0 {
		return nil
	}
	is, err := a.forge.Issue(id)
	if err != nil {
		return err
	}
	body := backlog.WithBlockers(is.Body, inBody)
	if body == is.Body {
		return nil
	}
	if err := b.SetBody(id, body); err != nil || !digest {
		return err
	}
	t := forge.Target{Kind: "issue", ID: id}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, a.role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", id, err)
	}
	st.Keep(body)
	return a.forge.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(a.role), false)
}

// refining applies a refine, a ready or an ask, against the issue as it is
// now (docs/spec/backlog-acts.md, "Refining to ready").
func refining(f forge.Forge, role string, c backlog.Proposal) error {
	b := f.(forge.Backlog)
	is, err := f.Issue(c.Issue)
	if err != nil {
		return err
	}
	t := forge.Target{Kind: "issue", ID: c.Issue}
	switch c.Do {
	case "ask":
		return f.Comment(t, backlog.Inert(backlog.Ask(is.Author, c.Questions, c.Round)), backlog.AskMarker(role, c.Round))
	case "refine":
		if !c.ToReporter {
			break
		}
		// An outsider's issue: the text proposed to its reporter, nothing
		// written in the body until they or a person of the project agree.
		if err := b.EnsureLabel(backlog.AcceptedLabel(f), "0e8a16", backlog.AcceptedSays); err != nil {
			return err
		}
		if err := f.Label(t, []string{backlog.LabelToRefine}, nil); err != nil {
			return err
		}
		return f.Comment(t, backlog.Inert(backlog.ProposalComment(is.Author, c, role)), backlog.ProposalMarker(role, c.Round))
	case "ready":
		accepted := backlog.Accepted(*is)
		comments, err := b.Comments(t)
		if err != nil {
			return err
		}
		st, _, _ := backlog.ReadState(comments, role) // its drafts told from a person's edit
		if missing := backlog.NotReady(*is, st, accepted); len(missing) > 0 {
			return nil // changed since the plan: it stays to refine
		}
		if strings.Contains(is.Body, backlog.DraftMarker) {
			// The drafts are the person's, accepted or edited: their marks go.
			body := backlog.StripDrafts(is.Body)
			if err := b.SetBody(c.Issue, body); err != nil {
				return err
			}
			if err := keepBody(f, role, t, body, nil, nil); err != nil {
				return err
			}
		}
		add, remove := []string{backlog.LabelReady}, backlog.ReadyRemoves(*is)
		if c.Card.Labels() {
			// Its card to the board's ready column, out of the others
			// (ADR-0057), in the same call.
			add = append(add, c.Card.Column)
			for _, l := range c.Card.Off {
				if slices.Contains(is.Labels, l) {
					remove = append(remove, l)
				}
			}
		}
		if err := f.Label(t, add, remove); err != nil {
			return err
		}
		if c.Card.Writes() {
			moveCard(f, c.Issue, c.Card.Project, c.Card.Column)
		}
		return nil
	}
	body, added, _ := backlog.Refine(is.Body, c, role)
	if len(added) > 0 {
		if err := b.SetBody(c.Issue, body); err != nil {
			return err
		}
	}
	labels := []string{backlog.LabelToRefine}
	if slices.ContainsFunc(added, backlog.Drafted) {
		labels = append(labels, backlog.LabelDraft)
		// The label a person accepts the drafts with, there to be picked
		// from the forge's list.
		if err := b.EnsureLabel(backlog.AcceptedLabel(f), "0e8a16", backlog.AcceptedSays); err != nil {
			return err
		}
	}
	if err := f.Label(t, labels, nil); err != nil {
		return err
	}
	return keepBody(f, role, t, body, c.Sources, added)
}

// keepBody records in an issue's state the body the engine left, so only a
// person's change has it read again, the files its scope names when it
// had none, and the sections it wrote, so a text there is told from a
// person's (#128).
func keepBody(f forge.Forge, role string, t forge.Target, body string, sources, wrote []string) error {
	b := f.(forge.Backlog)
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	st, _, err := backlog.ReadState(comments, role)
	if err != nil {
		return fmt.Errorf("#%d: its state comment: %w", t.ID, err)
	}
	if len(st.Sources) == 0 {
		st.Sources = sources
	}
	for _, name := range wrote {
		if name == backlog.DescriptionName {
			continue // its text on top taken into the Need: no description left to keep
		}
		st.SetWrote(name, backlog.SectionDigest(body, name))
	}
	st.Keep(body)
	return f.Sticky(t, backlog.FormatState(*st), backlog.StateMarker(role), false)
}

// moveCard sets the Status of issue's card in a GitHub project to option
// (ADR-0057). A card not moved — the issue in no item of the project, a
// write refused — is said, and changes no verdict: workline's labels
// stay the roles' marks.
func moveCard(f forge.Forge, issue, project int, option string) {
	if project == 0 || option == "" {
		return
	}
	why, err := forge.SetStatus(f, issue, project, option)
	if err != nil {
		why = forge.Said(err)
	}
	if why != "" {
		fmt.Fprintf(os.Stderr, "workline: #%d's card not moved to %q on the project #%d: %s\n", issue, option, project, why)
	}
}

// readyMove is the move a ready makes on the forge's board: its ready
// column's label on, the other columns' off (ADR-0057); nil with none. A
// board out of reach is said, and moves nothing.
func readyMove(f forge.Forge, repo string) *board.Move {
	if f == nil {
		return nil
	}
	pc, err := role.LoadProjectConfig(repo)
	if err != nil {
		return nil
	}
	mv, err := board.To(f, "ready", pc.Board)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workline: the forge's board could not be read, no card moved: %s\n", forge.Said(err))
	}
	return mv
}

// planActs decides what becomes of the run's acts on the backlog, once: a
// resumed run reads the plan the first attempt wrote, as the forge it reads
// has changed since. Nil when the run proposes none.
func planActs(f forge.Forge, r *role.Role, settings map[string]any, st runState, runDir string, intents []intent.Intention) (*backlog.Plan, error) {
	role := r.Name
	closes := map[int]backlog.Proposal{}
	for i, in := range intents {
		if !slices.Contains(backlog.Kinds, in.Kind) {
			continue
		}
		var c backlog.Proposal
		data, _ := yaml.Marshal(in.Value)
		if err := yaml.Unmarshal(data, &c); err != nil {
			c = backlog.Proposal{Reason: "unreadable: " + err.Error()}
		}
		c.Do = in.Kind
		closes[i] = c
	}
	// A role keeping a backlog reads its record on every run, acts or not:
	// a closing a person undid is found at the next run, whatever it does.
	if len(closes) == 0 && !slices.ContainsFunc(r.Intentions, func(k string) bool { return slices.Contains(backlog.Kinds, k) }) {
		return nil, nil
	}
	if len(closes) == 0 && f == nil {
		return nil, nil
	}
	file := filepath.Join(runDir, "out", "acts.yaml")
	if data, err := os.ReadFile(file); err == nil {
		var p backlog.Plan
		return &p, yaml.Unmarshal(data, &p)
	}
	if f == nil {
		return nil, fmt.Errorf("an act on an issue needs a forge; %s", forge.Missing)
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, errors.New("this forge cannot list or close issues")
	}
	// The issues pre gave the agent, read only once it answered (#427).
	var read []int
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "issues-read")); err == nil && !st.Unasked {
		for _, f := range strings.Fields(string(data)) {
			if n, err := strconv.Atoi(f); err == nil {
				read = append(read, n)
			}
		}
	}
	// The second judge's answers on the issues announced obsolete, as the
	// engine wrote them beside pre's questions (ADR-0024).
	judged := map[int]backlog.Judged{}
	answers, _ := filepath.Glob(filepath.Join(runDir, "in", "judge", backlog.JudgeKeyPrefix+"*", "answer.yaml"))
	for _, file := range answers {
		id, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(filepath.Dir(file)), backlog.JudgeKeyPrefix))
		if err != nil {
			continue
		}
		var j backlog.Judged
		if data, err := os.ReadFile(file); err == nil && yaml.Unmarshal(data, &j) == nil {
			judged[id] = j
		}
	}
	cfg, err := backlog.ReadConfig(settings)
	if err != nil {
		return nil, err
	}
	// The reviewer after the role in a line: ready waits on its spec
	// review (#128).
	line, err := routing.Load(st.Repo)
	if err != nil {
		return nil, err
	}
	cfg.SpecReview = line.Follows(role, backlog.SpecReviewer)
	// The board's ready column a ready moves the card to (ADR-0057); a
	// board out of reach is said and moves nothing.
	for _, c := range closes {
		if c.Do == "ready" {
			cfg.Ready = readyMove(f, st.Repo) // read once, when a ready is proposed
			break
		}
	}
	// What the run read — the commit, each issue's body — and the agents
	// and models it asked, kept with each act (ADR-0046).
	if data, err := os.ReadFile(filepath.Join(runDir, "in", backlog.ReadFile)); err == nil {
		if err := yaml.Unmarshal(data, &cfg.Read); err != nil {
			return nil, fmt.Errorf("in/%s: %v", backlog.ReadFile, err)
		}
	}
	cfg.Read.Model = strings.Join(st.Models, ", ")
	// The issues whose code pre gave the agent: an effort is estimated on
	// them only (ADR-0040).
	if data, err := os.ReadFile(filepath.Join(runDir, "in", backlog.CodeReadFile)); err == nil && !st.Unasked {
		for _, f := range strings.Fields(string(data)) {
			if n, err := strconv.Atoi(f); err == nil {
				cfg.CodeRead = append(cfg.CodeRead, n)
			}
		}
	}
	// What open issues were built on that changed, and those read again
	// for it (ADR-0032).
	var changes []backlog.Change
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "changes.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &changes); err != nil {
			return nil, fmt.Errorf("in/changes.yaml: %v", err)
		}
	}
	// A person's comments pre read as "revise", on the issues waiting on
	// them (ADR-0038).
	revise := map[int]*backlog.Answer{}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "answers.yaml")); err == nil && !st.Unasked {
		if err := yaml.Unmarshal(data, &revise); err != nil {
			return nil, fmt.Errorf("in/answers.yaml: %v", err)
		}
	}
	p, err := backlog.Decide(b, st.Repo, role, cfg, closes, read, judged, changes, revise)
	if err != nil {
		return nil, err
	}
	if cfg.Ready != nil && cfg.Ready.Said != "" && slices.ContainsFunc(p.Decisions, func(d backlog.Decision) bool {
		return d.Mode == backlog.Act && d.Act.Do == "ready"
	}) {
		// Said once a run, by the role that moves a card (ADR-0057).
		p.Findings = append(p.Findings, verdict.Finding{Rule: "board-workflow-off", Level: "info", Where: cfg.Ready.URL, Message: cfg.Ready.Said})
	}
	data, _ := yaml.Marshal(p)
	return p, os.WriteFile(file, data, 0o644)
}

// planAhead decides a run's acts on the backlog when it is judged and not
// applied (NoApply), reading the forge only: the plan is kept in the run
// folder, so the job that applies does what the judge's summary says. A
// forge that cannot be read is said; the apply decides then.
func planAhead(r *role.Role, settings map[string]any, st runState, runDir string, intents []intent.Intention, res *Result) error {
	f, err := forge.Open(st.Forge, st.Repo)
	if err != nil {
		return err
	}
	plan, err := planActs(f, r, settings, st, runDir, intents)
	if errors.Is(err, forge.ErrUnreachable) {
		res.Findings = append(res.Findings, verdict.Finding{Rule: "plan-unread", Level: "warn",
			Message: "the backlog could not be read to say what will be done and proposed (" + err.Error() + "): the job that applies decides it"})
		return nil
	}
	if err != nil || plan == nil {
		return err
	}
	res.Findings = append(res.Findings, plan.Findings...)
	return nil
}

// onIssues says what a backlog role's run leaves for people on the issues,
// counted by issue as the summary groups them (ADR-0038): done alone,
// done as a person accepted, proposed, set aside — never the writes it
// takes, its state comments among them. A run whose agent was not asked
// says so first, never "nothing to do" (#427).
func onIssues(fs []verdict.Finding, ahead, unasked bool) string {
	if slices.ContainsFunc(fs, func(f verdict.Finding) bool { return f.Rule == "plan-unread" }) {
		return "what it does on the issues is decided when applied"
	}
	count := func(rule string) int {
		seen := map[string]bool{}
		for _, f := range fs {
			if f.Rule == rule && f.Where != "" {
				seen[f.Where] = true
			}
		}
		return len(seen)
	}
	says := []struct{ rule, done, ahead string }{
		{"done", "done alone", "to do alone"},
		{"done-as-accepted", "done as a person accepted", "to do as a person accepted"},
		{"proposed", "proposed to a person", "to propose to a person"},
		{"set-aside", "set aside by a person", "set aside by a person"},
	}
	var parts []string
	for _, s := range says {
		if n := count(s.rule); n > 0 {
			what := s.done
			if ahead {
				what = s.ahead
			}
			parts = append(parts, fmt.Sprintf("%d %s", n, what))
		}
	}
	switch {
	case unasked && len(parts) == 0:
		return "the agent was not asked: no issue read"
	case unasked:
		return "the agent was not asked, no issue read; issues: " + strings.Join(parts, ", ")
	case len(parts) == 0 && ahead:
		return "nothing to do on the issues"
	case len(parts) == 0:
		return "nothing done on the issues"
	}
	return "issues: " + strings.Join(parts, ", ")
}

// links gives the job's summary the forge's addresses: each issue's page,
// the saved filter of what waits on a person (ADR-0038).
func links(f forge.Forge, res *Result) {
	if f == nil || res.Issues != "" {
		return
	}
	res.Issues = forge.IssuePages(f)
	res.Waiting = forge.LabelFilter(f, backlog.ProposedLabel(f))
}

// report writes back what the run leaves on each issue: its state, its
// label workline:proposed, a person's comments read; and closes an old
// report issue whose record moved to the issues (ADR-0038).
func report(f forge.Forge, role string, p *backlog.Plan) error {
	return p.Persist(f, role)
}
