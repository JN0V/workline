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
	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/codepick"
	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/gate"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/routing"
	"github.com/JN0V/workline/internal/verdict"
	"github.com/JN0V/workline/internal/work"
)

const (
	exitNothing  = 10 // pre: no question for the agent, its verdict final
	exitExternal = 3  // an outside service failed
	testFilesMax = 3  // test files beside the Scope's code given whole
	listedMax    = 40 // test files named, not given
	// testShare: the part of code-lines-max kept for a test beside the
	// code when the code alone is past it, 1 in testShare (#1059: 1000
	// lines of headers, no test seen, no test possible).
	testShare = 5
)

// Scenario is one of an issue's Validation scenarios: its name, as the
// test names it, and its Given / When / Then.
type Scenario = work.Scenario

// task is what pre leaves for post, in in/tests-task.json.
type task struct {
	Issue     int        `json:"issue"`
	Title     string     `json:"title"`
	Base      string     `json:"base"`
	Scenarios []Scenario `json:"scenarios"`
	// Labels: the issue's, as pre read them; its state, which the tests'
	// result moves on (ADR-0059).
	Labels []string `json:"labels,omitempty"`
	// Body: the digest of the issue's body pre read, kept with a "no
	// test possible" (#815).
	Body string `json:"body,omitempty"`
	// Author: who opened the issue, asked how to reproduce a bug its
	// tests do not show (ADR-0060).
	Author string `json:"author,omitempty"`
}

const taskFile = "tests-task.json"

// Scenarios reads the named scenarios of a Validation section, in the
// forms people write them (work.Scenarios): the one parser the product
// owner, the tester and the developer read them with.
func Scenarios(validation string) []Scenario { return work.Scenarios(validation) }

