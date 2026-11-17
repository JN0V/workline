package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
)

// retryPromptOf is the prompt of a tester asked again after its answer
// was refused: the run folder holds the refusal's findings
// (out/feedback.md), and the refused answer as the engine keeps it
// (out/refused-1.yaml).
func retryPromptOf(t *testing.T, refused []intent.Intention, feedback string) string {
	t.Helper()
	dir, run := t.TempDir(), t.TempDir()
	for _, err := range []error{
		os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Write the tests."), 0o644),
		os.MkdirAll(filepath.Join(run, "in"), 0o755),
		os.MkdirAll(filepath.Join(run, "out"), 0o755),
		os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("Issue 12: the fetch closes its body."), 0o644),
		os.WriteFile(filepath.Join(run, "out", "feedback.md"), []byte(feedback), 0o644),
		intent.Write(filepath.Join(run, "out", "refused-1.yaml"), refused),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	r := &role.Role{Name: "tester", Dir: dir, Intentions: []string{"patch", "note"}}
	_, user, err := Prompt(Request{RunDir: run, Repo: t.TempDir(), Role: r})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

// Given the tester's tests refused for one errcheck finding, When the
// agent is asked again, Then it sees its own refused test and the
// finding: the prompt holds the finding and the refused test's lines.
func TestRetryShowsTheRefusedTestAndItsFinding(t *testing.T) {
	finding := "- tests-lint, internal/fetch/fetch_test.go:9: the gate lint, internal/fetch/fetch_test.go:9: Error return value of `resp.Body.Close` is not checked (errcheck)\n"
	refused := []intent.Intention{{Kind: "patch", Value: map[string]any{
		"file": "internal/fetch/fetch_test.go",
		"content": "package fetch\n\nimport \"testing\"\n\nfunc TestFetchClosesItsBody(t *testing.T) {\n" +
			"\tresp := get()\n\tdefer resp.Body.Close()\n}\n",
	}}}
	user := retryPromptOf(t, refused, finding)
	for _, want := range []string{"errcheck", "internal/fetch/fetch_test.go:9", "TestFetchClosesItsBody", "resp.Body.Close()"} {
		if !strings.Contains(user, want) {
			t.Errorf("the retry's prompt lacks %q:\n%s", want, user)
		}
	}
}

// Given a refusal for lint findings alone, When the agent is asked
// again, Then the retry starts from the refused tests: each refused
// file, with its lines, is given under the refusal, before the rules, so
// the agent changes lines instead of reading the code again to rewrite
// them. (The tokens in under a tenth of the first answer's is measured
// on a real agent's usage report, not here.)
func TestRetryStartsFromTheRefusedTests(t *testing.T) {
	refused := []intent.Intention{
		{Kind: "patch", Value: map[string]any{
			"file":    "internal/fetch/fetch_test.go",
			"content": "package fetch\n\nfunc TestFetchClosesItsBody() {\n\tdefer resp.Body.Close()\n}\n",
		}},
		{Kind: "patch", Value: map[string]any{
			"file":    "internal/fetch/helper_test.go",
			"content": "package fetch\n\nfunc stubServerForTheFetch() {\n\tserveCannedBody()\n}\n",
		}},
		{Kind: "note", Value: "assumes: serveCannedBody: read in the stub"},
	}
	user := retryPromptOf(t, refused, "- tests-lint, internal/fetch/fetch_test.go:4: errcheck\n")
	head := strings.Index(user, "# Your previous answer was refused")
	if head < 0 {
		t.Fatalf("no refusal in the prompt:\n%s", user)
	}
	after := user[head:]
	for _, want := range []string{"internal/fetch/fetch_test.go", "TestFetchClosesItsBody", "resp.Body.Close()",
		"internal/fetch/helper_test.go", "stubServerForTheFetch", "serveCannedBody()", "assumes:"} {
		if !strings.Contains(after, want) {
			t.Errorf("the refused tests are not given back under the refusal: lacks %q:\n%s", want, user)
		}
	}
}
