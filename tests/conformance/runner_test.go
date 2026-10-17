// Package conformance runs the cases in cases/ against an engine binary, from
// the outside, as docs/spec/conformance.md describes. Any engine that passes
// them obeys the contract.
//
// Cases listed in pending.txt are not expected to pass yet. They still run: a
// pending case that starts passing fails the suite until it is removed from
// the list, so the list never lies.
package conformance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
	"go.yaml.in/yaml/v3"
)

var engineBin, rolesDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "workline-engine-")
	if err != nil {
		panic(err)
	}
	engineBin = os.Getenv("WORKLINE_ENGINE")
	if engineBin == "" {
		engineBin = filepath.Join(dir, "workline")
		build := exec.Command("go", "build", "-o", engineBin, "../../cmd/workline")
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			panic("building the engine: " + err.Error())
		}
	}
	// The shipped roles, and the roles only the tests use (fixtures/roles):
	// a mechanism no shipped role needs is still tried on one of those.
	rolesDir = filepath.Join(dir, "roles")
	if err := linkRoles(rolesDir, "../../roles", "fixtures/roles"); err != nil {
		panic("gathering the roles: " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type repoSpec struct {
	Repo  string   `yaml:"repo"`
	Setup []string `yaml:"setup"`
}

// caseFile is a case as written. It is read strictly (readCase): a key no
// field names, at any level, fails the case, never passes unread.
type caseFile struct {
	Case   string     `yaml:"case"`
	About  string     `yaml:"about"`
	Given  givenSpec  `yaml:"given"`
	Run    runSpec    `yaml:"run"`
	Expect expectSpec `yaml:"expect"`
}

// givenSpec is what the case starts from.
type givenSpec struct {
	Repo   string              `yaml:"repo"`
	Repos  map[string]repoSpec `yaml:"repos"`
	Setup  []string            `yaml:"setup"`
	Config map[string]any      `yaml:"config"`
	Forge  map[string]any      `yaml:"forge"`
	Seen   map[string]any      `yaml:"models-seen"` // the models this machine saw answer, before the run
	Env    map[string]string   `yaml:"env"`         // variables set for the run, e.g. a PATH without a tool; $VAR expands
	Tags   []string            `yaml:"engine-tags"` // the engine built with these Go build tags: a change of the tests only (engineFor)
}

// runSpec is what the case runs.
type runSpec struct {
	Role    string            `yaml:"role"`
	Target  map[string]int    `yaml:"target"`
	Branch  string            `yaml:"branch"` // the branch the targeted merge request comes from, as --branch
	NoApply bool              `yaml:"no-apply"`
	OpenMR  bool              `yaml:"open-merge-request"`
	PushMR  bool              `yaml:"push-to-merge-request"`
	Event   string            `yaml:"event"`
	Input   map[string]string `yaml:"input"`
	AI      string            `yaml:"ai"`
	Scope   []string          `yaml:"scope"`
	Tamper  string            `yaml:"tamper"`
	Then    string            `yaml:"then"`
	Between map[string]any    `yaml:"between"` // with then: again, a person's edits to the forge before the second run
	// Pushed: with then: judged-twice or applied-between, what lands on
	// the branch after the run judged, before it is applied (#375).
	Pushed []string `yaml:"pushed"`
	Route   string            `yaml:"route"`
	Item    int               `yaml:"item"`
	Gate    string            `yaml:"gate"`
	Doctor  bool              `yaml:"doctor"`
	Init    bool              `yaml:"init"`
	InitOpt []string          `yaml:"init-options"`  // workline init, with these options
	Setup   []string          `yaml:"setup"`         // workline setup, with these options
	Import  []string          `yaml:"issues-import"` // workline issues import, with these arguments
	Review  []string          `yaml:"review"`        // workline review, with these options
	Sample  []string          `yaml:"sample"`        // workline sample, with these options; then: apply writes what it found
	Reports bool              `yaml:"reports"`       // also write --sarif and --code-quality
	JobSum  bool              `yaml:"summary"`       // also write --summary, Markdown and HTML, the apply after it too
	Forge   string            `yaml:"forge"`         // a forge spec passed as --forge (local, cmd:…), instead of the simulated one
	Follow  []string          `yaml:"follow"`        // workline follow, with these options
	// ThenLine: after the run, the line on a branch of origin, checked
	// out, as CI runs it on that branch's push: what the case expects of
	// status, steps and calls is its; the findings are both runs'.
	ThenLine *thenLine `yaml:"then-line"`
	CLI     []string          `yaml:"cli"`           // workline with these arguments, as typed: its exit code, what it prints
	// ApplyFrom: with then: resume, the runs applied as the base's job
	// applies a merge request's own CI run (apply --from-merge-request).
	ApplyFrom *applyFrom `yaml:"apply-from"`
}

// applyFrom is a merge request's own run applied by a job of the base's:
// the base checked out, the head the tip of a branch of origin, the run's
// folders first changed as the merge request's code could have.
type applyFrom struct {
	MergeRequest int               `yaml:"merge-request"`
	Base         string            `yaml:"base"`    // the branch checked out before applying
	Head         string            `yaml:"head"`    // a branch of origin: its tip is --head
	Run          map[string]any    `yaml:"run"`     // keys of out/run.yaml written over
	Intents      []map[string]any  `yaml:"intents"` // proposals added to out/intentions.yaml
	Listed       string            `yaml:"listed"`  // the run listed at this path, not its own
	Setup        []string          `yaml:"setup"`   // commands run in the repository first: what its branches became since the run
	Env          map[string]string `yaml:"env"`     // variables of the applying job (WORKLINE_APP)
	// Line: apply the line's result, as CI's job does (`apply --line`),
	// the runs it lists, none when it proposed nothing.
	Line bool `yaml:"line"`
}

// thenLine is the line run after a case's run: the event, the branch of
// origin checked out, the range given, the agent.
type thenLine struct {
	Event  string `yaml:"event"`
	Branch string `yaml:"branch"`
	Range  string `yaml:"range"`
	AI     string `yaml:"ai"`
}

// expectSpec is what the case expects.
type expectSpec struct {
	Status      string                       `yaml:"status"`
	Findings    []findingWant                `yaml:"findings"`
	NoFindings  []findingWant                `yaml:"no-findings"` // findings that must not be there
	Checks      []checkWant                  `yaml:"checks"`      // the doctor's checks, ok ones too
	AgentCalls  *int                         `yaml:"agent-calls"`
	Applied     []string                     `yaml:"applied"`
	Refused     []string                     `yaml:"refused"`
	Files       map[string]fileWant          `yaml:"files"`
	Pushed      map[string]map[string]string `yaml:"pushed"`         // branch -> path -> a text the remote's branch holds there
	PushedMsg   map[string]string            `yaml:"pushed-message"` // branch -> a text the message of the remote branch's tip holds
	NotPushed   []string                     `yaml:"not-pushed"`     // branches of the remote the run left where they were
	OnTop       map[string]string            `yaml:"on-top"`         // branch -> the branch of the remote its tip holds
	Merged      map[string]string            `yaml:"merged"`         // branch -> base: the remote branch's tip merges base's tip into its tip before the run, its first parent
	Gone        []string                     `yaml:"gone"`           // branches of the remote absent after the run
	Forge       *forgeWant                   `yaml:"forge"`
	Steps       []string                     `yaml:"steps"`
	Calls       []map[string]string          `yaml:"calls"`
	SARIF       []map[string]any             `yaml:"sarif"`         // results, by rule, uri, line, level
	CodeQuality []map[string]any             `yaml:"code-quality"`  // issues, by check_name, path, line, severity
	LeftOut     []string                     `yaml:"left-out"`      // wheres found in neither report
	SARIFLacks  []string                     `yaml:"sarif-lacks"`   // rules no SARIF result has
	Notes       []string                     `yaml:"notes"`         // texts the agent's notes hold
	RefusedKept int                          `yaml:"refused-kept"`  // refused answers kept in the run folders
	CallsKept   int                          `yaml:"calls-kept"`    // agent calls recorded in the run folders
	RunFiles    map[string]runFileWant       `yaml:"run-files"`     // a file of the run folder -> the texts it holds, or not
	Branches    map[string]map[string]string `yaml:"branches"`      // a local branch -> path -> a text it holds there
	Listed      []string                     `yaml:"issues-listed"` // texts `workline issues list` prints afterwards
	Summary     string                       `yaml:"summary"`       // a text the result's summary holds
	SummaryFile []string                     `yaml:"summary-file"`  // texts the --summary file holds, in this order; with cli, workline-summary.md
	SummaryHTML []string                     `yaml:"summary-html"`  // texts the --summary .html file holds, in this order
	Coverage    []map[string]string          `yaml:"coverage"`      // an import's map: its items, by lines, state, issue, words, why
	NotCovered  []map[string]string          `yaml:"not-covered"`   // an import's items left with no issue nor reason
	Exit        *int                         `yaml:"exit"`          // with cli: the exit code
	Stdout      []string                     `yaml:"stdout"`        // with cli: texts printed on stdout
	Stderr      []string                     `yaml:"stderr"`        // with cli: texts printed on stderr
}

// findingWant is a finding the run reports, or must not: its rule and
// level, texts its where, message and fix hold (one not wanted: where equal).
type findingWant struct {
	Rule    string `yaml:"rule"`
	Where   string `yaml:"where"`
	Level   string `yaml:"level"` // block or warn; any when not given
	Message string `yaml:"message"`
	Fix     string `yaml:"fix"`
}

// checkWant is a doctor's check: its rule and level, texts the rest hold.
type checkWant struct {
	Rule    string `yaml:"rule"`
	Where   string `yaml:"where"`
	Level   string `yaml:"level"`
	Message string `yaml:"message"`
	Fix     string `yaml:"fix"`
}

// fileWant is a file of the repository after the run.
type fileWant struct {
	Exists   *bool `yaml:"exists"`
	Contains texts `yaml:"contains"`
	Lacks    texts `yaml:"lacks"`
}

// runFileWant is a file of the run folders: texts one holds, none holds.
type runFileWant struct {
	Contains    texts `yaml:"contains"`
	NotContains texts `yaml:"not-contains"`
}

// forgeWant is the simulated forge's state after the run, by kind.
type forgeWant struct {
	Issues        []itemWant `yaml:"issues"`
	MergeRequests []itemWant `yaml:"merge-requests"`
	Labels        []itemWant `yaml:"labels"` // labels defined, their name as id
	// Statuses: each must be the last status of its name on its commit (a
	// revision of the repository) — by, state and a part of its description
	// when given; absent: none of that name there.
	Statuses []statusWant `yaml:"statuses"`
	// CheckRuns: each must be the last check of its name on its commit (a
	// revision of the repository), holding each annotation listed;
	// absent: none of that name there.
	CheckRuns []checkRunWant `yaml:"check-runs"`
}

// checkRunWant is a check the simulated forge holds on a commit.
type checkRunWant struct {
	Commit      string           `yaml:"commit"`
	Name        string           `yaml:"name"`
	Annotations []annotationWant `yaml:"annotations"`
	Absent      bool             `yaml:"absent"`
}

// annotationWant is an annotation of a check: its path, line and level,
// a text its message holds.
type annotationWant struct {
	Path     string `yaml:"path"`
	Line     int    `yaml:"line"`
	Level    string `yaml:"level"`
	Contains string `yaml:"contains"`
}

// statusWant is a status the simulated forge holds on a commit.
type statusWant struct {
	Commit      string `yaml:"commit"`
	Name        string `yaml:"name"`
	State       string `yaml:"state"`
	By          string `yaml:"by"`
	Description string `yaml:"description"`
	Absent      bool   `yaml:"absent"`
}

// itemWant is an item of the forge, by id: each field given is checked
// (compareForge); `absent: true`, no item with that id.
type itemWant struct {
	ID              int                 `yaml:"id"`
	Absent          bool                `yaml:"absent"`
	Title           *string             `yaml:"title"`
	Branch          *string             `yaml:"branch"`
	Base            *string             `yaml:"base"`
	Reason          *string             `yaml:"reason"` // why it was closed
	Milestone       *string             `yaml:"milestone"`
	Parent          *int                `yaml:"parent"` // the issue it is a sub-issue of, 0 for none
	Closed          *bool               `yaml:"closed"`
	AutoMerge       *bool               `yaml:"auto-merge"` // a merge request's auto-merge on (ADR-0042)
	Merged          *bool               `yaml:"merged"`     // a merge request the forge merged
	BodyContains    texts               `yaml:"body-contains"`
	BodyLacks       texts               `yaml:"body-lacks"`
	Comments        *int                `yaml:"comments"` // how many
	Posted          *int                `yaml:"posted"`   // comments posted, each mailing the issue's followers; an edit posts none
	CommentContains texts               `yaml:"comment-contains"`
	CommentLacks    texts               `yaml:"comment-lacks"`
	BlockedBy       *[]int              `yaml:"blocked-by"` // the forge's own relation
	Labels          *[]string           `yaml:"labels"`     // the exact set
	Reactions       map[string][]string `yaml:"reactions"`  // on its comments, by the comment's id
	Threads         *[]threadWant       `yaml:"threads"`    // a merge request's comments on lines, exactly these, in the order posted
}

// threadWant is a comment on a line of a merge request: its file and
// line, texts it holds and does not, whether it is resolved.
type threadWant struct {
	Path     string `yaml:"path"`
	Line     int    `yaml:"line"`
	Contains texts  `yaml:"contains"`
	Lacks    texts  `yaml:"lacks"`
	Resolved *bool  `yaml:"resolved"`
}

type result struct {
	Status   string `json:"status"`
	Findings []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Level   string `json:"level"`
		Message string `json:"message"`
		Fix     string `json:"fix"`
	} `json:"findings"`
	Checks []struct {
		Rule    string `json:"rule"`
		Where   string `json:"where"`
		Level   string `json:"level"`
		Message string `json:"message"`
		Fix     string `json:"fix"`
	} `json:"checks"`
	AgentCalls int              `json:"agent-calls"`
	Calls      []map[string]any `json:"calls"`
	RunDir     string           `json:"run-dir"`
	Pending    []string         `json:"pending"` // runs a line judged and did not apply
	Notes      []string         `json:"notes"`
	Summary    string           `json:"summary"`
	Coverage   struct {
		Items      []map[string]any `json:"items"`
		NotCovered []map[string]any `json:"not-covered"`
	} `json:"coverage"`
	Applied []string `json:"applied"`
	Refused []string `json:"refused"`
	Steps   []struct {
		Name string `json:"name"`
	} `json:"steps"`
}

func TestConformance(t *testing.T) {
	pending := readPending(t)
	files, _ := filepath.Glob("cases/*/*.yaml")
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no cases found")
	}
	for _, f := range files {
		c, err := readCase(f)
		if err != nil {
			// Failed whatever pending.txt says: a case misread checks nothing.
			name := c.Case
			if name == "" {
				name = strings.TrimSuffix(strings.TrimPrefix(f, "cases/"), ".yaml")
			}
			t.Run(name, func(t *testing.T) { t.Error(err) })
			continue
		}
		t.Run(c.Case, func(t *testing.T) {
			problems := runCase(t, &c)
			switch {
			case pending[c.Case] && len(problems) == 0:
				t.Errorf("passes now: remove it from pending.txt")
			case pending[c.Case]:
				t.Skipf("pending: %s", problems[0])
			case len(problems) > 0:
				t.Errorf("%s\n  %s", c.About, strings.Join(problems, "\n  "))
			}
		})
	}
}

