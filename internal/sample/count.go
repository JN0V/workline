package sample

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// Every week's verdicts are kept on the tracking issue and counted
// (ADR-0046): the week's comment holds a row per act, numbered, the same
// rows hidden below as JSON the engine reads back; the maintainer agrees
// with a 👍 on it, or contests rows with a 👎 and a reply naming them
// (`not 3, 7`). One comment, the count, is written again each week from
// every week's rows and answers, never added to.

// Row is one act of a week, as its comment keeps it.
type Row struct {
	N            int    `json:"n"`
	Role         string `json:"role"`
	Kind         string `json:"kind"`
	Ref          string `json:"ref"`
	Where        string `json:"where,omitempty"`
	Verdict      string `json:"verdict"`
	Model        string `json:"model,omitempty"`
	Independence string `json:"independence,omitempty"`
	Panel        string `json:"panel,omitempty"`
	Issue        int    `json:"issue,omitempty"`
}

// rows are a week's acts, numbered: its docs first, then the others'.
func rows(res *Result) []Row {
	var out []Row
	for _, r := range res.Reads {
		row := Row{N: len(out) + 1, Role: docsRole, Kind: "checked", Ref: r.Doc, Verdict: r.Verdict, Model: r.Model, Independence: r.Independence}
		if r.Panel != nil {
			row.Panel = r.Panel.Words()
		}
		out = append(out, row)
	}
	for _, a := range res.Drawn {
		row := Row{N: len(out) + 1, Role: a.Role, Kind: a.Kind, Ref: a.Ref, Where: a.Where, Verdict: a.Verdict, Model: a.Model, Independence: a.Independence}
		if a.Panel != nil {
			row.Panel = a.Panel.Words()
		}
		if a.Defect != nil {
			row.Issue = a.Defect.Issue
		}
		out = append(out, row)
	}
	return out
}

// hiddenRows are a week's rows as its comment hides them: JSON on one
// line, a dash written - and `<`, `>` escaped, so nothing in a row
// ends the comment hiding them.
func hiddenRows(rs []Row) string {
	if rs == nil {
		rs = []Row{}
	}
	data, _ := json.Marshal(rs)
	return "<!-- workline:rows " + strings.ReplaceAll(string(data), "-", `-`) + " -->"
}

var (
	rowsLine = regexp.MustCompile(`<!-- workline:rows (.*?) -->`)
	weekMark = regexp.MustCompile(`<!-- workline:sample:(\d{4}-W\d{2}) -->`)
	notRows  = regexp.MustCompile(`(?i)\bnot\s+(\d+(?:\s*(?:,|and)\s*\d+)*)`)
	weekName = regexp.MustCompile(`\b\d{4}-W\d{2}\b`)
)

// parseRows reads a week's hidden rows; false for a comment written before
// they were kept.
func parseRows(body string) ([]Row, bool) {
	m := rowsLine.FindStringSubmatch(body)
	if m == nil {
		return nil, false
	}
	var rs []Row
	return rs, json.Unmarshal([]byte(m[1]), &rs) == nil
}

// week is a week's comment as the count reads it: its rows, and the
// maintainer's answer.
type week struct {
	Name      string
	Rows      []Row
	Kept      bool         // its rows are kept: written since ADR-0046
	Up, Down  []string     // who agreed, who contested: people who may write
	Contested map[int]bool // the rows a 👎's reply names
	Unnamed   bool         // a 👎 and no reply naming rows
}

// answer is what the maintainer said of a row: agreed, contested, or
// unanswered.
func (w week) answer(n int) string {
	switch {
	case w.Contested[n]:
		return "contested"
	case len(w.Up) > 0, len(w.Down) > 0 && !w.Unnamed:
		return "agreed"
	}
	return "unanswered"
}

