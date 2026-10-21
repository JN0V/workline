package documentalist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JN0V/workline/internal/builtin/committer"
	"github.com/JN0V/workline/internal/role"
)

// Every title the documentalist gives a merge request passes the
// committer's rules once a squash adds " (#N)" (ADR-0051), for each task,
// its docs' paths short or long.
func TestMergeRequestTitlesPassTheSquashRule(t *testing.T) {
	r, err := role.Load("../../../roles", "committer")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r.MergedSettings(&role.ProjectConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	var s committer.Settings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{"docs/a.md", "docs/gitlab/environment.md", "docs/gitlab/a-page-whose-path-alone-is-long-enough.md"} {
		other := "docs/b.md"
		for _, c := range []struct {
			kind, file, task string
			parts, byAgent bool
		}{
			{kind: "suspect", byAgent: true},
			{kind: "derived"},
			{kind: "stale", byAgent: true},
			{kind: "propagate", byAgent: true},
			{kind: "fix", parts: true, byAgent: true},
			{kind: "condense", file: "condense.yaml", task: "doc: " + doc, byAgent: true},
			{kind: "split", file: "condense.yaml", task: "doc: " + doc + "\nsplit: true", byAgent: true},
			{kind: "duplicates", file: "dedupe.yaml", task: "docs: [" + doc + ", " + other + "]", byAgent: true},
			{kind: "reader", file: "reader.yaml", task: "pages: [" + doc + "]", byAgent: true},
			{kind: "reader", file: "reader.yaml", task: "pages: [" + doc + ", " + other + "]", byAgent: true},
			{kind: "merge-card", file: "merge-card.yaml", task: "card: " + doc, byAgent: true},
			{kind: recordsTask, byAgent: true},
		} {
			runDir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte(c.kind), 0o644); err != nil {
				t.Fatal(err)
			}
			if c.file != "" {
				if err := os.WriteFile(filepath.Join(runDir, "in", c.file), []byte(c.task), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var inParts map[string]string
			if c.parts {
				inParts = map[string]string{doc: "part", other: "part"}
			}
			mr := mergeRequest(runDir, map[string]map[string]string{doc: {}}, inParts, c.byAgent, nil, nil)
			found, err := committer.Check(mr["title"]+" (#12345)", s)
			if err != nil {
				t.Fatal(err)
			}
			if len(found) > 0 {
				t.Errorf("%s, %s: %q fails the committer's rules once squashed: %v", c.kind, doc, mr["title"], found)
			}
		}
	}
}
