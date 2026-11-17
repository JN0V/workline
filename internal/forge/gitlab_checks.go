package forge

// The project's own checks on GitLab (#1381): the jobs of the head
// pipeline of a commit, apart from workline's own jobs, and a failed
// job's trace.

import (
	"fmt"
	"net/url"
	"strconv"
)

var _ = []ProjectChecked{&gitlab{}}
var _ = []GatesTraced{&gitlab{}}

// GatesTraced is a forge whose gates run as a job of the head's
// pipeline, not as a comment: it reads the trace of the gates job that
// failed at head.
type GatesTraced interface {
	// GatesTrace is the trace of the failed gates job of head's pipeline;
	// "" when there is none.
	GatesTrace(head string) (string, error)
}

type gitlabJob struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	AllowFailure bool   `json:"allow_failure"`
}

// headJobs reads the jobs of the latest pipeline that ran on head.
func (g *gitlab) headJobs(head string) ([]gitlabJob, error) {
	out, err := g.api("projects/:id/pipelines?sha=" + url.QueryEscape(head))
	if err != nil {
		return nil, err
	}
	var ps []struct {
		ID  int64  `json:"id"`
		SHA string `json:"sha"`
	}
	if err := decode(out, &ps); err != nil {
		return nil, err
	}
	for _, p := range ps {
		if !sameHead(p.SHA, head) {
			continue
		}
		out, err := g.api("--paginate", "projects/:id/pipelines/"+strconv.FormatInt(p.ID, 10)+"/jobs?per_page=100")
		if err != nil {
			return nil, err
		}
		return pages[gitlabJob](out)
	}
	return nil, nil
}

func (g *gitlab) ProjectChecks(head string) ([]HeadCheck, error) {
	jobs, err := g.headJobs(head)
	if err != nil {
		return nil, err
	}
	var all []HeadCheck
	for _, j := range jobs {
		c := HeadCheck{Name: j.Name, Job: j.ID}
		switch j.Status {
		case "success":
			c.State = "success"
		case "failed":
			c.State = "failure"
			if j.AllowFailure {
				c.State = "success"
			}
		case "canceled", "canceling":
			c.State = "cancelled"
		case "created", "pending", "running", "waiting_for_resource", "preparing", "scheduled":
			c.State = "pending"
		default: // skipped, manual: not run, nothing to wait for
			continue
		}
		all = append(all, c)
	}
	return projectOnly(all), nil
}

func (g *gitlab) CheckLog(c HeadCheck) (string, error) {
	if c.Job == 0 {
		return "", nil
	}
	out, err := g.api(fmt.Sprintf("projects/:id/jobs/%d/trace", c.Job))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (g *gitlab) GatesTrace(head string) (string, error) {
	jobs, err := g.headJobs(head)
	if err != nil {
		return "", err
	}
	for _, j := range jobs {
		if j.Name == "workline:gates" && j.Status == "failed" {
			return g.CheckLog(HeadCheck{Name: j.Name, Job: j.ID})
		}
	}
	return "", nil
}
