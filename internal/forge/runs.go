package forge

// The CI runs of the line's jobs, by the name each was started under:
// what `workline tick` reads to know a job going, and when one last ran
// (ADR-0069 §5). Read only.

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"
)

// Run is one CI run: Name, the name it was started under (a GitHub
// run-name, its display title); Workflow, the file it runs; Status,
// queued, waiting, pending, in_progress or completed; Success, it ended
// green; Started and Updated, RFC 3339.
type Run struct {
	Name     string `json:"name"`
	Workflow string `json:"workflow,omitempty"`
	Status   string `json:"status"`
	Success  bool   `json:"success,omitempty"`
	Started  string `json:"started"`
	Updated  string `json:"updated,omitempty"`
}

// Going says whether the run is queued, waiting or running.
func (r Run) Going() bool { return r.Status != "completed" }

// RunsPages bounds a workflow's listing: past it, older runs are not
// read, and a job whose last run is older is said due.
const RunsPages = 5

// Runs is a forge that lists its CI runs.
type Runs interface {
	// RunsGoing lists every run queued, waiting or running, of any
	// workflow.
	RunsGoing() ([]Run, error)
	// RunsSince lists workflow's runs started at since or after, newest
	// first, RunsPages pages at most.
	RunsSince(workflow string, since time.Time) ([]Run, error)
}

// ErrNoRuns: the forge lists no CI runs (a forge kept in this clone, a
// command's): no timed job is said due on a guess.
var ErrNoRuns = errors.New("this forge lists no CI runs")

// RunsOf is f's runs, or ErrNoRuns.
func RunsOf(f Forge) (Runs, error) {
	if r, ok := f.(Runs); ok && f != nil {
		return r, nil
	}
	return nil, ErrNoRuns
}

func (f *Fake) runs() ([]Run, error) {
	s, err := f.load()
	if err != nil {
		return nil, err
	}
	if s.RunsUnread != "" {
		return nil, fmt.Errorf("%w: %s", ErrUnreachable, s.RunsUnread)
	}
	return s.Runs, nil
}

// RunsGoing: the file's runs not completed.
func (f *Fake) RunsGoing() ([]Run, error) {
	all, err := f.runs()
	return slices.DeleteFunc(all, func(r Run) bool { return !r.Going() }), err
}

// RunsSince: the file's runs of workflow started at since or after,
// newest first.
func (f *Fake) RunsSince(workflow string, since time.Time) ([]Run, error) {
	all, err := f.runs()
	all = slices.DeleteFunc(all, func(r Run) bool {
		t, err := time.Parse(time.RFC3339, r.Started)
		return r.Workflow != workflow || err != nil || t.Before(since)
	})
	slices.SortStableFunc(all, func(a, b Run) int { return -compareTimes(a.Started, b.Started) })
	return all, err
}

func compareTimes(a, b string) int {
	ta, _ := time.Parse(time.RFC3339, a)
	tb, _ := time.Parse(time.RFC3339, b)
	return ta.Compare(tb)
}

// githubRunJQ keeps of a run what Run holds.
const githubRunJQ = `{name: .display_title, workflow: (.path | split("/") | last | split("@") | first), status, success: (.conclusion == "success"), started: (.run_started_at // .created_at), updated: .updated_at}`

// RunsGoing reads the runs of each status not completed, one page of 100
// each: 4 calls.
func (g *github) RunsGoing() ([]Run, error) {
	var all []Run
	for _, status := range []string{"queued", "waiting", "pending", "in_progress"} {
		out, err := g.api("repos/{owner}/{repo}/actions/runs?per_page=100&status="+status, "--jq", ".workflow_runs[] | "+githubRunJQ)
		if err != nil {
			return nil, err
		}
		runs, err := lines[Run](out)
		if err != nil {
			return nil, err
		}
		all = append(all, runs...)
	}
	return all, nil
}

// RunsSince reads workflow's runs created since, page by page, until a
// page holds fewer than 100 or RunsPages are read.
func (g *github) RunsSince(workflow string, since time.Time) ([]Run, error) {
	var all []Run
	created := url.QueryEscape(">=" + since.UTC().Format(time.RFC3339))
	for page := 1; page <= RunsPages; page++ {
		out, err := g.api(fmt.Sprintf("repos/{owner}/{repo}/actions/workflows/%s/runs?per_page=100&page=%d&created=%s", url.PathEscape(workflow), page, created),
			"--jq", ".workflow_runs[] | "+githubRunJQ)
		if err != nil {
			return nil, err
		}
		runs, err := lines[Run](out)
		if err != nil {
			return nil, err
		}
		all = append(all, runs...)
		if len(runs) < 100 {
			break
		}
	}
	return all, nil
}
