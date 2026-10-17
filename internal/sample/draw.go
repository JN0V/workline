package sample

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/forge"
)

// Every role's acts in the weekly sample (ADR-0046): one in ten of the
// window's acts per role and kind, at least one when a kind has any, each
// ranked by a digest of the week and the act, so a rerun draws the same.
// Each is drawn from the record its role keeps, read and never written:
// the product owner's `did` in each issue's state; the lines the
// reviewer's summary comment hides for the findings judged, every run's,
// on each merge request merged in the window; the committer's trailers on
// the commits.

// The roles whose acts are drawn, in the order they are said.
const (
	docsRole   = "documentalist"
	reviewRole = "reviewer"
	commitRole = "committer"
)

var roleOrder = []string{docsRole, productOwner, reviewRole, commitRole}

// Act is an act of a role the sample drew.
type Act struct {
	Role string `json:"role"`
	// Kind: the product owner's act as its record names it (ready,
	// refine…); a finding the reviewer's judge kept or dropped; a commit
	// the committer rewrote.
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`              // its issue or merge request as the forge names it (#205, !7), or its commit
	In     string `json:"in,omitempty"`     // the merge request a commit came in by, read on its head
	Where  string `json:"where,omitempty"`  // a finding's place: path:line
	Commit string `json:"commit,omitempty"` // the commit it was done at, when its record says
	Day    string `json:"day,omitempty"`    // the day it was done, YYYY-MM-DD, when its record says
	By     string `json:"by,omitempty"`     // the level it was done at, or the models that acted, when its record says
	Line   string `json:"line,omitempty"`   // what its record says of it
	Lens   string `json:"lens,omitempty"`   // a finding's lens
	Title  string `json:"title,omitempty"`  // a finding's title, all its judge's answer left out
	// A product owner's act: the agents and models that acted and the
	// digest of the issue's text it read, when its record says
	// (ADR-0046); what shows a person undid it since (ADR-0033).
	Acted  string `json:"acted,omitempty"`
	Body   string `json:"body,omitempty"`
	Undone string `json:"undone,omitempty"`
	// Verdict: true (it stands), false (proven wrong, with the
	// material's words), unproven (said wrong, not proven: for a person),
	// or not-read (for a person, Why says why).
	Verdict      string  `json:"verdict"`
	Why          string  `json:"why,omitempty"`
	Judge        string  `json:"judge,omitempty"`
	Model        string  `json:"model,omitempty"`
	Independence string  `json:"independence,omitempty"`
	Proofs       []Proof `json:"proofs,omitempty"`
	Unproven     []Proof `json:"unproven,omitempty"`
	Panel        *Panel  `json:"panel,omitempty"`  // how a false verdict was settled
	Defect       *Defect `json:"defect,omitempty"` // a true finding its judge dropped, for an issue
	key          string  // what it is ranked by, with the week
	older        bool    // a product owner's act read without the record ADR-0046 completes: as its issue reads now
}

// Count is how many acts of a role's kind the window held, and how many
// were drawn.
type Count struct {
	Role  string `json:"role"`
	Kind  string `json:"kind"`
	Acts  int    `json:"acts"`
	Drawn int    `json:"drawn"`
}

// drawEach draws one in OneIn of the acts of each role and kind, rounded
// up, and counts them.
func drawEach(week string, acts []Act) ([]Act, []Count) {
	groups := map[string][]int{}
	var order []string
	for i, a := range acts {
		k := a.Role + "\x00" + a.Kind
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], i)
	}
	rank := func(role string) int {
		for i, r := range roleOrder {
			if r == role {
				return i
			}
		}
		return len(roleOrder)
	}
	sort.Slice(order, func(i, j int) bool {
		ri, ki, _ := strings.Cut(order[i], "\x00")
		rj, kj, _ := strings.Cut(order[j], "\x00")
		if rank(ri) != rank(rj) {
			return rank(ri) < rank(rj)
		}
		return ki < kj
	})
	var drawn []Act
	var counts []Count
	for _, k := range order {
		idx := groups[k]
		keys := make([]string, len(idx))
		for i, j := range idx {
			keys[i] = acts[j].key
		}
		picked := pick(week, keys)
		role, kind, _ := strings.Cut(k, "\x00")
		counts = append(counts, Count{Role: role, Kind: kind, Acts: len(idx), Drawn: len(picked)})
		for _, p := range picked {
			a := acts[idx[p]]
			a.Verdict = "not-read"
			drawn = append(drawn, a)
		}
	}
	return drawn, counts
}

// productOwnerActs are the acts the product owner did alone on the days
// [first, end), from each issue's state (ADR-0033's record), each with
// what it read when the record says (ADR-0046) and what shows a person
// undid it; and the level its acts done alone suggest, from those at the
// level of its last, up to end (until, the last day counted). A person's
// yes makes an act theirs, never drawn.
func productOwnerActs(b forge.Backlog, first, end, until string) ([]Act, *Autonomy, error) {
	acts, err := backlog.ReadActs(b, productOwner)
	if err != nil || acts == nil {
		return nil, nil, err
	}
	au := &Autonomy{Level: acts.Level, Until: until}
	var out []Act
	for i, d := range acts.Did {
		if end != "" && d.Day >= end {
			continue
		}
		undone, isUndone := acts.Undone(i)
		if d.Level == acts.Level {
			au.Counted++
			if isUndone {
				au.Undone++
			}
		}
		if d.Day < first {
			continue
		}
		out = append(out, Act{Role: productOwner, Kind: d.Act, Ref: fmt.Sprintf("#%d", d.Issue), Day: d.Day, By: d.Level, Line: d.Line,
			Commit: d.At, Acted: d.Model, Body: d.Body, Undone: undone,
			key: fmt.Sprintf("%d\x00%s\x00%s", d.Issue, d.Act, d.Day)})
	}
	au.Suggest, au.Why = backlog.SuggestFromActs(au.Level, au.Counted, au.Undone)
	return out, au, nil
}

// summaryMarker marks the reviewer's one comment on a merge request.
var summaryMarker = forge.Marker("sticky=reviewer/summary")

var (
	// independence is what a judged finding's row says of the models:
	// the lens's, then the judge's.
	independence = regexp.MustCompile(`independence: \S+ \(([^()]*?) → ([^()]*?)\)`)
	blocksLine   = regexp.MustCompile(`<!-- workline:blocks (.*?) -->`)
)

// reviewerActs are the important findings the reviewer's judge kept or
// dropped on each merge request, read from the lines its summary comment
// hides for them, every run's, each at the head its review read
// (ADR-0046); and what they leave unread: the merge requests whose
// comment no longer holds some lines, with how many, and those whose
// comment an engine before the lines wrote.
func reviewerActs(repo string, f forge.Forge, b forge.Backlog, mrs []forge.MergeRequest) ([]Act, []string, error) {
	var out []Act
	var unread, before []string
	for _, m := range mrs {
		ref := forge.MergeRef(f, m.ID)
		comments, err := b.Comments(forge.Target{Kind: "merge-request", ID: m.ID})
		if err != nil {
			return nil, nil, fmt.Errorf("the comments of %s: %w", ref, err)
		}
		for _, c := range comments {
			if !strings.Contains(c, summaryMarker) {
				continue
			}
			judged, gone, held := reviewer.ReadJudged(c)
			if held {
				for _, j := range judged {
					out = append(out, judgedAct(ref, j))
				}
				if gone > 0 {
					unread = append(unread, fmt.Sprintf("%s on %s whose lines its summary comment no longer holds: past its size (%d characters, the oldest gone first), or not readable", plural(gone, "finding")+" judged", ref, reviewer.SummaryMax))
				}
				break
			}
			// An engine before the lines: the last run's rows of its table.
			before = append(before, ref)
			head, _ := git(repo, "rev-parse", "-q", "--verify", fmt.Sprintf("refs/workline/sample/%d^{commit}", m.ID))
			for _, a := range summaryActs(ref, c) {
				if a.Commit == "" {
					a.Commit = head
				}
				out = append(out, a)
			}
			break
		}
	}
	if len(before) > 0 {
		unread = append(unread, "the reviewer's findings of a run before the last on a merge request whose summary comment an engine before its judged lines wrote, its table read alone: "+strings.Join(before, ", "))
	}
	return out, unread, nil
}

// judgedAct is a finding judged, as its line says: kept or dropped, at the
// head its review read, by the models of its lens and its judge.
func judgedAct(ref string, j reviewer.Answered) Act {
	kind := "dropped"
	if j.Yes {
		kind = "kept"
	}
	by := j.Finder
	if j.Judge != "" {
		by += " → " + j.Judge
	}
	return Act{Role: reviewRole, Kind: kind, Ref: ref, Where: j.Where, Commit: j.At, By: by, Lens: j.Lens, Title: j.Title,
		Line: fmt.Sprintf("the %s lens: Important: %s", j.Lens, j.Title),
		key:  strings.Join([]string{ref, kind, j.Where, j.Lens, j.At, j.Title}, "\x00")}
}

// summaryActs reads the rows of one summary comment: a finding the judge
// dropped (`finding-judged-no`), and an important one kept, which no
// "not verified" says was left unjudged; one kept that blocks, with the
// head it was found at, as the record's blocks say.
func summaryActs(ref, comment string) []Act {
	var blocks []struct {
		Lens  string `json:"lens"`
		At    string `json:"at"`
		Where string `json:"where"`
	}
	if m := blocksLine.FindStringSubmatch(comment); m != nil {
		json.Unmarshal([]byte(m[1]), &blocks)
	}
	at := func(lens, where string) string {
		for _, b := range blocks {
			if b.Lens == lens && b.Where == where {
				return b.At
			}
		}
		return ""
	}
	models := func(message string) string {
		if m := independence.FindStringSubmatch(message); m != nil {
			return m[1] + " → " + m[2]
		}
		return ""
	}
	var out []Act
	for _, l := range strings.Split(comment, "\n") {
		cells := row(l)
		if len(cells) != 4 {
			continue
		}
		rule, where, message := cells[1], strings.Trim(cells[2], "`"), cells[3]
		switch {
		case rule == "finding-judged-no":
			head, _, ok := strings.Cut(message, " — the judge: ")
			lens, rest, found := strings.Cut(strings.TrimPrefix(head, "dropped, from the "), " lens: ")
			if !ok || !found {
				continue
			}
			title := rest
			if i := strings.LastIndex(rest, " at "); i >= 0 {
				title, where = rest[:i], rest[i+len(" at "):]
			}
			if t, err := strconv.Unquote(title); err == nil {
				title = t
			}
			out = append(out, Act{Role: reviewRole, Kind: "dropped", Ref: ref, Where: where, By: models(message), Line: message, Lens: lens, Title: title,
				key: ref + "\x00dropped\x00" + where + "\x00" + message})
		case strings.HasPrefix(message, "Important: ") && !strings.Contains(message, "(not verified"):
			title, _, _ := strings.Cut(strings.TrimPrefix(message, "Important: "), " — ")
			title, _, _ = strings.Cut(title, " (verified")
			out = append(out, Act{Role: reviewRole, Kind: "kept", Ref: ref, Where: where, Commit: at(rule, where), By: models(message),
				Line: fmt.Sprintf("the %s lens: %s", rule, message), Lens: rule, Title: title,
				key: ref + "\x00kept\x00" + where + "\x00" + rule + "\x00" + message})
		}
	}
	return out
}

