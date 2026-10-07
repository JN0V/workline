package productowner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
)

// due is an open issue pre may read, with what it knows of it.
type due struct {
	is       forge.Issue
	st       *backlog.State
	comments []string
	notes    []forge.Note // the comments with their authors
}

// changesFound reads, with no agent, what open issues were built on that
// changed (ADR-0032): an issue's Need or Scope a person rewrote since its
// state kept them, and the lines of a file an issue was imported from,
// changed by a commit since it was last read. front holds, for each issue
// to read first, the change it is read with; rebase, the issues whose
// state is to keep their sections now — rewritten, or never kept.
// unread are the imported issues whose lines git could not follow, said
// in the run's findings, never read as unchanged. A file archived — no
// longer a source, as the setting says — flags nothing.
func changesFound(repo string, open []forge.Issue, report int, all []due, archived []string) (found []backlog.Change, front map[int]string, rebase []due, unread []verdict.Finding) {
	front = map[int]string{}
	slices.SortFunc(all, func(a, b due) int { return a.is.ID - b.is.ID })
	sources := map[int][]string{}
	for _, d := range all {
		sources[d.is.ID] = d.st.Sources
	}
	var others []forge.Issue
	for _, is := range open {
		if is.ID != report {
			others = append(others, is)
		}
	}
	for _, d := range all {
		what := backlog.Rewritten(d.st, d.is.Body)
		switch {
		case len(what) > 0:
			rebase = append(rebase, d)
			now := backlog.Basis(d.is.Body)
			var was, is []string
			for _, name := range what {
				was = append(was, "## "+name+"\n\n"+d.st.Sections[name])
				is = append(is, "## "+name+"\n\n"+now[name])
			}
			c := backlog.Change{Issue: d.is.ID, What: what, Touch: backlog.Touched(d.is.ID, others, sources, slices.Contains(what, "Scope")),
				Was: strings.Join(was, "\n\n"), Now: strings.Join(is, "\n\n")}
			if len(c.Touch) == 0 {
				break // nothing built on it: its sections kept, nothing flagged
			}
			found = append(found, c)
			for _, t := range c.Touch {
				if t.How == backlog.TouchPart {
					front[t.Issue] += changeBlock(fmt.Sprintf("#%d, which it is a part of, had its %s rewritten by a person", d.is.ID, strings.Join(what, " and ")), c)
				}
			}
		case d.st.Sections == nil && backlog.Basis(d.is.Body) != nil:
			rebase = append(rebase, d) // its sections kept from now on, once
		}
		path, from, to, ok := backlog.Imported(d.is.Body)
		if !ok || d.st.Confirmed == "" || pathglob.Any(archived, path) {
			continue
		}
		base := d.st.Judged
		if base == "" {
			base = d.st.Confirmed
		}
		was, now, nfrom, nto, changed, err := backlog.LinesChange(repo, path, from, to, d.st.Confirmed, base, "HEAD")
		if err != nil {
			unread = append(unread, verdict.Finding{Rule: "lines-unread", Level: "warn", Where: fmt.Sprintf("#%d", d.is.ID),
				Message: fmt.Sprintf("whether the lines of %s it was imported from changed could not be read (%v): not said unchanged; a full clone reads them", path, err)})
			continue
		}
		if !changed {
			continue
		}
		lines := fmt.Sprintf("%d to %d", nfrom, nto)
		if now == "" {
			lines = "gone"
		}
		c := backlog.Change{Issue: d.is.ID, Path: path, Lines: lines,
			Touch: []backlog.Touch{{Issue: d.is.ID, How: backlog.TouchImport}}, Was: was, Now: now}
		found = append(found, c)
		front[d.is.ID] += changeBlock(fmt.Sprintf("the lines of `%s` it was opened from changed (now: %s)", path, lines), c)
	}
	return found, front, rebase, unread
}

// changeBlock is what an issue read again for a change is given with it:
// the change, the text before and after, and what the agent may propose.
func changeBlock(what string, c backlog.Change) string {
	quote := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "> (nothing: gone)"
		}
		var out []string
		for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
			out = append(out, strings.TrimRight("> "+l, " "))
		}
		return strings.Join(out, "\n")
	}
	return fmt.Sprintf("Read again for a change: %s. Every act you propose on it goes to a person, never done: say what the change asks of it — `unready` to move it back to refine when it no longer fits, a `refine` for what it lacks, or nothing.\nWas:\n%s\nNow:\n%s\n", what, quote(c.Was), quote(c.Now))
}

// changeLine says a change in a finding.
func changeLine(c backlog.Change) string {
	var touched []string
	for _, t := range c.Touch {
		state := "listed for a person"
		if t.Read {
			state = "read again"
		}
		touched = append(touched, fmt.Sprintf("#%d (%s, %s)", t.Issue, t.How, state))
	}
	what := fmt.Sprintf("a person rewrote its %s", strings.Join(c.What, " and "))
	if c.Path != "" {
		what = fmt.Sprintf("the lines of %s it was opened from changed", c.Path)
	}
	if len(touched) == 0 {
		return what + ": no open issue built on it"
	}
	return what + ": " + strings.Join(touched, ", ") + "; nothing written to them, every act proposed"
}
