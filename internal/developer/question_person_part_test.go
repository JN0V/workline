package developer

import (
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// Mixed Scope: Given a Scope naming a CI workflow and code, When it is ready,
// Then the developer's task says which part a person makes: the workflow,
// never the code.
func TestTaskTextSaysWhichPartAPersonMakes(t *testing.T) {
	is := forge.Issue{ID: 5, Title: "Lint in CI", Body: "## Need\n\nRun the linter in CI."}
	b := Bounds{Issue: 5, Scope: []string{".github/workflows/ci.yml", "internal/lint.go"}}
	task := taskText(is, b, "", nil, Settings{})
	said := false
	for _, para := range strings.Split(task, "\n\n") {
		if strings.Contains(para, ".github/workflows/ci.yml") && strings.Contains(para, "person") {
			said = true
			if strings.Contains(para, "a person makes") && strings.Contains(para, "internal/lint.go") && !strings.Contains(para, "- `internal/lint.go`") {
				t.Errorf("the part a person makes names the code too:\n%s", para)
			}
		}
	}
	if !said {
		t.Errorf("the task does not say a person makes the workflow:\n%s", task)
	}
}