// readCase reads a case file strictly, with the YAML library's own check:
// a key no field of caseFile names is an error naming the key and the file.
// The forge a case gives, and a person's edits to it, are the simulated
// forge's to read: checked against its own type, by the JSON library's.
func readCase(file string) (caseFile, error) {
	var c caseFile
	data, err := os.ReadFile(file)
	if err != nil {
		return c, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("%s: %v", file, err)
	}
	for name, state := range map[string]map[string]any{"given.forge": c.Given.Forge, "run.between": c.Run.Between} {
		if err := forgeReads(state); err != nil {
			return c, fmt.Errorf("%s: %s: %v", file, name, err)
		}
	}
	return c, nil
}

// forgeReads says what of a forge's state the simulated forge would not
// read: a key no field of its state names (GitLab's mock's members
// included), or a value of another type.
func forgeReads(state map[string]any) error {
	if state == nil {
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(&gitlabState{})
}

// engines are the engines built with build tags of the tests, by their tags.
var engines = map[string]string{}

// engineFor is the engine a case runs: the suite's, or, for a case naming
// engine-tags, one built from this source with those Go build tags — a
// change only the tests make, which no shipped engine has (the table of a
// section's headings given one more name, workline_alias). An engine
// built already (WORKLINE_ENGINE) cannot run such a case: said, never
// passed.
func engineFor(tags []string) (string, error) {
	if len(tags) == 0 {
		return engineBin, nil
	}
	key := strings.Join(tags, ",")
	if bin, ok := engines[key]; ok {
		return bin, nil
	}
	if os.Getenv("WORKLINE_ENGINE") != "" {
		return "", fmt.Errorf("needs the engine built from source with -tags %s, not the one WORKLINE_ENGINE names", key)
	}
	bin := filepath.Join(filepath.Dir(engineBin), "workline-"+strings.Join(tags, "-"))
	if out, err := exec.Command("go", "build", "-tags", key, "-o", bin, "../../cmd/workline").CombinedOutput(); err != nil {
		return "", fmt.Errorf("building the engine with -tags %s: %v\n%s", key, err, out)
	}
	engines[key] = bin
	return bin, nil
}

// runCase returns what differs from the expectation; empty means it passes.
func runCase(t *testing.T, c *caseFile) []string {
	engineBin, err := engineFor(c.Given.Tags)
	if err != nil {
		return []string{err.Error()}
	}
	work := t.TempDir()
	seen := filepath.Join(work, "models-seen.yaml")
	// The user's config folder stays out: their own lists and defaults are not the case's.
	env := append(hermeticEnv(), "WORKLINE_MODELS_SEEN="+seen, "XDG_CONFIG_HOME="+filepath.Join(work, "config"))
	if c.Given.Seen != nil {
		data, _ := yaml.Marshal(c.Given.Seen)
		if err := os.WriteFile(seen, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	names := make([]string, 0, len(c.Given.Repos))
	for n := range c.Given.Repos {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		dir := filepath.Join(work, n)
		if err := build(dir, c.Given.Repos[n].Repo, c.Given.Repos[n].Setup, env); err != nil {
			return []string{err.Error()}
		}
		env = append(env, n+"="+dir)
	}
	// The simulated forge first: the repository's setup may write into it
	// (FORGE_FILE) what only the built repository knows, a commit's id.
	forgeFile := ""
	setupEnv := env
	if len(c.Given.Forge) > 0 {
		forgeFile = filepath.Join(work, "forge.json")
		data, _ := json.Marshal(c.Given.Forge)
		os.WriteFile(forgeFile, data, 0o644)
		setupEnv = append(slices.Clone(env), "FORGE_FILE="+forgeFile)
	}
	repo := filepath.Join(work, "repo")
	if err := build(repo, c.Given.Repo, c.Given.Setup, setupEnv); err != nil {
		return []string{err.Error()}
	}
	if c.Given.Config != nil {
		data, _ := yaml.Marshal(c.Given.Config)
		os.MkdirAll(filepath.Join(repo, ".workline"), 0o755)
		os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), data, 0o644)
	}

	forgeSpec := ""
	if forgeFile != "" {
		forgeSpec = "fake:" + forgeFile
	}
	if c.Run.Forge != "" {
		forgeSpec = c.Run.Forge
	}
	if c.Run.Forge == "gitlab" {
		// GitLab's REST API, simulated over the same file (gitlab_test.go).
		if forgeFile == "" {
			return []string{"run.forge gitlab needs given.forge"}
		}
		srv := httptest.NewServer(&gitlabMock{file: forgeFile, repo: repo})
		defer srv.Close()
		env = append(env, "CI_API_V4_URL="+srv.URL+"/api/v4", "CI_PROJECT_ID=1", "CI_PROJECT_PATH=", "GITLAB_TOKEN=conformance")
	}
	if c.Run.CLI != nil {
		return runCLI(engineBin, c, repo, env)
	}
	roles := rolesDir
	args := []string{"run-role", c.Run.Role, "--event", c.Run.Event, "--repo", repo, "--roles", roles, "--json"}
	switch {
	case c.Run.Gate != "":
		args = []string{"gate", c.Run.Gate, "--repo", repo, "--json"}
	case c.Run.Doctor:
		args = []string{"doctor", "--repo", repo, "--json"}
	case c.Run.Init:
		args = append([]string{"init", "--repo", repo, "--roles", roles, "--json"}, c.Run.InitOpt...)
	case c.Run.Setup != nil:
		args = append(append([]string{"setup"}, c.Run.Setup...), "--json")
	case c.Run.Review != nil:
		args = append(append([]string{"review"}, c.Run.Review...), "--repo", repo, "--roles", roles, "--json")
	case c.Run.Import != nil:
		args = append(append([]string{"issues", "import"}, c.Run.Import...), "--repo", repo, "--json")
	case c.Run.Sample != nil:
		// The read reads the forge, the acts it draws (ADR-0046), and
		// writes nothing to it.
		args = append(append([]string{"sample"}, c.Run.Sample...), "--repo", repo, "--out", filepath.Join(work, "sample.json"), "--json")
		for i, a := range args {
			if strings.HasPrefix(a, "fake:") {
				p, _ := filepath.Abs(filepath.Join("fixtures", "agents", strings.TrimPrefix(a, "fake:")+".yaml"))
				args[i] = "fake:" + p
			}
		}
		if forgeSpec != "" {
			args = append(args, "--forge", forgeSpec)
		}
	case c.Run.Route == "ready" && c.Run.Item != 0:
		args = []string{"item", "ready", fmt.Sprint(c.Run.Item), "--repo", repo, "--json"}
		if forgeSpec != "" {
			args = append(args, "--forge", forgeSpec)
		}
	case c.Run.Route != "":
		args = []string{"route", c.Run.Route, "--repo", repo, "--roles", roles, "--json"}
	case c.Run.Follow != nil:
		args = append(append([]string{"follow"}, c.Run.Follow...), "--repo", repo, "--json")
	}
	ai := c.Run.AI
	if strings.HasPrefix(ai, "fake:") {
		p, _ := filepath.Abs(filepath.Join("fixtures", "agents", strings.TrimPrefix(ai, "fake:")+".yaml"))
		ai = "fake:" + p
	}
	if ai != "" {
		args = append(args, "--ai", ai)
	}
	for k, v := range c.Run.Input {
		args = append(args, "--input", k+"="+v)
	}
	if forgeSpec != "" && c.Run.Item == 0 && c.Run.Sample == nil {
		args = append(args, "--forge", forgeSpec)
	}
	for kind, id := range c.Run.Target {
		args = append(args, "--target", fmt.Sprintf("%s:%d", kind, id))
	}
	if c.Run.Branch != "" {
		args = append(args, "--branch", c.Run.Branch)
	}
	for _, s := range c.Run.Scope {
		args = append(args, "--scope", s)
	}
	if c.Run.OpenMR {
		args = append(args, "--open-merge-request")
	}
	if c.Run.PushMR {
		args = append(args, "--push-to-merge-request")
	}
	if c.Run.NoApply {
		args = append(args, "--no-apply")
	}
	if c.Run.Tamper != "" {
		args = append(args, "--test-tamper-before-apply")
	}
	sarifFile, cqFile := filepath.Join(work, "findings.sarif"), filepath.Join(work, "code-quality.json")
	if c.Run.Reports {
		args = append(args, "--sarif", sarifFile, "--code-quality", cqFile)
	}
	summaryFile, summaryHTML := filepath.Join(work, "summary.md"), filepath.Join(work, "summary.html")
	if c.Run.JobSum {
		args = append(args, "--summary", summaryFile, "--summary", summaryHTML)
	}
	// Judged on one machine, applied on another (CI's two jobs): each its own
	// cache, the first gone by the time the second applies.
	// With the roles built into the engine, as CI installs it: no --roles.
	judgeCache := ""
	if c.Run.Then == "resume-elsewhere" {
		judgeCache = t.TempDir()
		env = append(env, "XDG_CACHE_HOME="+judgeCache)
		for i, a := range args {
			if a == "--roles" {
				args = append(args[:i:i], args[i+2:]...)
				break
			}
		}
	}
	cmd := exec.Command(engineBin, args...)
	cmd.Dir = repo
	cmd.Env = env
	if c.Run.Setup != nil { // the machine's git config, which setup changes, is the case's own
		global := filepath.Join(work, "gitconfig")
		os.WriteFile(global, nil, 0o644)
		cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL="+global)
	}
	for k, v := range c.Given.Env {
		cmd.Env = append(cmd.Env, k+"="+os.Expand(v, func(name string) string { return lookup(cmd.Env, name) }))
	}
	tips := map[string]string{} // the remote's branches before the run
	for _, b := range c.Expect.NotPushed {
		out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output()
		tips[b] = string(out)
	}
	before := map[string]string{} // the tips of the branches merged into, before the run
	for b := range c.Expect.Merged {
		out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output()
		before[b], _, _ = strings.Cut(string(out), "\t")
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // the exit code mirrors the status, which is checked below
	var r result
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		return []string{fmt.Sprintf("engine printed no result: %v\n%s", err, stderr.String())}
	}
	if c.Run.Then == "again" {
		// The same run once more, on the forge and the repository as the
		// first left them, after a person's edits (between): what a run
		// records, the next reads back.
		if err := editForge(forgeFile, c.Run.Between); err != nil {
			return []string{err.Error()}
		}
		again := exec.Command(engineBin, args...)
		again.Dir, again.Env = repo, cmd.Env
		stdout.Reset()
		again.Stdout, again.Stderr = &stdout, &stderr
		_ = again.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("the second run printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Calls = append(r.Calls, r2.Calls...)
		r2.Findings = append(r.Findings, r2.Findings...)
		r = r2
	}
	if c.Run.Then == "judged-twice" || c.Run.Then == "applied-between" {
		var p []string
		if r, p = pushedBeforeApplied(c, engineBin, args, cmd.Env, repo, forgeSpec, r); len(p) > 0 {
			return p
		}
	}
	if l := c.Run.ThenLine; l != nil {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		if out, err := exec.Command("git", "-C", repo, "checkout", "-q", "--detach", "origin/"+l.Branch).CombinedOutput(); err != nil {
			return []string{fmt.Sprintf("then-line: origin/%s not checked out: %s", l.Branch, out)}
		}
		lineArgs := []string{"route", l.Event, "--repo", repo, "--roles", roles, "--json", "--no-apply", "--forge", "none", "--input", "range=" + l.Range}
		if l.AI != "" {
			ai := l.AI
			if strings.HasPrefix(ai, "fake:") {
				p, _ := filepath.Abs(filepath.Join("fixtures", "agents", strings.TrimPrefix(ai, "fake:")+".yaml"))
				ai = "fake:" + p
			}
			lineArgs = append(lineArgs, "--ai", ai)
		}
		line := exec.Command(engineBin, lineArgs...)
		line.Dir, line.Env = repo, cmd.Env
		stdout.Reset()
		line.Stdout, line.Stderr = &stdout, &stderr
		_ = line.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("then-line printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Findings = append(r.Findings, r2.Findings...)
		r = r2
	}
	if c.Run.Then == "resume" || c.Run.Then == "resume-elsewhere" {
		dirs := r.Pending
		if len(dirs) == 0 && r.RunDir != "" {
			dirs = []string{r.RunDir}
		}
		if len(dirs) == 0 && (c.Run.ApplyFrom == nil || !c.Run.ApplyFrom.Line) {
			return []string{"the first run reported no run folder to resume"}
		}
		var fromArgs []string
		if a := c.Run.ApplyFrom; a != nil {
			var problems []string
			if dirs, fromArgs, problems = poisoned(a, dirs, repo, forgeSpec); len(problems) > 0 {
				return problems
			}
		}
		resumeArgs := append(append(append([]string{"apply"}, dirs...), "--json"), fromArgs...)
		if a := c.Run.ApplyFrom; a != nil && a.Line {
			// The line's result as the judge printed it, its runs listed
			// where the run's code could have changed them (poisoned).
			lineFile := filepath.Join(work, "line.json")
			var l map[string]any
			json.Unmarshal(stdout.Bytes(), &l)
			if len(dirs) > 0 {
				l["pending"] = dirs
			}
			data, _ := json.Marshal(l)
			os.WriteFile(lineFile, data, 0o644)
			resumeArgs = append([]string{"apply", "--line", lineFile, "--json"}, fromArgs...)
		}
		if c.Run.JobSum {
			resumeArgs = append(resumeArgs, "--summary", summaryFile, "--summary", summaryHTML)
		}
		resume := exec.Command(engineBin, resumeArgs...)
		resume.Env = env
		resume.Dir = repo
		if a := c.Run.ApplyFrom; a != nil {
			for k, v := range a.Env {
				resume.Env = append(slices.Clone(resume.Env), k+"="+v)
			}
		}
		if judgeCache != "" {
			os.RemoveAll(judgeCache)
			resume.Env = append(env, "XDG_CACHE_HOME="+t.TempDir())
		}
		stdout.Reset()
		resume.Stdout, resume.Stderr = &stdout, &stderr
		_ = resume.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("resume printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Calls = append(r.Calls, r2.Calls...)
		if len(r2.Coverage.Items) == 0 && len(r2.Coverage.NotCovered) == 0 {
			r2.Coverage = r.Coverage // an import's map is the judge's: apply prints none
		}
		r = r2
	}
	if c.Run.Sample != nil && c.Run.Then == "apply" {
		// The write, as CI's job holding the forge's token and no AI key.
		applyArgs := []string{"sample", "--apply", filepath.Join(work, "sample.json"), "--repo", repo, "--forge", forgeSpec, "--json"}
		if c.Run.JobSum {
			applyArgs = append(applyArgs, "--summary", summaryFile, "--summary", summaryHTML)
		}
		apply := exec.Command(engineBin, applyArgs...)
		apply.Dir, apply.Env = repo, cmd.Env
		stdout.Reset()
		apply.Stdout, apply.Stderr = &stdout, &stderr
		_ = apply.Run()
		var r2 result
		if err := json.Unmarshal(stdout.Bytes(), &r2); err != nil {
			return []string{fmt.Sprintf("sample --apply printed no result: %v\n%s", err, stderr.String())}
		}
		r2.AgentCalls += r.AgentCalls
		r2.Calls = append(r.Calls, r2.Calls...)
		r2.Findings = append(r.Findings, r2.Findings...)
		r = r2
	}
	problems := compare(c, &r, repo)
	for b, before := range tips {
		if out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output(); string(out) != before {
			problems = append(problems, fmt.Sprintf("origin's %s moved: %q, was %q", b, out, before))
		}
	}
	for _, b := range c.Expect.Gone {
		if out, _ := exec.Command("git", "-C", repo, "ls-remote", "origin", "refs/heads/"+b).Output(); len(out) > 0 {
			problems = append(problems, fmt.Sprintf("origin's %s is there: %s", b, out))
		}
	}
	for b, base := range c.Expect.Merged {
		// A merge commit: the branch's tip before the run, its commits
		// kept, as first parent; base's tip as second.
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		out, _ := exec.Command("git", "-C", repo, "rev-list", "--parents", "-n", "1", "origin/"+b).Output()
		baseTip, _ := exec.Command("git", "-C", repo, "rev-parse", "origin/"+base).Output()
		if got := strings.Fields(string(out)); len(got) != 3 || got[1] != before[b] || got[2] != strings.TrimSpace(string(baseTip)) {
			problems = append(problems, fmt.Sprintf("origin's %s is not %s merged into its tip %s: commit and parents %v", b, base, before[b], got))
		}
	}
	if len(c.Expect.Listed) > 0 {
		list := exec.Command(engineBin, "issues", "list", "--repo", repo)
		list.Env = env
		out, err := list.CombinedOutput()
		for _, text := range c.Expect.Listed {
			if err != nil || !strings.Contains(string(out), text) {
				problems = append(problems, fmt.Sprintf("workline issues list does not print %q (%v):\n%s", text, err, out))
			}
		}
	}
	if c.Run.Reports {
		problems = append(problems, compareReports(c, sarifFile, cqFile)...)
	}
	if len(c.Expect.SummaryFile) > 0 {
		problems = append(problems, compareSummaryFile(c.Expect.SummaryFile, summaryFile)...)
	}
	if len(c.Expect.SummaryHTML) > 0 {
		problems = append(problems, compareSummaryFile(c.Expect.SummaryHTML, summaryHTML)...)
	}
	if c.Expect.Forge != nil {
		problems = append(problems, compareForge(c.Expect.Forge, forgeFile)...)
		problems = append(problems, compareStatuses(c.Expect.Forge.Statuses, forgeFile, repo)...)
		problems = append(problems, compareCheckRuns(c.Expect.Forge.CheckRuns, forgeFile, repo)...)
	}
	return problems
}

// runCLI runs workline with the case's arguments, as a person or a script
// types them, and checks its exit code and what it prints where.
func runCLI(engineBin string, c *caseFile, repo string, env []string) []string {
	cmd := exec.Command(engineBin, c.Run.CLI...)
	cmd.Dir, cmd.Env = repo, env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		return []string{err.Error()}
	}
	var p []string
	if c.Expect.Exit != nil && code != *c.Expect.Exit {
		p = append(p, fmt.Sprintf("exit = %d, want %d\nstdout: %s\nstderr: %s", code, *c.Expect.Exit, stdout.String(), stderr.String()))
	}
	for _, text := range c.Expect.Stdout {
		if !strings.Contains(stdout.String(), text) {
			p = append(p, fmt.Sprintf("stdout lacks %q:\n%s", text, stdout.String()))
		}
	}
	for _, text := range c.Expect.Stderr {
		if !strings.Contains(stderr.String(), text) {
			p = append(p, fmt.Sprintf("stderr lacks %q:\n%s", text, stderr.String()))
		}
	}
	if len(c.Expect.SummaryFile) > 0 { // the arguments name it, as CI's templates do
		p = append(p, compareSummaryFile(c.Expect.SummaryFile, filepath.Join(repo, "workline-summary.md"))...)
	}
	return append(p, compareFiles(c.Expect.Files, repo)...)
}

func compare(c *caseFile, r *result, repo string) []string {
	var p []string
	e := c.Expect
	if e.Status != "" && e.Status != r.Status {
		p = append(p, fmt.Sprintf("status = %s, want %s (findings: %v)", r.Status, e.Status, r.Findings))
	}
	for _, want := range e.Findings {
		found := false
		for _, f := range r.Findings {
			if f.Rule == want.Rule && (want.Level == "" || levelOf(f.Level, r.Status) == want.Level) && strings.Contains(f.Where, want.Where) &&
				strings.Contains(f.Message, want.Message) && strings.Contains(f.Fix, want.Fix) {
				found = true
			}
		}
		if !found {
			p = append(p, fmt.Sprintf("missing finding %+v (got %+v)", want, r.Findings))
		}
	}
	for _, want := range e.Checks {
		found := false
		for _, ch := range r.Checks {
			if ch.Rule == want.Rule && (want.Level == "" || ch.Level == want.Level) && strings.Contains(ch.Where, want.Where) &&
				strings.Contains(ch.Message, want.Message) && strings.Contains(ch.Fix, want.Fix) {
				found = true
			}
		}
		if !found {
			p = append(p, fmt.Sprintf("missing check %+v (got %+v)", want, r.Checks))
		}
	}
	for _, bad := range e.NoFindings {
		for _, f := range r.Findings {
			if f.Rule == bad.Rule && f.Where == bad.Where && (bad.Level == "" || levelOf(f.Level, r.Status) == bad.Level) &&
				strings.Contains(f.Message, bad.Message) && strings.Contains(f.Fix, bad.Fix) {
				p = append(p, fmt.Sprintf("finding %+v should not be there: %s", bad, f.Message))
			}
		}
	}
	if e.AgentCalls != nil && *e.AgentCalls != r.AgentCalls {
		p = append(p, fmt.Sprintf("agent calls = %d, want %d", r.AgentCalls, *e.AgentCalls))
	}
	if e.Calls != nil {
		if len(r.Calls) != len(e.Calls) {
			p = append(p, fmt.Sprintf("calls = %v, want %v", r.Calls, e.Calls))
		} else {
			for i, want := range e.Calls {
				for k, v := range want {
					if fmt.Sprint(r.Calls[i][k]) != v {
						p = append(p, fmt.Sprintf("call %d: %s = %v, want %s", i+1, k, r.Calls[i][k], v))
					}
				}
			}
		}
	}
	if e.Applied != nil && fmt.Sprint(e.Applied) != fmt.Sprint(r.Applied) {
		p = append(p, fmt.Sprintf("applied = %v, want %v", r.Applied, e.Applied))
	}
	if e.Steps != nil {
		var got []string
		for _, s := range r.Steps {
			got = append(got, s.Name)
		}
		if fmt.Sprint(got) != fmt.Sprint(e.Steps) {
			p = append(p, fmt.Sprintf("steps = %v, want %v", got, e.Steps))
		}
	}
	if e.Refused != nil && fmt.Sprint(e.Refused) != fmt.Sprint(r.Refused) {
		p = append(p, fmt.Sprintf("refused = %v, want %v", r.Refused, e.Refused))
	}
	for branch, files := range e.Pushed {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		for path, text := range files {
			out, err := exec.Command("git", "-C", repo, "show", "origin/"+branch+":"+path).Output()
			if err != nil || !strings.Contains(string(out), text) {
				p = append(p, fmt.Sprintf("origin's %s does not hold %q in %s", branch, text, path))
			}
		}
	}
	for branch, files := range e.Branches {
		for path, text := range files {
			out, err := exec.Command("git", "-C", repo, "show", "refs/heads/"+branch+":"+path).Output()
			if err != nil || !strings.Contains(string(out), text) {
				p = append(p, fmt.Sprintf("the local branch %s does not hold %q in %s", branch, text, path))
			}
		}
	}
	for branch, text := range e.PushedMsg {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%B", "origin/"+branch).Output()
		if err != nil || !strings.Contains(string(out), text) {
			p = append(p, fmt.Sprintf("the tip of origin's %s does not say %q in its message: %s", branch, text, out))
		}
	}
	for branch, base := range e.OnTop {
		exec.Command("git", "-C", repo, "fetch", "-q", "origin").Run()
		if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", "origin/"+base, "origin/"+branch).Run() != nil {
			p = append(p, fmt.Sprintf("origin's %s is not on top of origin's %s", branch, base))
		}
	}
	if e.RefusedKept > 0 {
		kept, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", "out", "refused-*.yaml"))
		if len(kept) != e.RefusedKept {
			p = append(p, fmt.Sprintf("%d refused answers kept, want %d", len(kept), e.RefusedKept))
		}
	}
	if e.CallsKept > 0 {
		n := 0
		files, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", "out", "calls.jsonl"))
		for _, f := range files {
			data, _ := os.ReadFile(f)
			n += strings.Count(string(data), "\n")
		}
		if n != e.CallsKept {
			p = append(p, fmt.Sprintf("%d agent calls kept, want %d", n, e.CallsKept))
		}
	}
	for name, want := range e.RunFiles {
		found, _ := filepath.Glob(filepath.Join(repo, ".git", "workline", "runs", "*", filepath.FromSlash(name)))
		if len(found) == 0 {
			p = append(p, fmt.Sprintf("no run folder keeps %s", name))
			continue
		}
		for _, text := range want.Contains {
			held := false
			for _, f := range found {
				data, _ := os.ReadFile(f)
				held = held || strings.Contains(string(data), text)
			}
			if !held {
				p = append(p, fmt.Sprintf("%s does not hold %q", name, text))
			}
		}
		for _, text := range want.NotContains {
			for _, f := range found {
				if data, _ := os.ReadFile(f); strings.Contains(string(data), text) {
					p = append(p, fmt.Sprintf("%s holds %q", name, text))
				}
			}
		}
	}
	if e.Summary != "" && !strings.Contains(r.Summary, e.Summary) {
		p = append(p, fmt.Sprintf("summary %q does not hold %q", r.Summary, e.Summary))
	}
	p = append(p, mapped("coverage", e.Coverage, r.Coverage.Items)...)
	p = append(p, mapped("not-covered", e.NotCovered, r.Coverage.NotCovered)...)
	for _, text := range e.Notes {
		if !strings.Contains(strings.Join(r.Notes, "\n"), text) {
			p = append(p, fmt.Sprintf("no note holds %q (notes: %q)", text, r.Notes))
		}
	}
	return append(p, compareFiles(e.Files, repo)...)
}

// compareFiles says how the files the case names differ from what it
// wants of them, each path from the repository.
func compareFiles(files map[string]fileWant, repo string) []string {
	var p []string
	for path, want := range files {
		data, err := os.ReadFile(filepath.Join(repo, path))
		if want.Exists != nil && *want.Exists != (err == nil) {
			p = append(p, fmt.Sprintf("%s exists = %v, want %v", path, err == nil, *want.Exists))
		}
		for _, text := range want.Contains {
			if !strings.Contains(string(data), text) {
				p = append(p, fmt.Sprintf("%s does not contain %q", path, text))
			}
		}
		for _, text := range want.Lacks {
			if strings.Contains(string(data), text) {
				p = append(p, fmt.Sprintf("%s contains %q", path, text))
			}
		}
	}
	return p
}

// linkRoles makes dir a folder of roles: a link to each role of each source.
func linkRoles(dir string, sources ...string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, src := range sources {
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			abs, _ := filepath.Abs(filepath.Join(src, e.Name()))
			if err := os.Symlink(abs, filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// build creates a fixture repository in dir and runs the case's setup in it.
func build(dir, fixture string, setup, env []string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	script, _ := filepath.Abs(filepath.Join("fixtures", "repos", fixture+".sh"))
	steps := append([]string{"sh " + script}, setup...)
	for _, s := range steps {
		cmd := exec.Command("sh", "-c", s)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fixture %s: %q failed: %v\n%s", fixture, s, err, out)
		}
	}
	return nil
}

// lookup is the value a variable has in env, the last one set winning.
func lookup(env []string, name string) string {
	v := ""
	for _, kv := range env {
		if k, val, ok := strings.Cut(kv, "="); ok && k == name {
			v = val
		}
	}
	return v
}

// hermeticEnv keeps the machine's git config and hooks out of the tests,
// and the CI the suite may run in: a case sets CI's variables itself.
func hermeticEnv() []string {
	fixtures, _ := filepath.Abs("fixtures")
	var env []string
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); name != "CI" && name != "GITHUB_ACTIONS" && name != "GITLAB_CI" {
			env = append(env, kv)
		}
	}
	return append(env, "FIXTURES="+fixtures, // for a cmd: agent's script
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-03T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-03T00:00:00Z",
	)
}

func readPending(t *testing.T) map[string]bool {
	out := map[string]bool{}
	f, err := os.Open("pending.txt")
	if os.IsNotExist(err) {
		return out
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			out[line] = true
		}
	}
	return out
}

// pushedBeforeApplied is a push landing before a judged run is applied
// (#375): `pushed`, the same run judged again on the new head, both
// applied, the first first; `judged-twice` judges both before applying,
// `applied-between` applies the first before the second judges. With
// apply-from, each --head is the head its run judged. The result is the
// last apply's; the calls, both judges'; the findings, all four's
// (docs/spec/conformance/cases.md).
func pushedBeforeApplied(c *caseFile, engineBin string, args, env []string, repo, forgeSpec string, first result) (result, []string) {
	exe := func(dir string, cmdArgs ...string) (result, []string) {
		cmd := exec.Command(engineBin, cmdArgs...)
		cmd.Dir, cmd.Env = dir, env
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		var r result
		if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
			return r, []string{fmt.Sprintf("workline %s printed no result: %v\n%s", cmdArgs[0], err, stderr.String())}
		}
		return r, nil
	}
	git := func(gitArgs ...string) (string, []string) {
		out, err := exec.Command("git", append([]string{"-C", repo}, gitArgs...)...).CombinedOutput()
		if err != nil {
			return "", []string{fmt.Sprintf("git %s: %v: %s", strings.Join(gitArgs, " "), err, out)}
		}
		return strings.TrimSpace(string(out)), nil
	}
	apply := func(r result, head string) (result, []string) {
		dirs := r.Pending
		if len(dirs) == 0 && r.RunDir != "" {
			dirs = []string{r.RunDir}
		}
		if len(dirs) == 0 {
			return r, []string{"a judging run reported no run folder to apply"}
		}
		applyArgs := append(append([]string{"apply"}, dirs...), "--json")
		if a := c.Run.ApplyFrom; a != nil {
			if a.Base != "" {
				if _, p := git("checkout", "-q", "--detach", a.Base); p != nil {
					return r, p
				}
			}
			applyArgs = append(applyArgs, "--from-merge-request", fmt.Sprint(a.MergeRequest), "--head", head, "--forge", forgeSpec)
		}
		return exe(repo, applyArgs...)
	}
	heads := make([]string, 2)
	var p []string
	if heads[0], p = git("rev-parse", "HEAD"); p != nil {
		return first, p
	}
	for _, s := range c.Run.Pushed {
		cmd := exec.Command("sh", "-c", s)
		cmd.Dir, cmd.Env = repo, env
		if out, err := cmd.CombinedOutput(); err != nil {
			return first, []string{fmt.Sprintf("pushed: %q failed: %v\n%s", s, err, out)}
		}
	}
	if heads[1], p = git("rev-parse", "HEAD"); p != nil {
		return first, p
	}
	judge := func() (result, []string) {
		if _, p := git("checkout", "-q", "--detach", heads[1]); p != nil {
			return result{}, p
		}
		return exe(repo, args...)
	}
	var second, applied1, applied2 result
	if c.Run.Then == "judged-twice" {
		if second, p = judge(); p != nil {
			return first, p
		}
		if applied1, p = apply(first, heads[0]); p != nil {
			return first, p
		}
	} else {
		if applied1, p = apply(first, heads[0]); p != nil {
			return first, p
		}
		if second, p = judge(); p != nil {
			return first, p
		}
	}
	if applied2, p = apply(second, heads[1]); p != nil {
		return first, p
	}
	out := applied2
	out.AgentCalls = first.AgentCalls + second.AgentCalls
	out.Calls = append(slices.Clone(first.Calls), second.Calls...)
	out.Findings = slices.Concat(first.Findings, applied1.Findings, second.Findings, applied2.Findings)
	return out, nil
}

