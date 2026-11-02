package forge

// The project's own checks on a commit (#917): what its CI runs on the
// developer's pull request — its tests, its build —, apart from
// workline's own checks, which its own records already say (the gates'
// comment, the reviewer's, the verdict). A red one is a round's input.

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// HeadCheck is a check on a commit: its name, the workflow that ran it
// (GitHub Actions; "" for another app's check or a commit status), its
// state as the merge policy reads it (success, failure, pending,
// cancelled), and its job, whose log the forge reads (0: none).
type HeadCheck struct {
	Name     string `json:"name"`
	Workflow string `json:"workflow,omitempty"`
	State    string `json:"state"`
	Job      int64  `json:"job,omitempty"`
}

// ProjectChecked is a forge that reads the project's checks on a commit,
// and a failed one's log. GitHub and the simulated forge do; GitLab not
// yet (roles/developer/docs/status.md).
type ProjectChecked interface {
	// ProjectChecks reads the checks on head that are the project's own,
	// workline's left out (OwnCheck).
	ProjectChecks(head string) ([]HeadCheck, error)
	// CheckLog reads what a failed check printed in its failed steps; ""
	// when the forge keeps no log of it.
	CheckLog(c HeadCheck) (string, error)
}

var _ = []ProjectChecked{&github{}, &Fake{}}

// OwnCheck says a check is workline's own, not the project's: a commit
// status workline/…, a check its App writes (workline, workline
// findings), or a job of one of its workflows, each named "workline …"
// in its templates.
func OwnCheck(c HeadCheck) bool {
	own := func(s string) bool { return strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "workline") }
	return own(c.Name) || own(c.Workflow)
}

func projectOnly(all []HeadCheck) []HeadCheck {
	var out []HeadCheck
	for _, c := range all {
		if !OwnCheck(c) && c.State != "" {
			out = append(out, c)
		}
	}
	return out
}

// headChecksQuery asks a commit's check suites, each with the workflow
// that ran it, their latest runs, and its commit statuses.
const headChecksQuery = `query($owner: String!, $name: String!, $oid: GitObjectID!) { repository(owner: $owner, name: $name) { object(oid: $oid) { ... on Commit { checkSuites(first: 100) { nodes { workflowRun { workflow { name } } checkRuns(first: 100, filterBy: {checkType: LATEST}) { nodes { __typename name status conclusion databaseId } } } } status { contexts { __typename context state } } } } } }`

func (g *github) ProjectChecks(head string) ([]HeadCheck, error) {
	out, err := g.api("graphql", "-f", "query="+headChecksQuery, "-F", "owner={owner}", "-F", "name={repo}", "-f", "oid="+head)
	if err != nil {
		return nil, err
	}
	type run struct {
		checkContext
		ID int64 `json:"databaseId"`
	}
	var v struct {
		Data struct {
			Repository struct {
				Object *struct {
					Suites struct {
						Nodes []struct {
							Run *struct {
								Workflow struct {
									Name string `json:"name"`
								} `json:"workflow"`
							} `json:"workflowRun"`
							Runs struct {
								Nodes []run `json:"nodes"`
							} `json:"checkRuns"`
						} `json:"nodes"`
					} `json:"checkSuites"`
					Status *struct {
						Contexts []checkContext `json:"contexts"`
					} `json:"status"`
				} `json:"object"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := decode(out, &v); err != nil {
		return nil, err
	}
	o := v.Data.Repository.Object
	if o == nil {
		return nil, fmt.Errorf("%w: no commit %s on GitHub", errNotFound, head)
	}
	var all []HeadCheck
	for _, s := range o.Suites.Nodes {
		wf := ""
		if s.Run != nil {
			wf = s.Run.Workflow.Name
		}
		for _, r := range s.Runs.Nodes {
			c := HeadCheck{Name: r.Name, Workflow: wf, Job: r.ID}
			if rs := runsOf([]checkContext{r.checkContext}); len(rs) == 1 {
				c.State = rs[0].State
			}
			if wf == "" {
				c.Job = 0 // another app's check: no Actions job to read
			}
			all = append(all, c)
		}
	}
	if o.Status != nil {
		for _, r := range runsOf(o.Status.Contexts) { // workline/… left out
			all = append(all, HeadCheck{Name: r.Name, State: r.State})
		}
	}
	return projectOnly(all), nil
}

// CheckLog reads a job's failed steps as `gh run view --log-failed`
// prints them, its escapes shown as text; each line's job and step names
// cut. Not through run, which asks `gh api`'s headers: a call of its own,
// two minutes at most.
func (g *github) CheckLog(c HeadCheck) (string, error) {
	if c.Job == 0 {
		return "", nil
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("%w: gh is not installed", ErrUnreachable)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "run", "view", "--job", strconv.FormatInt(c.Job, 10), "--log-failed")
	cmd.Dir = g.repo
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		said, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		return "", fmt.Errorf("%w: gh run view --job %d: %s", ErrUnreachable, c.Job, cmp.Or(said, err.Error()))
	}
	var b strings.Builder
	for _, l := range strings.Split(string(out), "\n") {
		if _, rest, ok := strings.Cut(l, "\t"); ok {
			if _, rest2, ok := strings.Cut(rest, "\t"); ok {
				l = rest2
			}
		}
		b.WriteString(l + "\n")
	}
	return b.String(), nil
}

func (f *Fake) ProjectChecks(head string) ([]HeadCheck, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	if s.HeadChecksUnread != "" {
		return nil, fmt.Errorf("%w: %s", ErrUnreachable, s.HeadChecksUnread)
	}
	var all []HeadCheck
	for rev, cs := range s.HeadChecks {
		if c := f.resolve(rev); c != "" && sameHead(c, head) || rev == head {
			for _, x := range cs {
				all = append(all, x.HeadCheck)
			}
		}
	}
	return projectOnly(all), nil
}

func (f *Fake) CheckLog(c HeadCheck) (string, error) {
	s, err := f.load()
	if err != nil {
		return "", err
	}
	for _, cs := range s.HeadChecks {
		for _, x := range cs {
			if x.Name == c.Name && x.Job == c.Job {
				return x.Log, nil
			}
		}
	}
	return "", nil
}

// FakeHeadCheck is a check on a commit, and the log of its failed steps.
type FakeHeadCheck struct {
	HeadCheck
	Log string `json:"log,omitempty"`
}

func sameHead(a, b string) bool {
	return len(a) >= 7 && len(b) >= 7 && (strings.HasPrefix(a, b) || strings.HasPrefix(b, a))
}
