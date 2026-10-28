package tester

// The curator's removals (ADR-0055, step 7): pre picks one folder of
// tests, from the measures, and gives the agent its test files and each
// test's measures; the agent, the judgement, proposes a few tests to
// remove or merge, a reason each. post proves it with no agent: the
// touched folders' tests run before and after in one checkout, with
// their coverage and, where the project's language has a mutation tool,
// its mutants; a covered line or a killed mutant lost, a test red, no
// reason given: nothing is proposed. Proven, the engine opens the pull
// request, merged by the project's merge policy.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/pathglob"
	"github.com/JN0V/workline/internal/verdict"
)

// HistoryEnv names the curator's history file for a curate run's pre: the
// measures it weighs, and where it keeps the folder it weighed.
const HistoryEnv = "WORKLINE_CURATOR_HISTORY"

// curateKey opens the key of a curator's merge request, one a folder
// (workline/tester/curate-<folder>).
const curateKey = "curate-"

// curateTask is what pre leaves for post, in in/curate-task.json.
type curateTask struct {
	Base   string         `json:"base"`
	Dir    string         `json:"dir"`
	Frozen map[string]int `json:"frozen,omitempty"` // test file -> its issue
}

const curateTaskFile = "curate-task.json"

// curatePre picks the folder the curator weighs this run and writes the
// agent's task: the folder's test files whole, each test's measures.
func curatePre(runDir, repo string, s Settings) int {
	if v := unset(s); v != nil {
		v.Summary = "the curator did not run"
		return final(runDir, *v)
	}
	missing := func(where, msg string) int {
		return final(runDir, *refuse("the curator did not run", verdict.Finding{Rule: "setting-missing", Where: where, Message: msg}))
	}
	if strings.TrimSpace(s.CoverageReport) == "" {
		return missing("coverage-report", "the proof compares the coverage before and after a removal: set coverage-report, "+
			"the file the curator-proof-command (else curator-command, else test-command) writes, Go's profile or LCOV; no proof, no pull request")
	}
	if lang, tools := mutationTool(repo); lang != "" && strings.TrimSpace(s.MutationCommand) == "" {
		return missing("curator-mutation-command", fmt.Sprintf("%s has %s: set curator-mutation-command, run on each touched folder ({package}) "+
			"and writing curator-mutation-report ({report}), e.g. `gremlins unleash -s -o {report} {package}`; the proof keeps every killed mutant. "+
			"`none` runs the proof on coverage alone, said in the pull request", lang, tools))
	}
	f, err := forge.Open(os.Getenv("WORKLINE_FORGE"), repo)
	if err != nil {
		return fail(err)
	}
	if f == nil {
		return final(runDir, verdict.Verdict{Status: verdict.Block, Summary: "no forge to propose a removal on",
			Findings: []verdict.Finding{{Rule: "no-forge", Message: "the curator reads the tester's frozen tests and opens its pull request on a forge; " + forge.Missing}}})
	}
	frozen, curating, err := protected(f)
	if errors.Is(err, forge.ErrUnreachable) {
		fmt.Fprintln(os.Stderr, err)
		return exitExternal
	} else if err != nil {
		return fail(err)
	}
	hist := os.Getenv(HistoryEnv)
	h, _, err := readHistory(hist)
	if err != nil {
		return final(runDir, *refuse("the curator did not run", verdict.Finding{Rule: "history-unreadable", Where: hist, Message: err.Error()}))
	}
	dir, files, err := pickFolder(repo, s, h, curating)
	if err != nil {
		return fail(err)
	}
	if dir == "" {
		return final(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no folder of tests to weigh",
			Findings: []verdict.Finding{{Rule: "nothing-to-curate", Level: "warn", Where: "curator-paths",
				Message: "each folder of tests is weighed already (an open pull request, or within curator-window measures), or over code-lines-max"}}})
	}
	if hist != "" {
		if h.Curated == nil {
			h.Curated = map[string]int{}
		}
		h.Curated[dir] = h.Count
		if err := writeHistory(hist, h); err != nil {
			return fail(err)
		}
	}
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return fail(err)
	}
	t := curateTask{Base: strings.TrimSpace(head), Dir: dir, Frozen: frozen}
	if data, err := json.Marshal(t); err != nil {
		return fail(err)
	} else if err := os.WriteFile(filepath.Join(runDir, "in", curateTaskFile), data, 0o644); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", "task.md"), []byte(curateTaskText(repo, dir, files, frozen, h, s)), 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// protected reads, on the forge, the test files the frozen guard records
