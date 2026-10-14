package sample

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// The judge is never a model that acted: of Claude, the first of its
// models none of them is — a third one for a finding its lens and judge
// read; another agent stands as it is set.
func TestApart(t *testing.T) {
	for _, c := range []struct {
		spec  string
		acted []string
		want  string
	}{
		{"claude:opus", []string{"claude-sonnet-5"}, "claude:opus"},
		{"claude:opus", []string{"claude-opus-5-5"}, "claude:sonnet"},
		{"claude:opus@high", []string{"claude-sonnet-5", "claude-opus-5"}, "claude:haiku@high"},
		{"claude:opus", []string{"claude-sonnet-5", "claude-opus-5", "claude-haiku-4-5"}, "claude:opus"},
		{"cmd:other-provider", []string{"claude-opus-5"}, "cmd:other-provider"},
		{"claude:opus", nil, "claude:opus"},
	} {
		if got, _ := apart(c.spec, c.acted); got != c.want {
			t.Errorf("apart(%s, %v) = %s, want %s", c.spec, c.acted, got, c.want)
		}
	}
}

// An answer against the act is false only with a quote found again in the
// item it names, outside the code's comments; the act stands when the
// note says it holds — for a dropped finding, `false`.
func TestSettle(t *testing.T) {
	m := &material{items: []item{
		{name: "#7", full: "### Validation\nTBD"},
		{name: "a.go", full: "package a\n\n// the rows are kept\nconst Rows = 2\n", code: true},
	}}
	claim := func(path, quote string) intent.Intention {
		return intent.Intention{Kind: "claim", Value: map[string]any{"source": map[string]any{"path": path, "quote": quote}}}
	}
	note := func(n string) intent.Intention { return intent.Intention{Kind: "note", Value: n} }
	for _, c := range []struct {
		answer  []intent.Intention
		holds   string
		verdict string
		proof   string
	}{
		{[]intent.Intention{note("true: it holds")}, "true", "true", ""},
		{[]intent.Intention{claim("#7", "Validation TBD"), note("false: no validation")}, "true", "false", `#7 says "Validation TBD"`},
		{[]intent.Intention{claim("a.go", "the rows are kept"), note("false: kept")}, "true", "unproven", ""},
		{[]intent.Intention{claim("#8", "TBD"), note("false: no")}, "true", "unproven", ""},
		{[]intent.Intention{claim("a.go", "const Rows = 2"), note("true: it fails")}, "false", "false", `a.go:4 says "const Rows = 2"`},
		{[]intent.Intention{note("false: it does not fail")}, "false", "true", ""},
		{[]intent.Intention{note("maybe")}, "true", "not-read", ""},
		{nil, "true", "not-read", ""},
	} {
		m.holds = c.holds
		a := &Act{}
		settle(a, m, c.answer)
		var proofs []string
		for _, p := range a.Proofs {
			proofs = append(proofs, p.String())
		}
		if a.Verdict != c.verdict || strings.Join(proofs, "; ") != c.proof {
			t.Errorf("settle(%v, holds %s) = %s %q, want %s %q", c.answer, c.holds, a.Verdict, proofs, c.verdict, c.proof)
		}
	}
}
