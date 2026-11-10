package tester

// Which test covers which scenario, as the agent declares it (ADR-0065,
// change 4): an entry for every scenario of the issue, its tests' ids or
// a reason none can check it. The engine guesses nothing from a test's
// name: it finds each declared id among the tests the red proof found
// new, a report's, or, its build broken on the base, one a test file of
// the change declares.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

// CoversKind is the intention the tester's agent declares a scenario's
// tests with: read by post, never applied.
const CoversKind = "covers"

// declared is the agent's word on one scenario: the ids of the tests that
// check it, or why none can.
type declared struct {
	Scenario string
	Tests    []string
	Reason   string
}

// coversOf reads the agent's covers entries; an entry it cannot read is
// a finding, the answer's to mend.
func coversOf(intents []intent.Intention) ([]declared, []verdict.Finding) {
	var out []declared
	var bad []verdict.Finding
	for i, in := range intents {
		if in.Kind != CoversKind {
			continue
		}
		m, ok := in.Value.(map[string]any)
		sc, _ := m["scenario"].(string)
		if !ok || strings.TrimSpace(sc) == "" {
			bad = append(bad, verdict.Finding{Rule: "covers-unreadable", Where: fmt.Sprintf("covers %d", i+1),
				Message: "a covers entry is `{scenario: <its name>, tests: [<test id>, …]}`, or `{scenario: <its name>, reason: <why no test can check it>}`"})
			continue
		}
		c := declared{Scenario: strings.TrimSpace(sc)}
		switch t := m["tests"].(type) {
		case string:
			c.Tests = []string{t}
		case []any:
			for _, x := range t {
				c.Tests = append(c.Tests, fmt.Sprint(x))
			}
		}
		c.Tests = slices.DeleteFunc(c.Tests, func(s string) bool { return strings.TrimSpace(s) == "" })
		for k, s := range c.Tests {
			c.Tests[k] = strings.Trim(strings.TrimSpace(s), "`")
		}
		c.Reason, _ = m["reason"].(string)
		c.Reason = strings.TrimSpace(c.Reason)
		out = append(out, c)
	}
	return out, bad
}

// sameScenario says whether an entry names the scenario: its name as the
// issue writes it, its case and the bold or colon around it aside.
func sameScenario(entry, name string) bool {
	clean := func(s string) string {
		return strings.TrimSpace(strings.TrimSuffix(strings.Trim(strings.TrimSpace(s), "*`"), ":"))
	}
	return strings.EqualFold(clean(entry), clean(name))
}

// held holds the entries to the issue's scenarios: tested, the ids each
// declares; said, the reason of each no test can check; the findings of
// a scenario with no entry, an entry naming none, or one with neither
// tests nor reason, each the answer's to mend.
func held(covers []declared, scenarios []Scenario) (tested map[string][]string, said map[string]string, findings []verdict.Finding) {
	tested, said = map[string][]string{}, map[string]string{}
	for _, c := range covers {
		i := slices.IndexFunc(scenarios, func(sc Scenario) bool { return sameScenario(c.Scenario, sc.Name) })
		switch {
		case i < 0:
			var names []string
			for _, sc := range scenarios {
				names = append(names, sc.Name)
			}
			findings = append(findings, verdict.Finding{Rule: "covers-unknown-scenario", Where: c.Scenario,
				Message: "a covers entry names no scenario of the issue: give each its name as the issue writes it, one of " + strings.Join(names, ", ")})
		case len(c.Tests) > 0:
			name := scenarios[i].Name
			for _, t := range c.Tests {
				if !slices.Contains(tested[name], t) {
					tested[name] = append(tested[name], t)
				}
			}
		case c.Reason != "":
			if _, ok := said[scenarios[i].Name]; !ok {
				said[scenarios[i].Name] = c.Reason
			}
		default:
			findings = append(findings, verdict.Finding{Rule: "covers-empty", Where: scenarios[i].Name,
				Message: "its covers entry names no test and gives no reason: name the tests that check it, or say why none can"})
		}
	}
	for _, sc := range scenarios {
		if _, ok := tested[sc.Name]; ok {
			delete(said, sc.Name)
			continue
		}
		if _, ok := said[sc.Name]; ok {
			continue
		}
		if slices.ContainsFunc(findings, func(f verdict.Finding) bool { return f.Where == sc.Name }) {
			continue
		}
		findings = append(findings, verdict.Finding{Rule: "scenario-without-test", Where: sc.Name,
			Message: "no covers entry for this scenario: declare the tests that check it, `- covers: {scenario: " + sc.Name +
				", tests: [<test id>]}`, or why none can, `reason:` in place of `tests:`; one test may cover several scenarios"})
	}
	return tested, said, findings
}