// on the tester's open Drafts, by issue, and the folders an open curator's
// merge request already weighs, by branch.
func protected(f forge.Forge) (map[string]int, map[string]bool, error) {
	mrs, err := f.ListMergeRequests()
	if err != nil {
		return nil, nil, err
	}
	frozen, curating := map[string]int{}, map[string]bool{}
	for _, mr := range mrs {
		if strings.HasPrefix(mr.Branch, "workline/tester/"+curateKey) {
			curating[mr.Branch] = true
		}
		if issueOf(mr.Branch) == 0 {
			continue
		}
		r, missing, err := readRecord(f, mr.ID)
		if err != nil {
			return nil, nil, err
		}
		if missing == nil {
			for file := range r.Files {
				frozen[file] = r.Issue
			}
		}
	}
	return frozen, curating, nil
}

// pickFolder is the folder of tests weighed this run: the largest by lines
// that fits code-lines-max, with no curator's merge request open, not
// weighed within curator-window measures; and its test files.
func pickFolder(repo string, s Settings, h History, curating map[string]bool) (string, []string, error) {
	out, err := git(repo, "ls-files")
	if err != nil {
		return "", nil, err
	}
	paths := curatorPaths(s)
	byDir := map[string][]string{}
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		if f != "" && pathglob.Any(paths, f) {
			byDir[path.Dir(f)] = append(byDir[path.Dir(f)], f)
		}
	}
	window := max(s.CuratorWindow, 1)
	best, bestLines := "", 0
	for _, dir := range slices.Sorted(func(yield func(string) bool) {
		for d := range byDir {
			if !yield(d) {
				return
			}
		}
	}) {
		if curating["workline/tester/"+intent.Slug(curateKey+dir)] {
			continue
		}
		if n, ok := h.Curated[dir]; ok && h.Count-n < window {
			continue
		}
		lines := 0
		for _, f := range byDir[dir] {
			data, _ := os.ReadFile(filepath.Join(repo, filepath.FromSlash(f)))
			lines += strings.Count(string(data), "\n")
		}
		if lines > bestLines && (s.CodeLinesMax <= 0 || lines <= s.CodeLinesMax) {
			best, bestLines = dir, lines
		}
	}
	return best, byDir[best], nil
}

func curatorPaths(s Settings) []string {
	if len(s.CuratorPaths) > 0 {
		return s.CuratorPaths
	}
	return s.Paths
}

