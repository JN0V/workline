package documentalist

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// condensable are the budget problems condensing answers, strongest first.
// A folder over budget, or a card under it, needs another kind of move.
var condensable = []string{"doc-too-long", "agent-file-too-long", "card-too-long", "section-too-long"}

// condenseTask is what the judge needs to know about a condense run.
type condenseTask struct {
	Doc  string   `yaml:"doc"`
	Keys []string `yaml:"keys"` // the budget problems the patch must resolve
}

// pickCondense chooses the one doc a gardening run condenses: the first with
// the strongest budget problem. One doc a run keeps each change reviewable.
func pickCondense(problems []Problem) *condenseTask {
	for _, rule := range condensable {
		for _, p := range problems {
			if p.Rule != rule {
				continue
			}
			doc, _, _ := strings.Cut(p.Where, "#")
			return &condenseTask{Doc: doc, Keys: []string{p.Key}}
		}
	}
	return nil
}

// writeCondenseTask writes the question for the agent: the doc with its line
// numbers, what is over budget, and the docs around it.
func writeCondenseTask(c *condenseTask, problems []Problem, t Tree) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Kind: condense\n\n%s is over its budget. Resolve this:\n\n", c.Doc)
	var others []string
	for _, p := range problems {
		switch w, _, _ := strings.Cut(p.Where, "#"); {
		case contains(c.Keys, p.Key):
			fmt.Fprintf(&b, "- %s %s: %s\n", p.Rule, p.Where, p.Message)
		case w == c.Doc && contains(condensable, p.Rule):
			others = append(others, fmt.Sprintf("- %s %s: %s", p.Rule, p.Where, p.Message))
		}
	}
	if len(others) > 0 {
		b.WriteString("\nAlso over budget in this doc — do not make these worse:\n\n" + strings.Join(others, "\n") + "\n")
	}
	b.WriteString(`
Bring it within budget by moving whole parts — a section, a list, a table —
into a new doc, and leaving a sentence and a link in their place. Move the
lines as they are: they are checked to be found, unchanged, in the new doc.
Do not squeeze sentences, drop a rule written as MUST or SHOULD, or touch any
other existing doc. One patch, a unified diff; a new doc is created with
--- /dev/null and +++ b/<path>.

The docs next to it, with their line counts:

`)
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

// judgeCondense checks a condense patch: the doc and new docs only, quotes at
// the lines cited, every removed line found unchanged in a new doc (headings
// may change level), no MUST or SHOULD lost, each new doc linked from the
// doc, the budget problems resolved, and no new problem anywhere.
func judgeCondense(repo string, s Settings, c *condenseTask, intents, fallback []intent.Intention) ([]verdict.Finding, error) {
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
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, so what it moves can be checked against the doc")
			continue
		}
		if _, err := gitIn(repo, intent.NormalizeDiff(diff), "apply", "--recount", "--check", "-"); err != nil {
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
				old := after[f.path]
				if why := misquoted(old, f); why != "" {
					refuse("misquoted", f.path, why)
					continue
				}
				if touchesDerived(old, f) {
					refuse("derived-block", f.path, "the patch changes lines between workline:derive markers; they are regenerated from the code, never written")
					continue
				}
				now := applyHunks(old, f)
				if byGit, err := gitApplied(f.path, old, diff, false); err != nil || byGit != now {
					refuse("patch-ambiguous", f.path, "git would apply this diff differently from how it reads; send a plain unified diff")
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
				files[f.path] = true
				created = append(created, f.path)
			default:
				refuse("patch-off-task", f.path, "condensing touches the doc and the new docs it creates, nothing else; new docs go under the docs folders, as .md")
			}
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	if len(created) == 0 {
		return []verdict.Finding{{Rule: "nothing-moved", Where: c.Doc, Message: "condensing moves parts into a new doc; none was created"}}, nil
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
	for _, p := range Hygiene(Tree{Docs: after, Files: files}, s.Budgets, s.Duplicates) {
		switch prev, was := old[p.Key]; {
		case contains(c.Keys, p.Key):
			refuse("still-over-budget", p.Where, p.Rule+": "+p.Message)
		case !was || p.Size > prev.Size:
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	return refused, nil
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

// normal is a line as condensing may move it: trimmed, and a heading at any level.
func normal(l string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "#"))
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
