package doctor

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/steer"
)

// eventPolicyDoc is where a person reads how to allow pull_request_target.
const eventPolicyDoc = "docs/github/pull-request-target.md"

// prTarget is the event GitHub blocks by default in public repositories.
const prTarget = "pull_request_target"

// blockedEvent says, of a public github.com repository whose workflows run
// on pull_request_target, that GitHub blocks the event by default from
// December 2, 2026 (brownouts before) unless an Actions event policy allows
// it: the bounds check never runs and no pull request merges (#1684). An
// event policy is an allow list, so it also says which events a policy
// covering a workflow file leaves out of it. Private and internal
// repositories are not affected, nor GitHub Enterprise Server.
func blockedEvent(r *Report, o Options) {
	files := map[string][]string{} // workflow file → the events of its on:
	var users []string             // the files on pull_request_target
	paths, _ := filepath.Glob(filepath.Join(o.Repo, ".github", "workflows", "*.y*ml"))
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		f := ".github/workflows/" + filepath.Base(p)
		files[f] = workflowEvents(data)
		if slices.Contains(files[f], prTarget) {
			users = append(users, f)
		}
	}
	if len(users) == 0 {
		return
	}
	out, err := exec.Command("git", "-C", o.Repo, "remote", "get-url", "origin").Output()
	if err != nil || steer.OriginHost(string(out)) != "github.com" {
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
	out, err = cmd.Output()
	if err != nil || json.Unmarshal(out, &repo) != nil {
		r.add(Check{Area: "repository", Rule: "event-policy-unread", Where: where, Level: Note,
			Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026; whether this repository is public could not be read",
			Fix:     eventPolicyDoc})
		return
	}
	if repo.Visibility == "private" || repo.Visibility == "internal" || (repo.Visibility == "" && repo.Private != nil && *repo.Private) {
		return
	}
	// has_parents is true by default: the organization's and the
	// enterprise's policies that apply to this repository come too.
	cmd = exec.Command("gh", "api", "repos/{owner}/{repo}/actions/policies?per_page=100")
	cmd.Dir = o.Repo
	raw, err := cmd.Output()
	var doc struct {
		Policies []actionsPolicy `json:"policies"`
	}
	if err == nil {
		err = json.Unmarshal(raw, &doc)
	}
	if err != nil {
		r.add(Check{Area: "repository", Rule: "event-policy-unread", Where: where, Level: Note,
			Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026; the Actions policies could not be read",
			Fix:     "allow pull_request_target for these workflow files in Settings → Actions → Policies (" + eventPolicyDoc + ")"})
		return
	}
	judgeEventPolicies(r, doc.Policies, files, users)
}

// judgeEventPolicies adds the checks: each workflow file on
// pull_request_target needs an active event policy covering it, and every
// active event policy covering a file must list all of the file's events.
func judgeEventPolicies(r *Report, policies []actionsPolicy, files map[string][]string, users []string) {
	var unallowed, evaluateOnly []string
	for _, f := range users {
		if len(coveringEventPolicies(policies, f, "active")) > 0 {
			continue
		}
		unallowed = append(unallowed, f)
		for _, p := range coveringEventPolicies(policies, f, "evaluate") {
			if allowed, _ := p.allowedEvents(); slices.Contains(allowed, prTarget) {
				evaluateOnly = append(evaluateOnly, f)
				break
			}
		}
	}
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	slices.Sort(names)
	var blocks []string // "file: events (policy)"
	blocked := map[string]bool{}
	for _, f := range names {
		for _, p := range coveringEventPolicies(policies, f, "active") {
			allowed, _ := p.allowedEvents()
			var missing []string
			for _, e := range files[f] {
				if e != "workflow_call" && !slices.Contains(allowed, e) {
					missing = append(missing, e)
				}
			}
			if len(missing) > 0 {
				blocks = append(blocks, fmt.Sprintf("%s: %s (policy %q)", f, strings.Join(missing, ", "), p.Name))
				blocked[f] = true
			}
		}
	}
	if len(blocks) > 0 {
		r.add(Check{Area: "repository", Rule: "event-policy-blocks", Where: strings.Join(slices.Sorted(maps.Keys(blocked)), ", "), Level: Warn,
			Message: "an Actions event policy is an allow list: an active one covering a workflow file blocks every event it does not list — " + strings.Join(blocks, "; "),
			Fix:     "add these events to the policy's allowed events, or narrow its workflow file targeting, in Settings → Actions → Policies (" + eventPolicyDoc + ")"})
	}
	if len(unallowed) > 0 {
		where := strings.Join(unallowed, ", ")
		msg := where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026 (brownouts before): workline bounds ends in startup_failure and no pull request merges"
		if len(evaluateOnly) > 0 {
			msg += "; the policy allowing it for " + strings.Join(evaluateOnly, ", ") + " is in Evaluate mode, which allows nothing: set it Active"
		}
		r.add(Check{Area: "repository", Rule: "event-blocked", Where: where, Level: Warn, Message: msg,
			Fix: "Settings → Actions → Policies: an Active policy with a rule \"Restrict events\" allowing pull_request_target and every other event of these files, targeted at " + where + " (" + eventPolicyDoc + ")"})
	}
	var fine []string
	for _, f := range users {
		if !slices.Contains(unallowed, f) && !blocked[f] {
			fine = append(fine, f)
		}
	}
	if len(fine) > 0 {
		where := strings.Join(fine, ", ")
		r.add(Check{Area: "repository", Rule: "event-policy", Where: where, Level: OK,
			Message: "an active Actions event policy allows pull_request_target and every other event of " + where + ": GitHub's block of the event in public repositories does not stop them"})
	}
}