// row is the cells of a row of a Markdown table, `\|` read as `|`; nil
// for a line that is no row, or the table's head or rule.
func row(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") || strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| | |") {
		return nil
	}
	var cells []string
	var cell strings.Builder
	for i := 1; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteByte(line[i])
		}
	}
	return cells
}

// committerActs are the commits the committer rewrote, as their own
// trailers say (`Workline-Role: committer`), among the commits given; one
// rebased twice — the same author, moment and subject — counts once.
func committerActs(repo string, commits []string, in map[string]string) ([]Act, error) {
	var out []Act
	seen := map[string]bool{}
	for _, c := range commits {
		info, err := git(repo, "log", "-1", "--format=%(trailers:key=Workline-Role,valueonly,separator=%x2C)%x00%(trailers:key=Workline-Model,valueonly,separator=%x2C%x20)%x00%an%x00%aI%x00%s", c)
		if err != nil {
			return nil, err
		}
		f := strings.Split(info, "\x00")
		if len(f) != 5 || !hasValue(f[0], commitRole) {
			continue
		}
		id := f[2] + "\x00" + f[3] + "\x00" + f[4]
		if seen[id] {
			continue
		}
		seen[id] = true
		day := ""
		if t, err := time.Parse(time.RFC3339, f[3]); err == nil {
			day = t.UTC().Format(time.DateOnly)
		}
		out = append(out, Act{Role: commitRole, Kind: "rewrite", Ref: short(repo, c), In: in[c], Commit: c, Day: day, By: f[1], Line: f[4], key: c})
	}
	return out, nil
}

