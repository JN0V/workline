// Package tick says which of the line's jobs are due now (ADR-0069 §2):
// read from the forge and the routing, no agent, nothing written. What
// runs the tick, a workflow, a pipeline or a timer, starts those and
// nothing else.
package tick

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/line"
	"github.com/JN0V/workline/internal/routing"
)

// Item is one job due: Job, its name; Role, the role it runs, when one;
// Target, the issue (#n) or merge request (!n) it runs on; Why, in words;
// Key, the name its run is started under, the same from tick to tick: a
// run going by that name keeps it off the list.
type Item struct {
	Job    string `json:"job"`
	Role   string `json:"role,omitempty"`
	Target string `json:"target,omitempty"`
	Why    string `json:"why"`
	Key    string `json:"key"`
}

// Result is a tick's list, and what it did not list and why: a job going,
// or not due on its own rule (Said); a read that failed (Unread): never a
// job listed on a guess.
type Result struct {
	Items  []Item
	Said   []string
	Unread []string
	// Calls: the forge's API calls this tick sent (GitHub's); Free, those
	// answered 304 Not Modified, which cost no quota (#1762).
	Calls, Free int
}

// Options: the repository, its routing, the forge (nil: none), when now
// is; Issues: the reconcile's issues, rounds and answers listed each as
// its own job, in the place of the reconcile.
type Options struct {
	Repo      string
	Config    *routing.Config
	Forge     forge.Forge
	ForgeSpec string
	Now       time.Time
	Issues    bool
}

// job is a timed job's run: the name it is started under, the workflow it
// runs in on GitHub (other forges ignore it), and the older names a run
// of it was started under, read as its runs too.
type job struct {
	key, workflow string
	aliases       []string
}

var jobs = map[string]job{
	"reconcile": {"workline develop reconcile", "workline-develop.yml", []string{"workline develop heartbeat"}},
	// The gardening's runs before its run-name (#1761) have no name of
	// their own: read as the backlog's. Since, each is named by its job;
	// the nightly and the template's full Sunday run (the product owner
	// too) keep the backlog too (#1765); only a succeeded one counts.
	"backlog": {"workline gardening backlog", "workline-gardening.yml", []string{"workline gardening", "workline gardening nightly", "workline gardening docs and backlog"}},
	"nightly": {"workline gardening nightly", "workline-gardening.yml", nil},
	"docs":    {"workline gardening docs", "workline-gardening.yml", []string{"workline gardening docs and backlog"}},
	"merge":   {"workline merge", "workline-merge.yml", nil},
	"sample":  {"workline sample", "workline-sample.yml", nil},
	"seeds":   {"workline test seeds", "test.yml", nil},
}

// backlogLookBack: how far back the last backlog run is looked for; past
// it, none before, and one is due.
const backlogLookBack = 24 * time.Hour

func (j job) names() []string { return append([]string{j.key}, j.aliases...) }

// Due lists the jobs due at o.Now, in routing.TimedJobs' order.
func Due(o Options) (r Result) {
	sent, free := forge.Calls()
	defer func() {
		now, nowFree := forge.Calls()
		r.Calls, r.Free = now-sent, nowFree-free
	}()
	runs, runsErr := forge.RunsOf(o.Forge)
	var going []forge.Run
	if runsErr == nil {
		going, runsErr = runs.RunsGoing()
	}
	if runsErr != nil {
		r.Unread = append(r.Unread, "the runs going: "+forge.Said(runsErr)+": no timed job listed on a guess")
	}
	goingAs := func(names []string) (forge.Run, bool) {
		for _, g := range going {
			if slices.Contains(names, g.Name) {
				return g, true
			}
		}
		return forge.Run{}, false
	}
	// Each workflow read once, as far back as its jobs need.
	since := map[string]time.Time{}
	need := func(wf string, t time.Time) {
		if s, ok := since[wf]; !ok || t.Before(s) {
			since[wf] = t
		}
	}
	type timed struct {
		name string
		routing.Timed
		from time.Time // runs started from then on count
	}
	var todo []timed
	for _, tj := range routing.TimedJobs {
		t := o.Config.Timing(tj.Name)
		if t.Off {
			continue
		}
		j := jobs[tj.Name]
		from := time.Time{}
		switch {
		case t.At != "":
			c, err := routing.ParseCron(t.At)
			last, ok := c.Last(o.Now)
			if err != nil || !ok {
				continue // checked when the routing was read; a 31 February never comes
			}
			from = last
		case t.Gap > 0:
			from = o.Now.Add(-time.Duration(t.Gap) * time.Minute)
		case tj.Name == "backlog":
			from = o.Now.Add(-backlogLookBack)
		}
		if !from.IsZero() {
			need(j.workflow, from)
		}
		todo = append(todo, timed{tj.Name, t, from})
	}
	read := map[string][]forge.Run{}
	readErr := map[string]error{}
	runsOf := func(wf string) ([]forge.Run, error) {
		if runsErr != nil {
			return nil, runsErr
		}
		if _, ok := read[wf]; !ok && readErr[wf] == nil {
			read[wf], readErr[wf] = runs.RunsSince(wf, since[wf])
			if readErr[wf] != nil {
				r.Unread = append(r.Unread, wf+"'s runs: "+forge.Said(readErr[wf]))
			}
		}
		return read[wf], readErr[wf]
	}
	for _, t := range todo {
		j := jobs[t.name]
		if g, ok := goingAs(j.names()); ok {
			r.Said = append(r.Said, fmt.Sprintf("%s: %q %s since %s", t.name, g.Name, g.Status, g.Started))
			continue
		}
		switch {
		case t.name == "reconcile" && t.Gap == 0 && t.At == "":
			r.add(o, Item{Job: "reconcile", Why: "every tick", Key: j.key})
		case t.name == "backlog":
			if why, ok := backlog(o, &r, runsOf, j, t.from, runsErr); ok {
				r.Items = append(r.Items, Item{Job: "backlog", Role: "product-owner", Why: why, Key: j.key})
			}
		default:
			past, err := runsOf(j.workflow)
			if err != nil {
				continue // said once, by runsOf or above
			}
			if t.name == "nightly" { // only a succeeded run counts (#1765)
				past = slices.DeleteFunc(slices.Clone(past), func(r forge.Run) bool { return !r.Success })
			}
			if run, ran := lastOf(past, j.names(), t.from); ran {
				r.Said = append(r.Said, fmt.Sprintf("%s: ran at %s (%q), not due", t.name, run.Started, run.Name))
				continue
			}
			why := ""
			if t.At != "" {
				why = fmt.Sprintf("its time %s (%s) passed, no run since", t.from.UTC().Format("2006-01-02 15:04 UTC"), t.At)
			} else {
				why = fmt.Sprintf("no run in the last %d min (gap %d)", t.Gap, t.Gap)
			}
			if t.name == "reconcile" {
				r.add(o, Item{Job: "reconcile", Why: why, Key: j.key})
				continue
			}
			r.Items = append(r.Items, Item{Job: t.name, Why: why, Key: j.key})
		}
	}
	r.Items = slices.DeleteFunc(r.Items, func(it Item) bool {
		if g, ok := goingAs([]string{it.Key}); ok && it.Target != "" {
			r.Said = append(r.Said, fmt.Sprintf("%s %s: %q %s since %s", it.Job, it.Target, g.Name, g.Status, g.Started))
			return true
		}
		return false
	})
	return r
}