// declaredIn is the new tests a declared id is, among those the red proof
// found (newTests) in the test files the change writes (written):
//   - a report's id: itself, or one it ends as a whole dotted part
//     (`CartTest.addsItem` for `shop.CartTest.addsItem`, `test_x` for
//     `tests.test_x`), its parameters aside (`test_x` for every row of
//     `tests.test_x[1]`, `addsItem` for `CartTest.addsItem(int)[1]`);
//   - file:name, a test its file declares, its build broken on the base:
//     that id, its name, an id holding the name qualified by the file
//     (`CartTest.addsItem` for `src/test/CartTest.kt:addsItem`), or the
//     file, each test it declares.
//
// An id the whole id of none and the tail of two tests or more names none
// of them: those are its others, the agent's to tell apart.
func declaredIn(id string, newTests, written []string) (found, others []string) {
	id = strings.TrimSuffix(id, "()")
	var exact, tails []string
	for _, t := range newTests {
		if file, name, ok := fileNamed(t, written); ok {
			switch {
			case id == t:
				exact = append(exact, t)
			case id == file:
				exact = append(exact, t) // the file's tests, each
			case id == name || namedAs(id, name, file):
				tails = append(tails, t)
			}
			continue
		}
		b := bare(t)
		switch {
		case id == t || id == b:
			exact = append(exact, t)
		case strings.HasSuffix(b, "."+id) || strings.HasSuffix(t, "."+id):
			tails = append(tails, t)
		}
	}
	if len(exact) > 0 {
		return exact, nil
	}
	var tests []string
	for _, t := range tails {
		if b := bare(t); !slices.Contains(tests, b) {
			tests = append(tests, b)
		}
	}
	if len(tests) > 1 {
		return nil, tests
	}
	return tails, nil
}

// bare is a report's id without its parameters: a parametrised test's row
// (pytest's `test_x[1]`, JUnit 5's `addsItem(int)[1]`) is its test, and
// `addsItem()` is `addsItem`.
func bare(id string) string {
	if i := strings.Index(id, "["); i > 0 && id[i-1] != ' ' && strings.HasSuffix(id, "]") {
		id = id[:i]
		if j := strings.LastIndex(id, "("); j > 0 && strings.HasSuffix(id, ")") {
			id = id[:j]
		}
	}
	return strings.TrimSuffix(id, "()")
}

// fileNamed splits a new test named from its file, file:name, the file
// one the change writes.
func fileNamed(t string, written []string) (file, name string, ok bool) {
	i := strings.LastIndex(t, ":")
	if i <= 0 || t[i+1:] == "" || !slices.Contains(written, t[:i]) {
		return "", "", false
	}
	return t[:i], t[i+1:], true
}

// found holds each scenario's declared ids to the new tests: matched, the
// tests each scenario is proven by; the findings of an id found nowhere,
// or the tail of several tests, the answer's to mend, asked again.
func found(tested map[string][]string, scenarios []Scenario, newTests, written []string) (matched map[string][]string, findings []verdict.Finding) {
	matched = map[string][]string{}
	for _, sc := range scenarios {
		for _, id := range tested[sc.Name] {
			ids, others := declaredIn(id, newTests, written)
			if len(others) > 0 {
				findings = append(findings, verdict.Finding{Rule: "declared-test-ambiguous", Where: sc.Name,
					Message: fmt.Sprintf("`%s`, declared for this scenario, ends several new tests (%s): declare the one meant by a longer id, its module or class too", id, codes(others))})
				continue
			}
			if len(ids) == 0 {
				findings = append(findings, verdict.Finding{Rule: "declared-test-missing", Where: sc.Name,
					Message: fmt.Sprintf("`%s`, declared for this scenario, is none of the new tests red on the base (%s), nor of the test files written: "+
						"declare it as the project's test report names it, its class and name dot-joined, or its name alone", id, sample(newTests))})
				continue
			}
			for _, t := range ids {
				if !slices.Contains(matched[sc.Name], t) {
					matched[sc.Name] = append(matched[sc.Name], t)
				}
			}
		}
	}
	return matched, findings
}