// Pre prepares the task: the issue the run targets, ready, its named
// scenarios, the Scope's code and the tests beside it.
func Pre(runDir, repo string) int {
	s, err := settingsIn(runDir)
	if err != nil {
		return fail(err)
	}
	switch os.Getenv("WORKLINE_EVENT") {
	case "merge-request": // the frozen guard, no agent
		return guardPre(runDir, repo, s)
	case "objection":
		return answerPre(runDir, repo, s)
	case "curate":
		return curatePre(runDir, repo, s)
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
	if !backlog.Readied(*is) {
		return refuseIssue(runDir, id, verdict.Verdict{Status: verdict.Block, Summary: "the issue is not ready",
			Findings: []verdict.Finding{{Rule: "issue-not-ready", Where: where,
				Message: "the tester takes an issue waiting for its tests only, labelled " + backlog.LabelNeedsTests + " (or " + backlog.LabelReady + "): its scenarios are a person's, accepted"}}},
			"the product owner, or a person, readies it; the tester writes its tests at a run after that")
	}
	sections := work.ByMeaning(is.Body)
	scenarios := Scenarios(sections[work.Validation].Text)
	if len(scenarios) == 0 {
		return refuseIssue(runDir, id, verdict.Verdict{Status: verdict.Block, Summary: "no scenario to test",
			Findings: []verdict.Finding{{Rule: "no-named-scenario", Where: where,
				Message: "its " + work.Name(work.Validation) + " holds no named scenario (`- **Name:** Given …, When …, Then …`): each test names the one it checks; " +
					"the product owner names them, the tester invents none"}}},
			"the product owner, or a person, names its scenarios in its "+work.Name(work.Validation)+"; the tester writes a test for each at a run after that")
	}
	// The issue's comments: the record of a "no test possible", and the
	// code the product owner named for it, its sources (#1047).
	var comments, sources []string
	if bl, ok := f.(forge.Backlog); ok {
		comments, err = bl.Comments(forge.Target{Kind: "issue", ID: id})
		if errors.Is(err, forge.ErrUnreachable) {
			fmt.Fprintln(os.Stderr, err)
			return exitExternal
		}
		if err != nil {
			return fail(err)
		}
		if st, found, err := backlog.ReadState(comments, "product-owner"); found && err == nil {
			sources = st.Sources
		}
	}
	// The files its agent said it lacked at its last run, given now
	// (ADR-0063, §6).
	for _, p := range neededFiles(runDir) {
		if !slices.Contains(sources, p) {
			sources = append(sources, p)
		}
	}
	if code, done := sentBack(runDir, repo, *is, comments); done {
		return code
	}
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return fail(err)
	}
	t := task{Issue: id, Title: is.Title, Base: strings.TrimSpace(head), Scenarios: scenarios, Labels: is.Labels, Body: backlog.BodyDigest(is.Body), Author: is.Author}
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
	intro := b.String()
	text := fitTask(runDir, repo, s, func(s Settings) string {
		var b strings.Builder
		b.WriteString(intro)
		writeCode(&b, repo, is.Title+"\n"+is.Body, sections[work.Scope].Text, sources, s)
		return b.String()
	})
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(text), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// writeCode gives the code the issue names, found by the shared picker
// (ADR-0063 §8): the Scope's files, then those the product owner named
// for the issue, its sources, that the Scope does not (#1047: a Scope in
// words, nothing given, no test possible), and the code its title, Need
// and Scope name, up to code-lines-max lines in all, a share kept for the
// test files beside it; then those tests, and names the project's
// others.
func writeCode(b *strings.Builder, repo, text, scope string, sources []string, s Settings) {
	pk, err := codepick.New(repo, "")
	if err != nil {
		fmt.Fprintf(b, "\n## The code the %s names\n\nNot read: %v.\n", work.Name(work.Scope), err)
		return
	}
	files := pk.Files()
	var paths, owners []string
	for _, w := range work.ScopeWords(scope) {
		paths = append(paths, w)
	}
	scoped := map[string]bool{}
	for _, w := range paths {
		for _, f := range work.Files(w, files) {
			scoped[f] = true
		}
	}
	for _, src := range sources {
		src = work.LineOff(strings.TrimSpace(src))
		if !slices.Contains(paths, src) {
			paths = append(paths, src)
			if f := work.InRepo(src, files); f != "" && !scoped[f] {
				owners = append(owners, f)
			}
		}
	}
	budget := s.CodeLinesMax
	if budget <= 0 {
		budget = 1500
	}
	pick := pk.Pick(codepick.Ask{Text: text + "\n" + scope, Paths: paths, Tests: s.Paths, Budget: budget, BudgetName: "code-lines-max"})
	if len(pick.Named()) > 0 {
		fmt.Fprintf(b, "\n## The code the %s names\n", work.Name(work.Scope))
		if len(owners) > 0 {
			fmt.Fprintf(b, "\nNamed by the product owner for this issue, not in the %s: `%s`.\n", work.Name(work.Scope), strings.Join(owners, "`, `"))
		}
		if pick.Kept > 0 {
			fmt.Fprintf(b, "\nThe code fills code-lines-max: %d of the %d lines of code-lines-max kept for the tests beside it.\n", pick.Kept, budget)
		}
		pick.Write(b, codepick.Style{NoTest: true})
	}
	var tests []string
	for _, f := range files {
		if pathglob.Any(s.Paths, f) {
			tests = append(tests, f)
		}
	}
	beside := codepick.NearestTests(files, s.Paths, pick.Anchors())
	if t := pick.Test(); t != nil {
		beside = append([]string{t.Path}, slices.DeleteFunc(beside, func(f string) bool { return f == t.Path })...)
	}
	beside = beside[:min(len(beside), testFilesMax)]
	if len(beside) == 0 && len(tests) > 0 { // none beside: the project's first, its style
		beside = tests[:min(len(tests), testFilesMax)]
	}
	left := pick.Left
	if t := pick.Test(); t != nil {
		left += t.GivenLines() // given again below, with the tests beside
	}
	give := func(heading string, list []string) {
		if len(list) == 0 {
			return
		}
		fmt.Fprintf(b, "\n## %s\n", heading)
		var unread []string
		for _, p := range list {
			data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(p)))
			if err != nil {
				continue
			}
			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			if left <= 0 {
				// Named, never shown empty: a file given as 0 lines read
				// as an empty file to the agent (#801).
				unread = append(unread, fmt.Sprintf("- `%s` (%d lines)", p, len(lines)))
				continue
			}
			cut := ""
			if len(lines) > left {
				lines, cut = lines[:left], fmt.Sprintf("\n(cut: the first %d lines given)", left)
			}
			left -= len(lines)
			fmt.Fprintf(b, "\n### `%s`\n\n````\n%s\n````%s\n", p, strings.Join(lines, "\n"), cut)
		}
		if len(unread) > 0 {
			fmt.Fprintf(b, "\nNot given, past code-lines-max (%d lines in all), not empty:\n\n%s\n", budget, strings.Join(unread, "\n"))
		}
	}
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
	switch os.Getenv("WORKLINE_EVENT") {
	case "objection":
		return answerPost(runDir, repo, s)
	case "curate":
		return curatePost(runDir, repo, s)
	}
	if code, ok := saidPost(runDir); ok {
		return code
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
		return refusePost(runDir, t.Issue, *v)
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
		// No new test fits: the kind of proof the agent named instead,
		// recorded (ADR-0059, decision 4).
		if paths, why, ok := needsOf(notes); ok {
			return needed(runDir, repo, t, intents, paths, why)
		}
		if p, ok := proofOf(notes); ok {
			return proved(runDir, repo, t, intents, p)
		}
		if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no agent: the issue waits for its tests",
				Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where,
					Message: "writing a scenario's test is judgement: with no agent, nothing is written; a person writes them, or the tester runs with one (`--ai`)"}}})
		}
		msg := "no test file the tester may write was proposed for the issue's scenarios"
		if why := os.Getenv("WORKLINE_AGENT_OVER_BUDGET"); why != "" {
			// Not asked: no word of the agent's, the budget's (#1017).
			msg = "the agent was not asked, its prompt over the tester's context budget (prompt-over-budget: " + why +
				"): lower code-lines-max in the tester's settings, or narrow the issue's " + work.Name(work.Scope)
		} else if why := os.Getenv("WORKLINE_AGENT_UNREAD"); why != "" {
			msg = "the agent's answer could not be read, asked again once with the error: " + why
		} else if why := os.Getenv("WORKLINE_AGENT_REFUSED"); why != "" {
			msg = "the agent's answer was refused whole: " + why
		} else if os.Getenv("WORKLINE_AGENT_ANSWERED") == "true" {
			// Read, and no test written nor kind named: no test possible,
			// its notes the reason, back to the product owner (#794).
			reason := strings.Join(notes, " ")
			if strings.TrimSpace(reason) == "" {
				reason = msg
			}
			return proved(runDir, repo, t, intents, Proof{Kind: KindNone, Reason: reason})
		}
		return refusePost(runDir, t.Issue, verdict.Verdict{Status: verdict.Block, Summary: "no test written",
			Findings: []verdict.Finding{{Rule: "no-test-written", Where: where, Message: msg}}})
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
		return refusePost(runDir, t.Issue, verdict.Verdict{Status: verdict.Block, Summary: "the tests do not hold to the issue's scenarios", Findings: findings})
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
		if notShown(t, *proof, len(newTests)) {
			return unreproduced(runDir, t, intents, *proof)
		}
		// A test that does not build but for a missing name is the
		// agent's to mend, as a lint finding is: asked once more with the
		// build's error (#1048: a wrong constructor call, blocked a week).
		if slices.ContainsFunc(proof.Findings, func(f verdict.Finding) bool { return f.Rule == "build-broken" }) {
			proof.Again = true
		}
		return refusePost(runDir, t.Issue, *proof)
	}
	mr := map[string]any{
		"key": fmt.Sprintf("issue-%d", t.Issue), "draft": true,
		"title": fmt.Sprintf("test: the acceptance tests of #%d", t.Issue),
		"body":  draftBody(t, tested, said, newTests, proof),
		// The tests frozen to whoever builds (ADR-0055, guards): kept in
		// workline's own comment on the Draft once it is opened.
		"record": record{Issue: t.Issue, Base: t.Base, Files: blobs, Scenarios: t.Scenarios}.intention()["comment"],
	}
	data, err := yaml.Marshal(mr)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	// Each scenario has its row, the tests red on the base and linted: the
	// issue moves to the state its tests lead to, ready (ADR-0059,
	// decision 3), said on it; no agent decides it.
	moved, err := testsDone(runDir, repo, t, intents, len(said))
	if err != nil {
		return fail(err)
	}
	if moved != "" {
		proof.Findings = append(proof.Findings, verdict.Finding{Rule: "state-moved", Level: "info", Where: where,
			Message: "each scenario has its test or its reason in the Draft: the issue moves to " + moved})
	}
	return write(runDir, *proof)
}

