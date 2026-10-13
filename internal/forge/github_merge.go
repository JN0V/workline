package forge

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// mergeStateQuery asks a pull request for what the merge policy weighs
// (ADR-0042): a draft, its head, its auto-merge on and who turned it on,
// and when auto-merge was last turned on, the last hundred such events.
const mergeStateQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { isDraft headRefOid autoMergeRequest { enabledAt enabledBy { login __typename } } timelineItems(itemTypes: [AUTO_MERGE_ENABLED_EVENT, AUTO_SQUASH_ENABLED_EVENT, AUTO_REBASE_ENABLED_EVENT], last: 100) { nodes { ... on AutoMergeEnabledEvent { createdAt } ... on AutoSquashEnabledEvent { createdAt } ... on AutoRebaseEnabledEvent { createdAt } } } } } }`

// checksQuery asks the head's checks as GitHub shows them on the pull
// request, each: check runs and commit statuses both.
const checksQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { commits(last: 1) { nodes { commit { statusCheckRollup { contexts(first: 100) { nodes { __typename ... on CheckRun { name status conclusion } ... on StatusContext { context state } } } } } } } } } }`

// MergeState reads a pull request's state in one GraphQL call, its
// checks in another — a token that may not read them leaves them unread,
// said —, its label events, and, for a branch of this repository, its
// pushes from the repository's activity: when each head was pushed, which
// a commit's own dates do not tell.
func (g *github) MergeState(mr MergeRequest) (MergeState, error) {
	vars := []string{"-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", mr.ID)}
	out, err := g.api(append([]string{"graphql", "-f", "query=" + mergeStateQuery}, vars...)...)
	if err != nil {
		return MergeState{}, err
	}
	var v struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Draft bool   `json:"isDraft"`
					Head  string `json:"headRefOid"`
					Auto  *struct {
						At string `json:"enabledAt"`
						By *struct {
							Login string `json:"login"`
							Type  string `json:"__typename"`
						} `json:"enabledBy"`
					} `json:"autoMergeRequest"`
					Timeline struct {
						Nodes []struct {
							At string `json:"createdAt"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return MergeState{}, err
	}
	pr := v.Data.Repository.PullRequest
	st := MergeState{Head: pr.Head, Draft: pr.Draft}
	for _, n := range pr.Timeline.Nodes {
		st.AutoMergeAt = max(st.AutoMergeAt, n.At)
	}
	if pr.Auto != nil {
		st.AutoMerge, st.AutoMergeAt = true, max(st.AutoMergeAt, pr.Auto.At)
		if pr.Auto.By != nil {
			st.AutoMergeBy, st.AutoMergeBot = pr.Auto.By.Login, pr.Auto.By.Type == "Bot"
		}
	}
	out, err = g.api(append([]string{"graphql", "-f", "query=" + checksQuery}, vars...)...)
	var c struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Commits struct {
						Nodes []struct {
							Commit struct {
								Rollup *struct {
									Contexts struct {
										Nodes []checkContext `json:"nodes"`
									} `json:"contexts"`
								} `json:"statusCheckRollup"`
							} `json:"commit"`
						} `json:"nodes"`
					} `json:"commits"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err == nil {
		err = decode(out, &c)
	}
	switch nodes := c.Data.Repository.PullRequest.Commits.Nodes; {
	case err != nil:
		st.ChecksUnread = err.Error()
	case len(nodes) == 1 && nodes[0].Commit.Rollup != nil:
		st.Checks = sumChecks(nodes[0].Commit.Rollup.Contexts.Nodes)
	}
	if st.Labels, err = g.LabelEvents(mr.ID); err != nil {
		return MergeState{}, err
	}
	if mr.Here {
		out, err := g.api("--paginate", "repos/{owner}/{repo}/activity?per_page=100&ref="+url.QueryEscape("refs/heads/"+mr.Branch),
			"--jq", `.[] | select(.activity_type == "push" or .activity_type == "force_push" or .activity_type == "branch_creation") | {head: .after, at: .timestamp}`)
		if err != nil {
			return MergeState{}, err
		}
		pushes, err := lines[Push](out)
		if err != nil {
			return MergeState{}, err
		}
		for i := len(pushes) - 1; i >= 0; i-- { // newest first: oldest first
			st.Pushes = append(st.Pushes, pushes[i])
		}
		if st.Pushes == nil {
			st.Pushes = []Push{} // the activity says none: not "unknown"
		}
	}
	return st, nil
}

// checkContext is a check run or a commit status on a head.
type checkContext struct {
	Type       string `json:"__typename"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	Context    string `json:"context"`
	State      string `json:"state"`
}

// sumChecks sums up a head's checks — failure when one failed, else
// pending when one runs, else success —, workline's own statuses left
// out: a verdict kept, a yes held, are no check of the change.
func sumChecks(all []checkContext) string {
	sum := ""
	for _, c := range all {
		state := ""
		switch {
		case c.Type == "StatusContext" && strings.HasPrefix(c.Context, "workline/"):
			continue
		case c.Type == "StatusContext":
			state = map[string]string{"SUCCESS": "success", "PENDING": "pending", "EXPECTED": "pending", "FAILURE": "failure", "ERROR": "failure"}[c.State]
		case c.Status != "COMPLETED":
			state = "pending"
		default:
			state = map[string]string{"SUCCESS": "success", "NEUTRAL": "success", "SKIPPED": "success",
				"FAILURE": "failure", "TIMED_OUT": "failure", "CANCELLED": "failure", "ACTION_REQUIRED": "failure", "STARTUP_FAILURE": "failure"}[c.Conclusion]
		}
		switch {
		case state == "failure":
			return "failure"
		case state == "pending" || sum == "pending":
			sum = "pending"
		case state == "success":
			sum = "success"
		}
	}
	return sum
}

// me is who this token writes as: the GitHub App's bot when the workflow
// names it (WORKLINE_APP), else the token's user, read once.
func (g *github) me() (string, error) {
	if app := strings.TrimSpace(os.Getenv("WORKLINE_APP")); app != "" {
		return app + "[bot]", nil
	}
	if g.login == "" {
		out, err := g.api("user", "--jq", ".login")
		if err != nil {
			return "", err
		}
		g.login = strings.TrimSpace(string(out))
	}
	return g.login, nil
}

// ownStatus is the last status of that name on a head set by this token:
// one by anyone else is never read.
func (g *github) ownStatus(head, name string) (state, description string, err error) {
	me, err := g.me()
	if err != nil {
		return "", "", err
	}
	out, err := g.api("--paginate", "repos/{owner}/{repo}/commits/"+head+"/statuses?per_page=100",
		"--jq", fmt.Sprintf(`.[] | select(.context == %q and .creator.login == %q) | {state, description}`, name, me))
	if err != nil {
		return "", "", err
	}
	found, err := lines[struct {
		State       string `json:"state"`
		Description string `json:"description"`
	}](out)
	if err != nil || len(found) == 0 {
		return "", "", err
	}
	return found[0].State, found[0].Description, nil // the newest first
}

// setStatus sets a status on a head, unless the last one this token set
// says the same.
func (g *github) setStatus(head, name, state, description string) error {
	was, said, err := g.ownStatus(head, name)
	if err != nil {
		return err
	}
	if was == state && said == statusText(description) {
		return nil
	}
	_, err = g.api("-X", "POST", "repos/{owner}/{repo}/statuses/"+head, "-f", "state="+state, "-f", "context="+name,
		"-f", "description="+statusText(description))
	return err
}

// KeepVerdict keeps the verdict as the status workline/verdict: success
// when it passed, pending for a release held for its docs, failure else.
func (g *github) KeepVerdict(head, class, why string) error {
	state := map[string]string{"pass": "success", "held": "pending"}[class]
	if state == "" {
		state = "failure"
	}
	return g.setStatus(head, VerdictStatus, state, verdictText(class, why))
}

// Verdict reads the status workline/verdict this token set on the head.
func (g *github) Verdict(head string) (string, string, error) {
	_, text, err := g.ownStatus(head, VerdictStatus)
	if err != nil {
		return "", "", err
	}
	class, why := readVerdict(text)
	return class, why, nil
}

// Accept sets the status workline/accepted on the head: success, the yes
// covering it; failure, why not.
func (g *github) Accept(head string, ok bool, why string) error {
	state := "failure"
	if ok {
		state = "success"
	}
	return g.setStatus(head, Accepted, state, why)
}

// Guarded: the base must require the status workline/accepted, which a
// new head lacks until `follow` sets it — in a ruleset, or the branch's
// protection —, pinned to the App (WORKLINE_APP_ID): unpinned, any token
// that may set a status could give a yes.
func (g *github) Guarded(base string) (string, error) {
	app := strings.TrimSpace(os.Getenv("WORKLINE_APP_ID"))
	out, err := g.api("repos/{owner}/{repo}/rules/branches/"+url.PathEscape(base), "--jq",
		`.[] | select(.type == "required_status_checks") | .parameters.required_status_checks[] | {context, app: ((.integration_id // "") | tostring)}`)
	if err != nil && !errors.Is(err, errNotFound) {
		return "", err
	}
	required, _ := lines[struct {
		Context string `json:"context"`
		App     string `json:"app"`
	}](out)
	out, err = g.api("repos/{owner}/{repo}/branches/"+url.PathEscape(base), "--jq",
		`(.protection.required_status_checks.checks // [])[] | {context, app: ((.app_id // "") | tostring)}`)
	if err == nil {
		more, _ := lines[struct {
			Context string `json:"context"`
			App     string `json:"app"`
		}](out)
		required = append(required, more...)
	}
	for _, r := range required {
		switch {
		case r.Context != Accepted:
			continue
		case app == "":
			return fmt.Sprintf("the GitHub App's id is not known (WORKLINE_APP_ID): whether %s is required from it cannot be told", Accepted), nil
		case r.App == "":
			return fmt.Sprintf("%s requires %s from any source: any token that may set a status could give a yes; pin it to the GitHub App (%s)", base, Accepted, app), nil
		case r.App != app:
			return fmt.Sprintf("%s requires %s from another app (%s), not workline's (%s)", base, Accepted, r.App, app), nil
		}
		return "", nil
	}
	return fmt.Sprintf("%s does not require the status %s: a push after the yes could merge before follow takes the yes back; require it, from the GitHub App, in a ruleset", base, Accepted), nil
}

// pullQuery asks a pull request's node id and the merge methods its
// repository allows, for the auto-merge mutations.
const pullQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { squashMergeAllowed mergeCommitAllowed rebaseMergeAllowed viewerDefaultMergeMethod pullRequest(number: $number) { id } } }`

// AutoMerge turns GitHub's auto-merge on with the repository's merge
// method, for head only (expectedHeadOid), or off. GitHub turns none on
// for a pull request it could merge now (mergeableNow): it is merged at
// once, at head, as `gh pr merge --auto` does — GitHub checking its rules
// on that merge as auto-merge would.
func (g *github) AutoMerge(mr MergeRequest, head string, on bool) (bool, error) {
	out, err := g.api("graphql", "-f", "query="+pullQuery, "-F", "owner={owner}", "-F", "name={repo}", "-F", fmt.Sprintf("number=%d", mr.ID))
	if err != nil {
		return false, err
	}
	var v struct {
		Data struct {
			Repository struct {
				Squash  bool   `json:"squashMergeAllowed"`
				Merge   bool   `json:"mergeCommitAllowed"`
				Rebase  bool   `json:"rebaseMergeAllowed"`
				Default string `json:"viewerDefaultMergeMethod"`
				Pull    struct {
					ID string `json:"id"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return false, err
	}
	r := v.Data.Repository
	if !on {
		_, err := g.api("graphql", "-f", "query=mutation($id: ID!) { disablePullRequestAutoMerge(input: {pullRequestId: $id}) { clientMutationId } }", "-f", "id="+r.Pull.ID)
		return false, graphqlRefusal(err)
	}
	allowed := map[string]bool{"SQUASH": r.Squash, "MERGE": r.Merge, "REBASE": r.Rebase}
	method := r.Default
	for _, m := range []string{"SQUASH", "MERGE", "REBASE"} {
		if !allowed[method] && allowed[m] {
			method = m
		}
	}
	_, err = g.api("graphql", "-f", "query=mutation($id: ID!, $head: GitObjectID!, $method: PullRequestMergeMethod!) { enablePullRequestAutoMerge(input: {pullRequestId: $id, expectedHeadOid: $head, mergeMethod: $method}) { clientMutationId } }",
		"-f", "id="+r.Pull.ID, "-f", "head="+head, "-f", "method="+method)
	if err == nil || !mergeableNow.MatchString(err.Error()) {
		return false, graphqlRefusal(err)
	}
	_, err = g.api("-X", "PUT", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/merge", mr.ID), "-f", "sha="+head, "-f", "merge_method="+strings.ToLower(method))
	return err == nil, err
}

// mergeableNow is GitHub's answer to auto-merge asked on a pull request it
// could merge now: its required checks passed ("clean"), only checks it
// does not require still running or red ("unstable", as the merge job's own
// check while it runs), or hooks left ("has_hooks").
var mergeableNow = regexp.MustCompile(`is in (clean|unstable|has_hooks) status`)

// graphqlRefusal reads an error GitHub's GraphQL API answered — gh says
// "GraphQL: …", the HTTP status a 200 — as the refusal it is: auto-merge
// not allowed on the repository, a head moved meanwhile.
func graphqlRefusal(err error) error {
	if err != nil && strings.Contains(err.Error(), "GraphQL:") && !refusal(err) {
		return fmt.Errorf("%w: %s", errRefused, err.Error())
	}
	return err
}