// CoveredBy is the scenario a test id is declared for, by a record's
// covers: the one whose ids hold it, whole or as a whole dotted part
// either way; "" when none, or more than one.
func CoveredBy(covers map[string][]string, id string) string {
	if all := Covering(covers, id); len(all) == 1 {
		return all[0]
	}
	return ""
}

// Covering are the scenarios a test id is declared for, sorted.
func Covering(covers map[string][]string, id string) []string {
	id = strings.Trim(strings.TrimSpace(id), "`")
	if id == "" {
		return nil
	}
	var out []string
	for sc, ids := range covers {
		if slices.ContainsFunc(ids, func(t string) bool { return SameTest(t, id) }) {
			out = append(out, sc)
		}
	}
	slices.Sort(out)
	return out
}

// SameTest says whether two ids name the same test: equal, or one the
// other's whole dotted or file-named tail (`CartTest.addsItem` and
// `shop.CartTest.addsItem`; `addsItem` and `src/CartTest.kt:addsItem`),
// a parametrised row's parameters aside (`test_x` and `tests.test_x[1]`).
func SameTest(a, b string) bool {
	a, b = bare(a), bare(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	tail := func(long, short string) bool {
		return strings.HasSuffix(long, "."+short) || strings.HasSuffix(long, ":"+short)
	}
	return tail(a, b) || tail(b, a)
}

// answerMends are the rules of the covers entries, the answer's to mend:
// asked again once with them before any block.
var answerMends = map[string]bool{"scenario-without-test": true, "covers-unreadable": true, "covers-unknown-scenario": true, "covers-empty": true}

// recovered keeps the record's covers true once file is rewritten on an
// objection: a covers entry of the answer for a scenario of the record
// names its tests now, each found among the rewrite's new tests; every
// scenario keeps those of its ids the record still holds. Why the rewrite
// is refused, "" when it is not: an id declared and not found, or a
// scenario left with none of its tests. A record with no covers, written
// before, has none to keep; one whose test ids its run lost keeps them.
func (r *record) recovered(intents []intent.Intention, newTests []string, file string) string {
	var names []string
	for _, sc := range r.Scenarios {
		names = append(names, sc.Name)
	}
	for sc := range r.Covers {
		if !slices.Contains(names, sc) {
			names = append(names, sc)
		}
	}
	entries, _ := coversOf(intents)
	now := map[string][]string{}
	for _, c := range entries {
		i := slices.IndexFunc(names, func(n string) bool { return sameScenario(c.Scenario, n) })
		if i < 0 {
			continue
		}
		scenario := names[i]
		for _, id := range c.Tests {
			ids, others := declaredIn(id, newTests, []string{file})
			if len(others) > 0 {
				return fmt.Sprintf("`%s`, declared for %s, ends several of the rewrite's tests (%s): declare the one meant by a longer id", id, scenario, codes(others))
			}
			if len(ids) == 0 {
				return fmt.Sprintf("`%s`, declared for %s, is none of the rewrite's tests red on the base (%s)", id, scenario, sample(newTests))
			}
			now[scenario] = append(now[scenario], ids...)
		}
	}
	if len(r.Covers) == 0 && len(now) == 0 {
		return ""
	}
	covers := map[string][]string{}
	for sc, ids := range r.Covers {
		kept := ids
		if r.Tests != nil {
			kept = slices.DeleteFunc(slices.Clone(ids), func(t string) bool { return !slices.Contains(r.Tests, t) })
		}
		if len(ids) > 0 && len(kept) == 0 && len(now[sc]) == 0 {
			return fmt.Sprintf("the rewrite drops %s, declared for %s, and no covers entry names the tests that cover it now: "+
				"`- covers: {scenario: %s, tests: [<test id>]}`", codes(ids), sc, sc)
		}
		covers[sc] = kept
	}
	for sc, ids := range now {
		covers[sc] = slices.Compact(slices.Sorted(slices.Values(append(covers[sc], ids...))))
	}
	r.Covers = covers
	return ""
}

// reasons are the reasons the covers entries give, each its scenario's
// name first: an answer with no test says why by them too.
func reasons(intents []intent.Intention) []string {
	entries, _ := coversOf(intents)
	var out []string
	for _, c := range entries {
		if len(c.Tests) == 0 && c.Reason != "" {
			out = append(out, c.Scenario+": "+c.Reason)
		}
	}
	return out
}
