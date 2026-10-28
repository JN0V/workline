package tester

// The tests from the criteria (ADR-0055, step 2): pre reads a ready
// issue's Validation scenarios and asks the agent for one test each, in
// the project's own framework; post holds the tests to the test files,
// to the scenarios, and to the red-first check, before the engine commits
// them on the work branch and opens it as a Draft.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing  = 10 // pre: no question for the agent, its verdict final
	exitExternal = 3  // an outside service failed
	testFilesMax = 3  // test files beside the Scope's code given whole
	listedMax    = 40 // test files named, not given
)

// Scenario is one of an issue's Validation scenarios: its name, as the
// test names it, and its Given / When / Then.
type Scenario struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

// task is what pre leaves for post, in in/tests-task.json.
type task struct {
	Issue     int        `json:"issue"`
	Title     string     `json:"title"`
	Base      string     `json:"base"`
	Scenarios []Scenario `json:"scenarios"`
}

const taskFile = "tests-task.json"

// scenarioLine is a Validation bullet opening on its name in bold:
// `- **Old debt:** Given …`, the colon inside the bold or after it.
var scenarioLine = regexp.MustCompile(`^\s*[-*+]\s+\*\*([^*]+?):?\*\*:?\s*(.*)$`)

// Scenarios reads the named scenarios of a Validation section, one a
// bullet opening on its name in bold, its lines until the next bullet.
func Scenarios(validation string) []Scenario {
	var out []Scenario
	for _, l := range strings.Split(validation, "\n") {
		if m := scenarioLine.FindStringSubmatch(l); m != nil {
			out = append(out, Scenario{Name: strings.TrimSpace(m[1]), Text: strings.TrimSpace(m[2])})
			continue
		}
		t := strings.TrimSpace(l)
		switch {
		case t == "":
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			// a bullet with no name: not a scenario a test can cite
			out = append(out, Scenario{Text: strings.TrimSpace(t[2:])})
		case len(out) > 0:
			out[len(out)-1].Text += " " + t
		}
	}
	return slices.DeleteFunc(out, func(s Scenario) bool { return s.Name == "" })
}

