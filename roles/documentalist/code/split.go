package documentalist

import (
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/docmark"
	"github.com/JN0V/workline/internal/kit/verdict"
)

// splittable are the budget problems a split answers, strongest first. A
// folder over budget needs another kind of move.
var splittable = []string{"doc-too-long", "agent-file-too-long", "section-too-long"}

// splitTask is what the judge needs to know about a split run.
type splitTask struct {
	Doc  string   `yaml:"doc"`
	Keys []string `yaml:"keys"` // the budget problems the patch must resolve
}

// splitCandidates are the docs a gardening run may split, the first with the
// strongest budget problem first: the run takes the first whose task fits.
// One doc a run keeps each change reviewable. A history doc is never one:
// moving its parts away rewrites the record (WaterMeter's CHANGELOG.md,
// chosen and refused two nights running); its budget is
// reported, for a person.
func splitCandidates(problems []Problem) []*splitTask {
	var out []*splitTask
	seen := map[string]bool{}
	for _, rule := range splittable {
		of := slices.DeleteFunc(slices.Clone(problems), func(p Problem) bool { return p.Rule != rule })
		slices.SortStableFunc(of, func(a, b Problem) int { return b.Size - a.Size }) // the most over first
		for _, p := range of {
			doc, _, _ := strings.Cut(p.Where, "#")
			if isHistory(doc) || seen[doc] {
				continue
			}
			seen[doc] = true
			out = append(out, &splitTask{Doc: doc, Keys: []string{p.Key}})
		}
	}
	return out
}

// tooLargeToRewrite says why a doc was not given to the agent whole, to be
// split or read: its task would not fit the role's context budget
// (workline's docs/spec/backlog-acts.md, 58,000 characters, 2026-10-07).
// The doc's budget problems stay, for a person, who moves whole parts of
// it out by hand.
func tooLargeToRewrite(kind, doc string, chars, max int) verdict.Finding {
	return verdict.Finding{Rule: "too-large-to-" + kind, Where: doc, Level: "warn",
		Message: fmt.Sprintf("not given to the agent to %s: its task would be %d characters, over the %d a task holds to fit the role's context budget; a person moves whole parts of it into docs of their own", kind, chars, max)}
}

// inbound lists, by doc, the lines of the other docs linking to a part of
// doc: a part that moves takes these links with it.
func inbound(t Tree, doc string) map[string][]string {
	out := map[string][]string{}
	for p, content := range t.Docs {
		for _, l := range scan(content) {
			for _, target := range linkTargets(codeSpan.ReplaceAllString(l.text, "")) {
				if file, anchor := resolve(p, target); p != doc && !l.code && file == doc && anchor != "" && !scheme.MatchString(target) {
					out[p] = append(out[p], fmt.Sprintf("%s:%d: %s", p, l.n, l.text))
					break
				}
			}
		}
	}
	return out
}

// writeSplitTask writes the question for the agent: the doc with its line
// numbers, what is over budget, the docs around it, and the lines linking
// to its parts.
func writeSplitTask(c *splitTask, problems []Problem, t Tree, s Budgets) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Kind: split\n\n%s is over its budget. Resolve this:\n\n", c.Doc)
	var others []string
	for _, p := range problems {
		switch w, _, _ := strings.Cut(p.Where, "#"); {
		case contains(c.Keys, p.Key):
			fmt.Fprintf(&b, "- %s %s: %s\n", p.Rule, p.Where, p.Message)
		case w == c.Doc && contains(splittable, p.Rule):
			others = append(others, fmt.Sprintf("- %s %s: %s", p.Rule, p.Where, p.Message))
		}
	}
	if len(others) > 0 {
		b.WriteString("\nAlso over budget in this doc — do not make these worse:\n\n" + strings.Join(others, "\n") + "\n")
	}
	b.WriteString(`
Split it into linked pages: move whole parts — a section, a list, a table —
into new pages beside it, named after it (run.md, run-people.md), and leave a
sentence and a link in their place. Move the lines as they are: they are
checked to be found, unchanged but for where their links point, in the new
pages. Do not squeeze sentences, or drop a rule written as MUST or SHOULD.
Give each new page the doc's header: its ` + "`checked`" + ` as it is — the lines
moved are the lines it vouched for — and those of its ` + "`sources`" + ` the page
speaks of; none if the doc has none. One patch, a unified diff; a new page is
created with --- /dev/null and +++ b/<path>.

A new page among the user pages answers to the reader rules (paragraphs and
cells within their word budget, every decision, doc or issue it names linked,
reachable from the docs), and the patch is refused for each problem it adds.
`)
	fmt.Fprintf(&b, "\nA new page holds at least %d words: a smaller part stays in the doc, or goes with its neighbour.\n", s.PageWords)
	in := inbound(t, c.Doc)
	if len(in) > 0 {
		b.WriteString("\nThese lines of other docs link to its parts. A link to a part you move points at its new page: change where it points, and nothing else in those docs:\n\n")
		for _, p := range slices.Sorted(maps.Keys(in)) {
			b.WriteString(strings.Join(in[p], "\n") + "\n")
		}
	}
	b.WriteString("\nThe docs next to it, with their line counts:\n\n")
	dir := path.Dir(c.Doc)
	var near []string
	for p, content := range t.Docs {
		if path.Dir(p) == dir || strings.HasPrefix(p, dir+"/") {
			near = append(near, fmt.Sprintf("- %s (%d lines)", p, lineCount(content)))
		}
	}
	sort.Strings(near)
	b.WriteString(strings.Join(near, "\n") + "\n\nThe doc as it is now, with its line numbers:\n\n```\n")
	for i, l := range strings.Split(strings.TrimSuffix(t.Docs[c.Doc], "\n"), "\n") {
		fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
	}
	b.WriteString("```\n")
	return b.String()
}

