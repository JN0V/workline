package documentalist

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/kit/verdict"
	"go.yaml.in/yaml/v3"
)

// waiting is what the role's merge requests waiting for review hold, when
// gardening (ADR-0013): the tasks open on the forge, those an earlier round
// of this run proposed, and the files each changes, when the engine could
// read them (in/waiting.yaml). Judged again, a doc they hold would only
// propose what is already proposed.
type waiting struct {
	open, proposed map[string]bool
	files          map[string][]string // by task; a task missing is known by its name only
	held           map[string]int      // a doc another open merge request changes: its number (in/held.yaml)
}

// readWaiting reads what waits; nothing does unless gardening.
func readWaiting(runDir string) (waiting, error) {
	w := waiting{open: map[string]bool{}, proposed: map[string]bool{}, files: map[string][]string{}, held: map[string]int{}}
	if os.Getenv("WORKLINE_EVENT") != "schedule" {
		return w, nil
	}
	if data, err := os.ReadFile(filepath.Join(runDir, "in", "held.yaml")); err == nil {
		var held map[int][]string
		if err := yaml.Unmarshal(data, &held); err != nil {
			return w, err
		}
		for n, files := range held {
			for _, p := range files {
				if m := w.held[p]; m == 0 || n < m {
					w.held[p] = n
				}
			}
		}
	}
	for _, t := range strings.Fields(os.Getenv("WORKLINE_OPEN_MERGE_REQUEST_TASKS")) {
		w.open[t] = true
	}
	for _, t := range strings.Fields(os.Getenv("WORKLINE_PROPOSED_TASKS")) {
		w.proposed[t] = true
	}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "waiting.yaml"))
	if os.IsNotExist(err) {
		return w, nil
	}
	if err != nil {
		return w, err
	}
	return w, yaml.Unmarshal(data, &w.files)
}

// holds says whether the task of this merge request key waits, and
// whether an earlier round of this run proposed it.
func (w waiting) holds(key string) (task string, waits, earlier bool) {
	task = intent.Slug(key)
	return task, w.open[task] || w.proposed[task], w.proposed[task]
}

// read says whether the engine could tell what the task changes.
func (w waiting) read(task string) bool {
	_, ok := w.files[task]
	return ok
}

// holdWaitingTasks keeps from the agent, when gardening, the docs whose
// judgement waits for review. A doc judged whole waits while its kind's
// task does (`suspect`, `stale`): one merge request for all such docs,
// which judging another would replace (#356).
func holdWaitingTasks(w waiting, maps ...map[string]*suspectDoc) {
	for i, docs := range maps {
		kind := []string{"suspect", "stale"}[i]
		for _, sd := range docs {
			if sd.note != "" {
				continue
			}
			// Another open merge request changes it: left to it, so that
			// neither conflicts with the other (ADR-0051).
			if n := w.held[sd.doc.Path]; n > 0 {
				sd.note = fmt.Sprintf("(open merge request #%d changes it: left to it until it is merged or closed)", n)
				continue
			}
			task, waits, earlier := kind, w.open[kind] || w.proposed[kind], w.proposed[kind]
			switch {
			case !waits || sd.tooLarge: // too large: a person's
			case earlier: // an earlier round of this run judged it
				sd.note, sd.earlier = "(judged in an earlier round of this run)", true
			default:
				sd.note = "(its task's merge request, workline/documentalist/" + task + ", waits for review: not put before an agent again until it is merged or closed)"
			}
		}
	}
}

// The key of each gardening task's merge request (ADR-0006): the docs it
// rewrites name it, so that the task waits for it while another goes; and
// the findings a rewrite resolves, which say so.

func (c *splitTask) key() string { return "split " + c.Doc }

func (c *splitTask) resolves(f verdict.Finding) bool {
	doc, _, _ := strings.Cut(f.Where, "#")
	return slices.Contains(splittable, f.Rule) && doc == c.Doc
}

func (d *dedupeTask) key() string { return "duplicates " + strings.Join(d.Docs, " ") }

func (d *dedupeTask) resolves(f verdict.Finding) bool {
	return f.Rule == "duplicate" && f.Where == d.Docs[0] && strings.Contains(f.Message, " "+d.Docs[1]+" ")
}