// testsDone moves the issue whose tests are written to the state next
// after its own in the routing's table, its other states' labels off,
// and says so on it, beside the agent's intentions; "" when its state is
// not the tester's (an issue labelled ready by hand keeps its state).
func testsDone(runDir, repo string, t task, intents []intent.Intention, untested int) (string, error) {
	line, err := routing.Load(repo)
	if err != nil {
		return "", err
	}
	s, ok := line.StateOf(t.Labels)
	if !ok || s.Role != "tester" || s.Next == "" {
		return "", nil
	}
	add, remove, err := line.Move(s.Next)
	if err != nil {
		return "", err
	}
	// Only the labels it bears come off: one call less each on GitHub.
	remove = slices.DeleteFunc(remove, func(l string) bool { return !slices.Contains(t.Labels, l) })
	if slices.Contains(t.Labels, backlog.LabelNeedsRepro) {
		remove = append(remove, backlog.LabelNeedsRepro) // reproduced now: the mark goes as it moves (ADR-0039)
	}
	why := fmt.Sprintf("each of its %d scenarios has its test in the Draft's Tests table, red on the base, and the tests pass the project's lint", len(t.Scenarios))
	if untested > 0 {
		why = fmt.Sprintf("each of its %d scenarios has its test in the Draft's Tests table, or its reason under Not tested (%d, for a real try), "+
			"the tests red on the base and passing the project's lint", len(t.Scenarios), untested)
	}
	intents = append(intents,
		intent.Intention{Kind: "label", Value: map[string]any{"issue": t.Issue, "add": add, "remove": remove, "card": s.Next}},
		intent.Intention{Kind: "comment", Value: map[string]any{"issue": t.Issue, "sticky": "issue",
			"body": backlog.Said("The tester", "wrote the acceptance tests of this issue, as a Draft", why,
				"it is "+s.Next+": whoever builds it starts from the Draft, workline/tester/"+taskKey(t.Issue))}})
	return s.Next, intent.Write(filepath.Join(runDir, "out", "intentions.yaml"), intents)
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
	if v, err := layPatches(tree, patches); v != nil || err != nil {
		return v, nil, nil, err
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
	// The tests alone change: their new tests are in the folders that hold
	// them, so a draft-command naming {packages} runs those alone, on the
	// base and with the tests, rather than the whole suite twice (#979).
	var pkgs []string
	if strings.Contains(s.DraftCommand, "{packages}") {
		if pkgs = Packages(tree, files, s.Paths); len(pkgs) > 0 {
			s.Command, s.packages = s.DraftCommand, pkgs
		}
	}
	v, newTests := Prove(tree, base, s)
	if len(pkgs) > 0 && v.Status == verdict.Pass {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "proof-scope", Level: "info", Where: "draft-command",
			Message: "run on the folders holding the tests alone, " + strings.Join(pkgs, " ") + ": the pull request's own checks run the whole suite"})
	}
	if v.Status == verdict.Pass {
		rules, _ := compileBuildErrors(s.BuildErrors) // Prove said a bad one
		if lint := lintTests(repo, tree, base, missingNamed(files, rules)); lint != nil {
			return lint, newTests, blobs, nil
		}
	}
	return v, newTests, blobs, nil
}

