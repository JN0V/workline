package forge

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// HoldLabel is the label an open issue carries to hold the release
// (ADR-0051): the person takes it off, or closes the issue, to let the
// release go. One spelling on every forge: GitLab's scoped `workline::`
// would take the place of the role's other labels on the issue.
const HoldLabel = "workline:hold-release"

// Tip is a branch's tip and its checks as the forge keeps them: State
// success, pending or failure as the merge policy sums checks up, ""
// when none is there; the checks red, and those not done, by name.
type Tip struct {
	Branch  string   `json:"branch"`
	Commit  string   `json:"commit"`
	State   string   `json:"state,omitempty"`
	Failing []string `json:"failing,omitempty"`
	Pending []string `json:"pending,omitempty"`
}

// Releasing is a forge that says what holds a release (ADR-0051): the
// open issues carrying HoldLabel, and a branch's tip with its checks,
// read, never run again. GitHub, GitLab and the simulated forge say both.
type Releasing interface {
	// Holding lists the open issues carrying HoldLabel, merge requests
	// left out.
	Holding() ([]Issue, error)
	// Tip reads a branch's tip and its checks, the default branch when
	// branch is "". Only what ran for the commit counts: on GitHub, the
	// runs a push started and other apps' checks, never a schedule's nor
	// an issue's, which GitHub files under the default branch's tip.
	Tip(branch string) (Tip, error)
}

var _ = []Releasing{&github{}, &gitlab{}, &Fake{}}

// tipOf sums a tip's runs up as the merge policy does (sumRuns), and
// names the checks red and those not done: a check cancelled with no
// other run of it is not done.
func tipOf(t Tip, runs []CheckRun) Tip {
	t.State = sumRuns(runs)
	done := map[string]bool{}
	for _, r := range runs {
		if r.State != "cancelled" && r.State != "pending" {
			done[r.Name] = true
		}
	}
	for _, r := range runs {
		switch {
		case r.State == "failure" && !slices.Contains(t.Failing, r.Name):
			t.Failing = append(t.Failing, r.Name)
		case (r.State == "pending" || r.State == "cancelled") && !done[r.Name] && !slices.Contains(t.Pending, r.Name):
			t.Pending = append(t.Pending, r.Name)
		}
	}
	return t
}

func (g *github) Holding() ([]Issue, error) {
	out, err := g.api("--paginate", "repos/{owner}/{repo}/issues?state=open&per_page=100&labels="+url.QueryEscape(HoldLabel),
		"--jq", ".[] | select(.pull_request == null) | {id: .number, title, labels: [.labels[].name]}")
	if err != nil {
		return nil, err
	}
	return lines[Issue](out)
}

// tipQuery asks a branch's tip for its check suites, each with the event
// and the workflow that started it when GitHub Actions ran it, and its
// commit statuses.
const tipQuery = `query($owner: String!, $name: String!, $ref: String!) { repository(owner: $owner, name: $name) { ref(qualifiedName: $ref) { target { ... on Commit { oid checkSuites(first: 100) { nodes { status conclusion app { slug } workflowRun { event workflow { name } } checkRuns(first: 100, filterBy: {checkType: LATEST}) { nodes { __typename name status conclusion } } } } status { contexts { __typename context state } } } } } } }`

