package evaluation

// The judge's replay (ADR-0045, #362): the 40 questions the reviewer's
// judge was asked on workline's pull requests, measured on 2026-10-14
// (docs/research/judge-measured.md). Each is kept in judge-replay/<item>/
// as the run asked it (question.yaml) and as the judge answered it then
// (answer.yaml), both copied from the run's workline-runs artifact, with
// the truth the measure settled (item.yaml). TestJudgeReplay asks them
// again, of the judge WORKLINE_EVAL names; the set is read, and its
// scoring proved, with no agent (checks_test.go).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/agent"
	wjudge "github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/role"
)

const replayDir = "judge-replay"

// replayItem is one verdict measured: where the judge was asked, what it
// said then, and the truth the measure settled.
type replayItem struct {
	Name      string `yaml:"-"` // its folder
	MeasureID string `yaml:"measure-id"`
	PR        int    `yaml:"pull-request"`
	Lens      string `yaml:"lens"`
	Title     string `yaml:"title"`
	Where     string `yaml:"where"`  // the finding's cause, file:line at commit
	Commit    string `yaml:"commit"` // the head the reviewer read
	// artifact: question.yaml and answer.yaml copied from the run's
	// workline-runs artifact; rebuilt: remade from git at commit, the
	// artifact gone.
	Origin string `yaml:"origin"`
	From   struct {
		Run     int64  `yaml:"run"`     // the workflow run
		Path    string `yaml:"path"`    // the question's folder in its artifact
		Expires string `yaml:"expires"` // when the artifact was due to expire
	} `yaml:"from"`
	Said       string `yaml:"said"` // yes or no, as answer.yaml holds it
	SaidTokens struct {
		In  int `yaml:"in"`
		Out int `yaml:"out"`
	} `yaml:"said-tokens"`
	Truth     bool   `yaml:"truth"`      // the finding holds
	SettledBy string `yaml:"settled-by"` // agreement: the judge and a rater; panel: three agents, both readings
	Matters   *bool  `yaml:"matters"`    // worth raising to a person, by the panel's majority; nil when not asked
	Evidence  string `yaml:"evidence"`   // the code that settled it

	Question, Material string `yaml:"-"` // what the judge was asked (question.yaml)
	Then               struct {
		Yes   *bool  `yaml:"yes"`
		Why   string `yaml:"why"`
		Model string `yaml:"model"`
	} `yaml:"-"` // what it answered then (answer.yaml)
}

// readReplay reads the set, an item a folder, as strictly as a case: a key
// no field names fails it.
func readReplay(dir string) ([]replayItem, error) {
	files, _ := filepath.Glob(filepath.Join(dir, "*", "item.yaml"))
	sort.Strings(files)
	var items []replayItem
	for _, f := range files {
		var it replayItem
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(&it); err != nil {
			return nil, fmt.Errorf("%s: %v", f, err)
		}
		it.Name = filepath.Base(filepath.Dir(f))
		var q struct{ Question, Material, Author string }
		if data, err = os.ReadFile(filepath.Join(filepath.Dir(f), "question.yaml")); err == nil {
			err = yaml.Unmarshal(data, &q)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: question: %v", it.Name, err)
		}
		it.Question, it.Material = q.Question, q.Material
		if data, err = os.ReadFile(filepath.Join(filepath.Dir(f), "answer.yaml")); err == nil {
			err = yaml.Unmarshal(data, &it.Then)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: answer: %v", it.Name, err)
		}
		items = append(items, it)
	}
	return items, nil
}

// replayAnswer is what a judge said to an item: yes, no, or "" when it
// could not answer (Err).
type replayAnswer struct {
	Said string
	Why  string
	Err  error
	Call agent.Call
}

// right: yes to a true finding, no to a false one.
func (a replayAnswer) right(it replayItem) bool {
	return a.Said == map[bool]string{true: "yes", false: "no"}[it.Truth]
}

// askReplay puts an item's question to the judge spec names, as the
// engine asks it (askQuestions): the question, the material, the judge's
// facets of today.
func askReplay(spec string, judgeRole *role.Role, it replayItem) replayAnswer {
	ans, err := wjudge.Ask(spec, judgeRole, it.Question, it.Material)
	a := replayAnswer{Why: ans.Why, Err: err, Call: ans.Call}
	if err == nil {
		a.Said = map[bool]string{true: "yes", false: "no"}[ans.Yes]
	}
	return a
}