// editForge makes a person's edits to the simulated forge between two runs
// (between): each item listed, by kind and id, takes the fields given.
// poisoned changes the runs' folders as a merge request's own run could
// have (run.yaml's keys, proposals added), checks the base out, and gives
// the runs as listed and the options `apply --from-merge-request` takes.
func poisoned(a *applyFrom, dirs []string, repo, forgeSpec string) (listed, args []string, problems []string) {
	for _, d := range dirs {
		if len(a.Run) > 0 {
			file := filepath.Join(d, "out", "run.yaml")
			var st map[string]any
			data, err := os.ReadFile(file)
			if err == nil {
				err = yaml.Unmarshal(data, &st)
			}
			if err != nil {
				return nil, nil, []string{"apply-from: " + err.Error()}
			}
			maps.Copy(st, a.Run)
			data, _ = yaml.Marshal(st)
			os.WriteFile(file, data, 0o644)
		}
		if len(a.Intents) > 0 {
			file := filepath.Join(d, "out", "intentions.yaml")
			var in []map[string]any
			if data, err := os.ReadFile(file); err == nil {
				yaml.Unmarshal(data, &in)
			}
			data, _ := yaml.Marshal(append(in, a.Intents...))
			os.WriteFile(file, data, 0o644)
		}
	}
	listed = dirs
	if a.Listed != "" {
		listed = []string{a.Listed}
	}
	for _, s := range a.Setup {
		cmd := exec.Command("sh", "-c", s)
		cmd.Dir, cmd.Env = repo, hermeticEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, nil, []string{fmt.Sprintf("apply-from: %q failed: %v\n%s", s, err, out)}
		}
	}
	if a.Base != "" {
		if out, err := exec.Command("git", "-C", repo, "checkout", "-q", "--detach", a.Base).CombinedOutput(); err != nil {
			return nil, nil, []string{fmt.Sprintf("apply-from: %s not checked out: %s", a.Base, out)}
		}
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "origin/"+a.Head).Output()
	if err != nil {
		return nil, nil, []string{fmt.Sprintf("apply-from: no branch %s on origin", a.Head)}
	}
	return listed, []string{"--from-merge-request", fmt.Sprint(a.MergeRequest), "--head", strings.TrimSpace(string(head)), "--forge", forgeSpec}, nil
}

