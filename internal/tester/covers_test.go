package tester

import (
	"slices"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// Each scenario takes the tests its entry declares, or its reason; one
// test may cover two; a scenario with no entry, an entry naming none, or
// one with neither tests nor reason, is a finding (ADR-0065, change 4).
func TestHeldEntries(t *testing.T) {
	sc := []Scenario{{Name: "Disabled console"}, {Name: "Enabled console"}, {Name: "Restart"}, {Name: "Look"}}
	intents := []intent.Intention{
		{Kind: "patch", Value: "x"},
		{Kind: "covers", Value: map[string]any{"scenario": "Disabled console", "tests": "test_disabled"}},
		{Kind: "covers", Value: map[string]any{"scenario": "enabled console", "tests": []any{"test_enabled_console_and_restart"}}},
		{Kind: "covers", Value: map[string]any{"scenario": "**Restart:**", "tests": []any{"`test_enabled_console_and_restart`"}}},
		{Kind: "covers", Value: map[string]any{"scenario": "Look", "reason": "a person reads it by eye"}},
		{Kind: "covers", Value: map[string]any{"scenario": "Elsewhere", "tests": "t"}},
	}
	entries, unread := coversOf(intents)
	if len(unread) != 0 {
		t.Fatalf("unread %v", unread)
	}
	tested, said, findings := held(entries, sc)
	if got := tested["Restart"]; !slices.Equal(got, []string{"test_enabled_console_and_restart"}) {
		t.Errorf("Restart: %v", got)
	}
	if got := tested["Enabled console"]; !slices.Equal(got, []string{"test_enabled_console_and_restart"}) {
		t.Errorf("Enabled console: %v", got)
	}
	if said["Look"] != "a person reads it by eye" {
		t.Errorf("Look: %q", said["Look"])
	}
	if len(findings) != 1 || findings[0].Rule != "covers-unknown-scenario" {
		t.Errorf("findings %+v", findings)
	}
	_, _, findings = held(entries[:1], sc)
	var rules []string
	for _, f := range findings {
		rules = append(rules, f.Rule+":"+f.Where)
	}
	if want := []string{"scenario-without-test:Enabled console", "scenario-without-test:Restart", "scenario-without-test:Look"}; !slices.Equal(rules, want) {
		t.Errorf("rules %v, want %v", rules, want)
	}
	if _, bad := coversOf([]intent.Intention{{Kind: "covers", Value: "Restart"}}); len(bad) != 1 {
		t.Errorf("an entry with no scenario is read: %v", bad)
	}
}

// A declared id is found as a report names it, whole or by a dotted tail;
// from its file when the build broke on the base; by the file alone only
// when no report row and no test-name name the change's tests.
func TestDeclaredIn(t *testing.T) {
	report := []string{"shop.CartTest.addsItem", "tests.test_mul_by_zero"}
	for id, want := range map[string]string{
		"shop.CartTest.addsItem": "shop.CartTest.addsItem", "CartTest.addsItem": "shop.CartTest.addsItem", "addsItem()": "shop.CartTest.addsItem",
		"test_mul_by_zero": "tests.test_mul_by_zero", "Item": "", "by_zero": "", "tests/test_mul_by_zero.sh": "",
	} {
		got, _ := declaredIn(id, report, []string{"tests/test_mul_by_zero.sh"})
		if (want == "") != (len(got) == 0) || want != "" && got[0] != want {
			t.Errorf("declaredIn(%q) = %v, want %q", id, got, want)
		}
	}
	file := "calc/src/test/kotlin/calc/AddThreeTest.kt"
	broke := []string{file + ":two plus three plus four"}
	for _, id := range []string{"AddThreeTest.two plus three plus four", "calc.AddThreeTest.two plus three plus four", "two plus three plus four", broke[0]} {
		if got, byFile := declaredIn(id, broke, []string{file}); len(got) != 1 || got[0] != broke[0] || byFile {
			t.Errorf("declaredIn(%q) = %v, %v", id, got, byFile)
		}
	}
	if got, _ := declaredIn("OtherTest.two plus three plus four", broke, []string{file}); len(got) != 0 {
		t.Errorf("another class's test of the same name is found: %v", got)
	}
	if got, byFile := declaredIn(file, nil, []string{file}); len(got) != 1 || !byFile {
		t.Errorf("no report, no test-name: the file is not found: %v %v", got, byFile)
	}
	if got, _ := declaredIn(file, report, []string{file}); len(got) != 0 {
		t.Errorf("a report holds the change's tests, yet a file is found: %v", got)
	}
}

// An objection naming a declared test is on the scenario it covers; one
// covering two, or none, names none.
func TestCoveredBy(t *testing.T) {
	covers := map[string][]string{"Add item": {"shop.CartTest.addsItem"}, "Enabled console": {"t.both"}, "Restart": {"t.both"}}
	for id, want := range map[string]string{
		"CartTest.addsItem": "Add item", "shop.CartTest.addsItem": "Add item", "`addsItem`": "Add item",
		"both": "", "TestAddItem": "", "": "",
	} {
		if got := CoveredBy(covers, id); got != want {
			t.Errorf("CoveredBy(%q) = %q, want %q", id, got, want)
		}
	}
}

// A rewrite keeps its scenario's declared tests, or names the new ones;
// one that drops them with no entry is refused.
func TestRecovered(t *testing.T) {
	r := record{Tests: []string{"tests.test_b"}, Covers: map[string][]string{"By zero": {"tests.test_a"}, "Two": {"tests.test_b"}}}
	if why := r.recovered("By zero", nil, []string{"tests.test_c"}, "tests/test_a.sh"); why == "" {
		t.Error("a rewrite dropping its scenario's test is kept")
	}
	r = record{Tests: []string{"tests.test_b", "tests.test_c"}, Covers: map[string][]string{"By zero": {"tests.test_a"}, "Two": {"tests.test_b"}}}
	entry := []intent.Intention{{Kind: "covers", Value: map[string]any{"scenario": "By zero", "tests": "test_c"}}}
	if why := r.recovered("By zero", entry, []string{"tests.test_c"}, "tests/test_a.sh"); why != "" {
		t.Fatal(why)
	}
	if got := r.Covers["By zero"]; !slices.Equal(got, []string{"tests.test_c"}) {
		t.Errorf("By zero now %v", got)
	}
	old := record{Tests: []string{"tests.test_c"}}
	if why := old.recovered("By zero", nil, []string{"tests.test_c"}, "tests/test_a.sh"); why != "" || old.Covers != nil {
		t.Errorf("a record with no covers: %q %v", why, old.Covers)
	}
}
