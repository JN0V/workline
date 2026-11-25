package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Policy allows it: Given a public repository with an active event policy
// allowing pull_request_target for workline-bounds.yml and
// workline-merge.yml, whose listing carries no conditions or rules and
// whose own read by id returns its file targets and rules, When I run
// `workline doctor`, Then it prints no pull_request_target warning.
func TestDoctorReadsEachListedPolicyByItsIdForItsFileTargets(t *testing.T) {
	isolate(t)
	repo := eventPolicyRepo(t)
	merge := "name: workline merge\non:\n  pull_request_target: {types: [opened, synchronize, reopened]}\npermissions: {}\n" +
		"jobs:\n  merge:\n    runs-on: ubuntu-latest\n    steps:\n      - run: workline merge\n"
	if err := os.WriteFile(filepath.Join(repo, ".github", "workflows", "workline-merge.yml"), []byte(merge), 0o644); err != nil {
		t.Fatal(err)
	}
	// The list names the policy and says nothing of its file targets.
	list := `{"total_count":1,"policies":[{"id":42,"name":"allow-events","enforcement":"active"}]}`
	// The policy's own read carries its conditions and rules.
	one := `{"id":42,"name":"allow-events","enforcement":"active",` +
		`"conditions":{"workflow_path":{"include":[".github/workflows/workline-bounds.yml",".github/workflows/workline-merge.yml"],"exclude":[]}},` +
		`"rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["pull_request_target"]}}]}`
	fakeGh(t, "case \"$*\" in\n*actions/policies/42*) echo '"+one+"' ;;\n"+
		"*actions/policies*) echo '"+list+"' ;;\n"+
		"*rules/branches*|*rulesets*|*default_branch*) exit 1 ;;\n"+
		"*) echo '{\"visibility\":\"public\",\"private\":false}' ;;\nesac\n")

	r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
	found := false
	for _, c := range r.Checks {
		if !strings.Contains(c.Message, "pull_request_target") {
			continue
		}
		if c.Level == OK {
			found = true
			continue
		}
		t.Errorf("a policy read by its id allows pull_request_target for both files, yet a check about it is not OK: %+v", c)
	}
	if !found {
		t.Errorf("no OK check says the policy allows pull_request_target: %+v", r.Checks)
	}
}