// Pre prepares the task: the issue the run targets, ready, its named
// scenarios, the Scope's code and the tests beside it.
func Pre(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	switch os.Getenv("WORKLINE_EVENT") {
	case "merge-request": // the frozen guard, no agent
		return guardPre(runDir, repo)
	case "objection":
		return answerPre(runDir, repo, s)
	}
	if v := unset(s); v != nil { // no proof possible: the agent is not asked
		return final(runDir, *v)
	}
	kind, num, _ := strings.Cut(os.Getenv("WORKLINE_TARGET"), ":")
	id, _ := strconv.Atoi(num)
	if os.Getenv("WORKLINE_EVENT") == "schedule" { // the moment: the run picks its issue
		var code int
		if id, code = scheduled(runDir, repo); id == 0 {
			return code
		}
		kind = "issue"
	}
	if kind != "issue" || id <= 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no issue to write tests for",
			Findings: []verdict.Finding{{Rule: "no-issue", Message: "the tester writes the tests of one issue: run `workline test write --issue <n>`"}}})
	}
	where := fmt.Sprintf("#%d", id)
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to read the issue from",
			Findings: []verdict.Finding{{Rule: "no-forge", Where: where, Message: "the tester reads the issue and opens a Draft on a forge; " + forge.Missing}}})
	}
	is, err := f.Issue(id)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	}
	if err != nil {
		return fail(err)
	}
	if !slices.Contains(is.Labels, backlog.LabelReady) {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the issue is not ready",
			Findings: []verdict.Finding{{Rule: "issue-not-ready", Where: where,
				Message: "the tester takes a ready issue only, labelled " + backlog.LabelReady + ": its scenarios are a person's, accepted"}}})
	}
	sections := work.ByMeaning(is.Body)
	scenarios := Scenarios(sections[work.Validation].Text)
	if len(scenarios) == 0 {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no scenario to test",
			Findings: []verdict.Finding{{Rule: "no-named-scenario", Where: where,
				Message: "its " + work.Name(work.Validation) + " holds no named scenario (`- **Name:** Given …, When …, Then …`): each test names the one it checks; " +
					"the product owner names them, the tester invents none"}}})
	}
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return fail(err)
	}
	t := task{Issue: id, Title: is.Title, Base: strings.TrimSpace(head), Scenarios: scenarios}
	if data, err := json.Marshal(t); err != nil {
		return fail(err)
	} else if err := os.WriteFile(filepath.Join(runDir, "in", taskFile), data, 0o644); err != nil {
		return fail(err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Issue #%d: %s\n\n", id, is.Title)
	if need := sections[work.Need].Text; need != "" {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", work.Name(work.Need), strings.TrimSpace(need))
	}
	b.WriteString("## The scenarios, one test each\n\n")
	for _, sc := range scenarios {
		fmt.Fprintf(&b, "- **%s:** %s\n", sc.Name, sc.Text)
	}
	for _, m := range []work.Meaning{work.Verification, work.Scope} {
		if text := strings.TrimSpace(sections[m].Text); text != "" {
			fmt.Fprintf(&b, "\n## %s\n\n%s\n", work.Name(m), text)
		}
	}
	fmt.Fprintf(&b, "\n## The project's tests\n\n- Run by: `%s`\n- Test files are those matching: %s\n",
		s.Command, strings.Join(s.Paths, ", "))
	writeCode(&b, repo, sections[work.Scope].Text, s)
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(b.String()), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// pathWord is a path-like word in backticks: what a Scope names.
var pathWord = regexp.MustCompile("`([^`\\s]+)`")

// writeCode gives the Scope's files, then the test files beside them, whole
// up to code-lines-max lines in all, and names the project's other tests.
func writeCode(b *strings.Builder, repo, scope string, s Settings) {
	out, _ := git(repo, "ls-files", "-z")
	files := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var named, tests []string
	for _, m := range pathWord.FindAllStringSubmatch(scope, -1) {
		if p := path.Clean(m[1]); slices.Contains(files, p) && !slices.Contains(named, p) {
			named = append(named, p)
		}
	}
	dirs := map[string]bool{}
	for _, p := range named {
		dirs[path.Dir(p)] = true
	}
	stems := map[string]bool{} // ordinals.go's tests: ordinals_test.go, test_ordinals.py
	for _, p := range named {
		stems[norm(strings.TrimSuffix(path.Base(p), path.Ext(p)))] = true
	}
	var beside, near []string
	for _, f := range files {
		if f == "" || !pathglob.Any(s.Paths, f) {
			continue
		}
		tests = append(tests, f)
		base := norm(strings.TrimSuffix(path.Base(f), path.Ext(f)))
		switch {
		case !dirs[path.Dir(f)]:
		case stems[strings.TrimSuffix(strings.TrimPrefix(base, "test"), "test")]:
			beside = append(beside, f)
		default:
			near = append(near, f)
		}
	}
	beside = append(beside, near...)
	beside = beside[:min(len(beside), testFilesMax)]
	if len(beside) == 0 && len(tests) > 0 { // none beside: the project's first, its style
		beside = tests[:min(len(tests), testFilesMax)]
	}
	left := s.CodeLinesMax
	if left <= 0 {
		left = 1500
	}
	give := func(heading string, list []string) {
		if len(list) == 0 {
			return
		}
		fmt.Fprintf(b, "\n## %s\n", heading)
		for _, p := range list {
			data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
			if err != nil {
				continue
			}
			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			cut := ""
			if len(lines) > left {
				lines, cut = lines[:max(left, 0)], fmt.Sprintf("\n(cut: the first %d lines given)", max(left, 0))
			}
			left -= len(lines)
			fmt.Fprintf(b, "\n### `%s`\n\n````\n%s\n````%s\n", p, strings.Join(lines, "\n"), cut)
		}
	}
	give("The code the "+work.Name(work.Scope)+" names", named)
	give("Tests already there, to write alike", beside)
	if len(tests) > len(beside) {
		rest := slices.DeleteFunc(slices.Clone(tests), func(f string) bool { return slices.Contains(beside, f) })
		if len(rest) > listedMax {
			rest = append(rest[:listedMax], fmt.Sprintf("and %d more", len(rest)-listedMax))
		}
		fmt.Fprintf(b, "\n## The project's other test files\n\n%s\n", strings.Join(rest, "\n"))
	}
}

// Post holds the agent's tests to what the tester may write and to the
// scenarios, then proves them red on the base; passed, it names the Draft
// the engine opens.
func Post(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	if os.Getenv("WORKLINE_EVENT") == "objection" {
		return answerPost(runDir, repo, s)
	}
	var t task
	if data, err := os.ReadFile(filepath.Join(runDir, "in", taskFile)); err != nil {
		return fail(err)
	} else if err := json.Unmarshal(data, &t); err != nil {
		return fail(err)
	}
	where := fmt.Sprintf("#%d", t.Issue)
	intents, err := intent.Read(filepath.Join(runDir, "out", "intentions.yaml"))
	if err != nil {
		return fail(err)
	}
	if v := enginesOnly(intents); v != nil {
		return write(runDir, *v)
	}
	var patches []intent.Intention
	var notes []string
	for _, in := range intents {
		switch in.Kind {
		case "patch":
			patches = append(patches, in)
		case "note":
			notes = append(notes, noteText(in.Value))
		}
	}
	if len(patches) == 0 {
		if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no agent: the issue waits for its tests",
				Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where,
					Message: "writing a scenario's test is judgement: with no agent, nothing is written; a person writes them, or the tester runs with one (`--ai`)"}}})
		}
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no test written",
			Findings: []verdict.Finding{{Rule: "no-test-written", Where: where, Message: "no test file the tester may write was proposed for the issue's scenarios"}}})
	}
	// What the tests read: each patch's file and its lines, for the
	// scenarios they name.
	var findings []verdict.Finding
	written := map[string]string{}
	var order []string
	for _, p := range patches {
		// A file outside test-paths never comes here: the engine refuses
		// the proposal whole against duties.writes first.
		for _, file := range intent.PatchFiles(p.Value) {
			// A test already there is never changed: the tester adds its
			// own files (ADR-0055, guards; weakening needs a proof).
			if _, err := git(repo, "cat-file", "-e", t.Base+":"+file); err == nil {
				findings = append(findings, verdict.Finding{Rule: "test-file-exists", Where: file,
					Message: "the base already holds this file: the tester writes its tests in new files, and never changes a test already there"})
				continue
			}
			if _, ok := written[file]; !ok {
				order = append(order, file)
			}
			written[file] += added(p.Value)
		}
	}
	tested := map[string]string{} // scenario -> the first test file naming it
	said := map[string]string{}   // scenario -> the note saying no test checks it
	for _, n := range notes {
		if i := claim(words(n), t.Scenarios, hasPrefix); i >= 0 {
			if _, ok := said[t.Scenarios[i].Name]; !ok {
				said[t.Scenarios[i].Name] = n
			}
		}
	}
	for _, sc := range t.Scenarios {
		for _, file := range order {
			if names(written[file], sc.Name) {
				tested[sc.Name] = file
				break
			}
		}
		if _, ok := tested[sc.Name]; ok {
			delete(said, sc.Name)
			continue
		}
		if _, ok := said[sc.Name]; !ok {
			findings = append(findings, verdict.Finding{Rule: "scenario-without-test", Where: sc.Name,
				Message: "no test names this scenario, and no note says why none can check it: each scenario gets a test that names it"})
		}
	}
	if len(findings) > 0 {
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the tests do not hold to the issue's scenarios", Findings: findings})
	}
	proof, newTests, blobs, err := proveOnBase(repo, t.Base, patches, s)
	if err != nil {
		return fail(err)
	}
	if proof.Status == verdict.Pass && len(newTests) == 0 {
		proof.Status = verdict.Block
		proof.Findings = append(proof.Findings, verdict.Finding{Rule: "no-new-test", Where: s.Report,
			Message: "the report names no new test: nothing is proven red, and no Draft is opened as proven"})
	}
	if proof.Status != verdict.Pass {
		proof.Findings = slices.DeleteFunc(proof.Findings, func(f verdict.Finding) bool { return f.Rule == "no-new-test" && f.Level == "warn" })
		return write(runDir, *proof)
	}
	mr := map[string]any{
		"key": fmt.Sprintf("issue-%d", t.Issue), "draft": true,
		"title": fmt.Sprintf("test: the acceptance tests of #%d", t.Issue),
		"body":  draftBody(t, tested, said, newTests, proof),
		// The tests frozen to whoever builds (ADR-0055, guards): kept in
		// workline's own comment on the Draft once it is opened.
		"record": record{Issue: t.Issue, Base: t.Base, Files: blobs}.intention()["comment"],
	}
	data, err := yaml.Marshal(mr)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	return write(runDir, *proof)
}

