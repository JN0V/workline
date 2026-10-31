package tester

import "testing"

// A flaky-n-of-n note names its test and runs on lines of their own,
// taken out of its reason; another kind keeps its words whole.
func TestProofOfNamesTheFlakyTest(t *testing.T) {
	p, ok := proofOf([]string{"proof: flaky-n-of-n — it fails one run in five.\ntest: pkg.TestRetry\nruns: 20\n"})
	if !ok || p.Kind != KindFlaky || p.Test != "pkg.TestRetry" || p.Runs != 20 || p.Reason != "it fails one run in five." {
		t.Errorf("flaky: %+v, want its test, 20 runs, the reason alone", p)
	}
	p, _ = proofOf([]string{"proof: flaky-n-of-n — why\nruns: many"})
	if p.Test != "" || p.Runs != 0 {
		t.Errorf("no test, runs not a number: %+v, want neither", p)
	}
	p, _ = proofOf([]string{"proof: existing-suite — a refactor\ntest: x"})
	if p.Test != "" || p.Reason != "a refactor\ntest: x" {
		t.Errorf("existing-suite: %+v, want its words whole, no test", p)
	}
}

// A name a shell would read is no test name.
func TestTestName(t *testing.T) {
	for name, want := range map[string]bool{
		"TestRetry": true, "tests.test_retry": true, "pkg/x.TestA/sub-1": true,
		"": false, "a b": false, "x;rm -rf": false, "$(id)": false, "a'b": false,
	} {
		if TestName.MatchString(name) != want {
			t.Errorf("TestName(%q) = %v, want %v", name, !want, want)
		}
	}
}

// The named test is found by its id whole or its last part.
func TestOutcomePasses(t *testing.T) {
	o := Outcome{Passed: []string{"pkg/x.TestA", "tests.test_b"}, Red: []string{"pkg/x.TestC"}}
	cases := []struct {
		test           string
		reported, pass bool
	}{
		{"TestA", true, true}, {"pkg/x.TestA", true, true}, {"test_b", true, true},
		{"TestC", true, false}, {"TestD", false, false}, {"A", false, false},
	}
	for _, c := range cases {
		if r, p := o.Passes(c.test); r != c.reported || p != c.pass {
			t.Errorf("Passes(%q) = %v, %v; want %v, %v", c.test, r, p, c.reported, c.pass)
		}
	}
}
