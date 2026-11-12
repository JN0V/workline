package line

// The runs on a merge request, counted in the record of the issue it
// closes (ADR-0064, §1: an issue's normal holds all its runs, its
// reviews too): the reviewer's, the inspector's, the tester's guard and
// answers, the developer's review rounds.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/engine"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/verdict"
)

// CountRuns adds the fresh tokens of each run in runDirs that targeted a
// merge request to the record of the issue that merge request closes, by
// the run's role, once a run (Record.Counted). f: the forge, else each
// run's own; mr > 0: the merge request every run is on, a run from its
// own CI whose target is not taken from it. A merge request closing no
// issue, an issue the line keeps no record on, a run that spent nothing:
// nothing counted. What could not be counted is said, never blocking.
func CountRuns(f forge.Forge, mr int, runDirs []string) []verdict.Finding {
	var said []verdict.Finding
	for _, dir := range runDirs {
		if err := countRun(f, mr, dir); err != nil {
			said = append(said, verdict.Finding{Rule: "size-run-uncounted", Level: "warn", Where: filepath.Base(dir),
				Message: "its tokens are not counted on the issue it closes: " + err.Error()})
		}
	}
	return said
}

func countRun(f forge.Forge, mrID int, dir string) error {
	if dir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, "out", "run.yaml"))
	if err != nil {
		return nil // a run that never reached apply: nothing to count
	}
	var run struct {
		Role   string        `yaml:"role"`
		Repo   string        `yaml:"repo"`
		Forge  string        `yaml:"forge"`
		Target *forge.Target `yaml:"target"`
	}
	if yaml.Unmarshal(data, &run) != nil || run.Role == "" {
		return nil
	}
	if mrID == 0 {
		if run.Target == nil || run.Target.Kind != "merge-request" {
			return nil
		}
		mrID = run.Target.ID
	}
	spent := runFresh(dir)
	if spent == 0 {
		return nil
	}
	if f == nil {
		if f, _, err = openForge(engine.Options{Repo: run.Repo, Forge: run.Forge}); err != nil {
			return nil // no forge to count on
		}
	}
	b, ok := f.(forge.Backlog)
	if !ok {
		return nil
	}
	mr, err := f.MergeRequest(mrID)
	if err != nil {
		return err
	}
	issue := ClosedIssue(mr)
	if issue == 0 {
		return nil
	}
	t := forge.Target{Kind: "issue", ID: issue}
	comments, err := b.Comments(t)
	if err != nil {
		return err
	}
	body, rec, had := recordCommentIn(comments)
	if !had {
		return nil
	}
	id := runID(dir)
	if slices.Contains(rec.Counted, id) {
		return nil
	}
	rec.Spent = maps.Clone(rec.Spent)
	if rec.Spent == nil {
		rec.Spent = map[string][]int{}
	}
	rec.Spent[run.Role] = append(slices.Clone(rec.Spent[run.Role]), spent)
	rec.Counted = append(slices.Clone(rec.Counted), id)
	return f.Sticky(t, backlog.Inert(withRecord(body, rec)), forge.Marker("sticky="+recordKey), false)
}

// runDirs are the run folders of steps.
func runDirs(steps []Step) []string {
	var dirs []string
	for _, s := range steps {
		if s.Result != nil && s.Result.RunDir != "" {
			dirs = append(dirs, s.Result.RunDir)
		}
	}
	return dirs
}

// runFresh is the fresh tokens of a run's calls, as out/calls.jsonl
// keeps them.
func runFresh(dir string) int {
	file, err := os.Open(filepath.Join(dir, "out", "calls.jsonl"))
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()
	n := 0
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var c agent.Call
		if json.Unmarshal(sc.Bytes(), &c) == nil {
			n += c.Fresh()
		}
	}
	return n
}

// ClosedIssue is the issue a merge request closes: the first its title
// or description closes, else the one its branch of a role names
// (workline/developer/<n>, workline/tester/issue-<n>); 0 for none.
func ClosedIssue(mr forge.MergeRequest) int {
	if ids := forge.Closes(mr.Title, mr.Body); len(ids) > 0 {
		return ids[0]
	}
	for _, prefix := range []string{"workline/developer/", "workline/tester/issue-"} {
		if rest, ok := strings.CutPrefix(mr.Branch, prefix); ok {
			if n, err := strconv.Atoi(rest); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// runID is a run's short id: a hash of its folder's name.
func runID(dir string) string {
	sum := sha256.Sum256([]byte(filepath.Base(dir)))
	return hex.EncodeToString(sum[:4])
}

// recordCommentIn is the comment holding the line's record among an
// issue's comments, its marker taken off, and the record.
func recordCommentIn(comments []string) (string, Record, bool) {
	sticky := forge.Marker("sticky=" + recordKey)
	for _, c := range slices.Backward(comments) {
		if !strings.Contains(c, sticky) {
			continue
		}
		if r, had := recordIn([]string{c}); had {
			return strings.TrimSpace(strings.ReplaceAll(c, sticky, "")), r, true
		}
	}
	return "", Record{}, false
}

var (
	recordLine = regexp.MustCompile(`<!-- workline:` + regexp.QuoteMeta(recordMark) + `.*? -->`)
	spentSaid  = regexp.MustCompile(` [0-9.]+[kM]? fresh tokens spent on it so far\.`)
)

// withRecord is body, the comment holding the record, with r in its
// place and the issue's total said again.
func withRecord(body string, r Record) string {
	data, _ := json.Marshal(r)
	body = recordLine.ReplaceAllLiteralString(body, "<!-- workline:"+recordMark+string(data)+" -->")
	total := fmt.Sprintf(" %s fresh tokens spent on it so far.", Tokens(r.Fresh()))
	switch {
	case spentSaid.MatchString(body):
		body = spentSaid.ReplaceAllLiteralString(body, total)
	case strings.Contains(body, " Edited by the engine, not by hand."):
		body = strings.Replace(body, " Edited by the engine, not by hand.", total+" Edited by the engine, not by hand.", 1)
	}
	return body
}
