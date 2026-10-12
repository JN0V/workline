package forge

import (
	"fmt"
	"net/url"
	"strings"
)

// mergeStateQuery asks a pull request for what the merge policy weighs
// (ADR-0042): a draft, its head, its auto-merge on and who turned it on,
// and when auto-merge was last turned on, the last hundred such events.
const mergeStateQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { isDraft headRefOid autoMergeRequest { enabledAt enabledBy { login __typename } } timelineItems(itemTypes: [AUTO_MERGE_ENABLED_EVENT, AUTO_SQUASH_ENABLED_EVENT, AUTO_REBASE_ENABLED_EVENT], last: 100) { nodes { ... on AutoMergeEnabledEvent { createdAt } ... on AutoSquashEnabledEvent { createdAt } ... on AutoRebaseEnabledEvent { createdAt } } } } } }`

// checksQuery asks the head's checks, summed up as GitHub shows them on
// the pull request: check runs and commit statuses both.
const checksQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { pullRequest(number: $number) { commits(last: 1) { nodes { commit { statusCheckRollup { state } } } } } } }`

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
									State string `json:"state"`
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
		st.Checks = rollup(nodes[0].Commit.Rollup.State)
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

// rollup reads GitHub's sum of a commit's checks: success, pending,
// failure.
func rollup(state string) string {
	switch state {
	case "SUCCESS":
		return "success"
	case "PENDING", "EXPECTED":
		return "pending"
	case "FAILURE", "ERROR":
		return "failure"
	}
	return ""
}

// pullQuery asks a pull request's node id and the merge methods its
// repository allows, for the auto-merge mutations.
const pullQuery = `query($owner: String!, $name: String!, $number: Int!) { repository(owner: $owner, name: $name) { squashMergeAllowed mergeCommitAllowed rebaseMergeAllowed viewerDefaultMergeMethod pullRequest(number: $number) { id } } }`

// AutoMerge turns GitHub's auto-merge on with the repository's merge
// method, for head only (expectedHeadOid), or off. GitHub turns none on
// for a pull request it could merge now ("clean status"): it is merged at
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
	if err == nil || !strings.Contains(err.Error(), "clean status") {
		return false, graphqlRefusal(err)
	}
	_, err = g.api("-X", "PUT", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/merge", mr.ID), "-f", "sha="+head, "-f", "merge_method="+strings.ToLower(method))
	return err == nil, err
}

// graphqlRefusal reads an error GitHub's GraphQL API answered — gh says
// "GraphQL: …", the HTTP status a 200 — as the refusal it is: auto-merge
// not allowed on the repository, a head moved meanwhile.
func graphqlRefusal(err error) error {
	if err != nil && strings.Contains(err.Error(), "GraphQL:") && !refusal(err) {
		return fmt.Errorf("%w: %s", errRefused, err.Error())
	}
	return err
}
