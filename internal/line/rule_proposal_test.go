package line

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Proposed: Given the same defect found on two merge requests, When the
// inspector runs, Then it proposes a rule and its test in a pull request.
// The inspector's role declares a proposal kind for a rule, which carries
// its test: `rule` among the intentions it may propose.
func TestProposed(t *testing.T) {
	data, err := os.ReadFile("../../roles/inspector/role.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^intentions:\s*\[([^\]]*)\]`).FindStringSubmatch(string(data))
	if m == nil {
		t.Fatal("role.yaml of the inspector has no intentions")
	}
	for _, k := range strings.Split(m[1], ",") {
		if strings.TrimSpace(k) == "rule" {
			return
		}
	}
	t.Errorf("the inspector's intentions are [%s]: no proposal kind `rule` for a rule and its test", m[1])
}