// actionsPolicy is an Actions policy as GET /repos/{owner}/{repo}/actions/policies
// lists it (REST API version 2026-03-10).
type actionsPolicy struct {
	Name        string `json:"name"`
	Enforcement string `json:"enforcement"` // disabled, active, evaluate
	Conditions  *struct {
		WorkflowPath *struct {
			Include []string `json:"include"`
			Exclude []string `json:"exclude"`
		} `json:"workflow_path"`
	} `json:"conditions"`
	Rules []struct {
		Type       string `json:"type"`
		Parameters struct {
			AllowedEvents []string `json:"allowed_events"`
		} `json:"parameters"`
	} `json:"rules"`
}

// allowedEvents is what the policy's restrict_action_events rules allow;
// ok is false when it holds none (an actor rule alone restricts no event).
// Two such rules in one policy each have to pass: only the events both
// list are allowed.
func (p actionsPolicy) allowedEvents() (allowed []string, ok bool) {
	for _, rule := range p.Rules {
		if rule.Type != "restrict_action_events" {
			continue
		}
		if !ok {
			allowed, ok = slices.Clone(rule.Parameters.AllowedEvents), true
			continue
		}
		allowed = slices.DeleteFunc(allowed, func(e string) bool { return !slices.Contains(rule.Parameters.AllowedEvents, e) })
	}
	return allowed, ok
}

// covers says whether the policy's workflow_path condition reaches the
// file: no condition, an empty include or ~ALL include every file; an
// exclude matching the file leaves it out.
func (p actionsPolicy) covers(file string) bool {
	if p.Conditions == nil || p.Conditions.WorkflowPath == nil {
		return true
	}
	wp := p.Conditions.WorkflowPath
	for _, pat := range wp.Exclude {
		if globMatch(pat, file) {
			return false
		}
	}
	if len(wp.Include) == 0 {
		return true
	}
	for _, pat := range wp.Include {
		if pat == "~ALL" || globMatch(pat, file) {
			return true
		}
	}
	return false
}

// coveringEventPolicies are the policies of that enforcement that hold an
// event rule and cover the file.
func coveringEventPolicies(policies []actionsPolicy, file, enforcement string) []actionsPolicy {
	var out []actionsPolicy
	for _, p := range policies {
		if _, ok := p.allowedEvents(); ok && p.Enforcement == enforcement && p.covers(file) {
			out = append(out, p)
		}
	}
	return out
}

// globMatch matches a workflow path pattern as rulesets do: * and ? stay
// within a directory, ** crosses them. A leading ./ or / is ignored.
func globMatch(pat, file string) bool {
	pat = strings.TrimPrefix(strings.TrimPrefix(pat, "./"), "/")
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pat); i++ {
		switch c := pat[i]; c {
		case '*':
			if i+1 < len(pat) && pat[i+1] == '*' {
				i++
				if i+1 < len(pat) && pat[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(file)
}

// workflowEvents are the events a workflow's on: key names, as a scalar, a
// list or a map; a mention in a comment or a string does not count.
func workflowEvents(data []byte) []string {
	var wf struct {
		On yaml.Node `yaml:"on"`
	}
	if yaml.Unmarshal(data, &wf) != nil {
		return nil
	}
	n := &wf.On
	var events []string
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Value != "" {
			events = append(events, n.Value)
		}
	case yaml.SequenceNode:
		for _, c := range n.Content {
			if c.Kind == yaml.ScalarNode {
				events = append(events, c.Value)
			}
		}
	case yaml.MappingNode:
		for i := 0; i < len(n.Content); i += 2 {
			events = append(events, n.Content[i].Value)
		}
	}
	return events
}
