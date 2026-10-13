package forge

import "net/url"

// BranchPushers are who pushed to branch since it was created, oldest
// first, each push, as the forge names them — GitHub's repository
// activity —; nil when the forge does not say, or the branch's creation is
// not among what it keeps. A commit says anything of itself: only the
// forge's word tells who pushed it.
func BranchPushers(f any, branch string) ([]string, error) {
	if p, ok := f.(interface {
		branchPushers(branch string) ([]string, error)
	}); ok {
		return p.branchPushers(branch)
	}
	return nil, nil
}

func (g *github) branchPushers(branch string) ([]string, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/activity?per_page=100&ref="+url.QueryEscape("refs/heads/"+branch),
		"--jq", `.[] | select(.activity_type == "push" or .activity_type == "force_push" or .activity_type == "branch_creation") | {by: (.actor.login // ""), created: (.activity_type == "branch_creation")}`)
	if err != nil {
		return nil, err
	}
	events, err := lines[struct {
		By      string `json:"by"`
		Created bool   `json:"created"`
	}](out)
	if err != nil {
		return nil, err
	}
	var who []string
	for i := len(events) - 1; i >= 0; i-- { // newest first: oldest first
		if events[i].Created {
			who = []string{}
		}
		if who != nil {
			who = append(who, events[i].By)
		}
	}
	return who, nil
}

// branchPushers: as the file says (`branch-pushers`), nil for a branch it
// does not name.
func (f *Fake) branchPushers(branch string) ([]string, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	who, ok := s.BranchPushers[branch]
	if !ok {
		return nil, nil
	}
	if who == nil {
		who = []string{}
	}
	return who, nil
}
