package line

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/routing"
)

// Moment is the wait a line's run is part of, and its budget (ADR-0050,
// section 1): a check, never a block. Over its budget, the run says so
// once, naming the step that took the most of it.
type Moment struct {
	// Name is the moment's key in routing's budgets: commit,
	// commit-rewrite, push, first-result, verdict, ready-review, queue.
	Name    string  `json:"name"`
	Budget  float64 `json:"budget"`  // seconds
	Seconds float64 `json:"seconds"` // what the line took of it here
	Over    bool    `json:"over,omitempty"`
	Step    string  `json:"step,omitempty"` // over its budget: the step that took the most
}

// moments say each moment in words, by its key.
var moments = map[string]string{
	"commit":         "the commit",
	"commit-rewrite": "the commit, with the agent's rewrite,",
	"push":           "the push",
	"first-result":   "the first result, the gates alone,",
	"verdict":        "the head's verdict",
	"ready-review":   "the review apart, the lenses that warn,",
	"queue":          "the merge queue's or train's run",
}

// momentOf is the moment a run of event is part of, its key in routing's
// budgets; "" for one no person waits on (main after a merge, the night,
// the release). A commit whose agent rewrote its message has the
// rewrite's budget; on a merge request, a queue's or train's run is what
// workline adds to the queue's turn, its gates' job too; else the gates
// alone are its first result, the run apart its review posted when done,
// the line its verdict.
func momentOf(event string, inputs map[string]string, gatesOnly bool, asked int) string {
	switch {
	case event == "commit-msg" && asked > 0:
		return "commit-rewrite"
	case event == "commit-msg":
		return "commit"
	case event == "pre-push":
		return "push"
	case event != "merge-request":
		return ""
	case inputs[QueueInput] == "true":
		return "queue"
	case gatesOnly:
		return "first-result"
	case inputs[WarnInput] == WarnOnly:
		return "ready-review"
	}
	return "verdict"
}

// Measure sets r's moment: its budget in cfg, and, over it, the step that
// took the most. A moment with no budget leaves it unset.
func Measure(r *Result, cfg *routing.Config, moment string) {
	budget, ok := cfg.Budget(moment)
	if !ok {
		r.Moment = nil
		return
	}
	m := &Moment{Name: moment, Budget: budget, Seconds: r.Seconds, Over: r.Seconds > budget}
	if m.Over {
		m.Step = slowest(r.Steps).Name
	}
	r.Moment = m
}

// Join is the steps and seconds of several runs of a moment as one, the
// pre-push hook's ranges: their steps in order, their seconds added.
func Join(runs []*Result) *Result {
	all := &Result{Steps: []Step{}}
	for _, r := range runs {
		all.Steps = append(all.Steps, r.Steps...)
		all.Seconds = round(all.Seconds + r.Seconds)
		all.AgentCalls += r.AgentCalls
	}
	return all
}

func slowest(steps []Step) Step {
	var s Step
	for _, st := range steps {
		if st.Seconds > s.Seconds || s.Name == "" {
			s = st
		}
	}
	return s
}

// Times is each step's seconds, compactly, after the line's total: "66 s
// in all — committer 0.5 s, documentalist 0.8 s, reviewer 64 s (lenses
// 40 s, judges 20 s)"; "" when no step ran. A gate read from the job that
// ran it says so: its seconds are that job's.
func (r *Result) Times() string {
	if len(r.Steps) == 0 {
		return ""
	}
	var parts []string
	for _, s := range r.Steps {
		parts = append(parts, s.timed())
	}
	return Seconds(r.Seconds) + " in all — " + strings.Join(parts, ", ")
}

// OverBudget is what the run says when its moment went over its budget,
// once, naming the step that took the most; "" within it, or with none.
func (r *Result) OverBudget() string {
	m := r.Moment
	if m == nil || !m.Over {
		return ""
	}
	said := fmt.Sprintf("%s took %s, over its budget of %s (ADR-0050)", moments[m.Name], Seconds(m.Seconds), Seconds(m.Budget))
	for _, s := range r.Steps {
		if s.Name == m.Step {
			return said + fmt.Sprintf("; the step that took the most: %s, %s", s.Name, strings.TrimPrefix(s.timed(), s.Name+" "))
		}
	}
	return said
}

// timed is a step's name and seconds, with what its agents took of them:
// the reviewer's lenses, the committer's rewrite, the judges.
func (s Step) timed() string {
	t := s.Name + " " + Seconds(s.Seconds)
	if s.From != "" {
		t += " (its job)"
	}
	if s.Result == nil || len(s.Result.Calls) == 0 {
		return t
	}
	role, _, _ := strings.Cut(s.Name, " ")
	var order []string
	took := map[string]float64{}
	for _, c := range s.Result.Calls {
		kind := "agent"
		switch {
		case c.Task == "judge":
			kind = "judges"
		case c.Task == "part" && role == "reviewer":
			kind = "lenses"
		case c.Task == "part":
			kind = "parts"
		case role == "committer":
			kind = "rewrite"
		}
		if _, ok := took[kind]; !ok {
			order = append(order, kind)
		}
		took[kind] += c.Seconds
	}
	var parts []string
	for _, k := range order {
		parts = append(parts, k+" "+Seconds(took[k]))
	}
	return t + " (" + strings.Join(parts, ", ") + ")"
}

// Seconds says a time compactly: tenths under 10 s, whole seconds under a
// minute, then minutes and seconds.
func Seconds(s float64) string {
	switch {
	case s < 9.95 && round(s) != math.Trunc(round(s)):
		return fmt.Sprintf("%.1f s", s)
	case s < 59.5:
		return fmt.Sprintf("%.0f s", s)
	}
	whole := int(math.Round(s))
	if whole%60 == 0 {
		return fmt.Sprintf("%d min", whole/60)
	}
	return fmt.Sprintf("%d min %d s", whole/60, whole%60)
}

// Since is the time since start, in tenths of a second.
func Since(start time.Time) float64 { return round(time.Since(start).Seconds()) }

func round(s float64) float64 { return math.Round(s*10) / 10 }
