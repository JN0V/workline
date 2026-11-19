package main

import (
	"strings"
	"testing"
)

// sampleHelpAbout runs `workline sample --help` and returns the sentence
// under its usage line, the command's description.
func sampleHelpAbout(t *testing.T) string {
	t.Helper()
	out, code := captureStdout(t, func() int { return dispatch([]string{"sample", "--help"}) })
	if code != 0 {
		t.Fatalf("workline sample --help: exit %d, stdout %q", code, out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "usage: workline sample") || lines[1] != "" || lines[2] == "" {
		t.Fatalf("workline sample --help: no description under the usage line in:\n%s", out)
	}
	return lines[2]
}

// sampleSays checks a description names every role's acts, the docs passed
// among them.
func sampleSays(t *testing.T, where, about string) {
	t.Helper()
	low := strings.ToLower(about)
	if !strings.Contains(low, "acts") || !strings.Contains(low, "role") {
		t.Errorf("%s does not name every role's acts: %q", where, about)
	}
	if !strings.Contains(low, "docs") {
		t.Errorf("%s does not name the docs passed among the acts: %q", where, about)
	}
	if strings.Contains(low, "only") {
		t.Errorf("%s says the sample is limited: %q", where, about)
	}
}

// The help says what is drawn
//
// Given workline built from `main`,
// When a person runs `workline sample --help`,
// Then the first line, the one under the usage, names every role's acts,
// the docs passed among them.
func TestSampleHelpSaysWhatIsDrawn(t *testing.T) {
	sampleSays(t, "workline sample --help", sampleHelpAbout(t))
}

// The command list agrees
//
// Given workline built from `main`,
// When a person runs `workline --help`,
// Then the `sample` line says the same as `workline sample --help`: the
// same sentence, apart from the capital letter and the final period, and
// it names every role's acts too.
func TestSampleCommandListAgrees(t *testing.T) {
	about := sampleHelpAbout(t)

	top, code := captureStdout(t, func() int { return dispatch([]string{"--help"}) })
	if code != 0 {
		t.Fatalf("workline --help: exit %d, stdout %q", code, top)
	}
	listed := ""
	for _, l := range strings.Split(top, "\n") {
		if rest, ok := strings.CutPrefix(l, "  sample "); ok {
			listed = strings.TrimSpace(rest)
			break
		}
	}
	if listed == "" {
		t.Fatalf("no sample line in workline --help:\n%s", top)
	}
	sampleSays(t, "the sample line of workline --help", listed)
	if !strings.EqualFold(strings.TrimSuffix(about, "."), listed) {
		t.Errorf("the sample line of workline --help %q is not the sentence of workline sample --help %q", listed, about)
	}
}
