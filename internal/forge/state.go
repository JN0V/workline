package forge

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// One state comment an item. The forge mails everyone following an item
// for each new comment, but not for an edit: so a role's `sticky=` key is
// a section of one comment of workline's own, edited in place, and a
// role's update mails nobody. The last writes are listed, dated, at its
// end. A second comment takes the largest sections past the forge's
// size. Readers get each section as the comment it was (splitState).
var (
	stateMark, overMark = Marker("state-comment"), Marker("state-overflow")
	historyMark         = Marker("history")
	sectionRe           = regexp.MustCompile(`(?s)<!-- workline:section (\w+) -->\n[^\n]*\n\n(.*?)\n<!-- workline:section-end -->`)
	stickyRe            = regexp.MustCompile(`<!-- workline:sticky=[^>]*-->`)
)

const stateLimit, historyKept = 65536, 10 // GitHub's most characters a comment

// isSection: a role's `sticky=` key, but those waiting on a person, kept
// comments of their own until their calls are built (#2350, #2351).
func isSection(marker string) bool {
	return strings.HasPrefix(marker, "<!-- workline:sticky=") && !strings.HasSuffix(marker, "/state -->") && !strings.HasSuffix(marker, "/needs-spent -->")
}

// section: its marker, its text as the comment it was, and the digest of
// what workline wrote; one that no longer matches is someone's edit.
type section struct{ marker, content, sum string }

func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:8]) }

// sections reads a state comment's sections and history; an older comment
// of a key, as one section.
func sections(n Note, own Own) (out []section, hist []string) {
	if !strings.Contains(n.Body, stateMark) && !strings.Contains(n.Body, overMark) {
		ms := stickyRe.FindAllString(n.Body, -1)
		if len(ms) == 0 || !isSection(ms[len(ms)-1]) {
			return nil, nil
		}
		s := section{ms[len(ms)-1], n.Body, digest(n.Body)}
		if n.Editor != "" && !own.Wrote(Note{Author: n.Editor}) {
			s.sum = "edited"
		}
		return []section{s}, nil
	}
	for _, m := range sectionRe.FindAllStringSubmatch(n.Body, -1) {
		if ms := stickyRe.FindAllString(m[2], -1); len(ms) > 0 {
			out = append(out, section{ms[len(ms)-1], m[2], m[1]})
		}
	}
	_, h, _ := strings.Cut(n.Body, historyMark)
	for _, l := range strings.Split(h, "\n") {
		if strings.HasPrefix(l, "- ") {
			hist = append(hist, l)
		}
	}
	return out, hist
}

// splitState hands each section to readers as the comment it was: its
// author, its id; edited only when its text is no longer what workline
// wrote, another's edit leaving the other sections read. A role's record
// is read only from a comment of workline's own that no one else edited:
// a record lets a role skip work it already judged, so text a person
// wrote into it must never stand for workline's judgement.
func splitState(notes []Note) []Note {
	var out []Note
	for _, n := range notes {
		if !strings.Contains(n.Body, stateMark) && !strings.Contains(n.Body, overMark) {
			out = append(out, n)
			continue
		}
		secs, _ := sections(n, Own{})
		for _, s := range secs {
			v := n
			v.Body, v.Editor, v.section = s.content, "", intact
			if s.sum != digest(s.content) {
				v.Editor, v.section = cmp.Or(n.Editor, "someone"), altered
			}
			out = append(out, v)
		}
	}
	return out
}

const intact, altered = 1, 2 // Note.section

// statePlan: the main comment's body and id ("" to post it), the
// overflow's (body "" for none), and the comments folded in, to delete.
type statePlan struct {
	main, mainID, over, overID string
	stage                      string // the overflow written first, holding the sections leaving it too
	drop                       []string
	written                    bool // the section there already, as asked
}

