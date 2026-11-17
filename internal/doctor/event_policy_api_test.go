package doctor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// policyJSON is one policy as GET /repos/{owner}/{repo}/actions/policies
// lists it (REST API 2026-03-10); conditions "" leaves them out.
func policyJSON(name, source, enforcement, conditions string, allowed ...string) string {
	events, _ := json.Marshal(allowed)
	c := ""
	if conditions != "" {
		c = `"conditions":` + conditions + `,`
	}
	return fmt.Sprintf(`{"id":1,"name":%q,"target":"actions","source_type":%q,"source":"example","enforcement":%q,%s`+
		`"rules":[{"type":"restrict_action_events","parameters":{"allowed_events":%s}}],"node_id":"x","created_at":"2026-11-01T00:00:00Z"}`,
		name, source, enforcement, c, events)
}

func judgeJSON(t *testing.T, policies ...string) *Report {
	t.Helper()
	raw := fmt.Sprintf(`{"total_count":%d,"policies":[%s]}`, len(policies), strings.Join(policies, ","))
	var doc struct {
		Policies []actionsPolicy `json:"policies"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	files := map[string][]string{
		".github/workflows/workline-bounds.yml": {"pull_request_target"},
		".github/workflows/workline-merge.yml":  {"pull_request_target", "issues", "schedule"},
		".github/workflows/workline.yml":        {"schedule", "workflow_dispatch"},
	}
	r := &Report{}
	judgeEventPolicies(r, doc.Policies, files, []string{".github/workflows/workline-bounds.yml", ".github/workflows/workline-merge.yml"})
	return r
}

func levelOf(r *Report, rule string) string {
	for _, c := range r.Checks {
		if c.Rule == rule {
			return c.Level
		}
	}
	return ""
}

const bothFiles = `{"workflow_path":{"include":[".github/workflows/workline-bounds.yml",".github/workflows/workline-merge.yml"],"exclude":[]}}`

func TestEventPolicyActiveAllowingEveryEventIsOK(t *testing.T) {
	r := judgeJSON(t, policyJSON("workline", "Repository", "active", bothFiles, "pull_request_target", "issues", "schedule"))
	if levelOf(r, "event-policy") != OK || levelOf(r, "event-blocked") != "" || levelOf(r, "event-policy-blocks") != "" {
		t.Errorf("an active policy allowing every event is not OK alone: %+v", r.Checks)
	}
}

func TestEventPolicyLeavingOutAnEventOfTheFileWarns(t *testing.T) {
	r := judgeJSON(t, policyJSON("prt only", "Repository", "active", bothFiles, "pull_request_target"))
	var c Check
	for _, x := range r.Checks {
		if x.Rule == "event-policy-blocks" {
			c = x
		}
	}
	if c.Level != Warn || !strings.Contains(c.Message, "issues, schedule") || !strings.Contains(c.Where, "workline-merge.yml") {
		t.Errorf("a policy allowing only pull_request_target does not warn of issues and schedule in workline-merge.yml: %+v", r.Checks)
	}
	if strings.Contains(c.Where, "workline-bounds.yml") || levelOf(r, "event-policy") != OK {
		t.Errorf("workline-bounds.yml, whose one event is allowed, is not OK: %+v", r.Checks)
	}
}

func TestEventPolicyEvaluateOnlyWarns(t *testing.T) {
	r := judgeJSON(t, policyJSON("trial", "Repository", "evaluate", bothFiles, "pull_request_target", "issues", "schedule"))
	if levelOf(r, "event-blocked") != Warn || levelOf(r, "event-policy") != "" {
		t.Fatalf("an evaluate policy gives OK or no warning: %+v", r.Checks)
	}
	for _, c := range r.Checks {
		if c.Rule == "event-blocked" && !strings.Contains(c.Message, "Evaluate mode") {
			t.Errorf("the warning does not say the policy is only in Evaluate mode: %q", c.Message)
		}
	}
}

func TestEventPolicyDisabledIsIgnored(t *testing.T) {
	r := judgeJSON(t, policyJSON("off", "Repository", "disabled", "", "push"))
	if levelOf(r, "event-blocked") != Warn || levelOf(r, "event-policy-blocks") != "" {
		t.Errorf("a disabled policy counts: %+v", r.Checks)
	}
}

func TestEventPolicyAllIncludesEveryFile(t *testing.T) {
	r := judgeJSON(t, policyJSON("all", "Organization", "active", `{"workflow_path":{"include":["~ALL"],"exclude":[]}}`,
		"pull_request_target", "issues", "schedule"))
	if levelOf(r, "event-policy") != OK {
		t.Errorf("~ALL does not cover the line's files: %+v", r.Checks)
	}
	// ~ALL covers workline.yml too, whose workflow_dispatch it leaves out.
	if levelOf(r, "event-policy-blocks") != Warn {
		t.Errorf("~ALL leaving out workflow_dispatch of workline.yml does not warn: %+v", r.Checks)
	}
}

func TestEventPolicyCoveringOtherFilesDoesNotAllow(t *testing.T) {
	r := judgeJSON(t, policyJSON("others", "Repository", "active", `{"workflow_path":{"include":[".github/workflows/deploy-*.yml"],"exclude":[]}}`,
		"pull_request_target"))
	if levelOf(r, "event-blocked") != Warn || levelOf(r, "event-policy") != "" {
		t.Errorf("a policy on other files allows the line's: %+v", r.Checks)
	}
}

func TestEventPolicyExcludeLeavesTheFileOut(t *testing.T) {
	r := judgeJSON(t, policyJSON("all but bounds", "Repository", "active",
		`{"workflow_path":{"include":[],"exclude":[".github/workflows/workline-bounds.yml"]}}`, "pull_request_target", "issues", "schedule", "workflow_dispatch"))
	for _, c := range r.Checks {
		if c.Rule == "event-blocked" && c.Where != ".github/workflows/workline-bounds.yml" {
			t.Errorf("the excluded file alone is not blocked: %+v", c)
		}
	}
	if levelOf(r, "event-blocked") != Warn || levelOf(r, "event-policy") != OK {
		t.Errorf("exclude is not read: %+v", r.Checks)
	}
}

// Two active policies stack: an organization's allowing the event
// everywhere does not lift a repository's that leaves it out.
func TestEventPolicyStackedPoliciesEachHaveToAllow(t *testing.T) {
	r := judgeJSON(t,
		policyJSON("org allows", "Organization", "active", "", "pull_request_target", "issues", "schedule", "workflow_dispatch"),
		policyJSON("repo narrows", "Repository", "active", `{"workflow_path":{"include":[".github/workflows/**"],"exclude":[]}}`, "push", "schedule", "workflow_dispatch"))
	var c Check
	for _, x := range r.Checks {
		if x.Rule == "event-policy-blocks" {
			c = x
		}
	}
	if c.Level != Warn || !strings.Contains(c.Message, `"repo narrows"`) || !strings.Contains(c.Where, "workline-bounds.yml") {
		t.Errorf("the narrower stacked policy does not block: %+v", r.Checks)
	}
	if levelOf(r, "event-policy") != "" {
		t.Errorf("OK given while a stacked policy blocks: %+v", r.Checks)
	}
}

func TestEventPolicyActorRuleAloneRestrictsNoEvent(t *testing.T) {
	actors := `{"id":2,"name":"actors","target":"actions","source_type":"Repository","source":"example","enforcement":"active",` +
		`"rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":1234,"type":"Team"}]}}]}`
	r := judgeJSON(t, actors)
	if levelOf(r, "event-blocked") != Warn || levelOf(r, "event-policy-blocks") != "" {
		t.Errorf("an actor rule is read as an event policy: %+v", r.Checks)
	}
}

func TestGlobMatch(t *testing.T) {
	for _, c := range []struct {
		pat, file string
		want      bool
	}{
		{".github/workflows/workline-*.yml", ".github/workflows/workline-merge.yml", true},
		{".github/workflows/*.yml", ".github/workflows/sub/x.yml", false},
		{".github/**", ".github/workflows/x.yml", true},
		{"**/workline-bounds.yml", ".github/workflows/workline-bounds.yml", true},
		{"workline-bounds.yml", ".github/workflows/workline-bounds.yml", false},
		{".github/workflows/workline.yml", ".github/workflows/workline-merge.yml", false},
	} {
		if got := globMatch(c.pat, c.file); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pat, c.file, got, c.want)
		}
	}
}

// A remote whose path names github.com is not on github.com: only the host says.
func TestEventPolicyReadsTheRemoteHost(t *testing.T) {
	isolate(t)
	repo := eventPolicyRepo(t)
	if out, err := exec.Command("git", "-C", repo, "remote", "set-url", "origin", "https://gitlab.example.com/mirrors/github.com/public-repo.git").CombinedOutput(); err != nil {
		t.Fatalf("git remote set-url: %v: %s", err, out)
	}
	fakeGh(t, "echo '{\"visibility\":\"public\",\"private\":false}'\n")
	r := Run(Options{Repo: repo, RolesDir: rolesDir, Engine: "v0.71.0"})
	for _, c := range r.Checks {
		if strings.Contains(c.Message, "pull_request_target") {
			t.Errorf("a remote off github.com is checked for the block: %+v", c)
		}
	}
}