// proveOnBase commits the patches on base in a checkout apart and runs the
// red-first check on that commit: the tests alone, red on the base.
func proveOnBase(repo, base string, patches []intent.Intention, s Settings) (*verdict.Verdict, []string, map[string]string, error) {
	tree, err := os.MkdirTemp("", "workline-tests-")
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if _, err := git(repo, "worktree", "remove", "--force", tree); err != nil {
			_ = os.RemoveAll(tree)
			_, _ = git(repo, "worktree", "prune")
		}
	}()
	if _, err := git(repo, "-c", "core.hooksPath=/dev/null", "worktree", "add", "--detach", "--quiet", tree, base); err != nil {
		return nil, nil, nil, err
	}
	for _, p := range patches {
		if m, ok := p.Value.(map[string]any); ok {
			file, _ := m["file"].(string)
			content, _ := m["content"].(string)
			full := filepath.Join(tree, filepath.FromSlash(path.Clean(file)))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return nil, nil, nil, err
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				return nil, nil, nil, err
			}
			continue
		}
		diff, _ := p.Value.(string)
		cmd := exec.Command("git", "-C", tree, "apply", "--recount", "--unidiff-zero", "-")
		cmd.Stdin = strings.NewReader(intent.NormalizeDiff(diff))
		if out, err := cmd.CombinedOutput(); err != nil {
			return &verdict.Verdict{Status: verdict.Block, Summary: "a test's patch does not apply",
				Findings: []verdict.Finding{{Rule: "patch-unreadable", Where: strings.Join(intent.PatchFiles(diff), ", "), Message: strings.TrimSpace(string(out))}}}, nil, nil, nil
		}
	}
	if _, err := git(tree, "add", "-A"); err != nil {
		return nil, nil, nil, err
	}
	if _, err := git(tree, "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-c", "user.name=workline", "-c", "user.email=workline@invalid",
		"commit", "-q", "--no-verify", "-m", "test: the tests proven red"); err != nil {
		return nil, nil, nil, err
	}
	var files []string
	for _, p := range patches {
		files = append(files, intent.PatchFiles(p.Value)...)
	}
	blobs, err := blobsOf(tree, "HEAD", files)
	if err != nil {
		return nil, nil, nil, err
	}
	v, newTests := Prove(tree, base, s)
	return v, newTests, blobs, nil
}

