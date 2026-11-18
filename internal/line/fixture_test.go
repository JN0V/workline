package line

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/contract"
	"github.com/JN0V/workline/internal/forge"
)

// The line's tests run on a fixture role in place of the developer
// (docs/spec/layout.md, Tests): the real one is tried in tests/ and in
// its own package. A claimed issue holds the files of the open merge
// requests that close it; none overlaps; a claim keeps 24 hours.
func TestMain(m *testing.M) {
	r := contract.NewRegistry()
	r.Join("fixture-coder", contract.Role{Code: &contract.Code{
		RoundDue: func(forge.Forge, string, forge.MergeRequest) (bool, error) { return false, nil },
		Held: func(issues []forge.Issue, mrs []forge.MergeRequest, claimLabel string, letGo map[int]bool, _ []string, _ contract.Diffs) map[int][]string {
			held := map[int][]string{}
			for _, is := range issues {
				if !slices.Contains(is.Labels, claimLabel) || letGo[is.ID] {
					continue
				}
				for _, mr := range mrs {
					if strings.Contains(mr.Body, fmt.Sprintf("Closes #%d", is.ID)) {
						held[is.ID] = append(held[is.ID], fmt.Sprintf("mr-%d", mr.ID))
					}
				}
			}
			return held
		},
		Overlapping: func(forge.Issue, map[int][]string, []string) (int, string) { return 0, "" },
		Changed:     func(string, bool) contract.Diffs { return nil },
		RepoFiles:   func(string) []string { return nil },
		ClaimHours:  func(map[string]any) int { return 24 },
	}})
	restore := contract.UseForTest(r)
	code := m.Run()
	restore()
	os.Exit(code)
}
