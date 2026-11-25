package tester

// What the auditor reads of the tester's acts (ADR-0046, the tester's row):
// each act where it is done, completed only where a field lacked. Its
// tests: the commit (`Workline-Role: tester`, `Workline-Model`) and the
// record on the Draft, which keeps the scenarios as the run read them and
// the base it proved them red on. An answer kept or rewritten: a hidden
// line in its reply — the commit the test was read at, the models asked,
// the scenario, the test, a rewrite's blob.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/agent"
)

// asked are the agents and models the run asked, as <agent>:<model>,
// from its calls (out/calls.jsonl); a judge is not one.
func asked(runDir string) string {
	data, err := os.ReadFile(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return ""
	}
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		var c agent.Call
		if json.Unmarshal([]byte(l), &c) != nil || c.Task == "judge" || c.Model == "" {
			continue
		}
		if m := c.Agent + ":" + c.Model; !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return strings.Join(out, ", ")
}