// weeks reads every week's comment on the tracking issue, and the
// answers to it: the reactions of people who may write, and the reply of
// one who contested, naming rows.
func weeks(f forge.Forge, issue int) ([]week, error) {
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil, nil
	}
	notes, err := b.Notes(forge.Target{Kind: "issue", ID: issue})
	if err != nil {
		return nil, err
	}
	reacted, _ := f.(forge.Reacted)
	var out []week
	for i, n := range notes {
		m := weekMark.FindStringSubmatch(n.Body)
		if m == nil {
			continue
		}
		w := week{Name: m[1], Contested: map[int]bool{}}
		w.Rows, w.Kept = parseRows(n.Body)
		if reacted != nil && n.ID != "" {
			rs, err := reacted.Reactions(issue, n.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range rs {
				if r.Bot || !r.Writer {
					continue
				}
				switch r.Emoji {
				case "+1":
					w.Up = append(w.Up, r.Author)
				case "-1":
					w.Down = append(w.Down, r.Author)
				}
			}
		}
		if len(w.Down) > 0 {
			w.Unnamed = true
			later := false // past the next week's comment: a reply naming no week is that one's
			for _, reply := range notes[i+1:] {
				if weekMark.MatchString(reply.Body) {
					later = true
					continue
				}
				if !slices.Contains(w.Down, reply.Author) {
					continue
				}
				named := weekName.FindAllString(reply.Body, -1)
				if len(named) > 0 && !slices.Contains(named, w.Name) || len(named) == 0 && later {
					continue // about another week
				}
				r := notRows.FindStringSubmatch(reply.Body)
				if r == nil {
					continue
				}
				for _, s := range regexp.MustCompile(`\d+`).FindAllString(r[1], -1) {
					k, _ := strconv.Atoi(s)
					w.Contested[k] = true
				}
				w.Unnamed = false
				break
			}
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	// A week written again keeps one comment: the last read stands.
	var unique []week
	for _, w := range out {
		if len(unique) > 0 && unique[len(unique)-1].Name == w.Name {
			unique[len(unique)-1] = w
			continue
		}
		unique = append(unique, w)
	}
	return unique, nil
}

// countKey keeps the count's one comment.
const countKey = "sample:count"

// countComment is the count, written again from every week: per role and
// kind of act, the verdicts and the maintainer's answers; a line a week;
// the weeks that read nothing.
func countComment(ws []week) string {
	type tally struct {
		drawn                            int
		stand, false_, unproven, notRead int
		agreed, contested, unanswered    int
	}
	counts := map[string]*tally{}
	var keys []string
	var nothing, before []string
	var b strings.Builder
	b.WriteString("### The count, every week\n\n")
	b.WriteString("Written again each week from every week's rows, never added to; a rerun writes the same (ADR-0046). A 👍 on a week's comment agrees with all its rows; a 👎 and a reply naming rows (`not 3, 7`) contests those, the rest agreed.\n\n")
	var weekLines strings.Builder
	for _, w := range ws {
		if !w.Kept {
			before = append(before, w.Name)
			continue
		}
		read, falseOnes := 0, 0
		for _, r := range w.Rows {
			k := r.Role + "\x00" + r.Kind
			t := counts[k]
			if t == nil {
				t = &tally{}
				counts[k] = t
				keys = append(keys, k)
			}
			t.drawn++
			switch r.Verdict {
			case "true":
				t.stand++
			case "false":
				t.false_++
				falseOnes++
			case "not-read":
				t.notRead++
			default:
				t.unproven++
			}
			if r.Verdict != "not-read" {
				read++
			}
			switch w.answer(r.N) {
			case "agreed":
				t.agreed++
			case "contested":
				t.contested++
			default:
				t.unanswered++
			}
		}
		if read == 0 {
			nothing = append(nothing, w.Name)
		}
		answer := "unanswered"
		switch {
		case len(w.Down) > 0 && w.Unnamed:
			answer = "👎 by " + strings.Join(w.Down, ", ") + ", no row named yet"
		case len(w.Down) > 0:
			var named []int
			for n := range w.Contested {
				named = append(named, n)
			}
			sort.Ints(named)
			answer = fmt.Sprintf("👎 by %s: not %s", strings.Join(w.Down, ", "), strings.Trim(strings.Join(strings.Fields(fmt.Sprint(named)), ", "), "[]"))
		case len(w.Up) > 0:
			answer = "👍 by " + strings.Join(w.Up, ", ")
		}
		fmt.Fprintf(&weekLines, "| %s | %d of %d | %d | %s |\n", w.Name, read, len(w.Rows), falseOnes, answer)
	}
	order := func(k string) (int, string) {
		role, kind, _ := strings.Cut(k, "\x00")
		for i, r := range roleOrder {
			if r == role {
				return i, kind
			}
		}
		return len(roleOrder), kind
	}
	sort.Slice(keys, func(i, j int) bool {
		ri, ki := order(keys[i])
		rj, kj := order(keys[j])
		if ri != rj {
			return ri < rj
		}
		return ki < kj
	})
	if len(keys) == 0 {
		b.WriteString("No week's rows kept yet.\n")
	} else {
		b.WriteString("| Role | Kind of act | Drawn | Stand | False | Unproven | Not read | Agreed | Contested | Unanswered |\n|---|---|---|---|---|---|---|---|---|---|\n")
		for _, k := range keys {
			role, kind, _ := strings.Cut(k, "\x00")
			t := counts[k]
			fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d |\n", strings.TrimPrefix(roleWords(role), "the "), kindWords(Count{Role: role, Kind: kind}),
				t.drawn, t.stand, t.false_, t.unproven, t.notRead, t.agreed, t.contested, t.unanswered)
		}
		b.WriteString("\n| Week | Read | False | The maintainer's answer |\n|---|---|---|---|\n")
		b.WriteString(weekLines.String())
	}
	if len(nothing) > 0 {
		fmt.Fprintf(&b, "\nWeeks that read nothing: %s.\n", strings.Join(nothing, ", "))
	}
	if len(before) > 0 {
		fmt.Fprintf(&b, "\nWeeks written before their rows were kept, not counted: %s.\n", strings.Join(before, ", "))
	}
	return b.String()
}

// openedCount is how the issues the auditor opened fared, every week's
// (ADR-0046, amendment 2026-10-29): one closed as not planned is the
// product owner's or a person's no, a false alarm of the auditor's own.
func openedCount(all []forge.Issue) string {
	var open, done, no []string
	for _, is := range all {
		if backlog.OpenedBy(is.Body) != "auditor" {
			continue
		}
		ref := fmt.Sprintf("#%d", is.ID)
		switch {
		case !is.Closed:
			open = append(open, ref)
		case is.Reason == "not_planned":
			no = append(no, ref)
		default:
			done = append(done, ref)
		}
	}
	n := len(open) + len(done) + len(no)
	if n == 0 {
		return "\nIssues the auditor opened: none yet.\n"
	}
	refs := func(ids []string) string {
		if len(ids) == 0 {
			return ""
		}
		return " (" + strings.Join(ids, ", ") + ")"
	}
	return fmt.Sprintf("\nIssues the auditor opened: %d — closed as not planned, a false alarm: %d%s; closed otherwise: %d%s; open: %d%s.\n",
		n, len(no), refs(no), len(done), refs(done), len(open), refs(open))
}

// openDefects opens an issue for each true finding the reviewer's judge
// dropped whose cause is still on the branch, through the one way every
// role opens one (ADR-0018): `needs-triage`, at most three a week, the rest
// said; one an issue holds already is never opened again.
func openDefects(res *Result, repo string, f forge.Forge) error {
	var open []*Defect
	for i := range res.Drawn {
		if d := res.Drawn[i].Defect; d != nil && d.Issue == 0 {
			open = append(open, d)
		}
	}
	if len(open) == 0 {
		return nil
	}
	o, err := backlog.NewOpenings(f, defectsMax)
	if err != nil {
		return err
	}
	head, err := git(repo, "rev-parse", "--short", "HEAD")
	if err != nil {
		return err
	}
	for i := range res.Drawn {
		a := &res.Drawn[i]
		d := a.Defect
		if d == nil || d.Issue != 0 {
			continue
		}
		_, _, original, found := backlog.Locate(repo, d.Path, d.Quote)
		if !found {
			d.Said = fmt.Sprintf("its cause is no longer in %s: not opened", d.Path)
			continue
		}
		line, _, _ := strings.Cut(original, "\n")
		fence := backlog.FenceFor(original)
		body := fmt.Sprintf("The reviewer's %s lens found this on %s, and its judge dropped it; re-checked by the weekly sample (%s), it is true, and its cause is still here.\n\nAt `%s`:\n\n%s\n%s\n%s\n", d.Lens, a.Ref, res.Week, d.Path, fence, original, fence)
		for _, p := range a.Proofs {
			body += "\n- " + p.String()
		}
		op := backlog.Opening{Role: "auditor", Key: backlog.CodeKey(d.Path, line), Title: d.Title, Body: body, Sources: []string{d.Path}, Commit: head, Triage: true,
			Found: fmt.Sprintf("re-checking a finding the reviewer's judge dropped, in the weekly sample %s", res.Week)}
		outcome, id, err := o.Open(op)
		if err != nil {
			return err
		}
		d.Issue = id
		switch outcome {
		case backlog.Opened:
			d.Said = fmt.Sprintf("opened as #%d", id)
		case backlog.StillOpen:
			d.Said = fmt.Sprintf("held by #%d, open", id)
		case backlog.Capped:
			d.Said = fmt.Sprintf("not opened: at most %d a week", defectsMax)
		default:
			d.Said = fmt.Sprintf("held by #%d, closed", id)
		}
	}
	return nil
}

// defectsMax is the most issues a week opens for true findings dropped.
const defectsMax = 3
