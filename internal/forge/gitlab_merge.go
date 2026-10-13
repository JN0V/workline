package forge

import (
	"fmt"
	"regexp"
	"time"
)

// autoMergeOn matches GitLab's system note on a merge request when
// auto-merge is turned on: "enabled an automatic merge when the pipeline
// for 1a2b3c4d succeeds" (merge when pipeline succeeds), "enabled an
// automatic merge when all merge checks for 1a2b3c4d pass", "set to
// auto-merge" (later versions).
var autoMergeOn = regexp.MustCompile(`(?i)^(enabled an automatic merge\b|set to auto-merge\b)`)

// MergeState reads a merge request — draft, head, auto-merge and who set
// it, its head pipeline's status —, its label events, its versions (one a
// push, with when) and its system notes, for when auto-merge was last
// turned on.
func (g *gitlab) MergeState(mr MergeRequest) (MergeState, error) {
	t := Target{Kind: "merge-request", ID: mr.ID}
	out, err := g.api(path(t))
	if err != nil {
		return MergeState{}, err
	}
	var v struct {
		SHA     string `json:"sha"`
		Draft   bool   `json:"draft"`
		Auto    bool   `json:"merge_when_pipeline_succeeds"`
		Created string `json:"created_at"`
		User    *struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		} `json:"merge_user"`
		Pipeline *struct {
			Status string `json:"status"`
		} `json:"head_pipeline"`
	}
	if err := decode(out, &v); err != nil {
		return MergeState{}, err
	}
	st := MergeState{Head: v.SHA, Draft: v.Draft, AutoMerge: v.Auto}
	if v.Auto && v.User != nil {
		st.AutoMergeBy, st.AutoMergeBot = v.User.Username, botName.MatchString(v.User.Username)
		if !st.AutoMergeBot && v.User.ID != 0 {
			if st.AutoMergeBot, err = g.isBot(v.User.ID); err != nil {
				return MergeState{}, err
			}
		}
	}
	if v.Pipeline != nil {
		st.Checks = pipelineStatus(v.Pipeline.Status)
	}
	if st.Labels, err = g.labelEvents(t); err != nil {
		return MergeState{}, err
	}
	out, err = g.api("--paginate", path(t)+"/versions?per_page=100")
	if err != nil {
		return MergeState{}, err
	}
	versions, err := pages[struct {
		Head string `json:"head_commit_sha"`
		At   string `json:"created_at"`
	}](out)
	if err != nil {
		return MergeState{}, err
	}
	st.Pushes = []Push{}
	for i := len(versions) - 1; i >= 0; i-- { // newest first: oldest first
		st.Pushes = append(st.Pushes, Push{Head: versions[i].Head, At: versions[i].At})
	}
	if mr.Here {
		if st.Pushers, err = g.pushers(mr.Branch, v.Created); err != nil {
			return MergeState{}, err
		}
	}
	out, err = g.api("--paginate", path(t)+"/notes?sort=asc&order_by=created_at&per_page=100")
	if err != nil {
		return MergeState{}, err
	}
	notes, err := pages[struct {
		Body    string `json:"body"`
		System  bool   `json:"system"`
		Created string `json:"created_at"`
	}](out)
	if err != nil {
		return MergeState{}, err
	}
	for _, n := range notes {
		if n.System && autoMergeOn.MatchString(n.Body) {
			st.AutoMergeAt = max(st.AutoMergeAt, n.Created)
		}
	}
	return st, nil
}

// pushers are who pushed to branch, oldest first, as the project's push
// events name them, from a week before the merge request was opened:
// workline's own branch is pushed as its merge request is opened.
func (g *gitlab) pushers(branch, opened string) ([]string, error) {
	q := "projects/:id/events?action=pushed&per_page=100"
	if t, err := time.Parse(time.RFC3339, opened); err == nil {
		q += "&after=" + t.AddDate(0, 0, -7).Format("2006-01-02")
	}
	out, err := g.api("--paginate", q)
	if err != nil {
		return nil, err
	}
	events, err := pages[struct {
		Author string `json:"author_username"`
		Push   struct {
			Ref     string `json:"ref"`
			RefType string `json:"ref_type"`
		} `json:"push_data"`
	}](out)
	if err != nil {
		return nil, err
	}
	who := []string{}
	for i := len(events) - 1; i >= 0; i-- { // newest first: oldest first
		if e := events[i]; e.Push.RefType == "branch" && e.Push.Ref == branch {
			who = append(who, e.Author)
		}
	}
	return who, nil
}

// pipelineStatus reads a GitLab pipeline's status as the merge policy sums
// checks up: success, pending, failure; "" for one manual or skipped.
func pipelineStatus(status string) string {
	switch status {
	case "success":
		return "success"
	case "failed", "canceled":
		return "failure"
	case "created", "waiting_for_resource", "preparing", "pending", "running", "scheduled":
		return "pending"
	}
	return ""
}

// AutoMerge sets the merge request to merge once its pipeline succeeds,
// at head only (`sha`) — `auto_merge`, and the older name of the same,
// `merge_when_pipeline_succeeds` —; GitLab merges at once one it could
// merge now, said by the merge request it answers. Off cancels it.
func (g *gitlab) AutoMerge(mr MergeRequest, head string, on bool) (bool, error) {
	t := Target{Kind: "merge-request", ID: mr.ID}
	if !on {
		_, err := g.api("-X", "POST", path(t)+"/cancel_merge_when_pipeline_succeeds")
		return false, err
	}
	out, err := g.api("-X", "PUT", path(t)+"/merge", "-f", "sha="+head, "-f", "auto_merge=true", "-f", "merge_when_pipeline_succeeds=true")
	if err != nil {
		return false, err
	}
	var v struct {
		State string `json:"state"`
	}
	if err := decode(out, &v); err != nil {
		return false, fmt.Errorf("merge request !%d: %w", mr.ID, err)
	}
	return v.State == "merged", nil
}

// Guarded: GitLab merges a merge request set to merge once the pipeline
// of its head succeeds, and a push starts a new one, holding the
// template's workline:merge, which takes a yes back before that pipeline
// can succeed (ci/gitlab/workline.gitlab-ci.yml).
func (g *gitlab) Guarded(base string) (string, error) { return "", nil }

// Accept: nothing to mark on GitLab, which holds a yes as Guarded says.
func (g *gitlab) Accept(head string, ok bool, why string) error { return nil }

// KeepVerdict keeps nothing on GitLab: the verdict is in hand in the
// pipeline that judged the head, where workline:merge reads it; a status
// set from elsewhere could make a pipeline of its own the merge request's.
func (g *gitlab) KeepVerdict(head, class, why string) error { return nil }

// Verdict: none kept on GitLab.
func (g *gitlab) Verdict(head string) (string, string, error) { return "", "", nil }
