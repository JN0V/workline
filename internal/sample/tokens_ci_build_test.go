package sample

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

// tokensOfOneRunOn is the week 2026-W02's tokens when its only run, one
// agent call of 3,000 tokens in and 500 out, was on a merge request
// titled title that closes no issue.
func tokensOfOneRunOn(t *testing.T, title string) *Tokens {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "20260106T090000.000000000-reviewer")
	if err := os.MkdirAll(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "out", "run.yaml"), []byte("role: reviewer\ntarget: {kind: merge-request, id: 7}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "out", "calls.jsonl"), []byte(`{"agent":"claude","tokens-in":3000,"tokens-out":500}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := worksForge{mrs: map[int]forge.MergeRequest{7: {Title: title}}}
	return tokensOf(base, f, time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC))
}

// ci is upkeep
// Given a week whose only reviewed pull request is `ci: …` and closes no issue,
// When the weekly sample runs,
// Then its tokens count as upkeep, not unknown.
func TestCiMergeRequestTokensAreUpkeep(t *testing.T) {
	tk := tokensOfOneRunOn(t, "ci: cache the build")
	if tk.Upkeep != 3500 || tk.Unknown != 0 || tk.Product != 0 {
		t.Errorf("upkeep %d, unknown %d, product %d; want upkeep 3500 and nothing else (%+v)", tk.Upkeep, tk.Unknown, tk.Product, tk)
	}
}

// build is upkeep
// Given a pull request `build: bump Go`, closing no issue,
// When the weekly sample runs,
// Then its tokens count as upkeep.
func TestBuildMergeRequestTokensAreUpkeep(t *testing.T) {
	tk := tokensOfOneRunOn(t, "build: bump Go")
	if tk.Upkeep != 3500 || tk.Unknown != 0 || tk.Product != 0 {
		t.Errorf("upkeep %d, unknown %d, product %d; want upkeep 3500 and nothing else (%+v)", tk.Upkeep, tk.Unknown, tk.Product, tk)
	}
}
