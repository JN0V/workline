package developer

// A pull request that is a part of its issue, not all of it, by facts: a
// file refused as a person's, or a path its Scope section names that only
// a person writes, left. It says `Part of #n`, never `Closes #n`, and the
// issue is blocked for a person, what is left said on it
// (docs/pull-request.md#a-part-of-its-issue).

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

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

// leftToAPerson are the paths the issue's Scope section names (not the
// product owner's sources) that only a person changes — a gate's, the
// CI's, a steering file —, none refused already: its code wrote none of
// them, so they are left for a person.
func leftToAPerson(repo string, b Bounds, refused []devmark.Refusal) []string {
	c := newChecker(repo, b)
	var out []string
	for _, p := range b.Named {
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

// personsPart blocks the issue for a person once its pull request, a
// part of it, is open: what is left said on it. The pull request still
// merges by the line; the person makes the rest, then closes the issue.
func personsPart(repo string, a devmark.Act, s Settings) []intent.Intention {
	var left []string
	for _, r := range personsRefusals(a.Refused) {
		left = append(left, fmt.Sprintf("`%s`: %s", r.File, why(r)))
	}
	for _, p := range a.Left {
		left = append(left, fmt.Sprintf("`%s`: %s", p, leftWhy))
	}
	out := blockedForAPerson(repo, a.Issue, s,
		"its pull request is a part of it, `Part of #"+fmt.Sprint(a.Issue)+"`; left for a person: "+strings.Join(left, "; "),
		"the line reviews and merges the pull request as any; a person makes these changes, then closes this issue")
	out[1].Value.(map[string]any)["sticky"] = "partial" // one comment, edited: a round says it again in place
	return out
}

// asMaps are intentions as out/merge-request.yaml lists them, one kind
// each.
func asMaps(in []intent.Intention) []map[string]any {
	out := make([]map[string]any, len(in))
	for i, x := range in {
		out[i] = map[string]any{x.Kind: x.Value}
	}
	return out
}

// markPartOf asks the engine, in out/merge-request.yaml, to say its own
// pull request a part of the issue: `Closes #n` in its body made
// `Part of #n`, what the round asked there kept.
func markPartOf(runDir string, issue int) error {
	mr := map[string]any{}
	if data, err := os.ReadFile(filepath.Join(runDir, "out", "merge-request.yaml")); err == nil {
		if err := yaml.Unmarshal(data, &mr); err != nil {
			return err
		}
	}
	mr["part-of"] = issue
	return writeMergeRequest(runDir, mr)
}
