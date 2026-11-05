// Package overlap tests roles/tester/overlap on a planted fixture:
// coverage.txt and mutations.txt, no real mutation tool.
package overlap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixture plants two tests, alpha and beta, with identical covered lines
// and identical killed mutants (all four), and a third, gamma, with other
// lines that kills two of the four.
func fixture(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	coverage := "alpha a.go:1\nalpha a.go:2\nbeta a.go:1\nbeta a.go:2\ngamma b.go:7\ngamma b.go:8\n"
	mutations := "mutant m1\nmutant m2\nmutant m3\nmutant m4\n" +
		"alpha kills m1\nalpha kills m2\nalpha kills m3\nalpha kills m4\n" +
		"beta kills m1\nbeta kills m2\nbeta kills m3\nbeta kills m4\n" +
		"gamma kills m1\ngamma kills m2\n"
	for name, body := range map[string]string{"coverage.txt": coverage, "mutations.txt": mutations} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func measure(t *testing.T) []string {
	t.Helper()
	dir := fixture(t)
	script, err := filepath.Abs(filepath.Join("..", "..", "..", "roles", "tester", "overlap"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", script, dir).Output()
	if err != nil {
		t.Fatalf("roles/tester/overlap %s: %v\n%s", dir, err, out)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// Overlap found
//
// Given a fixture repository with two planted tests that run the same
// lines and kill the same mutants,
// When the measure runs,
// Then it lists the pair as a candidate to merge.
func TestOverlapFound(t *testing.T) {
	var got []string
	for _, l := range measure(t) {
		if strings.HasPrefix(l, "merge-candidate:") {
			got = append(got, l)
		}
	}
	if len(got) != 1 || got[0] != "merge-candidate: alpha beta" {
		t.Errorf("merge-candidate lines = %q, want exactly [\"merge-candidate: alpha beta\"]", got)
	}
}

// Distinct test spared
//
// Given a third test that covers different lines or kills a mutant the
// others do not,
// When the measure runs,
// Then it is not listed.
func TestDistinctTestSpared(t *testing.T) {
	lines := measure(t)
	found := false
	for _, l := range lines {
		if strings.HasPrefix(l, "merge-candidate:") {
			found = true
			if strings.Contains(l, "gamma") {
				t.Errorf("distinct test listed: %q", l)
			}
		}
	}
	if !found {
		t.Errorf("no merge-candidate line at all: %q", lines)
	}
}

// Score shown
//
// Given the fixture,
// When the measure runs,
// Then its output gives a mutation score for each test.
func TestScoreShown(t *testing.T) {
	lines := measure(t)
	have := map[string]bool{}
	for _, l := range lines {
		have[l] = true
	}
	for _, want := range []string{"score: alpha 100%", "score: beta 100%", "score: gamma 50%"} {
		if !have[want] {
			t.Errorf("no line %q in %q", want, lines)
		}
	}
}
