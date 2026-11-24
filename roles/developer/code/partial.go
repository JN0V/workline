package developer

// A pull request that is a part of its issue, not all of it: a file
// refused, a Scope path only a person writes left, or its agent's note of
// what it left out. It says `Part of #n`, never `Closes #n`, and an issue
// for a person, a sub-issue of it, holds the rest
// (docs/pull-request.md#a-part-of-its-issue).

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// personsRefusals are the refusals that leave work for a person: a file
// only a person changes (a gate's, the CI's, a steering file) or one the
// developer cannot write (not text). A frozen test it may not edit, a
// path out of the worktree, leave none: the issue's work is not there.
func personsRefusals(refused []devmark.Refusal) []devmark.Refusal {
	return slices.DeleteFunc(slices.Clone(refused), func(r devmark.Refusal) bool {
		return r.Rule != "gate-file" && r.Rule != "steering-file" && r.Rule != "not-text"
	})
}

// leftToAPerson are the paths the issue's Scope names that only a person
// changes — a gate's, the CI's, a steering file —, none refused already:
// its code wrote none of them, so they are left for a person.
func leftToAPerson(repo string, b Bounds, refused []devmark.Refusal) []string {
	c := newChecker(repo, b)
	var out []string
	for _, p := range b.Scope {
		switch c.rule(p) {
		case "gate-file", "steering-file":
		default:
			continue
		}
		if slices.ContainsFunc(refused, func(r devmark.Refusal) bool { return r.File == p || under(r.File, []string{p}) }) {
			continue
		}
		out = append(out, p)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// personsPart is the issue for a person that holds what its pull request
// left out, a sub-issue of its issue, written as the product owner writes
// one (roles/product-owner/docs/writing.md): the person's part on top,
// the paths below the line. Keyed by its title: a run again finds it.
func personsPart(a devmark.Act, title, note string) intent.Intention {
	t := fmt.Sprintf("A person's part of #%d", a.Issue)
	if title = strings.TrimSpace(title); title != "" {
		t += ": " + title
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Part of #%d.\n\n", a.Issue)
	b.WriteString("## Need\n\nA person finishes what the developer's pull request left out: files only a person may change, or work its agent could not do.\n\n")
	b.WriteString("## Validation\n\n- **The rest done:** Given the developer's pull request for the parent issue merged, " +
		"When a person makes each change the Scope lists, Then the parent issue's Validation passes in full.\n\n")
	b.WriteString("---\n\n## Scope\n\n")
	for _, r := range personsRefusals(a.Refused) {
		fmt.Fprintf(&b, "- `%s`: %s\n", r.File, why(r))
	}
	for _, p := range a.Left {
		fmt.Fprintf(&b, "- `%s`: %s\n", p, leftWhy)
	}
	if note != "" {
		b.WriteString("\nIts agent's note of what it left out:\n\n")
		for _, l := range strings.Split(strings.TrimSpace(devmark.InertText(note)), "\n") {
			b.WriteString("> " + l + "\n")
		}
	}
	fmt.Fprintf(&b, "\nThe developer's pull request for #%d says `Part of #%[1]d`: its merge leaves #%[1]d open.\n", a.Issue)
	return intent.Intention{Kind: "issue", Value: map[string]any{"title": t, "body": b.String(), "parent": a.Issue}}
}