// hasValue says whether a list a comma separates holds v.
func hasValue(list, v string) bool {
	for _, x := range strings.Split(list, ",") {
		if strings.TrimSpace(x) == v {
			return true
		}
	}
	return false
}

// heads is what the merge requests merged in the window add to it: each
// one's head fetched from the forge's ref for it, which a squash leaves
// (GitHub's refs/pull/<n>/head, GitLab's refs/merge-requests/<iid>/head),
// so a `checked` naming a commit of it is read (ADR-0046, W41); and the
// commits of the heads that the branch does not hold, whose trailers the
// squash dropped.
type heads struct {
	merged []forge.MergeRequest // merged in the window
	own    []string             // the heads' commits the branch does not hold, newest first
	in     map[string]string    // such a commit → its merge request, as the forge names it
	squash map[string]bool      // the commits merge requests read on their heads became: their heads stand for them
	unread []string             // why some could not be read
}

// mergedIn lists the merge requests merged in the window base..tip — the
// commit each became is in it — and fetches their heads.
func mergedIn(repo string, f forge.Forge, since time.Time, base, tip, after string) (*heads, error) {
	h := &heads{in: map[string]string{}, squash: map[string]bool{}}
	lister, ok := f.(forge.Merged)
	if !ok || tip == "" {
		return h, nil
	}
	all, err := lister.MergedSince(since)
	if err != nil {
		return h, err
	}
	args := []string{"rev-list", tip}
	if base != "" {
		args = append(args, "^"+base)
	}
	list, err := git(repo, args...)
	if err != nil {
		return h, err
	}
	window := map[string]bool{}
	for _, c := range strings.Fields(list) {
		window[c] = true
	}
	for _, m := range all {
		if window[m.MergedAs] {
			h.merged = append(h.merged, m)
		}
	}
	if len(h.merged) == 0 || forge.KeepsBranches(f) {
		return h, nil
	}
	if _, err := git(repo, "remote", "get-url", "origin"); err != nil {
		h.unread = append(h.unread, fmt.Sprintf("the heads of the %s merged in the window: no remote `origin` to fetch them from", plural(len(h.merged), "merge request")))
		return h, nil
	}
	ref := func(m forge.MergeRequest) string { return fmt.Sprintf("refs/workline/sample/%d", m.ID) }
	fetch := []string{"fetch", "-q", "--no-tags", "origin"}
	for _, m := range h.merged {
		if m.Head != "" {
			fetch = append(fetch, "+"+m.Head+":"+ref(m))
		}
	}
	if _, err := git(repo, fetch...); err != nil {
		for _, m := range h.merged { // one by one, to say which
			if m.Head == "" {
				continue
			}
			if _, err := git(repo, "fetch", "-q", "--no-tags", "origin", "+"+m.Head+":"+ref(m)); err != nil {
				h.unread = append(h.unread, fmt.Sprintf("the head of %s (%s): %v", forge.MergeRef(f, m.ID), m.Head, err))
			}
		}
	}
	seen := map[string]bool{}
	for _, m := range h.merged {
		head, err := git(repo, "rev-parse", "-q", "--verify", ref(m)+"^{commit}")
		if err != nil {
			continue
		}
		args := []string{"rev-list", "--no-merges", head, "^" + tip}
		if after != "" {
			args = append(args, "^"+after)
		}
		list, err := git(repo, args...)
		if err != nil {
			return h, err
		}
		if list != "" {
			h.squash[m.MergedAs] = true
		}
		for _, c := range strings.Fields(list) {
			if !seen[c] {
				seen[c] = true
				h.own = append(h.own, c)
				h.in[c] = forge.MergeRef(f, m.ID)
			}
		}
	}
	return h, nil
}

