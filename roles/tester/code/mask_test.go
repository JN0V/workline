package tester

import (
	"strings"
	"testing"
)

// A test command printing a token's value is quoted with *** in its
// place; a line holding no secret is quoted as it printed (#1215).
func TestCommandOutputMasked(t *testing.T) {
	t.Setenv("GH_TOKEN", "fake-value-for-test-1215")
	tree := reportTree(t, []string{"pom.xml"}, nil)
	s := Settings{Report: "junit.xml", Command: `echo "error: login refused for $GH_TOKEN"; echo "error: kapt failed"; exit 1`}
	r, f := runIn(tree, s, "here")
	if f == nil || f.Rule != "report-missing" {
		t.Fatalf("runIn: %+v, want report-missing", f)
	}
	for what, got := range map[string]string{"finding": f.Message, "output": r.output, "reason": errorLines(r.output, nil, 5)} {
		if strings.Contains(got, "fake-value") || !strings.Contains(got, "error: login refused for ***") {
			t.Errorf("%s not masked: %q", what, got)
		}
		if !strings.Contains(got, "error: kapt failed") {
			t.Errorf("%s lost a line holding no secret: %q", what, got)
		}
	}
}