// replayTally is a replay against the measure's truth, and against what
// the judge said then (docs/research/judge-measured.md).
type replayTally struct {
	Items, Right                   int
	TrueKept, TrueDropped, True    int // true findings: answered yes, no
	FalseDropped, FalseKept, False int // false findings: answered no, yes
	DroppedThen, DroppedThenKept   int // true findings the judge dropped then; of them, kept now
	Yes, YesRight                  int // answered yes; of them, true findings
	SameAsThen, Unanswered         int
	TokensIn, TokensOut            int
}

func tallyReplay(items []replayItem, now []replayAnswer) replayTally {
	var t replayTally
	for i, it := range items {
		a := now[i]
		t.Items++
		t.TokensIn += a.Call.TokensIn
		t.TokensOut += a.Call.TokensOut
		if a.right(it) {
			t.Right++
		}
		if a.Said == it.Said {
			t.SameAsThen++
		}
		if it.Truth && it.Said == "no" {
			t.DroppedThen++
			if a.Said == "yes" {
				t.DroppedThenKept++
			}
		}
		switch {
		case a.Said == "":
			t.Unanswered++
		case it.Truth && a.Said == "yes":
			t.TrueKept++
		case it.Truth:
			t.TrueDropped++
		case a.Said == "no":
			t.FalseDropped++
		default:
			t.FalseKept++
		}
		if it.Truth {
			t.True++
		} else {
			t.False++
		}
		if a.Said == "yes" {
			t.Yes++
			if it.Truth {
				t.YesRight++
			}
		}
	}
	return t
}

func (t replayTally) String() string {
	return fmt.Sprintf("%d/%d right; true findings kept %d of %d, dropped %d (of the %d dropped then, %d kept now); "+
		"false findings dropped %d of %d, kept %d; its yes right %d of %d; same as then %d of %d; unanswered %d; tokens in %d, out %d",
		t.Right, t.Items, t.TrueKept, t.True, t.TrueDropped, t.DroppedThen, t.DroppedThenKept,
		t.FalseDropped, t.False, t.FalseKept, t.YesRight, t.Yes, t.SameAsThen, t.Items, t.Unanswered, t.TokensIn, t.TokensOut)
}

// TestJudgeReplay asks the 40 questions again, each in a context of its
// own, of the judge WORKLINE_EVAL names (`claude:opus` was the judge
// then). A line of results.tsv an item, case judge/<item>: a point when
// it says yes to a true finding, no to a false one; the tally of the run
// is logged.
func TestJudgeReplay(t *testing.T) {
	spec := os.Getenv("WORKLINE_EVAL")
	if spec == "" {
		t.Skip("set WORKLINE_EVAL=claude:opus to replay the judge's 40 measured questions: it calls a real agent and costs tokens")
	}
	items, err := readReplay(replayDir)
	if err != nil {
		t.Fatal(err)
	}
	judgeRole, err := role.Load("../../roles", "judge")
	if err != nil {
		t.Fatal(err)
	}
	version := worklineVersion()
	now := make([]replayAnswer, len(items))
	t.Run("items", func(t *testing.T) {
		for i, it := range items {
			t.Run(it.Name, func(t *testing.T) {
				t.Parallel()
				start := time.Now()
				a := askReplay(spec, judgeRole, it)
				now[i] = a
				score, failed := "1/1", ""
				if !a.right(it) {
					score, failed = "0/1", replayFailed(it, a)
				}
				t.Logf("%s — %s; then %s, now %s: %.300s", it.Name, score, it.Said, orDash(a.Said), a.Why)
				models, efforts, tokensIn, tokensOut, cost := answeredBy([]call{{a.Call.Effort, a.Call.Model, a.Call.TokensIn, a.Call.TokensOut, a.Call.CostUSD, "judge"}})
				line := strings.Join([]string{time.Now().UTC().Format(time.RFC3339), version, spec, models, efforts,
					"judge/" + it.Name, score, "1", tokensIn, tokensOut, cost, fmt.Sprintf("%.0f", time.Since(start).Seconds()),
					failed, "", ""}, "\t")
				if err := appendResult(line); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
	t.Logf("judge replay, %s: %s", spec, tallyReplay(items, now))
}

// replayFailed says what an item's answer lost its point on, one line.
func replayFailed(it replayItem, a replayAnswer) string {
	if a.Said == "" {
		return fmt.Sprintf("no answer: %.300s", oneLine(fmt.Sprint(a.Err)))
	}
	kind := map[bool]string{true: "true", false: "false"}[it.Truth]
	return fmt.Sprintf("said %s (%s then) to a %s finding: %.300s", a.Said, it.Said, kind, oneLine(a.Why))
}

var spaces = regexp.MustCompile(`\s+`)

func oneLine(s string) string { return spaces.ReplaceAllString(strings.TrimSpace(s), " ") }
