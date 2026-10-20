// Package inspector is the inspector's built-in steps (roles/inspector):
// on a merge request, the findings the project's gates report the change
// adds, triaged by the agent — each explained in the project's words, a
// likely false positive said with why, the fixes put in order — and never
// their verdict: a finding that blocks still blocks (ADR-0049, step 4).
// It reads apart from the verdict, which never waits for it (ADR-0050).
// A finding triaged once is not asked again, by its fingerprint
// (ADR-0013). With no agent, nothing is triaged, and that is said.
package inspector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/gitrange"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
)

const (
	exitNothing = 10 // no question for the agent: the verdict pre wrote is final

	// part is the one part the agent is asked in: the findings to triage.
	part = "triage"

	// warnApart, warnOnly: the input `warn` of a line split from the run
	// apart (ADR-0050): the verdict's run leaves the triage, the run apart
	// is where it reads.
	warnApart = "apart"
	warnOnly  = "only"

	// textMax is the most of one text of the agent's shown or kept.
	textMax = 400
)

// Settings are the role's, as role.yaml and the project set them.
type Settings struct {
	FindingsMax int  `json:"findings-max"` // new findings asked a run; the rest counted, asked at the next
	LinesAround int  `json:"lines-around"` // lines of the head's code shown above and below each finding
	ForgeWrites bool `json:"forge-writes"` // on a merge request: the comment, the record in it
}

// Found is one finding the gates report the change adds, as the inspector
// reads it from their result: placed, named by its fingerprint, which
// holds no line (ADR-0049, step 2).
type Found struct {
	ID          string `json:"id"` // t1 to triage now, r1 triaged before: what the task names it by
	Gate        string `json:"gate"`
	Check       string `json:"check"`
	Tool        string `json:"tool,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Level       string `json:"level,omitempty"`
	Path        string `json:"path,omitempty"`
	Line        int    `json:"line,omitempty"`
	Message     string `json:"message,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Where is the finding's place, `path:line`.
func (f Found) Where() string {
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.Path, f.Line)
	}
	return f.Path
}

// name is the rule a finding is shown by, its tool beside it.
func (f Found) name() string {
	n := f.Rule
	if n == "" {
		n = f.Check
	}
	if f.Tool != "" && f.Tool != n {
		return fmt.Sprintf("%s (%s)", n, f.Tool)
	}
	return n
}

// Run is what the inspector settles, kept between pre's passes and read
// by post: the findings at the head, the record, what to say.
type Run struct {
	Head     string            `json:"head"`
	Found    []Found           `json:"found"`           // every new finding of the gates, triaged or not
	Asked    []string          `json:"asked,omitempty"` // the fingerprints put to the agent this run
	Record   Record            `json:"record"`
	Local    bool              `json:"local"` // the record is kept on this machine
	Status   string            `json:"status"`
	Summary  string            `json:"summary"`
	Findings []verdict.Finding `json:"findings,omitempty"`
	Said     []string          `json:"said,omitempty"` // lines the comment says beside the triage
}

// Pre prepares the triage: the gates' new findings read, those triaged
// before taken from the record, the others put to the agent as one part;
// then, its answer read, the comment and the verdict.
func Pre(runDir, repo string) int {
	s, err := settings(runDir)
	if err != nil {
		return fail(err)
	}
	if os.Getenv("WORKLINE_PARTS") == "answered" {
		return answered(runDir, repo, s)
	}
	return prepare(runDir, repo, s)
}

