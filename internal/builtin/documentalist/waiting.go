package documentalist

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
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
}

// readWaiting reads what waits; nothing does unless gardening.
func readWaiting(runDir string) (waiting, error) {
	w := waiting{open: map[string]bool{}, proposed: map[string]bool{}, files: map[string][]string{}}
	if os.Getenv("WORKLINE_EVENT") != "schedule" {
		return w, nil
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

// changing names a waiting task whose change touches the doc, those an
// earlier round proposed first; failing one, a task judging docs in parts
// (`fix`, `fix-…`) whose files are not known, which may hold it.
func (w waiting) changing(doc string) (task string, waits, earlier bool) {
	tasks := make([]string, 0, len(w.open)+len(w.proposed))
	for t := range w.open {
		tasks = append(tasks, t)
	}
	for t := range w.proposed {
		if !w.open[t] {
			tasks = append(tasks, t)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		if w.proposed[tasks[i]] != w.proposed[tasks[j]] {
			return w.proposed[tasks[i]]
		}
		return tasks[i] < tasks[j]
	})
	for _, t := range tasks {
		if slices.Contains(w.files[t], doc) {
			return t, true, w.proposed[t]
		}
	}
	for _, t := range tasks {
		if !w.read(t) && (t == "fix" || strings.HasPrefix(t, "fix-")) {
			return t, true, w.proposed[t]
		}
	}
	return "", false, false
}

// holdWaitingTasks keeps from the agent, when gardening, the docs whose
// judgement waits for review. A doc judged whole waits while its kind's
// task does (`suspect`, `stale`): one merge request for all such docs,
// which judging another would replace. A doc judged in parts waits while a
// merge request changes it, whatever its task — its fix, or only the
// record of when it was judged —, and the docs no merge request holds get
// their turn, each run's on a merge request of its own (#356).
func holdWaitingTasks(w waiting, maps ...map[string]*suspectDoc) {
	for i, docs := range maps {
		kind := []string{"suspect", "stale"}[i]
		for _, sd := range docs {
			if sd.note != "" {
				continue
			}
			task, waits, earlier := kind, w.open[kind] || w.proposed[kind], w.proposed[kind]
			if sd.tooLarge {
				task, waits, earlier = w.changing(sd.doc.Path)
			}
			// The release's own fix changes it, open or judged tonight
			// before the gardening (ADR-0050, §7): judged once a night.
			release := !waits && slices.Contains(w.files["release"], sd.doc.Path)
			switch {
			case release:
				sd.note = "(workline/documentalist/release, the release's fix, changes it and waits for review: not put before an agent again until it is merged or closed)"
			case !waits:
			case earlier: // an earlier round of this run judged it
				sd.note, sd.earlier = "(judged in an earlier round of this run)", true
			case sd.tooLarge && !w.read(task): // a task in parts whose change could not be read
				sd.note = "(workline/documentalist/" + task + ", judging docs in parts, waits for review, and what it changes could not be read: not put before an agent again until it is merged or closed)"
			case sd.tooLarge:
				sd.note = "(workline/documentalist/" + task + " changes it and waits for review: not put before an agent again until it is merged or closed)"
			default:
				sd.note = "(its task's merge request, workline/documentalist/" + task + ", waits for review: not put before an agent again until it is merged or closed)"
			}
		}
	}
}

// The key of each gardening task's merge request (ADR-0006): the docs it
// rewrites name it, so that the task waits for it while another goes; and
// the findings a rewrite resolves, which say so.

// partsKey is a run's judging in parts, by the first doc it judged. When
// the branch that names already waits — another doc's, two paths making one
// name, since the run's own docs wait with theirs —, a short hash of the
// doc's path tells them apart: never one proposal replacing the other.
func partsKey(docs []string, w waiting) string {
	key := "fix " + docs[0]
	if _, waits, _ := w.holds(key); waits {
		sum := sha256.Sum256([]byte(docs[0]))
		key += " " + hex.EncodeToString(sum[:3])
	}
	return key
}

func (c *condenseTask) key() string {
	if c.Split {
		return "split " + c.Doc
	}
	return "condense " + c.Doc
}

func (c *condenseTask) resolves(f verdict.Finding) bool {
	doc, _, _ := strings.Cut(f.Where, "#")
	if c.Split {
		return f.Rule == "card-too-long" && doc == c.Doc
	}
	return slices.Contains(condensable, f.Rule) && doc == c.Doc
}

func (d *dedupeTask) key() string { return "duplicates " + strings.Join(d.Docs, " ") }

func (d *dedupeTask) resolves(f verdict.Finding) bool {
	return f.Rule == "duplicate" && f.Where == d.Docs[0] && strings.Contains(f.Message, " "+d.Docs[1]+" ")
}

func (m *mergeCardTask) key() string { return "merge-card " + m.Card }

func (m *mergeCardTask) resolves(f verdict.Finding) bool {
	return f.Rule == "card-too-short" && f.Where == m.Card
}
