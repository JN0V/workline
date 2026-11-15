package github

import (
	"os"
	"strings"
	"testing"
)

// heartbeatHeaderEntry returns the header's entry for the heartbeat job
// (the comment lines from "heartbeat" up to "Why each part"), its comment
// marks taken off and its white space made single.
func heartbeatHeaderEntry(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var header []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "#") {
			break
		}
		header = append(header, strings.TrimPrefix(line, "#"))
	}
	text := strings.Join(header, " ")
	start := strings.Index(text, "heartbeat ")
	if start < 0 {
		t.Fatalf("%s: no heartbeat entry in the header", file)
	}
	text = text[start:]
	if end := strings.Index(text, "Why each part"); end >= 0 {
		text = text[:end]
	}
	return strings.Join(strings.Fields(text), " ")
}

// Header true
// Given the header of the develop workflow,
// When a maintainer reads how to set the heartbeat's gap,
// Then it names the environment's wait timer, and says the variable only
// gates a reconcile started by the schedule or an event. It reads the
// template (ci/github) and the project's copy (.github/workflows).
func TestDevelopHeaderNamesTheWaitTimerAndWhatTheVariableGates(t *testing.T) {
	const variable = "WORKLINE_RECONCILE_GAP_MINUTES"
	for _, file := range []string{
		"workline-develop.yml",
		"../../.github/workflows/workline-develop.yml",
	} {
		entry := heartbeatHeaderEntry(t, file)
		if !strings.Contains(entry, "wait timer") {
			t.Errorf("%s: the heartbeat's header does not name the environment's wait timer: %q", file, entry)
		}
		at := strings.Index(entry, variable)
		if at < 0 {
			t.Errorf("%s: the heartbeat's header does not name %s: %q", file, variable, entry)
			continue
		}
		if at < strings.Index(entry, "wait timer") {
			t.Errorf("%s: the header names the variable before the wait timer: %q", file, entry)
		}
		after := entry[at+len(variable):]
		for _, word := range []string{"only", "schedule", "event"} {
			if !strings.Contains(after, word) {
				t.Errorf("%s: after %s the header does not say it only gates a reconcile started by the schedule or an event (no %q): %q", file, variable, word, after)
			}
		}
	}
}
