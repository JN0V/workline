package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// Environment is the environment the GitLab template's jobs holding a
// token declare (`environment: {name: workline, action: prepare}`, #520):
// a variable scoped to it reaches them alone, never workline:gates nor
// workline:main, which run the project's tools.
const Environment = "workline"

// scopedTokens are the variables holding what the project's tools must
// never read: the bot's tokens, the agent's, the signing key.
var scopedTokens = []string{"WORKLINE_GITLAB_TOKEN", "WORKLINE_GITLAB_READ_TOKEN", "GITLAB_TOKEN",
	"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_API_KEY", "WORKLINE_SIGNING_KEY"}

// tokensScoped says, on GitLab, whether the project's variables holding
// workline's tokens are scoped to the environment its jobs declare: one
// left at `*` reaches every job, the gates' tools included. Read with the
// token the forge finds; only a Maintainer's may list the variables, and
// one that may not is said, never taken for none left open. No value is
// read nor said.
func tokensScoped(r *Report, o Options, projectForge string) {
	spec := o.Forge
	if spec == "" {
		spec = projectForge
	}
	if spec == "" && onGitLab(o.Repo) {
		spec = "gitlab"
	}
	if spec != "gitlab" {
		return
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil {
		return // said by findingsOnMergeRequests
	}
	v, ok := f.(forge.VariableScopes)
	if !ok {
		return
	}
	fix := fmt.Sprintf("Settings → CI/CD → Variables, each one's Environments: %s; the template's jobs that hold a token declare it (docs/ci-gitlab.md)", Environment)
	scopes, err := v.VariableScopes()
	if err != nil {
		r.add(Check{Area: "repository", Rule: "tokens-scope-unread", Level: Warn,
			Message: fmt.Sprintf("could not read whether workline's tokens are scoped to the environment %s: %s; listing the project's variables needs a Maintainer's token", Environment, forge.Said(err)),
			Fix:     fix})
		return
	}
	var open, scoped, elsewhere, found []string
	for _, key := range scopedTokens {
		s, set := scopes[key]
		if !set {
			continue
		}
		found = append(found, key)
		reaches := false
		for _, scope := range s {
			switch {
			case scope == "*":
				open = append(open, key)
			case matchScope(scope, Environment):
				reaches = true
			}
		}
		switch {
		case slices.Contains(open, key):
		case reaches:
			scoped = append(scoped, key)
		default:
			elsewhere = append(elsewhere, key)
		}
	}
	if len(open) > 0 {
		r.add(Check{Area: "repository", Rule: "tokens-not-scoped", Where: strings.Join(open, ", "), Level: Warn,
			Message: fmt.Sprintf("%s %s every job, workline:gates' and workline:main's tools included (environment scope *): scope %s to the environment %s",
				strings.Join(open, ", "), plural(open, "reaches", "reach"), plural(open, "it", "each"), Environment),
			Fix: fix})
	}
	if len(elsewhere) > 0 {
		r.add(Check{Area: "repository", Rule: "tokens-scoped-elsewhere", Where: strings.Join(elsewhere, ", "), Level: Note,
			Message: fmt.Sprintf("%s %s scoped to no environment named %s: the template's jobs receive %s only if your copy of it names that environment",
				strings.Join(elsewhere, ", "), plural(elsewhere, "is", "are"), Environment, plural(elsewhere, "it", "them")),
			Fix: fix})
	}
	if len(scoped) > 0 {
		r.add(Check{Area: "repository", Rule: "tokens-scoped", Where: strings.Join(scoped, ", "), Level: OK,
			Message: fmt.Sprintf("%s %s scoped to the environment %s: workline:gates and workline:main never receive %s",
				strings.Join(scoped, ", "), plural(scoped, "is", "are"), Environment, plural(scoped, "it", "them"))})
	}
	if len(found) == 0 {
		r.add(Check{Area: "repository", Rule: "tokens-not-project-variables", Level: Note,
			Message: "none of workline's tokens is a variable of the project: one set on a group or the instance reaches every job unless scoped, which a group's can be on Premium",
			Fix:     fix})
	}
}

// onGitLab says whether the repository's GitLab pipeline names workline:
// its forge is GitLab's, though its config names none (the template
// passes --forge gitlab).
func onGitLab(repo string) bool {
	data, err := os.ReadFile(filepath.Join(repo, ".gitlab-ci.yml"))
	return err == nil && strings.Contains(string(data), "workline")
}

// matchScope says whether a variable's environment scope, `*` a wildcard
// as GitLab reads it, takes the environment name.
func matchScope(scope, name string) bool {
	parts := strings.Split(scope, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(name)
}
