package developer

import (
	"slices"
	"testing"
)

// A red test is a frozen one when its name, its last part, is a frozen
// file's name or a word in it (#987); a name too short to tell is never.
func TestFrozenRed(t *testing.T) {
	repo := gitRepo(t, map[string]string{
		"internal/doctor/summary_plural_test.go": "package doctor\n\nfunc TestSeveralWarnings(t *testing.T) {}\n",
		"tests/test_mul_by_zero.sh":              "[ \"$(mul 5 0)\" = 0 ]\n",
		"internal/doctor/own_test.go":            "package doctor\n\nfunc TestOwn(t *testing.T) {}\n",
	})
	acceptance := []string{"internal/doctor/summary_plural_test.go", "tests/test_mul_by_zero.sh"}
	red := []string{
		"github.com/x/y/internal/doctor.TestSeveralWarnings",     // a Go test, in a frozen file
		"github.com/x/y/internal/doctor.TestSeveralWarnings/two", // its subtest
		"test_mul_by_zero",                       // a test named by its file
		"github.com/x/y/internal/doctor.TestOwn", // the developer's own
		"pkg.TestX/case_1.5",                     // its last part too short to tell
	}
	got := frozenRed(repo, "HEAD", acceptance, red)
	if want := red[:3]; !slices.Equal(got, want) {
		t.Errorf("frozenRed = %q, want %q", got, want)
	}
}
