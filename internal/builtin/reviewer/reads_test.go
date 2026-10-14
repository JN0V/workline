package reviewer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/judge"
)

// What a judge reads (ADR-0045): from git at the commit, within the lines
// left; a name by its definition, nearest first; a word's places, at most
// 50; a path outside, untracked, or past its end refused, said.
func TestServe(t *testing.T) {
	var many strings.Builder
	for i := range grepMax + 1 {
		fmt.Fprintf(&many, "var total%d = total\n", i)
	}
	repo := gitRepo(t, map[string]string{"calc/calc.go": calcGo, "calc/sum.go": sumGo, "other/sum.go": "package other\n\nfunc sum() {}\n", "wide.go": "package wide\n\nvar total = 1\n" + many.String(),
		"a/x.go": "package a\n\nfunc Shared() {}\n", "b/x.go": "package b\n\nfunc Shared() {}\n", "c/x.go": "package c\n\nfunc Shared() {}\n"})
	head := strings.TrimSpace(must(git(repo, "rev-parse", "HEAD")))

	rd := serve(repo, head, "calc/calc.go", []judge.Read{{Name: "sum"}, {Path: "calc/calc.go", Lines: "3-11"}}, 12, 400)
	if !strings.Contains(rd.Served, "### calc/sum.go, lines 3 to 10: `sum`, read") || !strings.Contains(rd.Served, "6  \tfor _, v := range values[1:] {") {
		t.Errorf("a name is read where its file's folder defines it:\n%s", rd.Served)
	}
	if !strings.Contains(rd.Served, "### calc/calc.go, lines 3 to 6: read") || !strings.Contains(rd.Served, "calc/calc.go: lines 7 to 11 not read: past judge-read-lines-max (400)") || rd.Lines != 12 {
		t.Errorf("past the lines left, a read is cut, the rest named (%d lines):\n%s", rd.Lines, rd.Served)
	}
	if len(rd.Shown) != 2 || rd.Shown[1] != (seen{"calc/calc.go", 3, 6}) {
		t.Errorf("what was read is kept, to find a no's quote in: %v", rd.Shown)
	}

	rd = serve(repo, head, "calc/calc.go", []judge.Read{{Path: "../x.go"}, {Path: "/etc/passwd"}, {Path: "notes.txt"}, {Path: "calc"}, {Path: "calc/sum.go", Lines: "40-50"}, {Path: "calc/sum.go", Lines: "ten"}, {Why: "nothing"}}, 400, 400)
	for _, want := range []string{"Not read: ../x.go: outside the repository.", "Not read: /etc/passwd: outside the repository.",
		"Not read: notes.txt: not a file the repository tracks at", "Not read: calc: not a file the repository tracks at",
		"Not read: calc/sum.go:40-50: past its end (12 lines).", "Not read: calc/sum.go:ten: its lines do not read",
		"Not read: nothing named: a read names a path, a name or a word to grep."} {
		if !strings.Contains(rd.Served, want) {
			t.Errorf("refused, said: %q not in\n%s", want, rd.Served)
		}
	}
	if rd.Lines != 0 || len(rd.Refused) != 7 {
		t.Errorf("nothing read: %d lines, %v", rd.Lines, rd.Refused)
	}

	rd = serve(repo, head, "calc/calc.go", []judge.Read{{Grep: "total"}, {Grep: "values"}, {Grep: "nowhere_said"}, {Name: "Nowhere"}}, 400, 400)
	if !strings.Contains(rd.Served, "Not read: grep `total`: written in 55 places, past 50") || !strings.Contains(rd.Served, "### grep `values`: ") ||
		!strings.Contains(rd.Served, "calc/sum.go:6: for _, v := range values[1:] {") || !strings.Contains(rd.Served, "Not read: grep `nowhere_said`: written nowhere at") ||
		!strings.Contains(rd.Served, "Not read: `Nowhere`: no definition found at") {
		t.Errorf("a word's places, at most 50; none, said:\n%s", rd.Served)
	}

	rd = serve(repo, head, "wide.go", []judge.Read{{Name: "Shared"}, {Name: "sum"}, {Name: "calc.Mean()"}}, 400, 400)
	if !strings.Contains(rd.Served, "Not read: `Shared`: defined in too many places to tell which") ||
		!strings.Contains(rd.Served, "Not read: `sum`: no definition found") || !strings.Contains(rd.Served, "### calc/calc.go, lines 3 to 6: `Mean`, read") {
		t.Errorf("a name defined far away in many places is refused, an unexported one is not seen from another folder, a qualified one is read:\n%s", rd.Served)
	}
}

