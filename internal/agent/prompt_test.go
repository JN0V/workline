package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/role"
)

// Every role's output contract says how to write code and quotes so that
// they read (#138), and an answer asked for again repeats it.
func TestPromptAsksForBlockScalars(t *testing.T) {
	dir, run := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Review."), 0o644)
	os.MkdirAll(filepath.Join(run, "in"), 0o755)
	os.MkdirAll(filepath.Join(run, "out"), 0o755)
	os.WriteFile(filepath.Join(run, "in", "task.md"), []byte("The change."), 0o644)
	req := Request{RunDir: run, Repo: t.TempDir(), Role: &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"finding"}}}
	_, user, err := Prompt(req)
	if err != nil || !strings.Contains(user, BlockScalars) {
		t.Errorf("the output contract does not ask for block scalars: %v\n%s", err, user)
	}
}

// An intention the engine applies but the agent is never shown the shape
// of is one no agent proposes: the product owner's refine went unproposed
// on DomoticsCore for that.
func TestEveryIntentionHasItsShape(t *testing.T) {
	for kind := range intent.Catalogue {
		if contracts[kind] == "" {
			t.Errorf("%s: no shape in the output contract (contracts, prompt.go)", kind)
		}
	}
}

// A role's budget stops a prompt really over it (#235): estimated at four
// characters a token, a reviewer's call of 96.6k tokens passed a budget of
// 40000 as 29k. Code over the budget at the measured ratio is refused;
// prose within it is not.
func TestPromptBudgetCountsCodeAtTheMeasuredRatio(t *testing.T) {
	ask := func(task string) error {
		dir, run := t.TempDir(), t.TempDir()
		os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Review."), 0o644)
		os.MkdirAll(filepath.Join(run, "in"), 0o755)
		os.MkdirAll(filepath.Join(run, "out"), 0o755)
		os.WriteFile(filepath.Join(run, "in", "task.md"), []byte(task), 0o644)
		r := &role.Role{Name: "reviewer", Dir: dir, Intentions: []string{"note"}}
		r.Context.Budget = 4000
		_, _, err := Prompt(Request{RunDir: run, Repo: t.TempDir(), Role: r})
		return err
	}
	// About 5,000 characters of Go: 1,250 tokens at four a token, 4,800 measured.
	code := strings.Repeat("\tif n, err := f(x[i]); err != nil {\n\t\treturn 0, err\n\t}\n", 90)
	err := ask(code)
	if err == nil || !strings.Contains(err.Error(), "over the role's budget of 4000") || !strings.Contains(err.Error(), TokensRatio) {
		t.Errorf("code over the budget at the measured ratio is not refused, or not said: %v", err)
	}
	prose := strings.Repeat("The reviewer reads the change and the files it touches. ", 35)
	if err := ask(prose); err != nil {
		t.Errorf("prose within the budget is refused: %v", err)
	}
	if got := Tokens(117000); got < 96000 || got > 97200 {
		t.Errorf("a lens's call of 117k characters (96.6k tokens reported, #147) is estimated at %d", got)
	}
}
