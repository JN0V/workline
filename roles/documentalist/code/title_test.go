package documentalist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/commitmsg"
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
	var s commitmsg.Settings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{"docs/a.md", "docs/gitlab/environment.md", "docs/gitlab/a-page-whose-path-alone-is-long-enough.md"} {
		other := "docs/b.md"
		for _, c := range []struct {
			kind, file, task string
			parts, byAgent   bool
		}{
			{kind: "suspect", byAgent: true},
			{kind: "derived"},
			{kind: "stale", byAgent: true},
			{kind: "propagate", byAgent: true},
			{kind: "fix", parts: true, byAgent: true},
			{kind: "fix", parts: true},
			{kind: "split", file: "split.yaml", task: "doc: " + doc, byAgent: true},
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
			found, err := commitmsg.Check(mr["title"]+" (#12345)", s)
			if err != nil {
				t.Fatal(err)
			}
			if len(found) > 0 {
				t.Errorf("%s, %s: %q fails the committer's rules once squashed: %v", c.kind, doc, mr["title"], found)
			}
		}
	}
}

// A merge request fixing nothing says so in its title, and what the parts
// found in a doc the fix changed nothing of is listed in its description,
// where the person reviewing reads it (#2197).
func TestMergeRequestOfRecordsListsWhatIsLeft(t *testing.T) {
	runDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(runDir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task-kind"), []byte("fix"), 0o644); err != nil {
		t.Fatal(err)
	}
	found := []string{`line 10: "Tokens last one hour.", found wrong — src/token.go, line 4: "TTL = 7200"`}
	inParts := map[string]string{"docs/a.md": "abc1234", "docs/b.md": "abc1234"}
	outcome := map[string]string{
		"docs/a.md": "judged in parts; " + partsLeftOpen + foundList(found, "  "),
		"docs/b.md": "judged in parts, nothing found wrong; `checked` not moved",
	}
	mr := mergeRequest(runDir, map[string]map[string]string{"docs/a.md": {}}, inParts, false, nil, outcome)
	if want := "docs: record when docs/a.md and 1 more were judged in parts"; mr["title"] != want {
		t.Errorf("title %q, want %q", mr["title"], want)
	}
	if !strings.Contains(mr["body"], "\n  - "+found[0]+"\n") {
		t.Errorf("the description lists no finding of docs/a.md:\n%s", mr["body"])
	}
}