func prepare(runDir, repo string, s Settings) int {
	switch side := input(runDir, "warn"); side {
	case "", warnOnly:
	case warnApart:
		return final(runDir, verdict.Pass, "the triage reads apart from the verdict, which never waits for it (ADR-0050): nothing asked in this run", nil)
	default:
		return fail(fmt.Errorf("input warn %q: %s, the verdict's run, or %s, the run apart", side, warnApart, warnOnly))
	}
	// A draft asks no agent (ADR-0050): the templates give it no gates'
	// result to wait for either.
	if os.Getenv("WORKLINE_AI_WAITS") == "ready" {
		return final(runDir, verdict.Pass, "a draft: the triage waits for it to be marked ready", []verdict.Finding{{Rule: "not-triaged", Level: "warn",
			Message: "the gates' new findings not triaged: a draft, the triage waits for it to be marked ready; they stand as the gates report them"}})
	}
	file := input(runDir, "gates")
	if file == "" {
		return final(runDir, verdict.Pass, "no gate's result reached the inspector: nothing to triage", []verdict.Finding{{Rule: "gates-not-given", Level: "warn",
			Message: "no gate ran before the inspector in the line, nor was the gates' result given (--gates-from): nothing to triage; list the inspector after the gates in `routing:`"}})
	}
	found, said, err := readGates(file)
	if err != nil {
		return final(runDir, verdict.Pass, "the gates' result could not be read: nothing triaged", []verdict.Finding{{Rule: "gates-unread", Level: "warn",
			Message: fmt.Sprintf("the gates' result, %s, could not be read (%v): their findings stand as the gates report them, none triaged", file, err)}})
	}
	head, err := headOf(repo, input(runDir, "range"))
	if err != nil {
		return fail(err)
	}
	run := Run{Head: head, Found: found, Said: said, Status: verdict.Pass}
	rec, where, local, err := loadRecord(repo, s)
	run.Local = local
	if err != nil {
		// Never read as none in silence: every finding is asked again, and
		// the comment written holds the record anew.
		run.Findings = append(run.Findings, verdict.Finding{Rule: "record-unread", Level: "warn",
			Message: "the record of what was triaged before, " + where + ", could not be read (" + err.Error() + "): every new finding is triaged again"})
		rec = Record{}
	}
	run.Record = rec
	var fresh []Found
	t, r := 0, 0
	for i, f := range run.Found {
		if _, ok := rec.find(f.Fingerprint); ok {
			r++
			run.Found[i].ID = "r" + strconv.Itoa(r)
			continue
		}
		t++
		run.Found[i].ID = "t" + strconv.Itoa(t)
		fresh = append(fresh, run.Found[i])
	}
	switch {
	case len(run.Found) == 0:
		run.Summary = "no new finding of the gates: nobody asked"
		return settled(runDir, s, run)
	case len(fresh) == 0:
		run.Summary = fmt.Sprintf("%d new %s of the gates, each triaged before: nobody asked", len(run.Found), plural(len(run.Found), "finding", "findings"))
		return settled(runDir, s, run)
	}
	if why := noAgent(); why != "" {
		run.Findings = append(run.Findings, verdict.Finding{Rule: "not-triaged", Level: "warn",
			Message: fmt.Sprintf("%d new %s of the gates not triaged: %s; %s as the gates report %s", len(fresh), plural(len(fresh), "finding", "findings"), why,
				plural(len(fresh), "it stands", "they stand"), plural(len(fresh), "it", "them"))})
		run.Said = append(run.Said, fmt.Sprintf("Not triaged: %d, %s.", len(fresh), why))
		run.Summary = fmt.Sprintf("%d new %s of the gates, %d not triaged: %s", len(run.Found), plural(len(run.Found), "finding", "findings"), len(fresh), why)
		return settled(runDir, s, run)
	}
	asked := fresh
	if s.FindingsMax > 0 && len(asked) > s.FindingsMax {
		left := len(asked) - s.FindingsMax
		asked = asked[:s.FindingsMax]
		run.Findings = append(run.Findings, verdict.Finding{Rule: "findings-max", Level: "warn",
			Message: fmt.Sprintf("%d more new %s of the gates not triaged this run (findings-max %d): asked at the next push", left, plural(left, "finding", "findings"), s.FindingsMax)})
		run.Said = append(run.Said, fmt.Sprintf("Not triaged yet: %d, past `findings-max` (%d), asked at the next push.", left, s.FindingsMax))
	}
	for _, f := range asked {
		run.Asked = append(run.Asked, f.Fingerprint)
	}
	dir := filepath.Join(runDir, "in", "parts", part)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task.md"), []byte(task(repo, s, run, asked)), 0o644); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "inspector-state.json"), run); err != nil {
		return fail(err)
	}
	return 0
}