// lintTests runs the project's lint on the tests, as their merge request
// will: each gate the project routes on merge-request, only what the
// tests add counted (ADR-0059, decision 3: a Draft that does not lint is
// not ready). Nil when each passes, or none is routed. A finding the
// proof reads as a missing name (missing) is not counted: the tests are
// red by it on purpose, and a lint that type-checks says it too (#988).
func lintTests(repo, tree, base string, missing func(path, message string) bool) *verdict.Verdict {
	line, err := routing.Load(repo)
	if err != nil {
		return &verdict.Verdict{Status: verdict.Block, Summary: "the project's lint could not be read",
			Findings: []verdict.Finding{{Rule: "check-error", Where: "routing", Message: err.Error()}}}
	}
	// The gates as this run reads them: the checkout's config, laid in
	// the tests' tree, uncommitted, never part of what they add.
	if data, err := os.ReadFile(filepath.Join(repo, ".workline", "config.yaml")); err == nil {
		if err := os.MkdirAll(filepath.Join(tree, ".workline"), 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(tree, ".workline", "config.yaml"), data, 0o644)
		}
	}
	var found []verdict.Finding
	var names []string
	mendable := true // each block a finding in the tests, none a gate that did not run
	for _, step := range line.Events["merge-request"] {
		name, ok := strings.CutPrefix(step, "gate:")
		if !ok {
			continue
		}
		v, add := gate.RunNew(tree, name, base)
		if v.Status == verdict.Pass {
			continue
		}
		g, _ := gate.Load(tree, name)
		var blocks []verdict.Finding
		lifted := false // a check over its limit only by missing names
		for _, f := range v.Findings {
			if f.Level != "" && f.Level != verdict.Block && f.Level != "error" {
				continue
			}
			if f.Rule == "over-threshold" && withinLeftOut(f.Where, g, add, missing) {
				lifted = true
				continue
			}
			blocks = append(blocks, f)
		}
		if lifted && len(blocks) == 0 {
			continue // each blocking finding a missing name: the gate passes on the tests
		}
		names = append(names, name)
		said := false
		for _, f := range blocks {
			said = true
			switch f.Rule {
			case "over-threshold":
				// The findings themselves, where a person finds them in
				// the tests, rather than the count and the gate's folder
				// on the machine that ran it, gone with the tests' tree.
				if named := lintFound(name, f.Where, g, add, missing); len(named) > 0 {
					found = append(found, named...)
					continue
				}
			case "check-failed":
			default: // the gate did not run: nothing the agent's tests can mend
				mendable = false
			}
			f.Rule, f.Message = "tests-lint", fmt.Sprintf("the gate %s: %s (%s)", name, ranOn(f.Message), f.Rule)
			found = append(found, f)
		}
		if !said { // a gate that blocks is always said, its own summary when no finding of it blocks
			mendable = false
			found = append(found, verdict.Finding{Rule: "tests-lint", Where: name, Message: fmt.Sprintf("the gate %s: %s", name, v.Summary)})
		}
	}
	// The reviewer's rules, as its review of the developer's pull request
	// will run them: a finding in a frozen test is the tester's to mend,
	// before its Draft opens (#1040).
	if slices.Contains(line.Events["merge-request"], reviewer.Name) {
		rules, err := reviewer.RulesOn(tree, base, "HEAD")
		if err != nil {
			mendable = false
			rules = []verdict.Finding{{Rule: "check-error", Where: reviewer.Name, Message: err.Error()}}
		}
		for _, f := range rules {
			f.Rule, f.Message = "tests-lint", fmt.Sprintf("the %s's rule: %s (%s)", reviewer.Name, f.Message, f.Rule)
			found = append(found, f)
		}
		if len(rules) > 0 {
			names = append(names, reviewer.Name+"'s rules")
		}
	}
	if len(names) == 0 {
		return nil
	}
	return &verdict.Verdict{Status: verdict.Block, Summary: "the tests do not pass the project's lint (" + strings.Join(names, ", ") + "): no Draft opened",
		Findings: found, Again: mendable}
}

