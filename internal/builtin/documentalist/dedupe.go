package documentalist

import (
	"fmt"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// dedupeTask is what the judge needs to know about a duplicates run: the two
// docs holding the passage, and the problem the patch must resolve.
type dedupeTask struct {
	Docs []string `yaml:"docs"`
	Key  string   `yaml:"key"`
}

// pickDedupe chooses the one repeated passage a gardening run merges: the
// first found. Two copies drift apart; one kept, and linked, cannot.
func pickDedupe(problems []Problem) *dedupeTask {
	for _, p := range problems {
		if p.Rule == "duplicate" && p.Other != "" {
			return &dedupeTask{Docs: []string{p.Where, p.Other}, Key: p.Key}
		}
	}
	return nil
}

// writeDedupeTask writes the question for the agent: the repeated passage,
// and both docs with their line numbers.
func writeDedupeTask(d *dedupeTask, problems []Problem, t Tree) string {
	var b strings.Builder
	b.WriteString("Kind: duplicates\n\nThe same passage is written in two docs:\n\n")
	for _, p := range problems {
		if p.Key == d.Key {
			fmt.Fprintf(&b, "- %s: %s\n", p.Where, p.Message)
		}
	}
	b.WriteString(`
Keep the passage in the one doc it belongs to, as it is, and in the other
replace it with a sentence and a link to where it is kept (to its heading).
If the two copies differ, keep what is true of both in the one kept, and say
nothing new. Touch no other doc. One patch, a unified diff whose hunks cite
the docs' lines by the numbers shown here.
`)
	for _, p := range d.Docs {
		fmt.Fprintf(&b, "\n%s, with its line numbers:\n\n```\n", p)
		for i, l := range strings.Split(strings.TrimSuffix(t.Docs[p], "\n"), "\n") {
			fmt.Fprintf(&b, "%4d | %s\n", i+1, l)
		}
		b.WriteString("```\n")
	}
	return b.String()
}

// judgeDedupe checks a duplicates patch: the two docs only, quoted at the
// lines cited; what leaves one doc is found in the other; no MUST or SHOULD
// lost; the doc that gave the passage up links to the one keeping it; the
// duplicate gone, and no new problem anywhere, dead links included.
func judgeDedupe(repo string, s Settings, d *dedupeTask, intents, fallback []intent.Intention) ([]verdict.Finding, error) {
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
	for _, in := range intents {
		if in.Kind != "patch" || isFallback(in, fallback) {
			continue
		}
		diff, ok := in.Value.(string)
		if !ok {
			refuse("patch-not-diff", "patch", "send a unified diff, so what it replaces can be checked against the docs")
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
			if !contains(d.Docs, f.path) {
				refuse("patch-off-task", f.path, "merging a repeated passage touches the two docs holding it, nothing else")
				continue
			}
			now, bad, why := applyDoc(after[f.path], f, diff)
			if bad != "" {
				refuse(bad, f.path, why)
				continue
			}
			after[f.path] = now
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	// What leaves a doc is the repeated passage, found in the other one.
	gave := ""
	for i, p := range d.Docs {
		other := d.Docs[1-i]
		kept := map[string]bool{}
		for _, w := range normalWords(after[other]) {
			kept[w] = true
		}
		now := map[string]int{}
		for _, l := range strings.Split(after[p], "\n") {
			now[normal(l)]++
		}
		for _, l := range strings.Split(tree.Docs[p], "\n") {
			n := normal(l)
			if n == "" {
				continue
			}
			if now[n] > 0 {
				now[n]--
				continue
			}
			gave = p
			ws := normalWords(l)
			found := 0
			for _, w := range ws {
				if kept[w] {
					found++
				}
			}
			if found*10 < len(ws)*7 {
				refuse("rewritten", p, fmt.Sprintf("%q left %s and is not in %s; only the repeated passage moves out", strings.TrimSpace(l), p, other))
				break
			}
		}
	}
	if gave == "" {
		refuse("nothing-merged", d.Docs[0], "the passage is still written in both docs")
	}
	before, now := 0, 0
	for _, p := range d.Docs {
		before += len(rule.FindAllString(tree.Docs[p], -1))
		now += len(rule.FindAllString(after[p], -1))
	}
	if now < before {
		refuse("rule-lost", d.Docs[0], fmt.Sprintf("the two docs held %d MUST or SHOULD, and %d remain", before, now))
	}
	if gave != "" {
		keeper := d.Docs[0]
		if keeper == gave {
			keeper = d.Docs[1]
		}
		if !linksFrom(gave, after[gave])[keeper] {
			refuse("not-linked", gave, fmt.Sprintf("%s does not link to %s, where the passage is kept", gave, keeper))
		}
	}
	if len(refused) > 0 {
		return refused, nil
	}
	old := map[string]Problem{}
	for _, p := range Hygiene(tree, s.Budgets, s.Duplicates) {
		old[p.Key] = p
	}
	for _, p := range Hygiene(Tree{Docs: after, Files: tree.Files}, s.Budgets, s.Duplicates) {
		switch prev, was := old[p.Key]; {
		case p.Key == d.Key:
			refuse("still-repeated", p.Where, p.Message)
		case !was || p.Size > prev.Size:
			refuse("patch-introduces", p.Where, p.Rule+": "+p.Message)
		}
	}
	return refused, nil
}