// noAgent says why no agent triages, "" when one does.
func noAgent() string {
	if os.Getenv("WORKLINE_AI") == "" || os.Getenv("WORKLINE_AI") == "none" {
		return "no agent (--ai none)"
	}
	return ""
}

// answered reads the agent's triage, keeps each one that reads in the
// record, and settles the comment and the verdict.
func answered(runDir, repo string, s Settings) int {
	var run Run
	if err := readJSON(filepath.Join(runDir, "in", "inspector-state.json"), &run); err != nil {
		return fail(err)
	}
	dir := filepath.Join(runDir, "in", "parts", part)
	notRead := func(why string) int {
		n := len(run.Asked)
		run.Findings = append(run.Findings, verdict.Finding{Rule: "triage-failed", Level: "warn",
			Message: fmt.Sprintf("the agent's triage %s: the %d new %s asked %s as the gates report %s, asked again at the next push", why, n, plural(n, "finding", "findings"),
				plural(n, "stands", "stand"), plural(n, "it", "them"))})
		run.Said = append(run.Said, fmt.Sprintf("Not triaged: %d, the agent's answer %s.", n, why))
		run.Summary = fmt.Sprintf("%d new %s of the gates, %d not triaged: the agent's answer %s", len(run.Found), plural(len(run.Found), "finding", "findings"), n, why)
		return settled(runDir, s, run)
	}
	if why, err := os.ReadFile(filepath.Join(dir, "unanswered")); err == nil {
		return notRead("did not read (" + strings.TrimSpace(string(why)) + ")")
	}
	if !exists(filepath.Join(dir, "answer.yaml")) {
		return notRead("was not asked")
	}
	answers, err := intent.Read(filepath.Join(dir, "answer.yaml"))
	if err != nil {
		return notRead(fmt.Sprintf("does not read (%v)", err))
	}
	byID := map[string]Found{}
	for _, f := range run.Found {
		if slices.Contains(run.Asked, f.Fingerprint) || strings.HasPrefix(f.ID, "r") {
			byID[f.ID] = f
		}
	}
	model := triageModel(runDir)
	order := map[string]int{} // fingerprint -> the place the agent gave it
	kept := map[string]bool{} // the ids whose answer was kept: one more is refused
	refuse := func(id, why string) {
		run.Findings = append(run.Findings, verdict.Finding{Rule: "triage-refused", Level: "warn", Where: id, Message: why})
	}
	for _, a := range answers {
		m, ok := a.Value.(map[string]any)
		if a.Kind != "triage" || !ok {
			refuse("", fmt.Sprintf("a %s, not a triage: not read", a.Kind))
			continue
		}
		id := strings.TrimSpace(fmt.Sprint(m["finding"]))
		f, known := byID[id]
		switch {
		case !known:
			refuse(id, fmt.Sprintf("names %q, no finding the task gave: not read", id))
			continue
		case kept[id]:
			refuse(id, "answered twice: the first kept")
			continue
		}
		place, placed := number(m["order"])
		if strings.HasPrefix(id, "r") { // triaged before: its place only
			if placed && place > 0 {
				order[f.Fingerprint], kept[id] = place, true
			}
			continue
		}
		t := Triaged{Fingerprint: f.Fingerprint, Gate: f.Gate, Check: f.Check, Tool: f.Tool, Rule: f.Rule, Where: f.Where(), At: run.Head, Model: model,
			Matters: text(m["matters"]), Fix: text(m["fix"]), Why: text(m["why"])}
		fp, _ := m["false-positive"].(bool)
		t.FalsePositive = fp
		switch {
		case t.Matters == "":
			refuse(id, "says not why it matters here (`matters`): not kept, asked again at the next push")
			continue
		case t.FalsePositive && t.Why == "":
			refuse(id, "calls it a likely false positive without saying why (`why`): not kept, asked again at the next push")
			continue
		}
		if !t.FalsePositive {
			t.Why = ""
		}
		kept[id] = true
		if placed && place > 0 {
			order[f.Fingerprint] = place
		}
		run.Record = run.Record.with(t)
	}
	left := 0
	for _, fp := range run.Asked {
		if _, ok := run.Record.find(fp); !ok {
			left++
		}
	}
	if left > 0 {
		run.Said = append(run.Said, fmt.Sprintf("Not triaged: %d, the agent's answer giving no triage that reads for %s, asked again at the next push.", left, plural(left, "it", "them")))
	}
	run.Record = run.Record.ordered(run.Found, order)
	return settled(runDir, s, run)
}

