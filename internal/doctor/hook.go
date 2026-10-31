package doctor

import (
	"fmt"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/role"
	"github.com/JN0V/workline/internal/routing"
)

// hookDoc is where a person reads how GitLab's issue webhook starts the line.
const hookDoc = "docs/gitlab-trigger.md#a-label-or-a-comment-with-gitlab-alone"

// IssueHookOf is the forge whose issue webhook starts the line of states
// (ADR-0059, decision 8): GitLab's, when a role of the line is routed;
// nil else. setup creates the hook through it, the doctor reads it.
func IssueHookOf(repo, spec string) forge.IssueHook {
	cfg, err := role.LoadProjectConfig(repo)
	if err != nil {
		return nil
	}
	line, err := routing.Load(repo)
	if err != nil {
		return nil
	}
	return issueHookOf(repo, spec, cfg, line)
}

func issueHookOf(repo, spec string, cfg *role.ProjectConfig, line *routing.Config) forge.IssueHook {
	if !routesRole(line, "product-owner") && !routesRole(line, "tester") && !routesRole(line, "developer") {
		return nil
	}
	if spec == "" {
		spec = cfg.Forge
	}
	if spec == "" && onGitLab(repo) {
		spec = "gitlab"
	}
	if spec != "gitlab" {
		return nil
	}
	f, err := forge.Open(spec, repo)
	if err != nil {
		return nil
	}
	h, _ := f.(forge.IssueHook)
	return h
}

// issueHook says whether GitLab's issue webhook starts the line at once:
// the hook to the project's own trigger URL, its issue events on, not
// disabled, on the default branch, and workline's trigger token. Read
// only; a token that may not read them (a Maintainer's may) is said.
func issueHook(r *Report, o Options, cfg *role.ProjectConfig, line *routing.Config) {
	h := issueHookOf(o.Repo, o.Forge, cfg, line)
	if h == nil {
		return
	}
	const setup = "workline setup --yes   # with a Maintainer's token: the webhook and its trigger token (" + hookDoc + ")"
	s, err := h.IssueHook()
	switch {
	case err != nil:
		r.add(Check{Area: "repository", Rule: "issue-hook-unread", Level: Warn,
			Message: "could not read whether GitLab's issue webhook starts the line: " + forge.Said(err) + "; reading the project's webhooks needs a Maintainer's token",
			Fix:     setup})
	case !s.Found:
		r.add(Check{Area: "repository", Rule: "issue-hook-missing", Level: Warn,
			Message: "no webhook starts a pipeline on an issue's label: the hourly reconciler alone moves the line",
			Fix:     setup})
	case !s.IssueEvents:
		r.add(Check{Area: "repository", Rule: "issue-hook-off", Level: Warn,
			Message: "the webhook to the project's trigger URL does not fire on issue events: the hourly reconciler alone moves the line",
			Fix:     setup})
	case s.Alert == "disabled" || s.Alert == "temporarily_disabled":
		r.add(Check{Area: "repository", Rule: "issue-hook-disabled", Level: Warn,
			Message: fmt.Sprintf("GitLab %s the issue webhook after its calls failed: the hourly reconciler alone moves the line", map[bool]string{true: "disabled", false: "paused"}[s.Alert == "disabled"]),
			Fix:     "Settings → Webhooks: read its recent events, fix the cause, then Enable (" + hookDoc + ")"})
	case s.Default != "" && s.Ref != s.Default:
		r.add(Check{Area: "repository", Rule: "issue-hook-ref", Where: s.Ref, Level: Warn,
			Message: fmt.Sprintf("the issue webhook starts pipelines on %q, not on the default branch %s: the job workline:issue runs on the default branch only", s.Ref, s.Default),
			Fix:     "Settings → Webhooks: its URL's /ref/<branch>/, the default branch (" + hookDoc + ")"})
	case !s.Trigger:
		r.add(Check{Area: "repository", Rule: "issue-hook", Level: Note,
			Message: fmt.Sprintf("GitLab's issue webhook starts the line at once, on %s; its trigger token is not one workline created, so whether it still exists is not checked", s.Ref)})
	default:
		r.add(Check{Area: "repository", Rule: "issue-hook", Level: OK,
			Message: fmt.Sprintf("GitLab's issue webhook starts the line at once, on %s, with workline's trigger token", s.Ref)})
	}
}