// actsOfTheWindow gathers the acts of the product owner, the reviewer and
// the committer in the window, and says what could not be read.
func actsOfTheWindow(o Options, res *Result, h *heads, commits []string, first, end string) []Act {
	var acts []Act
	notDrawn := func(why string) {
		res.NotDrawn = append(res.NotDrawn, why)
	}
	unreachable := func(what string, err error) {
		notDrawn(fmt.Sprintf("%s: the forge did not answer: %v", what, err))
		res.unreachable = true
	}
	b, isBacklog := o.Forge.(forge.Backlog)
	switch {
	case o.Forge == nil:
		notDrawn("the product owner's acts and the reviewer's findings: no forge to read them from (`--forge`, or the project's `forge`); nor the commits of a merge request squashed")
	case !isBacklog:
		notDrawn("the product owner's acts and the reviewer's findings: this forge keeps no issues to read them from")
	default:
		// The last day counted for the product owner's level: the window's,
		// or today for one that reaches now (--since).
		until := o.Now
		if until.IsZero() {
			until = time.Now()
		}
		if t, err := time.Parse(time.DateOnly, end); err == nil {
			until = t.AddDate(0, 0, -1)
		}
		if po, au, err := productOwnerActs(b, first, end, until.UTC().Format(time.DateOnly)); err != nil {
			unreachable("the product owner's acts", err)
		} else {
			acts = append(acts, po...)
			res.Autonomy = au
		}
		if _, ok := o.Forge.(forge.Merged); !ok {
			notDrawn("the reviewer's findings, and the commits of a merge request squashed: this forge does not list the merge requests merged")
		} else if rv, unread, err := reviewerActs(o.Repo, o.Forge, b, h.merged); err != nil {
			unreachable("the reviewer's findings", err)
		} else {
			acts = append(acts, rv...)
			res.NotDrawn = append(res.NotDrawn, unread...)
		}
	}
	res.NotDrawn = append(res.NotDrawn, h.unread...)
	var own []string
	for _, c := range commits {
		if !h.squash[c] {
			own = append(own, c)
		}
	}
	rw, err := committerActs(o.Repo, append(own, h.own...), h.in)
	if err != nil {
		notDrawn("the committer's rewrites: " + err.Error())
	}
	if len(rw) == 0 && err == nil {
		notDrawn("the committer's rewrites: no commit of the window carries `Workline-Role: committer`, and a rewrite without it cannot be told from its commit")
	}
	return append(acts, rw...)
}