// lintFound are the findings the tests add to a check of a gate, at a
// level it limits, each where it is from the repository's root: the
// file, its line, what the tool says and its rule.
func lintFound(name, check string, g *gate.Gate, add *gate.Added, missing func(path, message string) bool) []verdict.Finding {
	if g == nil || add == nil {
		return nil
	}
	limits := map[string]bool{}
	for _, c := range g.Checks {
		if c.ID == check {
			for level := range c.Max {
				limits[level] = true
			}
		}
	}
	var out []verdict.Finding
	for _, a := range add.Findings {
		if a.Check != check || !limits[a.Level] || missing(a.Path, a.Message) {
			continue
		}
		at := a.Path
		if a.Line > 0 {
			at = fmt.Sprintf("%s:%d", a.Path, a.Line)
		}
		out = append(out, verdict.Finding{Rule: "tests-lint", Where: at,
			Message: strings.TrimSpace(fmt.Sprintf("the gate %s, %s: %s (%s)", name, at, a.Message, a.Rule))})
	}
	return out
}

// withinLeftOut reports whether a check over its limit is within it once
// the findings naming missing names are left out; false when none is, or
// what the check added is not known.
func withinLeftOut(check string, g *gate.Gate, add *gate.Added, missing func(path, message string) bool) bool {
	if g == nil || add == nil {
		return false
	}
	counts, left := map[string]int{}, 0
	for _, a := range add.Findings {
		switch {
		case a.Check != check:
		case missing(a.Path, a.Message):
			left++
		default:
			counts[a.Level]++
		}
	}
	if left == 0 {
		return false
	}
	for _, c := range g.Checks {
		if c.ID != check {
			continue
		}
		for level, most := range c.Max {
			if counts[level] > most {
				return false
			}
		}
		return true
	}
	return false
}