// planState plans body written as marker's section on the comments notes;
// false when nothing is to write. Another's comment carrying the marker
// is left, and the section written beside it.
func planState(notes []Note, own Own, body, marker string, create bool, now time.Time) (statePlan, bool) {
	var p statePlan
	var secs, folded, oldOver []section
	var hist []string
	notes = slices.Clone(notes) // the overflow read last: of a section in both, the main comment's
	slices.SortStableFunc(notes, func(a, b Note) int {
		return cmp.Compare(strings.Count(a.Body, overMark), strings.Count(b.Body, overMark))
	})
	for _, n := range notes {
		s, h := sections(n, own)
		state, over := strings.Contains(n.Body, stateMark), strings.Contains(n.Body, overMark)
		switch {
		case !own.Wrote(n):
			create = create || strings.Contains(n.Body, marker)
			continue
		case state && p.mainID == "":
			p.mainID = n.ID
		case over && p.overID == "":
			p.overID, oldOver = n.ID, s
		case state || over || len(s) > 0: // an older one, folded in
			p.drop, folded = append(p.drop, n.ID), append(folded, s...)
			s = nil
		}
		for _, x := range s { // a section in both comments, of a write cut short, once
			if !slices.ContainsFunc(secs, func(o section) bool { return o.marker == x.marker }) {
				secs = append(secs, x)
			}
		}
		hist = append(hist, h...)
	}
	k := len(secs) // the state comment's own sections first, then the last folded of a key
	for _, x := range folded {
		if i := slices.IndexFunc(secs, func(o section) bool { return o.marker == x.marker }); i < 0 {
			secs = append(secs, x)
		} else if i >= k {
			secs[i] = x
		}
	}
	if n := utf8.RuneCountInString(body); n > stateLimit-2048 { // a shorter comment, not a failed write
		body = string([]rune(body)[:stateLimit-2048]) + fmt.Sprintf("\n\n_(Cut: %d characters past the forge's limit.)_", n-stateLimit+2048)
	}
	content := body + "\n\n" + marker
	at := slices.IndexFunc(secs, func(o section) bool { return o.marker == marker })
	same := at >= 0 && secs[at].content == content && secs[at].sum == digest(content)
	p.written = same
	if (same || at < 0 && !create) && len(p.drop) == 0 {
		return p, false
	}
	if !same && at >= 0 {
		secs[at] = section{marker, content, digest(content)}
	} else if !same && create {
		secs = append(secs, section{marker, content, digest(content)})
	}
	if !same && (at >= 0 || create) {
		role, _, _ := strings.Cut(strings.TrimPrefix(marker, "<!-- workline:sticky="), "/")
		hist = append(hist, fmt.Sprintf("- %s %s: %s", now.UTC().Format("2006-01-02 15:04 UTC"), role, sentence(body)))
	}
	hist = hist[max(0, len(hist)-historyKept):]
	// The history cut first, then the largest section the overflow still
	// holds moves there, at most one overflow; none fitting, the largest
	// is cut: a shorter comment, not a failed write.
	var over []section
	for {
		h := hist
		for len(h) > 0 && utf8.RuneCountInString(render(stateMark, secs, h)) > stateLimit {
			h = h[1:]
		}
		p.main = render(stateMark, secs, h)
		excess := utf8.RuneCountInString(p.main) - stateLimit
		if excess <= 0 {
			break
		}
		big, fit := 0, -1
		for j := range secs {
			if len(secs[j].content) > len(secs[big].content) {
				big = j
			}
			if len(secs) > 1 && utf8.RuneCountInString(render(overMark, append(slices.Clone(over), secs[j]), nil)) <= stateLimit && (fit < 0 || len(secs[j].content) > len(secs[fit].content)) {
				fit = j
			}
		}
		if fit >= 0 {
			over, secs = append(over, secs[fit]), slices.Delete(secs, fit, fit+1)
		} else if c := clip(secs[big], excess); c.content != secs[big].content {
			secs[big] = c
		} else {
			break // nothing left to cut
		}
	}
	if len(over) > 0 {
		p.over = render(overMark, over, nil)
	}
	// Sections leaving the overflow stay in it until the main comment
	// holding them is written: each is in a comment at every step.
	var leaving []section
	for _, x := range oldOver {
		if !slices.ContainsFunc(over, func(o section) bool { return o.marker == x.marker }) {
			leaving = append(leaving, x)
		}
	}
	if len(over) > 0 && len(leaving) > 0 {
		if st := render(overMark, append(slices.Clone(over), leaving...), nil); utf8.RuneCountInString(st) <= stateLimit {
			p.stage = st
		}
	}
	return p, true
}

// clip cuts a section's text by n characters, and the line saying so,
// its marker kept, so its record is still found.
func clip(s section, n int) section {
	body := []rune(strings.TrimSuffix(s.content, "\n\n"+s.marker))
	keep := max(0, len(body)-n-64)
	c := string(body[:keep]) + fmt.Sprintf("\n\n_(Cut: %d characters past the forge's limit.)_\n\n", len(body)-keep) + s.marker
	if len(c) >= len(s.content) {
		return s // nothing left to cut
	}
	if s.sum == digest(s.content) {
		s.sum = digest(c)
	}
	s.content = c
	return s
}