// settled writes what the run proposes — the comment, the record in it —
// and what post reads.
func settled(runDir string, s Settings, run Run) int {
	shown := current(run)
	fps := 0
	for _, t := range shown {
		if t.FalsePositive {
			fps++
		}
	}
	if run.Summary == "" {
		n := len(run.Found)
		run.Summary = fmt.Sprintf("%d new %s of the gates, %d triaged", n, plural(n, "finding", "findings"), len(shown))
		if before := len(shown) - countNew(run, shown); before > 0 {
			run.Summary += fmt.Sprintf(" (%d before)", before)
		}
		if left := n - len(shown); left > 0 {
			run.Summary += fmt.Sprintf(", %d not", left)
		}
	}
	if len(shown) > 0 {
		run.Summary += fmt.Sprintf("; %d likely %s", fps, plural(fps, "a false positive", "false positives"))
		run.Summary += fmt.Sprintf("; fix first: %s (%s)", shown[0].Where, shown[0].ruleName())
	}
	run.Summary += "; the gates' verdict stands"
	for i, t := range shown {
		msg := fmt.Sprintf("fix %d of %d, %s: %s", i+1, len(shown), t.ruleName(), sentence(t.Matters))
		if t.FalsePositive {
			msg = fmt.Sprintf("fix %d of %d, %s: likely a false positive: %s %s", i+1, len(shown), t.ruleName(), sentence(t.Why), sentence(t.Matters))
		}
		if t.Fix != "" {
			msg += " Fix: " + sentence(t.Fix)
		}
		run.Findings = append(run.Findings, verdict.Finding{Rule: "triage", Level: "warn", Where: t.Where, Message: msg})
	}
	var fallback []intent.Intention
	if !run.Local {
		body := comment(run, shown)
		// Nothing triaged to show: only a comment an earlier push left is
		// edited, never one opened to say nothing.
		value := map[string]any{"sticky": stickyKey, "body": body}
		if len(shown) == 0 {
			value["update-only"] = true
		}
		fallback = append(fallback, intent.Intention{Kind: "comment", Value: value})
	}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), fallback); err != nil {
		return fail(err)
	}
	if err := writeJSON(filepath.Join(runDir, "in", "inspector.json"), run); err != nil {
		return fail(err)
	}
	return 0
}

// current are the record's triages of the findings at the head, in the
// order to fix them, each at its place there.
func current(run Run) []Triaged {
	var out []Triaged
	for _, f := range run.Found {
		if t, ok := run.Record.find(f.Fingerprint); ok {
			t.Where = f.Where()
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return rank(out[a]) < rank(out[b]) })
	return out
}

// rank is a triage's place; none given, after those given.
func rank(t Triaged) int {
	if t.Order > 0 {
		return t.Order
	}
	return 1 << 30
}

// countNew is how many of the triages shown this run made.
func countNew(run Run, shown []Triaged) int {
	n := 0
	for _, t := range shown {
		if slices.Contains(run.Asked, t.Fingerprint) {
			n++
		}
	}
	return n
}

// Post writes the verdict pre settled, always a pass: the triage never
// holds a merge request, the gates' verdict is the line's. On a machine,
// it keeps the record there.
func Post(runDir, repo string) int {
	var run Run
	if err := readJSON(filepath.Join(runDir, "in", "inspector.json"), &run); err != nil {
		return fail(err)
	}
	if run.Local {
		if err := saveLocal(repo, run.Record); err != nil {
			return fail(err)
		}
	}
	if err := writeJSON(filepath.Join(runDir, "out", "inspector.json"), run); err != nil {
		return fail(err)
	}
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: verdict.Pass, Summary: run.Summary, Findings: run.Findings, Final: true}); err != nil {
		return fail(err)
	}
	return 0
}