func editForge(file string, edits map[string]any) error {
	if len(edits) == 0 {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("between: no forge to edit: %v", err)
	}
	var forge map[string]any
	if err := json.Unmarshal(data, &forge); err != nil {
		return fmt.Errorf("between: the forge does not read: %v", err)
	}
	for kind, items := range edits {
		list, _ := items.([]any)
		have, _ := forge[kind].([]any)
		for _, e := range list {
			edit, _ := e.(map[string]any)
			found := false
			for _, h := range have {
				if item, _ := h.(map[string]any); item != nil && fmt.Sprint(item["id"]) == fmt.Sprint(edit["id"]) {
					for k, v := range edit {
						item[k] = v
					}
					found = true
				}
			}
			if !found {
				return fmt.Errorf("between: no %s with id %v", kind, edit["id"])
			}
		}
	}
	if data, err = json.Marshal(forge); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

// compareStatuses checks the statuses the simulated forge holds: the last
// of each name on each commit, the commits revisions of the repository.
func compareStatuses(want []statusWant, file, repo string) []string {
	if len(want) == 0 {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{err.Error()}
	}
	var s forge.FakeState
	if err := json.Unmarshal(data, &s); err != nil {
		return []string{err.Error()}
	}
	commit := func(rev string) string {
		out, _ := exec.Command("git", "-C", repo, "rev-parse", "--verify", "-q", rev+"^{commit}").Output()
		return strings.TrimSpace(string(out))
	}
	var p []string
	for _, w := range want {
		var last *forge.FakeStatus
		for i := range s.Statuses {
			if st := &s.Statuses[i]; st.Name == w.Name && commit(st.Commit) == commit(w.Commit) {
				last = st
			}
		}
		switch {
		case w.Absent && last != nil:
			p = append(p, fmt.Sprintf("forge: status %s on %s is there (%+v), want none", w.Name, w.Commit, *last))
		case w.Absent:
		case last == nil:
			p = append(p, fmt.Sprintf("forge: no status %s on %s", w.Name, w.Commit))
		case w.State != "" && last.State != w.State, w.By != "" && last.By != w.By, !strings.Contains(last.Description, w.Description):
			p = append(p, fmt.Sprintf("forge: status %s on %s is %+v, want %+v", w.Name, w.Commit, *last, w))
		}
	}
	return p
}

// compareCheckRuns checks the checks the simulated forge holds: the last of
// each name on each commit, the commits revisions of the repository.
func compareCheckRuns(want []checkRunWant, file, repo string) []string {
	if len(want) == 0 {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{err.Error()}
	}
	var s forge.FakeState
	if err := json.Unmarshal(data, &s); err != nil {
		return []string{err.Error()}
	}
	commit := func(rev string) string {
		out, _ := exec.Command("git", "-C", repo, "rev-parse", "--verify", "-q", rev+"^{commit}").Output()
		return strings.TrimSpace(string(out))
	}
	var p []string
	for _, w := range want {
		var last *forge.FakeCheck
		for i := range s.Checks {
			if ch := &s.Checks[i]; ch.Name == w.Name && commit(ch.Commit) == commit(w.Commit) {
				last = ch
			}
		}
		switch {
		case w.Absent && last != nil:
			p = append(p, fmt.Sprintf("forge: check %s on %s is there (%+v), want none", w.Name, w.Commit, *last))
			continue
		case w.Absent:
			continue
		case last == nil:
			p = append(p, fmt.Sprintf("forge: no check %s on %s (checks: %+v)", w.Name, w.Commit, s.Checks))
			continue
		}
		for _, a := range w.Annotations {
			found := false
			for _, g := range last.Annotations {
				found = found || g.Path == a.Path && (a.Line == 0 || g.Line == a.Line) && (a.Level == "" || g.Level == a.Level) && strings.Contains(g.Message, a.Contains)
			}
			if !found {
				p = append(p, fmt.Sprintf("forge: check %s on %s has no annotation %+v (got %+v)", w.Name, w.Commit, a, last.Annotations))
			}
		}
	}
	return p
}

// compareForge checks the simulated forge's state: each item listed, by
// id, holds what itemWant says of it; `absent: true`, no item with that id.
func compareForge(want *forgeWant, file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{"no forge state to check: " + err.Error()}
	}
	var got map[string][]map[string]any
	json.Unmarshal(data, &got)
	var p []string
	for _, kind := range []struct {
		name  string
		items []itemWant
	}{{"issues", want.Issues}, {"merge-requests", want.MergeRequests}, {"labels", want.Labels}} {
		for _, wm := range kind.items {
			p = append(p, compareItem(kind.name, wm, got[kind.name])...)
		}
	}
	return p
}