func must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

// A no stands on its quote, found again at the commit in what the judge
// was shown or read; else it is not found, said why.
func TestQuoted(t *testing.T) {
	repo := gitRepo(t, map[string]string{"calc/calc.go": calcGo, "calc/sum.go": sumGo})
	head := strings.TrimSpace(must(git(repo, "rev-parse", "HEAD")))
	st := state{Head: head, Issues: []ClosedIssue{{ID: 4, Text: "## Need\n\nThe mean of every value."}}}
	r := reading{Material: "### calc/calc.go, lines 3 to 6: `Mean`\n\n```diff\n+++ b/calc/sum.go\n@@ -1,2 +4,2 @@\n```\n"}
	for _, c := range []struct {
		q          judge.Quote
		where, why string
	}{
		{judge.Quote{Path: "calc/calc.go", Text: "return sum(values) /\n len(values)"}, "calc/calc.go:5", ""},
		{judge.Quote{Path: "./calc/sum.go", Text: "func sum(values []int) int {"}, "calc/sum.go:4", ""},
		{judge.Quote{Path: "#4", Text: "The mean of every value."}, "#4", ""},
		{judge.Quote{Path: "calc/sum.go", Text: "for _, v := range values[1:] {"}, "", "its quote, at calc/sum.go:6, is not in what it was shown or read"},
		{judge.Quote{Path: "calc/calc.go", Text: "if len(values) == 0"}, "", "its quote is not found in calc/calc.go at "},
		{judge.Quote{Path: "calc/none.go", Text: "x"}, "", "its quote names calc/none.go, not a file at "},
		{judge.Quote{Path: "#5", Text: "x"}, "", "its quote names #5, not an issue the change closes"},
		{judge.Quote{Path: "calc/calc.go"}, "", "it quotes no line"},
	} {
		where, why := r.quoted(repo, st, &c.q)
		if where != c.where || !strings.HasPrefix(why, c.why) || (c.why == "") != (why == "") {
			t.Errorf("%+v: %q %q, want %q %q", c.q, where, why, c.where, c.why)
		}
	}
	r.Rounds = []readRound{{Shown: []seen{{"calc/sum.go", 6, 6}}}}
	if where, why := r.quoted(repo, st, &judge.Quote{Path: "calc/sum.go", Text: "for _, v := range values[1:] {"}); where != "calc/sum.go:6" || why != "" {
		t.Errorf("a quote from what the judge read is found: %q %q", where, why)
	}
	if _, why := r.quoted(repo, st, nil); why != "it quotes no line" {
		t.Errorf("a no with no quote: %q", why)
	}
}

// What a judge reads is bounded by its rounds and its lines, said in the
// question it is asked.
func TestReadingLeft(t *testing.T) {
	s := Settings{JudgeReadRounds: 2, JudgeReadMax: 400}
	r := reading{Rounds: []readRound{{Lines: 150}}}
	if rounds, lines := r.left(s); rounds != 1 || lines != 250 {
		t.Errorf("left: %d rounds, %d lines", rounds, lines)
	}
	if got := r.said(); got != "" {
		t.Errorf("nothing asked, nothing said: %q", got)
	}
	r.Rounds[0].Asked = []judge.Read{{Name: "sum"}, {Path: "a.go", Lines: "1-3"}}
	if got := r.said(); got != ", having read `sum`, a.go:1-3" {
		t.Errorf("what it read, said: %q", got)
	}
	for in, want := range map[string][2]int{"": {1, 12}, "4": {4, 4}, "4-90": {4, 12}, "40-50": {40, 12}} {
		if from, to, ok := lineRange(in, 12); !ok || from != want[0] || to != want[1] {
			t.Errorf("lines %q: %d-%d %v", in, from, to, ok)
		}
	}
	for _, in := range []string{"0-3", "9-3", "x", "3-x"} {
		if _, _, ok := lineRange(in, 12); ok {
			t.Errorf("lines %q read", in)
		}
	}
}