// render: the mark, each section headed by its role's name, the history.
func render(mark string, secs []section, hist []string) string {
	b := mark
	for _, s := range secs {
		role, key, _ := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(s.marker, "<!-- workline:sticky="), " -->"), "/")
		b += fmt.Sprintf("\n<!-- workline:section %s -->\n### %s (%s)\n\n%s\n<!-- workline:section-end -->\n", s.sum, role, key, s.content)
	}
	if len(hist) > 0 {
		b += fmt.Sprintf("\n%s\n**History** (UTC)\n\n%s", historyMark, strings.Join(hist, "\n"))
	}
	return b
}

// sentence: a body's first line a person reads, to its first full stop.
func sentence(body string) string {
	for _, l := range strings.Split(body, "\n") {
		if l = strings.TrimSpace(strings.TrimLeft(l, "#>-| \t")); l != "" && !strings.HasPrefix(l, "<!--") && !strings.HasPrefix(l, "```") {
			if i := strings.Index(l, ". "); i > 0 {
				l = l[:i+1]
			}
			if r := []rune(l); len(r) > 160 {
				l = string(r[:159]) + "…"
			}
			return Unmarked(l)
		}
	}
	return "wrote its section."
}

// vouched marks, in the comments a write reads, what one who is not
// workline's own account edited, so it is never written again as
// workline's: an older comment's editor, and each section of a state
// comment no longer as workline's own account last wrote it, as last
// says (nil where the forge keeps no versions: every section then). A
// section's digest alone does not tell: anyone may compute it.
func vouched(notes []Note, by map[string]string, own Own, last func(Note) (string, error)) error {
	for i, n := range notes {
		if who := by[n.ID]; who != "" && !own.Wrote(Note{Author: who}) {
			notes[i].Editor = who
			if !strings.Contains(n.Body, stateMark) && !strings.Contains(n.Body, overMark) {
				continue
			}
			l := ""
			if last != nil {
				var err error
				if l, err = last(n); err != nil {
					return err
				}
			}
			notes[i].Body = sectionRe.ReplaceAllStringFunc(n.Body, func(sec string) string {
				m := sectionRe.FindStringSubmatch(sec)
				if ownWrote(l, m[2]) {
					return sec
				}
				return strings.Replace(sec, "section "+m[1]+" -->", "section edited -->", 1)
			})
		}
	}
	return nil
}

// ownWrote says whether last, a state comment as workline's own account
// last wrote it, holds the section content as it is.
func ownWrote(last, content string) bool {
	return slices.ContainsFunc(sectionRe.FindAllStringSubmatch(last, -1), func(m []string) bool { return m[2] == content })
}

// knowsOwn: who workline writes as is read, so its state comment, and
// the comments to fold, are told from another's.
func (g *github) knowsOwn() bool { _, err := g.own(); return err == nil }
func (g *gitlab) knowsOwn() bool { o, err := g.own(); return err == nil && len(o.logins) > 0 }

// stateOps are the calls a forge writes the state comment with.
type stateOps struct {
	list       func() ([]Note, error)
	post, drop func(string) error
	edit       func(id, body string) error
}

// apply writes the overflow, the main comment, then deletes those folded
// in, or an overflow left empty: a section moving to the overflow is
// written there before the main comment drops it, and one leaving it
// kept there (stage) until the main comment holds it: nothing lost
// between, whichever write fails.
func (p statePlan) apply(o stateOps) error {
	var writes [][2]string
	if p.over != "" {
		writes = append(writes, [2]string{p.overID, cmp.Or(p.stage, p.over)})
	} else if p.overID != "" {
		p.drop = append(p.drop, p.overID)
	}
	if p.main != "" { // "": nothing to write
		writes = append(writes, [2]string{p.mainID, p.main})
	}
	if p.stage != "" {
		writes = append(writes, [2]string{p.overID, p.over})
	}
	for _, c := range writes {
		write := func() error { return o.post(c[1]) }
		if c[0] != "" {
			write = func() error { return o.edit(c[0], c[1]) }
		}
		if err := write(); err != nil {
			return err
		}
	}
	for _, id := range p.drop {
		if err := o.drop(id); err != nil {
			return err
		}
	}
	return nil
}

// writeState reads the comments just before a write and back after: its
// section lost to another writer meanwhile, it writes once more; lost
// again, it fails, and the next run writes it.
func writeState(o stateOps, own Own, body, marker string, create bool) error {
	for try := 0; ; try++ {
		notes, err := o.list()
		if err != nil {
			return err
		}
		p, ok := planState(notes, own, body, marker, create, time.Now())
		if !ok || try > 0 && p.written { // read back; a folded one the token may not delete left
			return nil
		}
		if try == 2 {
			return fmt.Errorf("its section %s was lost twice from workline's state comment: another wrote it meanwhile", marker)
		}
		if err := p.apply(o); err != nil {
			return err
		}
	}
}
