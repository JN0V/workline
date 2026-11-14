package developer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// One case, its group: given a change touching one conformance case
// (tests/conformance/cases/dev/x.yaml), when the developer's proof runs,
// then the proof command runs that case's group: its package carries a
// `-run` naming the group `dev`, not the whole suite.
func TestProveOneCaseRunsItsGroup(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	roles, err := filepath.Abs("../../roles")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_ROLES_DIR", roles)
	// The script writes one red test whose name is the arguments it got
	// after the report path.
	script := `f=$1
shift
printf '<testsuite name="s"><testcase classname="c" name="args %s"><failure message="x"/></testcase></testsuite>\n' "$*" > "$f"
exit 1
`
	repo := gitRepo(t, map[string]string{
		"run-tests.sh":                           script,
		"tests/conformance/runner_test.go":       "package conformance\n",
		"tests/conformance/cases/dev/x.yaml":     "case: x\n",
		"tests/conformance/cases/other/y.yaml":   "case: y\n",
		".workline/config.yaml": "roles:\n  tester:\n    settings:\n      test-command: \"sh run-tests.sh {report}\"\n",
	})
	patches := []intent.Intention{{Kind: "patch", Value: map[string]any{
		"file": "tests/conformance/cases/dev/x.yaml", "content": "case: x\nabout: changed\n",
	}}}
	s := Settings{ProofSeconds: 60, ProofCommand: "sh run-tests.sh {report} {packages}"}
	p, c := prove(repo, "", patches, nil, nil, s)
	if c != nil || len(p.Red) != 1 {
		t.Fatalf("proof = %+v, cap = %+v; want one red test naming the arguments", p, c)
	}
	args := p.Red[0]
	if !strings.Contains(args, "./tests/conformance") {
		t.Errorf("arguments %q lack the package ./tests/conformance", args)
	}
	if !strings.Contains(args, "-run") || !strings.Contains(args, "dev") {
		t.Errorf("arguments %q: want a -run naming the group dev, not the whole suite", args)
	}
	if strings.Contains(args, "other") {
		t.Errorf("arguments %q name a group the change does not touch", args)
	}
}
