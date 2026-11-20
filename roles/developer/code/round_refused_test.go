package developer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/devmark"
)

// Round note honest
// Given a developer round whose change was refused at apply (a patch to a
// frozen test),
// When its note is posted,
// Then the note says the refusal, not that the change landed.
func TestRoundNoteSaysTheRefusalAtApply(t *testing.T) {
	repo := gitRepo(t, map[string]string{"src/lib.go": "package lib\n", "src/lib_test.go": "x\n"})
	runDir := t.TempDir()
	for _, d := range []string{"in", "out"} {
		if err := os.MkdirAll(filepath.Join(runDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON := func(name string, v any) {
		t.Helper()
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "in", name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON(roundFile, round{MR: 5, Branch: "workline/developer/3", At: "HEAD", Record: devmark.Act{Act: "code", Issue: 3},
		Asked: []devmark.Asked{{Key: "correctness|src/lib.go|wrong", Lens: "correctness", Where: "src/lib.go:1", Holds: "blocks", Text: "wrong"}}})
	writeJSON(boundsFile, Bounds{Issue: 3, Scope: []string{"src/lib.go"}, Frozen: map[string]int{"src/lib_test.go": 3}})
	diff := "diff --git a/src/lib_test.go b/src/lib_test.go\n--- a/src/lib_test.go\n+++ b/src/lib_test.go\n@@ -1 +1 @@\n-x\n+y\n"
	if err := intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), []intent.Intention{{Kind: "patch", Value: diff}}); err != nil {
		t.Fatal(err)
	}

	reviewPost(runDir, repo, Settings{Rounds: 3})

	out, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	body := ""
	for _, in := range out {
		if m, ok := in.Value.(map[string]any); ok && in.Kind == "comment" && m["sticky"] == "round" {
			body, _ = m["body"].(string)
		}
	}
	if body == "" {
		t.Fatalf("no round note among the intentions: %+v", out)
	}
	if !strings.Contains(strings.ToLower(body), "refused") {
		t.Errorf("the round note does not say the refusal:\n%s", body)
	}
	if strings.Contains(body, "its fix pushed") {
		t.Errorf("the round note says the change landed:\n%s", body)
	}
}