func (g *github) Tip(branch string) (Tip, error) {
	if branch == "" {
		out, err := g.api("repos/{owner}/{repo}", "--jq", ".default_branch")
		if err != nil {
			return Tip{}, err
		}
		branch = strings.TrimSpace(string(out))
	}
	out, err := g.api("graphql", "-f", "query="+tipQuery, "-F", "owner={owner}", "-F", "name={repo}", "-f", "ref=refs/heads/"+branch)
	if err != nil {
		return Tip{}, err
	}
	var v struct {
		Data struct {
			Repository struct {
				Ref *struct {
					Target struct {
						OID    string `json:"oid"`
						Suites struct {
							Nodes []struct {
								Status     string `json:"status"`
								Conclusion string `json:"conclusion"`
								App        *struct {
									Slug string `json:"slug"`
								} `json:"app"`
								Run *struct {
									Event    string `json:"event"`
									Workflow struct {
										Name string `json:"name"`
									} `json:"workflow"`
								} `json:"workflowRun"`
								Runs struct {
									Nodes []checkContext `json:"nodes"`
								} `json:"checkRuns"`
							} `json:"nodes"`
						} `json:"checkSuites"`
						Status *struct {
							Contexts []checkContext `json:"contexts"`
						} `json:"status"`
					} `json:"target"`
				} `json:"ref"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return Tip{}, err
	}
	ref := v.Data.Repository.Ref
	if ref == nil {
		return Tip{}, fmt.Errorf("%w: no branch %q on GitHub", errNotFound, branch)
	}
	var all []checkContext
	for _, s := range ref.Target.Suites.Nodes {
		if s.Run != nil && s.Run.Event != "push" {
			continue
		}
		all = append(all, s.Runs.Nodes...)
		// A workflow that could not start ends with no run: its suite
		// failed, which is main red. One never begun, an app's suite
		// GitHub makes on each push, says nothing.
		if len(s.Runs.Nodes) == 0 && s.Status == "COMPLETED" {
			name := "a check suite"
			switch {
			case s.Run != nil && s.Run.Workflow.Name != "":
				name = s.Run.Workflow.Name
			case s.App != nil:
				name = s.App.Slug
			}
			all = append(all, checkContext{Type: "CheckRun", Name: name, Status: s.Status, Conclusion: s.Conclusion})
		}
	}
	if ref.Target.Status != nil {
		all = append(all, ref.Target.Status.Contexts...)
	}
	return tipOf(Tip{Branch: branch, Commit: ref.Target.OID}, runsOf(all)), nil
}

func (g *gitlab) Holding() ([]Issue, error) {
	out, err := g.api("--paginate", "projects/:id/issues?state=opened&per_page=100&labels="+url.QueryEscape(HoldLabel))
	if err != nil {
		return nil, err
	}
	found, err := pages[struct {
		IID    int      `json:"iid"`
		Title  string   `json:"title"`
		Labels []string `json:"labels"`
	}](out)
	if err != nil {
		return nil, err
	}
	var is []Issue
	for _, f := range found {
		is = append(is, Issue{ID: f.IID, Title: f.Title, Labels: f.Labels})
	}
	return is, nil
}

// Tip reads the newest pipeline on the branch's tip that no schedule
// started, a canceled one passed over for the one before it, as the
// merge policy does (pipelineBefore).
func (g *gitlab) Tip(branch string) (Tip, error) {
	if branch == "" {
		out, err := g.api("projects/:id")
		if err != nil {
			return Tip{}, err
		}
		var p struct {
			Default string `json:"default_branch"`
		}
		if err := decode(out, &p); err != nil {
			return Tip{}, err
		}
		branch = p.Default
	}
	out, err := g.api("projects/:id/repository/branches/" + url.PathEscape(branch))
	if err != nil {
		return Tip{}, err
	}
	var b struct {
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
	}
	if err := decode(out, &b); err != nil {
		return Tip{}, err
	}
	t := Tip{Branch: branch, Commit: b.Commit.ID}
	out, err = g.api("projects/:id/pipelines?per_page=100&sha=" + b.Commit.ID + "&ref=" + url.QueryEscape(branch))
	if err != nil {
		return t, err
	}
	all, err := pages[struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
		Source string `json:"source"`
	}](out)
	if err != nil {
		return t, err
	}
	var runs []CheckRun
	for _, p := range all { // newest first
		s := pipelineStatus(p.Status)
		if p.Source == "schedule" || s == "" {
			continue
		}
		runs = append(runs, CheckRun{Name: fmt.Sprintf("pipeline #%d", p.ID), State: s})
		if s != "cancelled" {
			break
		}
	}
	return tipOf(t, runs), nil
}

func (f *Fake) Holding() ([]Issue, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range s.Issues {
		if !it.Closed && slices.Contains(it.Labels, HoldLabel) {
			out = append(out, Issue{ID: it.ID, Title: it.Title, Labels: it.Labels})
		}
	}
	return out, nil
}

// Tip reads the runs kept for the branch, `main` when none is named; a
// branch with none has no check.
func (f *Fake) Tip(branch string) (Tip, error) {
	s, err := f.load()
	if err != nil {
		return Tip{}, err
	}
	if s.TipsUnread != "" {
		return Tip{}, fmt.Errorf("%w: %s", ErrUnreachable, s.TipsUnread)
	}
	if branch == "" {
		branch = "main"
	}
	commit := branch
	if sha := f.resolve(branch); f.Repo != "" && sha != "" {
		commit = sha
	}
	runs := s.Tips[branch]
	if then, ok := s.TipsThen[branch]; ok {
		if s.Tips == nil {
			s.Tips = map[string][]CheckRun{}
		}
		s.Tips[branch] = then
		delete(s.TipsThen, branch)
		if err := f.save(s); err != nil {
			return Tip{}, err
		}
	}
	return tipOf(Tip{Branch: branch, Commit: commit}, runs), nil
}
