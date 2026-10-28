// Package whys reads the five whys a role's agent wrote with the shared
// skill (skills/five-whys/SKILL.md): a note opening with `Five whys`, the
// problem, the whys numbered, the root cause, one action of a closed kind.
// The engine holds its shape, never its judgement: a role's post checks it
// and posts it where the next reader looks.
package whys

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Skill is the shared skill's name, as a role's pre names it in in/skills.
const Skill = "five-whys"

// MaxChars bounds an analysis, all lines together, as the skill says.
const MaxChars = 1500

// Kinds are the actions an analysis may propose.
var Kinds = []string{"issue", "test", "doc", "code", "decision"}

// Analysis is the five whys as read.
type Analysis struct {
	Problem string   `json:"problem"`
	Whys    []string `json:"whys"`
	Cause   string   `json:"cause"`
	Also    string   `json:"also,omitempty"`
	Kind    string   `json:"kind"`
	Action  string   `json:"action"`
}

var (
	heading = regexp.MustCompile(`(?i)^[#*\s]*five whys[*:\s]*$`)
	field   = regexp.MustCompile(`(?i)^[-*\s]*\**\s*(problem|root cause|also|action)\s*(?:\(\s*([a-z]+)\s*\))?\s*\**\s*:\s*\**\s*(.*)$`)
	step    = regexp.MustCompile(`^\s*(\d+)[.)]\s+(.+)$`)
)

// Is says whether a note is an analysis, by its first line.
func Is(note string) bool {
	first, _, _ := strings.Cut(strings.TrimSpace(note), "\n")
	return heading.MatchString(first)
}

// Parse reads an analysis; its error says what the skill's form lacks.
func Parse(note string) (Analysis, error) {
	var a Analysis
	text := strings.TrimSpace(note)
	if !Is(text) {
		return a, errors.New("an analysis opens with the line `Five whys`")
	}
	if n := len(text); n > MaxChars {
		return a, fmt.Errorf("%d characters, over the skill's %d: shorter lines", n, MaxChars)
	}
	_, rest, _ := strings.Cut(text, "\n")
	var last *string // the item a line that opens none continues
	for _, l := range strings.Split(rest, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if m := step.FindStringSubmatch(l); m != nil && a.Cause == "" {
			if n, _ := strconv.Atoi(m[1]); n != len(a.Whys)+1 {
				return a, fmt.Errorf("why %d follows why %d: number them from 1, in order", n, len(a.Whys))
			}
			a.Whys = append(a.Whys, m[2])
			last = &a.Whys[len(a.Whys)-1]
			continue
		}
		if m := field.FindStringSubmatch(l); m != nil {
			v := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(m[3]), "**"))
			switch strings.ToLower(m[1]) {
			case "problem":
				a.Problem, last = v, &a.Problem
			case "root cause":
				a.Cause, last = v, &a.Cause
			case "also":
				a.Also, last = v, &a.Also
			case "action":
				a.Kind, a.Action, last = strings.ToLower(m[2]), v, &a.Action
			}
			continue
		}
		if last == nil {
			return a, fmt.Errorf("%q is none of the skill's lines", l)
		}
		*last += " " + l
	}
	var missing []string
	for _, f := range []struct{ name, v string }{{"Problem:", a.Problem}, {"Root cause:", a.Cause}, {"Action (<kind>):", a.Action}} {
		if f.v == "" {
			missing = append(missing, f.name)
		}
	}
	switch {
	case len(missing) > 0:
		return a, fmt.Errorf("it lacks %s", strings.Join(missing, ", "))
	case len(a.Whys) == 0:
		return a, errors.New("it asks no why: `1. Why …? Because …`")
	case len(a.Whys) > 5:
		return a, fmt.Errorf("%d whys, five at most: stop at the first cause the project can change", len(a.Whys))
	case !slices.Contains(Kinds, a.Kind):
		return a, fmt.Errorf("the action's kind %q is none of %s", a.Kind, strings.Join(Kinds, ", "))
	}
	return a, nil
}

// Markdown is the analysis as posted, for a person or the next role.
func (a Analysis) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Five whys**\n\n- **Problem:** %s\n", a.Problem)
	for i, w := range a.Whys {
		fmt.Fprintf(&b, "%d. %s\n", i+1, w)
	}
	fmt.Fprintf(&b, "\n- **Root cause:** %s\n", a.Cause)
	if a.Also != "" {
		fmt.Fprintf(&b, "- **Also:** %s\n", a.Also)
	}
	fmt.Fprintf(&b, "- **Action** (%s): %s\n", a.Kind, a.Action)
	return b.String()
}