// curateTaskText is the agent's task: what to weigh, how to answer, each
// test's measures, the folder's test files whole.
func curateTaskText(repo, dir string, files []string, frozen map[string]int, h History, s Settings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Curate the tests of `%s`\n\n", dir)
	fmt.Fprintf(&b, `You keep the project's test corpus concise. In the test files below, find
at most %d tests that are redundant, and remove or merge them:

- **Redundant**: another test checks the same behaviour on the same code;
  a slow test a faster one dominates; one scenario tested twice. Merge
  two tests into one when each checks something the other does not.
- **Kept**: a test that is the only one checking a behaviour, a bug's
  regression whose scenario no other test holds, a test in a frozen file.
  When in doubt, keep it.
- **Answer**: each test file you change, whole, as a `+"`patch`"+` with `+"`file`"+`
  and `+"`content`"+`; a file you delete whole, as a diff deleting it. Change
  test files only, never the code. Then one `+"`note`"+` for each test you
  remove or merge: the test's name first, then the test that keeps its
  check, and why.
- **Nothing redundant**: one `+"`note`"+` saying so, no patch.

The engine then proves your change, with no agent: the folder's tests
run before and after, all green; no covered line lost; no killed mutant
lost. Unproven, nothing is proposed.
`, max(s.RemovalsMax, 1))
	if len(frozen) > 0 {
		var fs []string
		for f := range frozen {
			if path.Dir(f) == dir {
				fs = append(fs, f)
			}
		}
		sort.Strings(fs)
		if len(fs) > 0 {
			b.WriteString("\n## Frozen: never touched\n\n")
			for _, f := range fs {
				fmt.Fprintf(&b, "- `%s`: the acceptance tests of #%d\n", f, frozen[f])
			}
		}
	}
	b.WriteString("\n## Each test's measures\n\n")
	dotted := strings.ReplaceAll(dir, "/", ".")
	var ids []string
	for id := range h.Tests {
		if dir != "." && (strings.Contains(id, dir) || strings.Contains(id, dotted)) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		b.WriteString("None measured yet (`workline test measure --history`): weigh the code alone.\n")
	} else {
		window := max(s.CuratorWindow, 1)
		fmt.Fprintf(&b, "From %d measure%s of the corpus; never red: in the last %d.\n\n| Test | Time (s) | Runs | Red runs | Never red |\n|---|---|---|---|---|\n",
			h.Count, plural(h.Count), window)
		for _, id := range ids {
			k := h.Tests[id]
			quiet := "no"
			if k.First <= h.Count-window+1 && k.LastRed <= h.Count-window {
				quiet = "yes"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %s |\n", id, num(k.Seconds), k.Runs, k.Red, quiet)
		}
	}
	b.WriteString("\n## The test files\n")
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(f)))
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "\n### `%s`\n\n````\n%s````\n", f, data)
	}
	return b.String()
}

// curatePost holds the agent's removal to what the curator may touch, then
// proves it; proven, it names the pull request the engine opens.
func curatePost(runDir, repo string, s Settings) int {
	var t curateTask
	if data, err := os.ReadFile(filepath.Join(runDir, "in", curateTaskFile)); err != nil {
		return fail(err)
	} else if err := json.Unmarshal(data, &t); err != nil {
		return fail(err)
	}
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
	where := t.Dir
	if len(patches) == 0 {
		if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
			return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: "no agent: nothing removed",
				Findings: []verdict.Finding{{Rule: "no-agent", Level: "warn", Where: where,
					Message: "weighing which tests are redundant is judgement: with no agent, nothing is removed; the measures stay in the job's summary"}}})
		}
		said := "nothing redundant in " + where
		if len(notes) > 0 {
			said += ": " + notes[0]
		}
		return write(runDir, verdict.Verdict{Status: verdict.Pass, Summary: said})
	}
	kept, err := keptNames(repo, s.CuratorKeep)
	if err != nil {
		return fail(err)
	}
	var findings []verdict.Finding
	var touched []string
	for _, p := range patches {
		for _, file := range intent.PatchFiles(p.Value) {
			touched = append(touched, file)
			switch {
			case !pathglob.Any(curatorPaths(s), file):
				findings = append(findings, verdict.Finding{Rule: "curator-path", Where: file,
					Message: "not a test file the curator may change (curator-paths): a test whose coverage the report cannot see is kept out of it"})
			case t.Frozen[file] > 0:
				findings = append(findings, verdict.Finding{Rule: "test-frozen", Where: file,
					Message: fmt.Sprintf("the acceptance tests of #%d, frozen on the tester's open Draft: never removed while it is open", t.Frozen[file])})
			case keeps(kept, file):
				findings = append(findings, verdict.Finding{Rule: "test-kept", Where: file, Message: "listed in a curator-keep file: never removed"})
			}
		}
	}
	if len(findings) > 0 {
		return write(runDir, verdict.Verdict{Status: verdict.Block, Summary: "the removal touches what the curator may not", Findings: findings})
	}
	p, v, err := proveRemoval(repo, t.Base, patches, touched, s)
	if err != nil {
		return fail(err)
	}
	if v == nil {
		v = p.judge(s, kept, notes)
	}
	if v.Status != verdict.Pass {
		return write(runDir, *v)
	}
	mr := map[string]any{
		"key":   curateKey + t.Dir,
		"title": fmt.Sprintf("test: curate %s, %d test%s fewer", t.Dir, len(p.removed)-len(p.added), plural(len(p.removed)-len(p.added))),
		"body":  p.body(t, s, notes),
	}
	data, err := yaml.Marshal(mr)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "out", "merge-request.yaml"), data, 0o644); err != nil {
		return fail(err)
	}
	return write(runDir, *v)
}

