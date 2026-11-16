package doctor

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// eventPolicyDoc is where a person reads how to allow pull_request_target.
const eventPolicyDoc = "docs/ci-github.md#public-repositories-and-pull_request_target"

// blockedEvent says, of a public GitHub repository whose workflows run on
// pull_request_target, that GitHub blocks the event by default from
// December 2, 2026 (brownouts before): the bounds check never runs and no
// pull request merges (#1684). Private and internal repositories are not
// affected; an Actions event policy allowing the event is.
func blockedEvent(r *Report, o Options) {
	var users []string
	paths, _ := filepath.Glob(filepath.Join(o.Repo, ".github", "workflows", "*.y*ml"))
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil && triggersOn(data, "pull_request_target") {
			users = append(users, ".github/workflows/"+filepath.Base(p))
		}
	}
	if len(users) == 0 {
		return
	}
	if out, err := exec.Command("git", "-C", o.Repo, "remote", "get-url", "origin").Output(); err != nil || !strings.Contains(string(out), "github.com") {
		return
	}
	slices.Sort(users)
	where := strings.Join(users, ", ")
	var repo struct {
		Visibility string `json:"visibility"`
		Private    *bool  `json:"private"`
	}
	cmd := exec.Command("gh", "api", "repos/{owner}/{repo}")
	cmd.Dir = o.Repo
	out, err := cmd.Output()
	if err != nil || json.Unmarshal(out, &repo) != nil {
		r.add(Check{Area: "repository", Rule: "event-policy-unread", Where: where, Level: Note,
			Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026; whether this repository is public could not be read",
			Fix:     eventPolicyDoc})
		return
	}
	if repo.Visibility == "private" || repo.Visibility == "internal" || (repo.Visibility == "" && repo.Private != nil && *repo.Private) {
		return
	}
	cmd = exec.Command("gh", "api", "repos/{owner}/{repo}/actions/policies")
	cmd.Dir = o.Repo
	pol, err := cmd.Output()
	if err != nil {
		r.add(Check{Area: "repository", Rule: "event-policy-unread", Where: where, Level: Note,
			Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026; the Actions policies could not be read",
			Fix:     "allow pull_request_target for these workflow files in Settings → Actions → Policies (" + eventPolicyDoc + ")"})
		return
	}
	if policyAllows(pol, "pull_request_target", users) {
		r.add(Check{Area: "repository", Rule: "event-policy", Where: where, Level: OK,
			Message: "an Actions policy allows pull_request_target for " + where + ": GitHub's block of the event in public repositories does not stop them"})
		return
	}
	r.add(Check{Area: "repository", Rule: "event-blocked", Where: where, Level: Warn,
		Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026 (brownouts before): workline bounds ends in startup_failure and no pull request merges",
		Fix:     "Settings → Actions → Policies: a rule \"Restrict events\" allowing pull_request_target, targeted at " + where + " (" + eventPolicyDoc + ")"})
}

// triggersOn says whether a workflow's on: key names the event, as a
// scalar, a list or a map; a mention in a comment or a string does not count.
func triggersOn(data []byte, event string) bool {
	var wf struct {
		On yaml.Node `yaml:"on"`
	}
	if yaml.Unmarshal(data, &wf) != nil {
		return false
	}
	n := &wf.On
	switch n.Kind {
	case yaml.ScalarNode:
		return n.Value == event
	case yaml.SequenceNode:
		for _, c := range n.Content {
			if c.Kind == yaml.ScalarNode && c.Value == event {
				return true
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(n.Content); i += 2 {
			if n.Content[i].Value == event {
				return true
			}
		}
	}
	return false
}

// policyAllows says whether the policies JSON holds, for every workflow
// file in files, a policy that allows the event (it names the event, is not a
// deny, and targets no workflow path or one matching the file) and no covering
// policy that denies it.
func policyAllows(raw []byte, event string, files []string) bool {
	var doc struct {
		Policies []map[string]any `json:"policies"`
		Rules    []map[string]any `json:"rules"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return false
	}
	doc.Policies = append(doc.Policies, doc.Rules...)
	for _, f := range files {
		// a deny that covers the file fails it, whatever else allows it
		for _, p := range doc.Policies {
			if policyNames(p, event) && policyCovers(p, f) && policyDenies(p, event) {
				return false
			}
		}
		ok := false
		for _, p := range doc.Policies {
			if policyNames(p, event) && policyCovers(p, f) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func policyNames(v any, event string) bool {
	switch x := v.(type) {
	case string:
		return x == event
	case []any:
		for _, c := range x {
			if policyNames(c, event) {
				return true
			}
		}
	case map[string]any:
		for k, c := range x {
			if k == "conditions" {
				continue
			}
			if policyNames(c, event) {
				return true
			}
		}
	}
	return false
}

// policyDenies says whether the policy blocks the event: its action, effect
// (or alike) field says deny, or a deny-like key holds a value that names the
// event. A deny-like key that does not name the event (another event, false,
// an empty list) is no deny of it.
func policyDenies(v any, event string) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			lk := strings.ToLower(k)
			if k == "conditions" {
				continue
			}
			switch lk {
			case "action", "effect", "type", "behavior", "decision", "mode", "access", "permission":
				if s, ok := c.(string); ok && denyWord(s) {
					return true
				}
			}
			if denyWord(lk) && policyNames(c, event) {
				return true
			}
			if policyDenies(c, event) {
				return true
			}
		}
	case []any:
		for _, c := range x {
			if policyDenies(c, event) {
				return true
			}
		}
	}
	return false
}

func denyWord(s string) bool {
	s = strings.ToLower(s)
	for _, w := range []string{"deny", "denied", "block", "disallow"} {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// policyCovers says whether the policy's workflow_path conditions reach the
// file: no condition covers every file; an include must match, an exclude must not.
func policyCovers(p map[string]any, file string) bool {
	cond, _ := p["conditions"].(map[string]any)
	wp, _ := cond["workflow_path"].(map[string]any)
	if wp == nil {
		return true
	}
	match := func(key string) (listed, hit bool) {
		list, _ := wp[key].([]any)
		for _, e := range list {
			if pat, ok := e.(string); ok {
				listed = true
				if m, _ := path.Match(pat, file); m || pat == file {
					hit = true
				}
			}
		}
		return
	}
	if _, hit := match("exclude"); hit {
		return false
	}
	if listed, hit := match("include"); listed && !hit {
		return false
	}
	return true
}