// compareItem checks one item of the forge against what is wanted of it.
func compareItem(kind string, wm itemWant, got []map[string]any) []string {
	var p []string
	var found map[string]any
	for _, g := range got {
		if fmt.Sprint(g["id"]) == fmt.Sprint(wm.ID) {
			found = g
		}
	}
	if wm.Absent {
		if found != nil {
			p = append(p, fmt.Sprintf("forge: %s %v exists, want none", kind, wm.ID))
		}
		return p
	}
	if found == nil {
		return append(p, fmt.Sprintf("forge: no %s with id %v", kind, wm.ID))
	}
	body := fmt.Sprint(found["body"])
	for _, t := range wm.BodyContains {
		if !strings.Contains(body, t) {
			p = append(p, fmt.Sprintf("forge: %s %v body = %q, want it to hold %q", kind, wm.ID, found["body"], t))
		}
	}
	for _, t := range wm.BodyLacks {
		if strings.Contains(body, t) {
			p = append(p, fmt.Sprintf("forge: %s %v body = %q, want it without %q", kind, wm.ID, found["body"], t))
		}
	}
	if wm.Closed != nil && (found["closed"] == true) != *wm.Closed {
		p = append(p, fmt.Sprintf("forge: %s %v closed = %v, want %v", kind, wm.ID, found["closed"] == true, *wm.Closed))
	}
	if wm.AutoMerge != nil && (found["auto-merge"] == true) != *wm.AutoMerge {
		p = append(p, fmt.Sprintf("forge: %s %v auto-merge = %v, want %v", kind, wm.ID, found["auto-merge"] == true, *wm.AutoMerge))
	}
	if wm.Merged != nil && (found["merged"] == true) != *wm.Merged {
		p = append(p, fmt.Sprintf("forge: %s %v merged = %v, want %v", kind, wm.ID, found["merged"] == true, *wm.Merged))
	}
	if found["parent"] == nil {
		found["parent"] = 0 // not a sub-issue
	}
	fields := []struct {
		key  string
		want any
	}{{"branch", given(wm.Branch)}, {"base", given(wm.Base)}, {"title", given(wm.Title)},
		{"reason", given(wm.Reason)}, {"milestone", given(wm.Milestone)}, {"parent", given(wm.Parent)}}
	for _, f := range fields {
		if f.want != nil && fmt.Sprint(found[f.key]) != fmt.Sprint(f.want) {
			p = append(p, fmt.Sprintf("forge: %s %v %s = %v, want %v", kind, wm.ID, f.key, found[f.key], f.want))
		}
	}
	comments, _ := found["comments"].([]any)
	if wm.Comments != nil && len(comments) != *wm.Comments {
		p = append(p, fmt.Sprintf("forge: %s %v has %d comments, want %d", kind, wm.ID, len(comments), *wm.Comments))
	}
	if posted, _ := found["posted"].(float64); wm.Posted != nil && int(posted) != *wm.Posted {
		p = append(p, fmt.Sprintf("forge: %s %v posted %d comments, want %d", kind, wm.ID, int(posted), *wm.Posted))
	}
	held := func(text string) bool {
		for _, c := range comments {
			if strings.Contains(fmt.Sprint(c), text) {
				return true
			}
		}
		return false
	}
	for _, t := range wm.CommentContains {
		if !held(t) {
			p = append(p, fmt.Sprintf("forge: no comment on %s %v holds %q (comments: %v)", kind, wm.ID, t, comments))
		}
	}
	for _, t := range wm.CommentLacks {
		if held(t) {
			p = append(p, fmt.Sprintf("forge: a comment on %s %v holds %q", kind, wm.ID, t))
		}
	}
	if wm.BlockedBy != nil {
		gb, _ := found["blocked-by"].([]any)
		if fmt.Sprint(gb) != fmt.Sprint(*wm.BlockedBy) {
			p = append(p, fmt.Sprintf("forge: %s %v blocked-by = %v, want %v", kind, wm.ID, gb, *wm.BlockedBy))
		}
	}
	if wm.Labels != nil {
		gl, _ := found["labels"].([]any)
		if fmt.Sprint(gl) != fmt.Sprint(*wm.Labels) {
			p = append(p, fmt.Sprintf("forge: %s %v labels = %v, want %v", kind, wm.ID, gl, *wm.Labels))
		}
	}
	if wm.Reactions != nil {
		gr, _ := found["reactions"].(map[string]any)
		if gr == nil {
			gr = map[string]any{}
		}
		if fmt.Sprint(gr) != fmt.Sprint(wm.Reactions) {
			p = append(p, fmt.Sprintf("forge: %s %v reactions = %v, want %v", kind, wm.ID, gr, wm.Reactions))
		}
	}
	if wm.Threads != nil {
		p = append(p, compareThreads(fmt.Sprintf("%s %v", kind, wm.ID), *wm.Threads, found["threads"])...)
	}
	return p
}