// side is what one run of the proof measured: each test, the suite's time,
// each covered block, each mutant's status.
type side struct {
	tests   map[string]timed
	seconds float64
	blocks  map[string]block
	unit    string
	mutants map[string]string
}

// proof is the removal's proof: the touched folders, before and after.
type proof struct {
	pkgs           []string
	command        string
	mutation       string
	before, after  side
	removed, added []string
	seconds        float64
	base           string
}

// proveRemoval runs the touched folders' tests, their coverage and their
// mutants on base, then again with the patches laid, in one checkout and
// within curator-proof-seconds. A run that did not hold is the verdict.
func proveRemoval(repo, base string, patches []intent.Intention, touched []string, s Settings) (*proof, *verdict.Verdict, error) {
	start := time.Now()
	if s.ProofSeconds > 0 {
		s.deadline = start.Add(time.Duration(s.ProofSeconds) * time.Second)
	}
	p := &proof{base: base, command: proofCommand(s), mutation: strings.TrimSpace(s.MutationCommand)}
	if p.mutation == "" { // no tool known for the language: pre let it through
		p.mutation = "none"
	}
	seen := map[string]bool{}
	for _, f := range touched {
		if d := path.Dir(f); !seen[d] {
			seen[d] = true
			p.pkgs = append(p.pkgs, d)
		}
	}
	sort.Strings(p.pkgs)
	tree, done, f := checkout(repo, base)
	if f != nil {
		return nil, refuse("the proof did not run", *f), nil
	}
	defer done()
	var vf *verdict.Finding
	if p.before, vf = p.measure(tree, s, "before the removal"); vf != nil {
		return nil, refuse("the proof did not run", *vf), nil
	}
	if v, err := layPatches(tree, patches); v != nil || err != nil {
		return nil, v, err
	}
	if p.after, vf = p.measure(tree, s, "after the removal"); vf != nil {
		return nil, refuse("the proof did not run", *vf), nil
	}
	p.seconds = time.Since(start).Seconds()
	parents := parentsOf(slices.Collect(func(yield func(string) bool) {
		for id := range p.before.tests {
			if !yield(id) {
				return
			}
		}
		for id := range p.after.tests {
			if !yield(id) {
				return
			}
		}
	}))
	for id := range p.before.tests {
		if _, ok := p.after.tests[id]; !ok && !parents[id] {
			p.removed = append(p.removed, id)
		}
	}
	for id := range p.after.tests {
		if _, ok := p.before.tests[id]; !ok && !parents[id] {
			p.added = append(p.added, id)
		}
	}
	sort.Strings(p.removed)
	sort.Strings(p.added)
	return p, nil, nil
}

func proofCommand(s Settings) string {
	for _, c := range []string{s.ProofCommand, s.CuratorCommand, s.Command} {
		if strings.TrimSpace(c) != "" {
			return c
		}
	}
	return ""
}