// lastOf is the newest of runs started under one of names at from or
// after.
func lastOf(runs []forge.Run, names []string, from time.Time) (forge.Run, bool) {
	for _, run := range runs {
		t, err := time.Parse(time.RFC3339, run.Started)
		if err == nil && !t.Before(from) && slices.Contains(names, run.Name) {
			return run, true
		}
	}
	return forge.Run{}, false
}

// backlog says why a backlog run is due, when it is, as line.BacklogDue
// reads it; a read that failed: in Unread.
func backlog(o Options, r *Result, runsOf func(string) ([]forge.Run, error), j job, from time.Time, runsErr error) (string, bool) {
	b, ok := o.Forge.(forge.Backlog)
	if !ok || o.Forge == nil {
		r.Unread = append(r.Unread, "backlog: this forge lists no issues")
		return "", false
	}
	open, err := b.Issues()
	if err != nil {
		r.Unread = append(r.Unread, "backlog: the issues: "+forge.Said(err))
		return "", false
	}
	last := "none"
	if runsErr != nil {
		last = "unread"
	} else if past, err := runsOf(j.workflow); err != nil {
		last = "unread"
	} else if run, ok := lastOf(slices.DeleteFunc(slices.Clone(past), func(r forge.Run) bool { return !r.Success }), j.names(), from); ok {
		last = run.Started + "/" + run.Updated
		if run.Updated == "" {
			last = run.Started + "/" + run.Started
		}
	}
	due, why, err := line.BacklogDue(o.Config, b, open, last)
	if err != nil {
		r.Unread = append(r.Unread, "backlog: "+forge.Said(err)+": no backlog run listed on a guess")
	} else if !due && why != "" {
		r.Said = append(r.Said, "backlog: "+why)
	}
	return why, due
}

// add lists the reconcile; with Issues, each issue's step, review round
// and answer it finds due, in its place, as its own job (line's
// reconcile, `--input reconcile=list`: no agent, nothing written).
func (r *Result) add(o Options, it Item) {
	if !o.Issues {
		r.Items = append(r.Items, it)
		return
	}
	res := line.Run(line.ReconcileEvent, engine.Options{Repo: o.Repo, Forge: o.ForgeSpec, AI: "none", NoApply: true,
		Inputs: map[string]string{line.ReconcileInput: "list"}})
	for _, f := range res.Findings {
		if strings.HasSuffix(f.Rule, "-unread") {
			r.Unread = append(r.Unread, "reconcile: "+f.Where+": "+f.Message)
		}
	}
	roles := map[int]string{}
	if b, ok := o.Forge.(forge.Backlog); ok && o.Forge != nil && len(res.Due) > 0 {
		if open, err := b.Issues(); err == nil {
			for _, is := range open {
				if _, name, _ := o.Config.Wakes(is.Labels, line.IssueEvent); name != "" {
					roles[is.ID] = name
				}
			}
		}
	}
	for _, n := range res.Due {
		r.Items = append(r.Items, Item{Job: "develop", Role: roles[n], Target: "#" + strconv.Itoa(n),
			Why: "its state's step is due (reconcile)", Key: "workline develop #" + strconv.Itoa(n)})
	}
	for _, rv := range res.Reviews {
		n := strconv.Itoa(rv.ID)
		r.Items = append(r.Items, Item{Job: "round", Role: "developer", Target: "!" + n,
			Why: "a review round is due on " + rv.Branch, Key: "workline develop round !" + n})
	}
	for _, a := range res.Answers {
		n := strconv.Itoa(a.ID)
		r.Items = append(r.Items, Item{Job: "answer", Role: "tester", Target: "!" + n,
			Why: "an objection waits on the tester's Draft " + a.Branch, Key: "workline answer !" + n})
	}
}
