package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/forge"
)

// findingsOnMergeRequests says how workline's findings reach a pull
// request, and whether the default branch requires every conversation
// resolved before a merge: code scanning opened one for each alert on a
// line the change touched, and held the merge on them (#412). Read from
// the forge, when it keeps such rules (GitHub); a rule it cannot read is
// said, never taken for none.
func findingsOnMergeRequests(r *Report, o Options, projectForge string) {
	spec := o.Forge
	if spec == "" {
		spec = projectForge
	}
	if spec == "" && onGitHub(o.Repo) {
		spec = "github"
	}
	f, err := forge.Open(spec, o.Repo)
	if err != nil {
		r.add(Check{Area: "repository", Rule: "forge-unknown", Level: Error, Message: err.Error()})
		return
	}
	rules, ok := f.(forge.ConversationRules)
	if !ok {
		return
	}
	const how = "the documentalist's findings reach pull requests as annotations on the check `workline findings`, which open none; " +
		"the reviewer's comments on lines are conversations to resolve, as a person's; a GitHub App writing the check needs the permission Checks: read and write"
	got, err := rules.Conversations("")
	branch := got.Branch
	if branch == "" {
		branch = "the default branch"
	}
	c := Check{Area: "repository", Rule: "findings-on-merge-requests", Where: got.Branch, Level: OK}
	switch {
	case err != nil:
		c.Level = Note
		c.Message = fmt.Sprintf("could not read whether %s requires every conversation resolved: %s; either way, %s", branch, forge.Said(err), how)
	case got.Required:
		c.Message = fmt.Sprintf("%s requires every conversation resolved before a merge: %s", branch, how)
	case got.Unread != "":
		c.Level = Note
		c.Message = fmt.Sprintf("could not read whether %s requires every conversation resolved: %s; either way, %s", branch, got.Unread, how)
	default:
		c.Message = fmt.Sprintf("%s does not require conversations resolved before a merge; %s", branch, how)
	}
	if applies, granted := applyWorkflow(o.Repo); applies && !granted {
		c.Level = Warn
		c.Message += "; but the workflow running `workline apply` grants no `checks: write`, as copied before the findings became annotations: they reach pull requests only in the judge's job summary"
		c.Fix = "copy ci/github/workline.yml and workline-apply.yml of this release again (docs/ci-github.md); a GitHub App writing the check needs the permission Checks: read and write"
	}
	r.add(c)
}

// workflows are the repository's GitHub workflows, each file's text.
func workflows(repo string) []string {
	files, _ := filepath.Glob(filepath.Join(repo, ".github", "workflows", "*.y*ml"))
	var out []string
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil {
			out = append(out, string(data))
		}
	}
	return out
}

// onGitHub says whether the repository's GitHub workflows run workline on
// pull requests, its origin on github.com: its forge is GitHub's, though
// its config names none (the templates pass --forge github).
func onGitHub(repo string) bool {
	runs := false
	for _, w := range workflows(repo) {
		runs = runs || strings.Contains(w, "workline route merge-request")
	}
	if !runs {
		return false
	}
	out, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").Output()
	return err == nil && strings.Contains(string(out), "github.com")
}

// applyWorkflow says whether a GitHub workflow applies a pull request's
// run (`--from-merge-request`), and whether each one doing so grants
// checks: write, which writing its check needs with the job's own token.
func applyWorkflow(repo string) (applies, granted bool) {
	granted = true
	for _, w := range workflows(repo) {
		if strings.Contains(w, "workline apply") && strings.Contains(w, "--from-merge-request") {
			applies = true
			granted = granted && strings.Contains(w, "checks: write")
		}
	}
	return applies, applies && granted
}
