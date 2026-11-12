package sample

// The week's tokens, by kind of work: what the line's own runs spent on
// product work and on upkeep, ADR-0052's third criterion to 1.0 (upkeep
// under 10% of tokens, four weeks in a row), measured as #610 asks.
// Counted by the engine from the runs folder (engine.Tallies), no agent.

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
)

// Tokens are the fresh tokens (agent.Call.Fresh) of the runs started in
// the window, by the work each was on, as catch-and-cost.md splits merge
// requests: product (a feature, a bug, or a feat, fix, refactor, perf,
// test, ci, build merge request), upkeep (a task, or a docs, chore, style
// one), a release apart; a run whose work is not recorded is unknown,
// and said, never guessed.
type Tokens struct {
	Runs    int `json:"runs"`
	Product int `json:"product"`
	Upkeep  int `json:"upkeep"`
	Release int `json:"release,omitempty"`
	Unknown int `json:"unknown,omitempty"`
	// Unknowns say each unknown run's role and why, with its tokens.
	Unknowns []string `json:"unknowns,omitempty"`
	// Unread: the runs folder could not be read whole.
	Unread string `json:"unread,omitempty"`
}

// Total is every fresh token counted.
func (t *Tokens) Total() int { return t.Product + t.Upkeep + t.Release + t.Unknown }

// Words say the week's tokens in a sentence.
func (t *Tokens) Words() string {
	if t.Runs == 0 {
		s := "no run of the line's agents recorded in the window, on this machine"
		if t.Unread != "" {
			s += " (the runs folder could not be read: " + t.Unread + ")"
		}
		return s
	}
	share := func(n int) string { return fmt.Sprintf("%s (%d%%)", line.Tokens(n), n*100/t.Total()) }
	s := fmt.Sprintf("%s fresh tokens over %s: upkeep %s, product %s", line.Tokens(t.Total()), plural(t.Runs, "run"), share(t.Upkeep), share(t.Product))
	if t.Release > 0 {
		s += ", releases " + share(t.Release)
	}
	if t.Unknown > 0 {
		s += ", unknown " + share(t.Unknown) + ": " + strings.Join(t.Unknowns, "; ")
	}
	if t.Unread != "" {
		s += "; the runs folder could not be read whole: " + t.Unread
	}
	return s
}

// tokensOf counts the runs of runsDir started in [from, to); f tells the
// work of each run's issue or merge request.
func tokensOf(runsDir string, f forge.Forge, from, to time.Time) *Tokens {
	t := &Tokens{}
	runs, err := engine.Tallies(runsDir)
	if err != nil {
		t.Unread = err.Error()
	}
	w := works{f: f, issues: map[int]workKind{}, mrs: map[int]workKind{}}
	unknown := map[string]int{}
	count := map[string]int{}
	for _, r := range runs {
		at := r.Started()
		if r.Calls == 0 || at.Before(from) || !at.Before(to) {
			continue
		}
		t.Runs++
		k := w.of(r.Target)
		switch k.kind {
		case product:
			t.Product += r.Fresh
		case upkeep:
			t.Upkeep += r.Fresh
		case release:
			t.Release += r.Fresh
		default:
			t.Unknown += r.Fresh
			key := r.Role + ", " + k.why
			unknown[key] += r.Fresh
			count[key]++
		}
	}
	for _, key := range slices.Sorted(maps.Keys(unknown)) {
		t.Unknowns = append(t.Unknowns, fmt.Sprintf("%s (%s, %s)", key, plural(count[key], "run"), line.Tokens(unknown[key])))
	}
	return t
}

// The kinds of work a run is on.
const (
	product = "product"
	upkeep  = "upkeep"
	release = "release"
)

type workKind struct{ kind, why string }

// works tells the work of an issue or a merge request, each read once.
type works struct {
	f      forge.Forge
	issues map[int]workKind
	mrs    map[int]workKind
}

func (w *works) of(t *forge.Target) workKind {
	switch {
	case t == nil:
		return workKind{why: "on no issue nor merge request"}
	case w.f == nil:
		return workKind{why: "no forge to read its " + t.Kind + " on"}
	case t.Kind == "issue":
		return w.issue(t.ID)
	case t.Kind == "merge-request":
		return w.mr(t.ID)
	}
	return workKind{why: "on a " + t.Kind}
}

// issue: a feature or a bug is product, a task upkeep, by its labels
// (backlog.KindOf).
func (w *works) issue(id int) workKind {
	if k, ok := w.issues[id]; ok {
		return k
	}
	k := workKind{}
	switch is, err := w.f.Issue(id); {
	case err != nil:
		k.why = "its issue unread"
	default:
		switch kind, _ := backlog.KindOf(is.Labels); kind {
		case backlog.KindFeature, backlog.KindBug:
			k.kind = product
		case backlog.KindTask:
			k.kind = upkeep
		default:
			k.why = "its issue has no kind"
		}
	}
	w.issues[id] = k
	return k
}

// conventional reads a title's type: `docs(x)!: …` is docs; a release is
// a chore whose subject starts with "release".
var conventional = regexp.MustCompile(`^([a-z]+)(\([^)]*\))?!?:\s*(\S*)`)

// mr: the issue it closes tells it (line.ClosedIssue); else its title's
// type.
func (w *works) mr(id int) workKind {
	if k, ok := w.mrs[id]; ok {
		return k
	}
	k := workKind{}
	mr, err := w.f.MergeRequest(id)
	switch {
	case err != nil:
		k.why = "its merge request unread"
	case line.ClosedIssue(mr) > 0:
		k = w.issue(line.ClosedIssue(mr))
	default:
		m := conventional.FindStringSubmatch(strings.ToLower(strings.TrimSpace(mr.Title)))
		switch {
		case m == nil:
			k.why = "its merge request closes no issue, its title no type"
		case m[1] == "chore" && strings.HasPrefix(m[3], "release"):
			k.kind = release
		case slices.Contains([]string{"feat", "fix", "refactor", "perf", "test", "ci", "build"}, m[1]):
			k.kind = product
		case slices.Contains([]string{"docs", "chore", "style"}, m[1]):
			k.kind = upkeep
		default:
			k.why = "its merge request closes no issue, its type `" + m[1] + "` neither product nor upkeep"
		}
	}
	w.mrs[id] = k
	return k
}
