package documentalist

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// What the agent says it could not confirm (#449). The policy tells it that
// moving `checked` vouches for every sentence a doc keeps, and that what it
// could not confirm is kept, `checked` left, `judged` set and said in a
// `note`. On the `drifted` fixture Sonnet kept a test count its note said it
// could not check, and moved `checked` all the same, once in ten runs: kept
// taken for confirmed. A note is the agent's own doubt, so the doc a note of
// the answer names is not vouched for by it: its move of `checked` is
// withheld (`checked-over-note`), and the note said on that doc.

// doubtedDocs are the docs put before the agent that a note of its answer
// names, each with the words of the notes naming it. A note names its doc in
// `doc` ({doc, text}): its path, or an end of it no other doc of the task
// shares (`status.md`). One given as text, or whose `doc` names no doc of
// the task, names every doc of the task whose path, or such an end, its
// text holds, as the run on `drifted` wrote "In status.md, …": a doubt is
// never lost to a path written wrong.
func doubtedDocs(intents []intent.Intention, judged map[string]map[string]string) map[string][]string {
	out := map[string][]string{}
	ends := pathEnds(judged) // an end of a doc's path -> the docs of the task ending so
	for _, in := range intents {
		if in.Kind != "note" {
			continue
		}
		text, docs, ok := noteOf(in.Value)
		if !ok {
			continue
		}
		var named []string
		for _, d := range docs {
			if p := ends[d]; len(p) == 1 && !contains(named, p[0]) {
				named = append(named, p[0])
			}
		}
		if len(named) == 0 {
			named = namedIn(text, ends)
		}
		for _, d := range named {
			if !contains(out[d], text) {
				out[d] = append(out[d], text)
			}
		}
	}
	return out
}

// noteOf reads a note: its words, and the docs it names in `doc` (a path or
// a list; nil when it names none). false: not the agent's note — the
// engine's own marks, which carry no text.
func noteOf(v any) (text string, docs []string, ok bool) {
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n), nil, strings.TrimSpace(n) != ""
	case map[string]any:
		if own, _ := n["own"].(bool); own {
			return "", nil, false
		}
		if t, has := n["text"]; has && t != nil {
			text = fmt.Sprint(t) // a text YAML read as a number or a list, as it reads
		}
		var raw []any
		switch d := n["doc"].(type) {
		case string:
			raw = []any{d}
		case []any:
			raw = d
		}
		for _, r := range raw {
			if s, isText := r.(string); isText && strings.TrimSpace(s) != "" {
				docs = append(docs, path.Clean(strings.TrimPrefix(strings.TrimSpace(s), "./")))
			}
		}
		if docs == nil && strings.TrimSpace(text) == "" {
			return "", nil, false
		}
		return strings.TrimSpace(text), docs, true // with no doc, its words are read as text
	}
	return "", nil, false
}

// pathEnds lists, for each doc of the task, its path and each end of it at a
// slash (docs/clock/status.md, clock/status.md, status.md), with the docs
// ending so.
func pathEnds(judged map[string]map[string]string) map[string][]string {
	ends := map[string][]string{}
	for p := range judged {
		for rest := p; ; {
			ends[rest] = append(ends[rest], p)
			i := strings.Index(rest, "/")
			if i < 0 {
				break
			}
			rest = rest[i+1:]
		}
	}
	return ends
}

// namedIn are the docs a note's text names: an end of a path only one doc
// of the task has, standing on its own — not inside a longer path or word.
func namedIn(text string, ends map[string][]string) []string {
	var out []string
	for end, docs := range ends {
		if len(docs) != 1 || contains(out, docs[0]) {
			continue
		}
		if regexp.MustCompile(`(?:^|[^\w./-])` + regexp.QuoteMeta(end) + `(?:$|[^\w./-]|\.(?:$|\W))`).MatchString(text) {
			out = append(out, docs[0])
		}
	}
	sort.Strings(out)
	return out
}

// onlyDoubted says whether all a doc's fix had withheld is its move of
// `checked`, over a note of the answer naming it: the doc is then said not
// vouched for, its note saying why, not fixed in part.
func onlyDoubted(withheld []verdict.Finding, doc string) bool {
	doubted := false
	for _, w := range withheld {
		if w.Where != doc {
			continue
		}
		if w.Rule != "checked-over-note" {
			return false
		}
		doubted = true
	}
	return doubted
}

// doubtSaid is what the notes naming a doc say, for a person: their words,
// on one line, cut short.
func doubtSaid(notes []string) string {
	said := strings.Join(strings.Fields(strings.Join(notes, " — ")), " ")
	if r := []rune(said); len(r) > 300 {
		said = string(r[:299]) + "…"
	}
	return "“" + said + "”"
}
