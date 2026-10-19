package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// A moment's budget is a number of seconds above 0: 0, below it, NaN or
// infinity is refused, as a setting the engine cannot trust; one given is
// laid over the shipped ones, the others kept.
func TestBudgets(t *testing.T) {
	for _, bad := range []string{"0", "-1", ".nan", ".inf"} {
		repo := t.TempDir()
		os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
		os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("routing: {budgets: {verdict: "+bad+"}}\n"), 0o644)
		_, err := Load(repo)
		if !role.IsConfigError(err) || !strings.Contains(err.Error(), "routing.budgets.verdict") {
			t.Errorf("a budget of %s: %v, want it refused", bad, err)
		}
	}
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
	os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte("routing: {budgets: {verdict: 300}}\n"), 0o644)
	cfg, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := cfg.Budget("verdict"); b != 300 {
		t.Errorf("verdict's budget = %v, want the project's 300", b)
	}
	if b, _ := cfg.Budget("commit"); b != 1 {
		t.Errorf("commit's budget = %v, want the shipped 1", b)
	}
	if _, ok := cfg.Budget("schedule"); ok {
		t.Error("the night has a budget")
	}
}