// missingNamed tells a lint's finding that names what the code lacks in
// one of the tests' files, read as the proof reads a build's error (the
// rules' missing-name): the tests red by it on purpose. golangci-lint
// gives the compiler's lines in its message, at the package's first
// file: each line it holds must be one; else the finding itself, in a
// test file, is read.
func missingNamed(tests []string, rules []buildRules) func(path, message string) bool {
	in := map[string]bool{}
	for _, t := range tests {
		in[path.Clean(t)] = true
	}
	return func(p, message string) bool {
		if errs := buildErrorsIn(message, tests, rules); len(errs) > 0 {
			for _, e := range errs {
				if !e.missing {
					return false
				}
			}
			return true
		}
		if !in[path.Clean(strings.TrimPrefix(filepath.ToSlash(p), "./"))] {
			return false
		}
		for _, r := range rules {
			for _, re := range r.missing {
				if re.MatchString(strings.TrimSpace(message)) {
					return true
				}
			}
		}
		return false
	}
}

// seeLog is where a check's message names its log, on the machine that ran it.
var seeLog = regexp.MustCompile(`; see (/\S+\.log)`)

// ranOn is a gate's message without the folders of the machine that ran
// it, gone with the tests' tree: a failed check's log given by its last
// lines instead, where the findings were marked left out.
func ranOn(msg string) string {
	msg = markedIn.ReplaceAllString(msg, "")
	return seeLog.ReplaceAllStringFunc(msg, func(m string) string {
		data, err := os.ReadFile(seeLog.FindStringSubmatch(m)[1])
		if err != nil || strings.TrimSpace(string(data)) == "" {
			return ""
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		if len(lines) > logTail {
			lines = lines[len(lines)-logTail:]
		}
		return "; its output ends:\n" + strings.Join(lines, "\n") + "\n"
	})
}

// markedIn is where a gate marked the findings it compared.
var markedIn = regexp.MustCompile(`; each marked in /[^)]*`)

// logTail is the lines of a failed check's output told.
const logTail = 20

// layPatches writes the patches in tree: a whole file, or a diff applied;
// a diff that does not apply is a blocking verdict.
func layPatches(tree string, patches []intent.Intention) (*verdict.Verdict, error) {
	for _, p := range patches {
		if m, ok := p.Value.(map[string]any); ok {
			file, _ := m["file"].(string)
			content, _ := m["content"].(string)
			full := filepath.Join(tree, filepath.FromSlash(path.Clean(file)))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				return nil, err
			}
			continue
		}
		diff, _ := p.Value.(string)
		cmd := exec.Command("git", "-C", tree, "apply", "--recount", "--unidiff-zero", "-")
		cmd.Env = freshEnv() // the tree's own git, not the machine's config
		cmd.Stdin = strings.NewReader(intent.NormalizeDiff(diff))
		if out, err := cmd.CombinedOutput(); err != nil {
			return &verdict.Verdict{Status: verdict.Block, Summary: "a test's patch does not apply",
				Findings: []verdict.Finding{{Rule: "patch-unreadable", Where: strings.Join(intent.PatchFiles(diff), ", "), Message: strings.TrimSpace(string(out))}}}, nil
		}
	}
	return nil, nil
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