var rule = regexp.MustCompile(`\b(MUST|SHOULD)\b`)

// judgeSplit checks a split patch: the doc, new pages carrying its header,
// and the docs linking to its parts on where those links point only; quotes
// at the lines cited, every removed line found unchanged in a new page
// (headings may change level, links where they point), no MUST or SHOULD
// lost, each new page linked from the doc, the budget problems resolved,
// and no new problem anywhere.
func judgeSplit(repo string, s Settings, c *splitTask, intents, fallback []intent.Intention) ([]verdict.Finding, error) {
	if !proposedPatch(intents, fallback) {
		return nil, nil // no agent, or a note instead: the findings are reported as they are
	}
	var refused []verdict.Finding
	refuse := func(rule, where, msg string) {
		refused = append(refused, verdict.Finding{Rule: rule, Where: where, Message: msg})
	}
	tree, err := loadTree(repo, s.Docs)
	if err != nil {
		return nil, err
	}
	after := map[string]string{}
	for p, content := range tree.Docs {
		after[p] = content
	}
	files := map[string]bool{}
	for f := range tree.Files {
		files[f] = true
	}
	var created []string
	linking := inbound(tree, c.Doc)
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, so what it moves can be checked against the doc")
			continue
		}
		if _, err := gitIn(repo, intent.NormalizeDiff(diff), "apply", "--recount", "--unidiff-zero", "--check", "-"); err != nil {
			refuse("patch-does-not-apply", "patch", err.Error())
			continue
		}
		parsed, err := parseDiff(diff)
		if err != nil {
			refuse("patch-unreadable", "patch", err.Error())
			continue
		}
		for _, f := range parsed {
			isNew := !files[f.path]
			if _, err := os.Stat(filepath.Join(repo, f.path)); err == nil {
				isNew = false
			}
			switch {
			case f.path == c.Doc:
				now, bad, why := applyDoc(after[f.path], f, diff)
				if bad != "" {
					refuse(bad, f.path, why)
					continue
				}
				after[f.path] = now
			case linking[f.path] != nil:
				now, bad, why := applyDoc(after[f.path], f, diff)
				if bad == "" && !relinked(after[f.path], now) {
					bad, why = "rewritten", "a doc linking to a part that moves changes where those links point, nothing else"
				}
				if bad != "" {
					refuse(bad, f.path, why)
					continue
				}
				after[f.path] = now
			case isNew && strings.HasSuffix(f.path, ".md") && matchAny(s.Docs, f.path):
				var lines []string
				for _, h := range f.hunks {
					for _, l := range h.lines {
						if l[0] == '+' {
							lines = append(lines, l[1:])
						}
					}
				}
				after[f.path] = strings.Join(lines, "\n") + "\n"
				if bad, why := bornHeader(tree.Docs[c.Doc], after[f.path]); bad != "" {
					refuse(bad, f.path, why)
					continue
				}
				files[f.path] = true
				created = append(created, f.path)
			default:
				refuse("patch-off-task", f.path, "a split touches the doc, the new pages it creates, and the links to its parts, nothing else; new pages go under the docs folders, as .md")
			}
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	if len(created) == 0 {
		return []verdict.Finding{{Rule: "nothing-moved", Where: c.Doc, Message: "a split moves parts into new pages; none was created"}}, nil
	}
	for _, p := range created { // a page this small is a fragment: it stays with its neighbour
		_, n := docmark.Header(after[p])
		if w := words(strings.Join(strings.Split(after[p], "\n")[n:], "\n")); w < s.Budgets.PageWords {
			refuse("page-too-small", p, fmt.Sprintf("%d words, under the %d a page holds (`budgets.page-words`): a part this small stays in the doc, or goes with its neighbour", w, s.Budgets.PageWords))
		}
	}
	// Moved, not rewritten.
	moved := map[string]bool{}
	for _, p := range created {
		for _, l := range strings.Split(after[p], "\n") {
			moved[normal(l)] = true
		}
	}
	kept := map[string]int{}
	for _, l := range strings.Split(after[c.Doc], "\n") {
		kept[normal(l)]++
	}
	was := map[string]int{}
	for _, l := range strings.Split(tree.Docs[c.Doc], "\n") {
		was[normal(l)]++
	}
	var gained []string // lines the doc did not have: the links, and references updated in place
	for _, l := range strings.Split(after[c.Doc], "\n") {
		if n := normal(l); was[n] > 0 {
			was[n]--
		} else {
			gained = append(gained, n)
		}
	}
	for _, l := range strings.Split(tree.Docs[c.Doc], "\n") {
		n := normal(l)
		if n == "" {
			continue
		}
		if kept[n] > 0 {
			kept[n]--
			continue
		}
		if !moved[n] && !editedInPlace(n, gained) {
			refuse("rewritten", c.Doc, fmt.Sprintf("%q left the doc and is found in no new doc; move lines as they are", strings.TrimSpace(l)))
			break
		}
	}
	// No rule lost.
	before := len(rule.FindAllString(tree.Docs[c.Doc], -1))
	now := len(rule.FindAllString(after[c.Doc], -1))
	for _, p := range created {
		now += len(rule.FindAllString(after[p], -1))
	}
	if now < before {
		refuse("rule-lost", c.Doc, fmt.Sprintf("the doc held %d MUST or SHOULD, and %d remain", before, now))
	}
	// Each new doc is reached from the doc.
	linked := linksFrom(c.Doc, after[c.Doc])
	for _, p := range created {
		if !linked[p] {
			refuse("not-linked", p, fmt.Sprintf("%s does not link to %s; leave a sentence and a link where the part was", c.Doc, p))
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	// The budget problems resolved, and nothing new.
	old := map[string]Problem{}
	for _, p := range Hygiene(tree, s.Budgets, s.Duplicates) {
		old[p.Key] = p
	}
	found := Hygiene(Tree{Docs: after, Files: files}, s.Budgets, s.Duplicates)
	// A part of the doc over its budget already, moved whole into a new
	// doc — its text the same, word for word, and gone from the doc — is
	// the same problem moved, not a new one.
	gone := map[string]int{} // a rule and a part's text -> the parts the doc no longer holds
	stays := map[string]bool{}
	for _, p := range found {
		stays[p.Key] = true
	}
	for _, p := range old {
		if doc, _, part := strings.Cut(p.Where, "#"); part && doc == c.Doc && !stays[p.Key] {
			if text := partText(tree.Docs[c.Doc], p.Where); text != "" {
				gone[p.Rule+"\x00"+text]++
			}
		}
	}
	movedWhole := func(p Problem) bool {
		doc, _, part := strings.Cut(p.Where, "#")
		if !part || !contains(created, doc) {
			return false
		}
		key := p.Rule + "\x00" + partText(after[doc], p.Where)
		if gone[key] == 0 {
			return false
		}
		gone[key]--
		return true
	}
	for _, p := range found {
		switch prev, was := old[p.Key]; {
		case contains(c.Keys, p.Key):
			refuse("still-over-budget", p.Where, p.Rule+": "+p.Message)
		case !was && movedWhole(p):
		case p.Rule == "folder-too-long" || p.Rule == "links-not-checked": // the pages' headers and links, the links moved
		case !was || p.Size > prev.Size:
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	// The user pages too: a new page among them answers to the reader rules.
	for _, p := range readerIntroduced(tree, Tree{Docs: after, Files: files}, s.Reader) {
		refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
	}
	return refused, nil
}

// applyDoc applies one file of a patch to an existing doc, as git will: the
// lines it cites, once the engine places its hunks (placed, mended), must be
// quoted as they are, and it must leave derived blocks alone. rule and why
// say what is wrong, if anything.
func applyDoc(old string, f fileDiff, diff string) (now, rule, why string) {
	f = placed(old, f)
	if why := misquoted(old, f); why != "" {
		return "", "misquoted", why
	}
	if touchesDerived(old, f) {
		return "", "derived-block", "the patch changes lines between workline:derive markers; they are regenerated from the code, never written"
	}
	now = applyHunks(old, f)
	byGit, err := gitApplied(f.path, old, diff, false)
	if err != nil {
		// Git cannot apply the hunks as written: what lands is the doc's
		// part written anew from them as placed — the reader task's own
		// patch, or the one placedPatch wrote —, which git is asked about
		// (#592).
		var part string
		if part, _, err = writtenAnew(old, f); err == nil {
			byGit, err = gitApplied(f.path, old, part, false)
		}
	}
	if err != nil || byGit != now {
		return "", "patch-ambiguous", "git would apply this diff differently from how it reads; send a plain unified diff"
	}
	// Moving text gives no source to read: `checked` stays.
	if checkedOf(now) != checkedOf(old) {
		return "", "checked-unread", "the patch moves `checked`, but this task gives none of the doc's sources: moving text vouches for nothing, so leave the header as it is"
	}
	return now, "", ""
}

// bornHeader says what a new page's header lacks to carry the doc's, or
// "": the doc's `checked`, as it is — the lines moved are the lines it
// vouched for, and no other —, and `sources` among the doc's.
func bornHeader(doc, page string) (rule, why string) {
	var d, p docmark.Frontmatter
	dm, _ := docmark.Header(doc)
	pm, _ := docmark.Header(page)
	if yaml.Unmarshal([]byte(pm), &p) != nil || checkedOf(page) != checkedOf(doc) {
		return "checked-unread", fmt.Sprintf("the new page's `checked` must be the doc's, as it is (%q): the lines moved are the lines it vouched for, and no other", checkedOf(doc))
	}
	_ = yaml.Unmarshal([]byte(dm), &d) // a broken header is reported by ParseDoc
	for _, src := range p.Sources {
		if !slices.Contains(d.Sources, src) {
			return "sources-not-carried", "the new page names " + src + " in `sources`, which the doc does not: a page split off speaks of the doc's sources"
		}
	}
	if len(d.Sources) > 0 && len(p.Sources) == 0 {
		return "sources-not-carried", "the doc names its `sources`: give the new page those its lines speak of"
	}
	return "", ""
}

// relinked says whether a doc changed only where its links point.
func relinked(old, now string) bool {
	return slices.EqualFunc(strings.Split(old, "\n"), strings.Split(now, "\n"), func(a, b string) bool { return normal(a) == normal(b) })
}

// editedInPlace says whether a line left the doc only to come back changed a
// little — a reference to a part that moved ("below" becoming "in x.md") —
// which a gained line shows by keeping most of its words.
func editedInPlace(line string, gained []string) bool {
	words := strings.Fields(line)
	for _, g := range gained {
		has := map[string]bool{}
		for _, w := range strings.Fields(g) {
			has[w] = true
		}
		kept := 0
		for _, w := range words {
			if has[w] {
				kept++
			}
		}
		if len(words) > 0 && kept*10 >= len(words)*7 {
			return true
		}
	}
	return false
}

// partText is the text of the part of a doc a finding's place names
// (`doc#slug`), its words alone, or "".
func partText(content, where string) string {
	_, slug, _ := strings.Cut(where, "#")
	for _, s := range sections(scan(content)) {
		if s.slug == slug {
			return strings.Join(strings.Fields(s.text), " ")
		}
	}
	return ""
}

// normal is a line as a split may move it: trimmed, a heading at any level,
// and its links pointing anywhere.
func normal(l string) string {
	return inlineLink.ReplaceAllString(strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "#")), "]()")
}

// linksFrom lists the files of the repository a doc links to.
func linksFrom(doc, content string) map[string]bool {
	out := map[string]bool{}
	for _, l := range scan(content) {
		if l.code {
			continue
		}
		for _, m := range inlineLink.FindAllStringSubmatch(codeSpan.ReplaceAllString(l.text, ""), -1) {
			file, _, _ := strings.Cut(m[1], "#")
			if file == "" || scheme.MatchString(file) {
				continue
			}
			if strings.HasPrefix(file, "/") {
				out[path.Clean(strings.TrimPrefix(file, "/"))] = true
			} else {
				out[path.Join(path.Dir(doc), file)] = true
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// proposedPatch says whether the agent proposed a patch of its own.
func proposedPatch(intents, fallback []intent.Intention) bool {
	for _, in := range intents {
		if in.Kind == "patch" && !isFallback(in, fallback) {
			return true
		}
	}
	return false
}
