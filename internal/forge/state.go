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

// One state comment an item, as the decision record ADR-0076
// (docs/adr/0076-*.md) sets it, its §1 the one comment and §2 its dated
// history: a role's `sticky=` key is a section of one comment of
// workline's own, edited in place, so an update mails nobody; the last writes dated at its end. A second comment takes
// the largest sections past the forge's size. Readers get each section as
// the comment it was (splitState).
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
// wrote, another's edit leaving the other sections read: a role's record
// is read only from a comment of workline's own that no one else edited
// (ADR-0013, "what was judged is not asked again", docs/adr/0013-*.md).
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
	drop                       []string
	written                    bool // the section there already, as asked
}

// planState plans body written as marker's section on the comments notes;
// false when nothing is to write. Another's comment carrying the marker
// is left, and the section written beside it.
func planState(notes []Note, own Own, body, marker string, create bool, now time.Time) (statePlan, bool) {
	var p statePlan
	var secs, folded []section
	var hist []string
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
			p.overID = n.ID
		case state || over || len(s) > 0: // an older one, folded in
			p.drop, folded = append(p.drop, n.ID), append(folded, s...)
			s = nil
		}
		secs, hist = append(secs, s...), append(hist, h...)
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
	// The history cut first, then the largest sections to the overflow.
	var over []section
	for {
		h := hist
		for len(h) > 0 && utf8.RuneCountInString(render(stateMark, secs, h)) > stateLimit {
			h = h[1:]
		}
		if p.main = render(stateMark, secs, h); utf8.RuneCountInString(p.main) <= stateLimit || len(secs) == 1 {
			break
		}
		i := 0
		for j := range secs {
			if len(secs[j].content) > len(secs[i].content) {
				i = j
			}
		}
		over, secs = append(over, secs[i]), slices.Delete(secs, i, i+1)
	}
	if len(over) > 0 {
		p.over = render(overMark, over, nil)
	}
	return p, true
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

// apply writes the main comment, the overflow, then deletes those folded
// in, or an overflow left empty: nothing lost between.
func (p statePlan) apply(o stateOps) (err error) {
	for _, c := range [][2]string{{p.mainID, p.main}, {p.overID, p.over}} {
		switch {
		case c[1] == "" && c[0] != "":
			p.drop = append(p.drop, c[0])
		case c[0] == "" && c[1] != "":
			err = o.post(c[1])
		case c[1] != "":
			err = o.edit(c[0], c[1])
		}
		if err != nil {
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
