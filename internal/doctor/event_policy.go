package doctor

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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
		if data, err := os.ReadFile(p); err == nil && strings.Contains(string(data), "pull_request_target") {
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
	if strings.Contains(string(pol), "pull_request_target") {
		r.add(Check{Area: "repository", Rule: "event-policy", Where: where, Level: OK,
			Message: "an Actions policy names pull_request_target: GitHub's block of the event in public repositories does not stop " + where})
		return
	}
	r.add(Check{Area: "repository", Rule: "event-blocked", Where: where, Level: Warn,
		Message: where + " runs on pull_request_target, which GitHub blocks by default in public repositories from December 2, 2026 (brownouts before): workline bounds ends in startup_failure and no pull request merges",
		Fix:     "Settings → Actions → Policies: a rule \"Restrict events\" allowing pull_request_target, targeted at " + where + " (" + eventPolicyDoc + ")"})
}