// actWords say an act drawn, for a person.
func actWords(a Act) string {
	var b strings.Builder
	switch a.Role {
	case productOwner:
		fmt.Fprintf(&b, "the product owner's `%s` on %s, %s, at %s", a.Kind, a.Ref, a.Day, a.By)
	case reviewRole:
		fmt.Fprintf(&b, "a finding the reviewer's judge %s on %s, at %s", a.Kind, a.Ref, a.Where)
	case commitRole:
		fmt.Fprintf(&b, "a commit the committer rewrote, %s", a.Ref)
		if a.In != "" {
			fmt.Fprintf(&b, ", on the head of %s", a.In)
		}
	default:
		fmt.Fprintf(&b, "the %s's %s, %s", a.Role, a.Kind, a.Ref)
	}
	if a.Line != "" {
		fmt.Fprintf(&b, ": %s", oneLine(a.Line, 200))
	}
	return b.String()
}

// roleWords name a role as a sentence does.
func roleWords(role string) string { return "the " + strings.ReplaceAll(role, "-", " ") }

// kindWords name a role's kind of act.
func kindWords(c Count) string {
	switch {
	case c.Role == docsRole:
		return "`checked` moved"
	case c.Role == reviewRole:
		return "finding " + c.Kind
	case c.Role == commitRole:
		return "message rewritten"
	}
	return "`" + c.Kind + "`"
}

// oneLine is a text on one line, cut past max characters.
func oneLine(t string, max int) string {
	t = strings.Join(strings.Fields(t), " ")
	if r := []rune(t); len(r) > max {
		t = string(r[:max]) + "…"
	}
	return t
}