// readGates reads the gates' result, a line's as `--gates-only --json`
// prints it: each gate step's new findings, once each by fingerprint. A
// check whose fix is a command, a formatter's, is said, not triaged: its
// fix is that command (gates.md).
func readGates(file string) ([]Found, []string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}
	var r struct {
		Steps []struct {
			Name string      `json:"name"`
			New  *gate.Added `json:"new"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, nil, err
	}
	var out []Found
	var said []string
	seen := map[string]bool{}
	for _, st := range r.Steps {
		name, isGate := strings.CutPrefix(st.Name, "gate:")
		if !isGate || st.New == nil {
			continue
		}
		fixes := map[string]string{}
		for _, c := range st.New.Checks {
			if c.Fix != "" && c.New > 0 {
				fixes[c.ID] = c.Fix
				said = append(said, fmt.Sprintf("Not triaged: `%s`'s %d new %s, its fix a command: `%s`.", oneLine(c.ID), c.New, plural(c.New, "finding", "findings"), oneLine(c.Fix)))
			}
		}
		for _, f := range st.New.Findings {
			if fixes[f.Check] != "" || f.Fingerprint == "" || seen[f.Fingerprint] {
				continue
			}
			seen[f.Fingerprint] = true
			out = append(out, Found{Gate: name, Check: f.Check, Tool: f.Tool, Rule: f.Rule, Level: f.Level, Path: f.Path, Line: f.Line, Message: f.Message, Fingerprint: f.Fingerprint})
		}
	}
	return out, said, nil
}

// headOf is the commit the range judges, else HEAD's.
func headOf(repo, rng string) (string, error) {
	head := "HEAD"
	if rng != "" {
		head = gitrange.Head(rng)
	}
	out, err := git(repo, "rev-parse", "--verify", head+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("the head %s: %v", head, err)
	}
	return strings.TrimSpace(out), nil
}

// triageModel is the model that wrote the triage, as the engine kept its
// call (out/calls.jsonl).
func triageModel(runDir string) string {
	f, err := os.Open(filepath.Join(runDir, "out", "calls.jsonl"))
	if err != nil {
		return ""
	}
	defer f.Close()
	model := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var c struct {
			For   string `json:"for"`
			Model string `json:"model"`
			Agent string `json:"agent"`
		}
		if json.Unmarshal(sc.Bytes(), &c) == nil && c.For == part {
			model = c.Model
			if model == "" {
				model = c.Agent
			}
		}
	}
	return model
}

// final ends the run with no question: the verdict is the one written
// here, always a pass.
func final(runDir, status, summary string, findings []verdict.Finding) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &verdict.Verdict{Status: status, Summary: summary, Findings: findings}); err != nil {
		return fail(err)
	}
	return exitNothing
}

func settings(runDir string) (Settings, error) {
	s := Settings{FindingsMax: 20, LinesAround: 8, ForgeWrites: true}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("settings: %w", err)
	}
	switch {
	case s.FindingsMax < 0:
		return s, fmt.Errorf("settings: findings-max: %d is below 0 (0: every new finding)", s.FindingsMax)
	case s.LinesAround < 0 || s.LinesAround > 50:
		return s, fmt.Errorf("settings: lines-around: %d is not a number from 0 to 50", s.LinesAround)
	}
	return s, nil
}

func input(runDir, name string) string {
	data, _ := os.ReadFile(filepath.Join(runDir, "in", "input", name))
	return strings.TrimSpace(string(data))
}

// text is an answer's text as shown and kept: one line, cut.
func text(v any) string {
	s, _ := v.(string)
	s = oneLine(s)
	if len(s) > textMax {
		s = strings.ToValidUTF8(s[:textMax], "") + "…"
	}
	return s
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// number reads an integer an answer gave, as YAML reads it.
func number(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), n == float64(int(n))
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		return i, err == nil
	}
	return 0, false
}

func git(repo string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok && len(e.Stderr) > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(e.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "inspector:", err)
	return 99
}
