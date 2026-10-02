package documentalist

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// mergeCardTask is what the judge needs to know about a merge-card run: the
// card too short to stand alone, the cards it may go into, and the docs
// linking to it, whose links must follow it.
type mergeCardTask struct {
	Card    string   `yaml:"card"`
	Cards   []string `yaml:"cards"`
	Linking []string `yaml:"linking"`
}

// mergeCardChars caps what the cards around the short one take in the task.
const mergeCardChars = 30000

// pickMergeCard chooses the one card a gardening run merges: the first too
// short, when another card exists to take it — those of its folder first.
func pickMergeCard(problems []Problem, t Tree) *mergeCardTask {
	for _, p := range problems {
		if p.Rule != "card-too-short" {
			continue
		}
		m := &mergeCardTask{Card: p.Where}
		var near, far []string
		for d, content := range t.Docs {
			switch {
			case d != p.Where && docType(content) == "card" && path.Dir(d) == path.Dir(p.Where):
				near = append(near, d)
			case d != p.Where && docType(content) == "card":
				far = append(far, d)
			}
			if d != p.Where && linksFrom(d, t.Docs[d])[p.Where] {
				m.Linking = append(m.Linking, d)
			}
		}
		sort.Strings(near)
		sort.Strings(far)
		size := 0
		for _, d := range append(near, far...) {
			if size += len(t.Docs[d]); size > mergeCardChars && len(m.Cards) > 0 {
				break
			}
			m.Cards = append(m.Cards, d)
		}
		sort.Strings(m.Linking)
		if len(m.Cards) > 0 {
			return m
		}
	}
	return nil
}

// writeMergeCardTask writes the question for the agent: the short card, the
// cards it may go into, and the docs linking to it, with their line numbers.
func writeMergeCardTask(m *mergeCardTask, problems []Problem, t Tree) string {
	var b strings.Builder
	b.WriteString("Kind: merge-card\n\n")
	for _, p := range problems {
		if p.Rule == "card-too-short" && p.Where == m.Card {
			fmt.Fprintf(&b, "%s is too short to stand alone: %s\n", m.Card, p.Message)
		}
	}
	fmt.Fprintf(&b, `
Merge it into the one card below it belongs with: add its text there, as it
is, under a heading one level below that card's title, delete it (--- a/%s then
+++ /dev/null), and point every link to it at the card it went into. Touch
nothing else, and say nothing new. One patch, a unified diff whose hunks cite
the docs' lines by the numbers shown here. If no card is the right place,
return a note instead.
`, m.Card)
	show := func(title, p string) {
		fmt.Fprintf(&b, "\n%s %s, with its line numbers:\n\n```\n", title, p)
		for i, l := range strings.Split(strings.TrimSuffix(t.Docs[p], "\n"), "\n") {
			fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
		}
		b.WriteString("```\n")
	}
	show("The short card,", m.Card)
	for _, p := range m.Cards {
		show("A card it may go into,", p)
	}
	for _, p := range m.Linking {
		show("A doc linking to it,", p)
	}
	return b.String()
}

// judgeMergeCard checks a merge-card patch: the short card deleted, its text
// found in one card, which loses nothing; the docs linking to it changed on
// those links only; no MUST or SHOULD lost; and no new problem anywhere, a
// link left pointing at the deleted card included.
func judgeMergeCard(repo string, s Settings, m *mergeCardTask, intents, fallback []intent.Intention) ([]verdict.Finding, error) {
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
	deleted := false
	var into []string
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, so what it moves can be checked against the cards")
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
			switch {
			case f.path == m.Card && f.deleted:
				delete(after, f.path)
				delete(files, f.path)
				deleted = true
				continue
			case f.path == m.Card:
				refuse("card-kept", f.path, "the short card is deleted once its text is in the card it goes into (+++ /dev/null)")
				continue
			case f.deleted || !contains(m.Cards, f.path) && !contains(m.Linking, f.path):
				refuse("patch-off-task", f.path, "merging a card touches the card, the card it goes into, and the docs linking to it, nothing else")
				continue
			}
			now, bad, why := applyDoc(after[f.path], f, diff)
			if bad != "" {
				refuse(bad, f.path, why)
				continue
			}
			after[f.path] = now
			if contains(m.Cards, f.path) && !contains(m.Linking, f.path) {
				into = append(into, f.path)
			}
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	if !deleted || len(into) == 0 {
		return []verdict.Finding{{Rule: "nothing-merged", Where: m.Card, Message: "merging a card adds its text to another card and deletes it; this patch does not do both"}}, nil
	}
	if len(into) > 1 {
		return []verdict.Finding{{Rule: "several-cards", Where: m.Card, Message: fmt.Sprintf("the card goes into one card, not %d: %s", len(into), strings.Join(into, ", "))}}, nil
	}
	target := into[0]
	// The card's text is found in the card it went into.
	has := map[string]bool{}
	for _, w := range normalWords(after[target]) {
		has[w] = true
	}
	for _, l := range scan(tree.Docs[m.Card]) {
		ws := normalWords(l.text)
		found := 0
		for _, w := range ws {
			if has[w] {
				found++
			}
		}
		if found*10 < len(ws)*7 {
			refuse("rewritten", m.Card, fmt.Sprintf("%q is not in %s; move the card's text as it is", strings.TrimSpace(l.text), target))
			break
		}
	}
	// The card it went into only gains; the docs linking to it change their links only.
	for _, p := range append([]string{target}, m.Linking...) {
		if after[p] == tree.Docs[p] {
			continue
		}
		var gained []string
		was := map[string]int{}
		for _, l := range strings.Split(tree.Docs[p], "\n") {
			was[normal(l)]++
		}
		for _, l := range strings.Split(after[p], "\n") {
			if n := normal(l); was[n] > 0 {
				was[n]--
			} else {
				gained = append(gained, n)
			}
		}
		kept := map[string]int{}
		for _, l := range strings.Split(after[p], "\n") {
			kept[normal(l)]++
		}
		for _, l := range strings.Split(tree.Docs[p], "\n") {
			n := normal(l)
			if n == "" {
				continue
			}
			if kept[n] > 0 {
				kept[n]--
				continue
			}
			if p != target && strings.Contains(l, path.Base(m.Card)) {
				continue // a link to the card, pointed elsewhere
			}
			if !editedInPlace(n, gained) {
				refuse("rewritten", p, fmt.Sprintf("%q left %s; only the card's text is added there, and links to it changed", strings.TrimSpace(l), p))
				break
			}
		}
	}
	before, now := len(rule.FindAllString(tree.Docs[m.Card], -1)), 0
	for _, p := range append([]string{target}, m.Linking...) {
		before += len(rule.FindAllString(tree.Docs[p], -1))
		now += len(rule.FindAllString(after[p], -1))
	}
	if now < before {
		refuse("rule-lost", m.Card, fmt.Sprintf("the card and the docs around it held %d MUST or SHOULD, and %d remain", before, now))
	}
	if len(refused) > 0 {
		return refused, nil
	}
	old := map[string]Problem{}
	for _, p := range Hygiene(tree, s.Budgets, s.Duplicates) {
		old[p.Key] = p
	}
	for _, p := range Hygiene(Tree{Docs: after, Files: files}, s.Budgets, s.Duplicates) {
		if prev, was := old[p.Key]; !was || p.Size > prev.Size {
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	return refused, nil
}
