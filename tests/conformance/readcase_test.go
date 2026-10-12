package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A case with a key nothing reads fails, naming the key and its file: it
// never passes while checking nothing (#311).
func TestReadCaseRefusesUnknownKeys(t *testing.T) {
	head := "case: x/y\nabout: z\ngiven: {repo: plain}\n"
	for _, tc := range []struct{ name, yaml, key string }{
		{"top level", head + "run: {role: committer, event: commit}\nexpects: {status: pass}\n", "expects"},
		{"in run", head + "run: {role: committer, evnt: commit}\n", "evnt"},
		{"a merge request's body", head + "run: {role: committer, event: commit}\nexpect:\n  forge:\n    merge-requests:\n      - {id: 1, body: \"a text\"}\n", "body"},
		{"a field misspelled", head + "run: {role: committer, event: commit}\nexpect:\n  forge:\n    merge-requests:\n      - {id: 1, body-contain: \"a text\"}\n", "body-contain"},
		{"a kind of the forge", head + "run: {role: committer, event: commit}\nexpect:\n  forge:\n    merge-request: [{id: 1, absent: true}]\n", "merge-request"},
		{"a finding", head + "run: {role: committer, event: commit}\nexpect:\n  findings: [{rule: internal-code, mesage: AC-3}]\n", "mesage"},
		{"a file", head + "run: {role: committer, event: commit}\nexpect:\n  files: {README.md: {contain: a text}}\n", "contain"},
		{"a run file", head + "run: {role: committer, event: commit}\nexpect:\n  run-files: {out/claims.yaml: {lacks: a text}}\n", "lacks"},
		{"the forge given", "case: x/y\nabout: z\ngiven:\n  repo: plain\n  forge:\n    issues: [{id: 1, title: t, labels: [], comments: [], ticks: [{done: true}]}]\nrun: {role: product-owner, event: schedule}\n", "ticks"},
		{"a person's edits", head + "run: {role: product-owner, event: schedule, then: again, between: {issues: [{id: 1, bdy: a text}]}}\n", "bdy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.yaml")
			os.WriteFile(file, []byte(tc.yaml), 0o644)
			_, err := readCase(file)
			if err == nil {
				t.Fatalf("read with %s unknown, want an error", tc.key)
			}
			if !strings.Contains(err.Error(), file) || !strings.Contains(err.Error(), `"`+tc.key+`"`) && !strings.Contains(err.Error(), "field "+tc.key+" ") {
				t.Errorf("the error does not name %s and %s: %v", tc.key, file, err)
			}
		})
	}
}

// The keys a case may hold read as before: a merge request's body checked
// by body-contains, a text or a list of them.
func TestReadCaseKnownKeys(t *testing.T) {
	file := filepath.Join(t.TempDir(), "case.yaml")
	os.WriteFile(file, []byte("case: x/y\nabout: z\ngiven: {repo: plain, forge: {merge-requests: []}}\n"+
		"run: {role: documentalist, event: schedule}\nexpect:\n  findings: [{rule: suspect, level: block}]\n  forge:\n    merge-requests:\n"+
		"      - {id: 1, base: main, body-contains: [\"a\", \"b\"], body-lacks: \"c\"}\n"), 0o644)
	c, err := readCase(file)
	if err != nil {
		t.Fatal(err)
	}
	mr := c.Expect.Forge.MergeRequests[0]
	if mr.ID != 1 || *mr.Base != "main" || strings.Join(mr.BodyContains, ",") != "a,b" || strings.Join(mr.BodyLacks, ",") != "c" || c.Expect.Findings[0].Level != "block" {
		t.Errorf("read %+v, %+v", mr, c.Expect.Findings)
	}
}