// draftBody is the Draft's description: the issue it refers to, each
// scenario and its test, what the proof said.
func draftBody(t task, tested, said map[string]string, newTests []string, proof *verdict.Verdict) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Refs #%d\n\nThe acceptance tests of #%d, written before the code: %s. "+
		"Whoever builds it pushes the code on this branch, and makes this pull request close the issue.\n\n## Tests\n\n| Scenario | Test |\n|---|---|\n",
		t.Issue, t.Issue, proof.Summary)
	for _, sc := range t.Scenarios {
		file, ok := tested[sc.Name]
		if !ok {
			continue
		}
		var ids []string
		for _, id := range newTests {
			if i := claim(words(id[strings.LastIndexAny(id, "./:")+1:]), t.Scenarios, hasSuffix); i >= 0 && t.Scenarios[i].Name == sc.Name {
				ids = append(ids, "`"+id+"`")
			}
		}
		test := "in `" + file + "`"
		if len(ids) > 0 {
			test = strings.Join(ids, ", ") + " " + test
		}
		fmt.Fprintf(&b, "| %s | %s |\n", sc.Name, test)
	}
	if len(said) > 0 {
		b.WriteString("\n**Not tested**, no test can check it:\n\n")
		for _, sc := range t.Scenarios {
			if n, ok := said[sc.Name]; ok {
				fmt.Fprintf(&b, "- %s: %s\n", sc.Name, n)
			}
		}
	}
	for _, f := range proof.Findings {
		fmt.Fprintf(&b, "\n> %s (%s): %s\n", f.Rule, f.Level, f.Message)
	}
	return b.String()
}

// enginesOnly refuses an agent's comment or label: the tester's are the
// engine's own, its record and its replies, never the agent's.
func enginesOnly(intents []intent.Intention) *verdict.Verdict {
	for _, in := range intents {
		if in.Kind == "comment" || in.Kind == "label" {
			return &verdict.Verdict{Status: verdict.Block, Summary: "the agent may propose only tests and notes",
				Findings: []verdict.Finding{{Rule: "not-the-agents", Where: in.Kind,
					Message: "a " + in.Kind + " is the engine's to write, from the tester's verdict: the agent answers with `patch` and `note` only"}}}
		}
	}
	return nil
}