// measure runs the proof's command and the mutation tool once.
func (p *proof) measure(tree string, s Settings, where string) (side, *verdict.Finding) {
	var dirs []string
	for _, d := range p.pkgs {
		dirs = append(dirs, "./"+d)
	}
	cover := filepath.Join(tree, filepath.FromSlash(s.CoverageReport))
	_ = os.Remove(cover) // never the last run's
	rs := s
	rs.Command = strings.ReplaceAll(p.command, "{packages}", strings.Join(dirs, " "))
	tests, total, f := timedRun(tree, rs, where)
	if f != nil {
		return side{}, f
	}
	out := side{tests: tests, seconds: total, mutants: map[string]string{}}
	file, err := os.Open(cover)
	if err != nil {
		return side{}, &verdict.Finding{Rule: "coverage-missing", Where: s.CoverageReport,
			Message: fmt.Sprintf("the proof's command wrote no coverage %s: no proof", where)}
	}
	out.blocks, out.unit, err = coveredBlocks(bufio.NewScanner(file))
	_ = file.Close()
	if err != nil || len(out.blocks) == 0 {
		return side{}, &verdict.Finding{Rule: "coverage-missing", Where: s.CoverageReport,
			Message: fmt.Sprintf("%s %s is neither Go's profile nor LCOV, or empty: no proof", s.CoverageReport, where)}
	}
	if p.mutation == "none" {
		return out, nil
	}
	report := filepath.Join(tree, filepath.FromSlash(s.MutationReport))
	for i, d := range p.pkgs {
		_ = os.Remove(report)
		line := strings.NewReplacer("{package}", dirs[i], "{report}", report).Replace(p.mutation)
		res, late, runErr := shell(tree, line, s.deadline)
		if late {
			return side{}, overTime(where)
		}
		data, err := os.ReadFile(report)
		if err != nil {
			said := "it exited 0"
			if runErr != nil {
				said = runErr.Error()
			}
			return side{}, &verdict.Finding{Rule: "mutation-report-missing", Where: s.MutationReport,
				Message: fmt.Sprintf("the mutation command wrote no report on %s %s (%s); its output ends: %s", d, where, said, tail(res.String(), 10))}
		}
		if err := readMutants(data, d, out.mutants); err != nil {
			return side{}, &verdict.Finding{Rule: "mutation-report-unread", Where: s.MutationReport, Message: err.Error()}
		}
	}
	return out, nil
}