// compareThreads checks a merge request's comments on lines, in the order
// posted: exactly as many as wanted, each on its path and line, holding
// what it should and not what it should not, resolved or not when said.
func compareThreads(item string, want []threadWant, got any) []string {
	have, _ := got.([]any)
	if len(have) != len(want) {
		return []string{fmt.Sprintf("forge: %s has %d threads, want %d (threads: %v)", item, len(have), len(want), have)}
	}
	var p []string
	for i, w := range want {
		g, _ := have[i].(map[string]any)
		if w.Path != "" && fmt.Sprint(g["path"]) != w.Path {
			p = append(p, fmt.Sprintf("forge: %s thread %d path = %v, want %s", item, i+1, g["path"], w.Path))
		}
		if w.Line != 0 && fmt.Sprint(g["line"]) != fmt.Sprint(w.Line) {
			p = append(p, fmt.Sprintf("forge: %s thread %d line = %v, want %d", item, i+1, g["line"], w.Line))
		}
		if w.Resolved != nil && (g["resolved"] == true) != *w.Resolved {
			p = append(p, fmt.Sprintf("forge: %s thread %d resolved = %v, want %v", item, i+1, g["resolved"] == true, *w.Resolved))
		}
		body := fmt.Sprint(g["body"])
		for _, t := range w.Contains {
			if !strings.Contains(body, t) {
				p = append(p, fmt.Sprintf("forge: %s thread %d does not hold %q: %q", item, i+1, t, body))
			}
		}
		for _, t := range w.Lacks {
			if strings.Contains(body, t) {
				p = append(p, fmt.Sprintf("forge: %s thread %d holds %q: %q", item, i+1, t, body))
			}
		}
	}
	return p
}