// added is the text a patch adds: a file's content, or a diff's + lines.
func added(v any) string {
	if m, ok := v.(map[string]any); ok {
		c, _ := m["content"].(string)
		return c
	}
	diff, _ := v.(string)
	var b strings.Builder
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			b.WriteString(l[1:] + "\n")
		}
	}
	return b.String()
}

func noteText(v any) string {
	if m, ok := v.(map[string]any); ok {
		t, _ := m["text"].(string)
		return t
	}
	t, _ := v.(string)
	return t
}

// names reports whether a test file names the scenario by a whole,
// delimited mention, never as a part of a longer name nor in prose: a
// "Scenario: <name>" line, a quoted string that is the name, or an
// identifier whose words are the name's, after a "test" prefix or an
// underscore (`TestMinusOne`, `test_minus_one`, `TestSub_MinusOne`).
// "Minus one" is thus not named by `TestMinusOneHundred`.
func names(text, name string) bool {
	want := words(name)
	if len(want) == 0 {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(strings.ToLower(line), "scenario:"); i >= 0 && slices.Equal(words(line[i+len("scenario:"):]), want) {
			return true
		}
		for _, q := range quotedRe.FindAllString(line, -1) {
			if slices.Equal(words(q[1:len(q)-1]), want) {
				return true
			}
		}
		for _, id := range identRe.FindAllString(line, -1) {
			w := words(id)
			if len(w) > 0 && (w[0] == "test" || w[0] == "tests") && slices.Equal(w[1:], want) {
				return true
			}
			// a camel-cased last segment, after a prefix: `TestSub_MinusOne`
			if seg := id[strings.LastIndex(id, "_")+1:]; seg != id && hasUpper(seg) && slices.Equal(words(seg), want) {
				return true
			}
		}
	}
	return false
}

var (
	quotedRe = regexp.MustCompile("\"[^\"]*\"|'[^']*'|`[^`]*`")
	identRe  = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

func hasUpper(s string) bool { return strings.ToLower(s) != s }

// words cuts a text into its lowercased words: at what is no letter nor
// digit, at a lower-to-upper case change and between letters and digits,
// so "Two by 3", `TwoBy3` and `two_by_3` give the same words, and
// "Space in a name" those of `TestSpaceInAName`.
func words(s string) []string {
	var out []string
	var cur []rune
	cut := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			cut()
			continue
		}
		if n := len(cur); n > 0 {
			p := cur[n-1]
			switch {
			case (unicode.IsUpper(r) && !unicode.IsUpper(p)) || unicode.IsLetter(r) != unicode.IsLetter(p):
				cut()
			case unicode.IsLower(r) && unicode.IsUpper(p) && n > 1 && unicode.IsUpper(cur[n-2]):
				// capitals then a word: `AName` is "a name", `HTTPServer` "http server"
				cur = cur[:n-1]
				cut()
				cur = append(cur, p)
			}
		}
		cur = append(cur, r)
	}
	cut()
	return out
}

func hasPrefix(have, want []string) bool {
	return len(have) >= len(want) && slices.Equal(have[:len(want)], want)
}

func hasSuffix(have, want []string) bool {
	return len(have) >= len(want) && slices.Equal(have[len(have)-len(want):], want)
}

// claim is the scenario a text's words mention, by the match given: the
// one with the most words when several do ("Minus one hundred" before
// "Minus one"); -1 when none does.
func claim(have []string, scenarios []Scenario, match func(have, want []string) bool) int {
	best, most := -1, 0
	for i, sc := range scenarios {
		if w := words(sc.Name); len(w) > most && match(have, w) {
			best, most = i, len(w)
		}
	}
	return best
}

// norm keeps a text's letters and digits, lowercased: a scenario's name
// read the same in a comment, `TestTwoByThree` or `test_two_by_three`.
func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func settingsIn(runDir string) (Settings, error) {
	m := map[string]any{}
	data, err := os.ReadFile(filepath.Join(runDir, "in", "settings.json"))
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return Settings{}, err
	}
	return SettingsFrom(m)
}

func final(runDir string, v verdict.Verdict) int {
	v.Final = true
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	return exitNothing
}

func write(runDir string, v verdict.Verdict) int {
	if err := verdict.Write(filepath.Join(runDir, "out", "verdict.yaml"), &v); err != nil {
		return fail(err)
	}
	switch v.Status { // the exit post's verdict says (role contract, "Exit codes")
	case verdict.Pass:
		return 0
	case verdict.Human:
		return 2
	case verdict.BlockedExternal:
		return exitExternal
	}
	return 1
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "tester:", err)
	return 1
}