// readMutants reads a mutation report into each mutant's status, keyed by
// folder, file, place and kind: Gremlins' JSON, or the mutation testing
// report schema (Stryker's, and the tools writing it).
func readMutants(data []byte, dir string, out map[string]string) error {
	var g struct {
		Files []struct {
			Name      string `json:"file_name"`
			Mutations []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
				Line   int    `json:"line"`
				Column int    `json:"column"`
			} `json:"mutations"`
		} `json:"files"`
	}
	if json.Unmarshal(data, &g) == nil && g.Files != nil {
		for _, f := range g.Files {
			for _, m := range f.Mutations {
				out[fmt.Sprintf("%s/%s:%d:%d:%s", dir, f.Name, m.Line, m.Column, m.Type)] = m.Status
			}
		}
		return nil
	}
	var schema struct {
		Files map[string]struct {
			Mutants []struct {
				Mutator  string `json:"mutatorName"`
				Status   string `json:"status"`
				Location struct {
					Start struct{ Line, Column int } `json:"start"`
				} `json:"location"`
			} `json:"mutants"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &schema); err != nil || schema.Files == nil {
		return errors.New("the mutation report is neither Gremlins' JSON nor the mutation testing report schema")
	}
	for name, f := range schema.Files {
		for _, m := range f.Mutants {
			out[fmt.Sprintf("%s:%d:%d:%s", name, m.Location.Start.Line, m.Location.Start.Column, m.Mutator)] = m.Status
		}
	}
	return nil
}

// killed: a mutant a test caught; counted: one a test could have caught.
func killed(status string) bool {
	switch strings.ToUpper(strings.ReplaceAll(status, " ", "")) {
	case "KILLED", "TIMEDOUT", "TIMEOUT":
		return true
	}
	return false
}

func counted(status string) bool {
	switch strings.ToUpper(strings.ReplaceAll(status, " ", "")) {
	case "LIVED", "SURVIVED", "NOTCOVERED", "NOCOVERAGE":
		return true
	}
	return killed(status)
}

// judge holds the proof: both runs green, a test removed, no more than
// curator-removals-max, none kept, each named in a note, no covered line
// and no killed mutant lost.
func (p *proof) judge(s Settings, kept []string, notes []string) *verdict.Verdict {
	var fs []verdict.Finding
	red := func(sd side) []string {
		var ids []string
		for id, t := range sd.tests {
			if t.res == failed {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		return ids
	}
	if ids := red(p.before); len(ids) > 0 {
		fs = append(fs, verdict.Finding{Rule: "red-before", Where: sample(ids), Message: "red before the removal: a corpus red is not weighed"})
	}
	if ids := red(p.after); len(ids) > 0 {
		fs = append(fs, verdict.Finding{Rule: "red-after", Where: sample(ids), Message: "red after the removal"})
	}
	switch {
	case len(p.removed) == 0:
		fs = append(fs, verdict.Finding{Rule: "nothing-removed", Where: strings.Join(p.pkgs, ", "), Message: "the report names no test fewer after the patches"})
	case len(p.removed) > max(s.RemovalsMax, 1):
		fs = append(fs, verdict.Finding{Rule: "too-many-removed", Where: sample(p.removed),
			Message: fmt.Sprintf("%d tests removed, over curator-removals-max (%d): a few a run, each weighed", len(p.removed), max(s.RemovalsMax, 1))})
	}
	for _, id := range p.removed {
		name, _, _ := strings.Cut(p.before.tests[id].name, "/") // Go's subtest: its test
		if keeps(kept, id) || slices.Contains(kept, name) {
			fs = append(fs, verdict.Finding{Rule: "test-kept", Where: id, Message: "listed in a curator-keep file: never removed"})
		}
		if !slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, name) }) {
			fs = append(fs, verdict.Finding{Rule: "removal-unexplained", Where: id, Message: "no note names it: each test removed says which test keeps its check"})
		}
	}
	if lost := p.lostLines(s); len(lost) > 0 {
		fs = append(fs, verdict.Finding{Rule: "coverage-lost", Where: sample(lost),
			Message: fmt.Sprintf("%d %s covered before the removal, not after", len(lost), p.before.unit)})
	}
	if lost := p.lostMutants(); len(lost) > 0 {
		fs = append(fs, verdict.Finding{Rule: "mutant-lost", Where: sample(lost),
			Message: fmt.Sprintf("%d mutant%s killed before the removal, not after", len(lost), plural(len(lost)))})
	}
	if len(fs) > 0 {
		return &verdict.Verdict{Status: verdict.Block, Summary: "the removal is not proven", Findings: fs}
	}
	return &verdict.Verdict{Status: verdict.Pass, Summary: fmt.Sprintf("%d test%s removed, %d added; no covered line, no killed mutant lost",
		len(p.removed), plural(len(p.removed)), len(p.added))}
}

// lostLines are the blocks covered before and not after, the test files'
// own left out.
func (p *proof) lostLines(s Settings) []string {
	var lost []string
	for key, b := range p.before.blocks {
		file := key[:strings.LastIndexByte(key, ':')]
		if b.hit && !p.after.blocks[key].hit && !pathglob.Any(s.Paths, file) && !pathglob.Any(curatorPaths(s), file) {
			lost = append(lost, key)
		}
	}
	sort.Strings(lost)
	return lost
}

func (p *proof) lostMutants() []string {
	var lost []string
	for key, st := range p.before.mutants {
		if killed(st) && !killed(p.after.mutants[key]) {
			lost = append(lost, key)
		}
	}
	sort.Strings(lost)
	return lost
}

// keptNames reads the curator-keep files: a name a line, # a comment.
func keptNames(repo string, files []string) ([]string, error) {
	var out []string
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(f)))
		if err != nil {
			return nil, fmt.Errorf("curator-keep %s: %w", f, err)
		}
		for _, l := range strings.Split(string(data), "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				out = append(out, l)
			}
		}
	}
	return out, nil
}

// keeps: a kept name is a file's path without its extension, or its end;
// a test's id.
func keeps(kept []string, name string) bool {
	bare := strings.TrimSuffix(name, path.Ext(name))
	for _, k := range kept {
		if bare == k || strings.HasSuffix(bare, "/"+k) || name == k {
			return true
		}
	}
	return false
}

// coverPct is a run's coverage of the touched folders, as a person reads it.
func coverPct(sd side) string {
	all, hit := 0, 0
	for _, b := range sd.blocks {
		all += b.stmts
		if b.hit {
			hit += b.stmts
		}
	}
	return fmt.Sprintf("%.1f %% of %d %s", 100*float64(hit)/float64(max(all, 1)), all, sd.unit)
}

func mutationScore(sd side) string {
	k, c := 0, 0
	for _, st := range sd.mutants {
		if killed(st) {
			k++
		}
		if counted(st) {
			c++
		}
	}
	return fmt.Sprintf("%d of %d (%.1f %%)", k, c, 100*float64(k)/float64(max(c, 1)))
}

// body is the pull request's description: the numbers before and after,
// each test removed and why.
func (p *proof) body(t curateTask, s Settings, notes []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "The tester's curator removes %d test%s of `%s` that other tests already hold "+
		"([ADR-0055](https://github.com/JN0V/workline/blob/main/docs/adr/0055-the-tester-writes-the-acceptance-tests-before-the-code.md)). "+
		"Proven with no agent, before and after, on %s at `%s`: nothing lost. Merged by the project's merge policy.\n\n",
		len(p.removed), plural(len(p.removed)), t.Dir, "`"+strings.Join(p.pkgs, "`, `")+"`", short(p.base))
	b.WriteString("| | Before | After |\n|---|---|---|\n")
	fmt.Fprintf(&b, "| Tests | %d | %d |\n", len(p.before.tests), len(p.after.tests))
	fmt.Fprintf(&b, "| Their time | %s s | %s s |\n", num(p.before.seconds), num(p.after.seconds))
	fmt.Fprintf(&b, "| Coverage | %s | %s |\n", coverPct(p.before), coverPct(p.after))
	if p.mutation == "none" {
		b.WriteString("| Mutants killed | not run: `curator-mutation-command` none, or no tool known for the language | |\n")
	} else {
		fmt.Fprintf(&b, "| Mutants killed | %s | %s |\n", mutationScore(p.before), mutationScore(p.after))
	}
	fmt.Fprintf(&b, "\n- **Covered %s lost**: none. **Killed mutants lost**: none.\n", p.before.unit)
	fmt.Fprintf(&b, "- **The proof**: %s s, `%s`", num(p.seconds), strings.ReplaceAll(p.command, "`", "'"))
	if p.mutation != "none" {
		fmt.Fprintf(&b, ", then `%s` on each folder", strings.ReplaceAll(p.mutation, "`", "'"))
	}
	b.WriteString(".\n\n## Removed\n\n")
	for _, id := range p.removed {
		fmt.Fprintf(&b, "- `%s`\n", id)
	}
	if len(p.added) > 0 {
		b.WriteString("\n## Added, merging them\n\n")
		for _, id := range p.added {
			fmt.Fprintf(&b, "- `%s`\n", id)
		}
	}
	b.WriteString("\n## Why\n\n")
	for _, n := range notes {
		fmt.Fprintf(&b, "- %s\n", strings.TrimSpace(n))
	}
	return b.String()
}

// mutationTool is the project's language and its mutation tools, when
// one is known for it.
func mutationTool(repo string) (string, string) {
	for _, m := range mutators {
		if _, err := os.Stat(filepath.Join(repo, m.marker)); err == nil {
			return m.lang, m.tools
		}
	}
	return "", ""
}