// given is a field's value when the case gives it, nil when it does not.
func given[T any](v *T) any {
	if v == nil {
		return nil
	}
	return *v
}

// levelOf is a finding's level as docs/spec/role-outcome.md reads it: one
// that says none blocks in a run that did not pass.
func levelOf(level, status string) string {
	if level == "" && status != "pass" {
		return "block"
	}
	return level
}

// compareSummaryFile checks the --summary file holds each text, in order.
func compareSummaryFile(want []string, file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return []string{fmt.Sprintf("no summary file: %v", err)}
	}
	rest := string(data)
	for _, w := range want {
		i := strings.Index(rest, w)
		if i < 0 {
			return []string{fmt.Sprintf("the summary file does not hold %q after what came before:\n%s", w, data)}
		}
		rest = rest[i+len(w):]
	}
	return nil
}

// compareReports checks the SARIF and Code Quality files a run wrote.
func compareReports(c *caseFile, sarifFile, cqFile string) []string {
	var p []string
	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				PartialFingerprints map[string]string `json:"partialFingerprints"`
			} `json:"results"`
		} `json:"runs"`
	}
	var cq []struct {
		CheckName   string `json:"check_name"`
		Severity    string `json:"severity"`
		Fingerprint string `json:"fingerprint"`
		Location    struct {
			Path  string `json:"path"`
			Lines struct {
				Begin int `json:"begin"`
			} `json:"lines"`
		} `json:"location"`
	}
	for file, into := range map[string]any{sarifFile: &sarif, cqFile: &cq} {
		data, err := os.ReadFile(file)
		if err == nil {
			err = json.Unmarshal(data, into)
		}
		if err != nil {
			return []string{fmt.Sprintf("%s: %v", filepath.Base(file), err)}
		}
	}
	if sarif.Version != "2.1.0" || len(sarif.Runs) != 1 {
		p = append(p, fmt.Sprintf("SARIF version %q with %d runs, want 2.1.0 with one run", sarif.Version, len(sarif.Runs)))
		return p
	}
	var got, gotCQ []map[string]any
	uris := map[string]bool{}
	for _, r := range sarif.Runs[0].Results {
		if len(r.Locations) != 1 || len(r.PartialFingerprints) == 0 {
			p = append(p, fmt.Sprintf("SARIF result %s has %d locations and %d fingerprints, want one and some", r.RuleID, len(r.Locations), len(r.PartialFingerprints)))
			continue
		}
		l := r.Locations[0].PhysicalLocation
		uris[l.ArtifactLocation.URI] = true
		got = append(got, map[string]any{"rule": r.RuleID, "uri": l.ArtifactLocation.URI, "line": l.Region.StartLine, "level": r.Level})
	}
	for _, i := range cq {
		if i.Fingerprint == "" {
			p = append(p, "a Code Quality issue has no fingerprint")
		}
		uris[i.Location.Path] = true
		gotCQ = append(gotCQ, map[string]any{"check_name": i.CheckName, "path": i.Location.Path, "line": i.Location.Lines.Begin, "severity": i.Severity})
	}
	has := func(list []map[string]any, want map[string]any) bool {
		for _, g := range list {
			ok := true
			for k, v := range want {
				ok = ok && fmt.Sprint(g[k]) == fmt.Sprint(v)
			}
			if ok {
				return true
			}
		}
		return false
	}
	for _, w := range c.Expect.SARIF {
		if !has(got, w) {
			p = append(p, fmt.Sprintf("SARIF lacks %v (got %v)", w, got))
		}
	}
	for _, w := range c.Expect.CodeQuality {
		if !has(gotCQ, w) {
			p = append(p, fmt.Sprintf("Code Quality lacks %v (got %v)", w, gotCQ))
		}
	}
	for _, rule := range c.Expect.SARIFLacks {
		for _, g := range got {
			if g["rule"] == rule {
				p = append(p, fmt.Sprintf("SARIF holds %v, and should not", g))
			}
		}
	}
	for _, w := range c.Expect.LeftOut {
		if uris[w] {
			p = append(p, fmt.Sprintf("%s is in a report, and should be left out", w))
		}
	}
	return p
}

// mapped says which wanted entries of an import's map no entry matches:
// each key equal, but words and why, which an entry need only hold.
func mapped(name string, want []map[string]string, got []map[string]any) []string {
	var p []string
	for _, w := range want {
		found := false
		for _, g := range got {
			ok := true
			for k, v := range w {
				have := fmt.Sprint(g[k])
				if k == "words" || k == "why" {
					ok = ok && strings.Contains(have, v)
				} else {
					ok = ok && have == v
				}
			}
			found = found || ok
		}
		if !found {
			p = append(p, fmt.Sprintf("%s: no entry %v (got %v)", name, w, got))
		}
	}
	return p
}

// texts is a text, or a list of them: what a run file holds, or does not.
type texts []string

func (t *texts) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		*t = texts{n.Value}
		return nil
	}
	var l []string
	if err := n.Decode(&l); err != nil {
		return err
	}
	*t = l
	return nil
}
