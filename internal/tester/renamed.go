package tester

// A frozen test named from its file (file:name: its build broke on the
// base, no report named it yet) that no head id holds, while the head
// holds new tests it may run as: did it run under another name (a display
// name, a runner's own naming), or did the change drop it? The name match
// is a guess, so its miss is the judge's, never a block (ADR-0065, rule
// A). The judge reads the raw facts — the ids, the head's new tests, the
// frozen files, the change beside them — as the red proof's judge does
// (judged.go): "ran" passes, said; "dropped" blocks, frozen-test-missing;
// unsure, failed or no agent, frozen-unproven, written on the pull
// request, never silent. A report's id, or no new test at all, converges:
// it blocks with no judge (green.go).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/intent"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// renamedKey is the folder pre asks the judge in (in/judge/<key>).
const renamedKey = "frozen-names"

// FrozenUnproven is the rule of a frozen test the judge did not settle.
const FrozenUnproven = "frozen-unproven"

// renamed is what the judge reads, and the proof's verdict kept while it
// reads.
type renamed struct {
	Missing    []string        `json:"missing"`
	Candidates []string        `json:"candidates"`
	Runs       int             `json:"runs"`
	Facts      string          `json:"facts"`
	Verdict    verdict.Verdict `json:"verdict"`
}

// The judge's material, bounded.
const (
	renameCandidatesMax = 200
	renameFileMax       = 6000
	renameDiffMax       = 12000
)

// renameFacts is what the judge reads, raw: the frozen ids not found,
// their files as the head holds them, the head's new tests, and the
// change beside the frozen files, masked and clipped.
func renameFacts(repo, mb string, rec record, d *renamed) string {
	var b strings.Builder
	b.WriteString("## The tester's tests not found under their names\n\n" +
		"Each as `file:name`, the name read from its file when its build broke on the base; as its file alone, its ecosystem naming no test, " +
		"met by any test of that file; or as the id the runner wrote, " +
		"with no class and failed, on the base with the tests: maybe its record of a file it could not collect, not a test case.\n\n")
	files := map[string]bool{}
	all := false // a runner's record names no file: each frozen file is shown
	for _, t := range d.Missing {
		fmt.Fprintf(&b, "- `%s`\n", t)
		switch {
		case fileAlone(rec, t):
			files[t] = true
		case named(rec, t):
			files[t[:strings.LastIndex(t, ":")]] = true
		default:
			all = true
		}
	}
	b.WriteString("\n## Their files, as the head holds them\n")
	for _, f := range rec.files() {
		if !files[f] && !all {
			continue
		}
		content, _ := git(repo, "show", "HEAD:"+f)
		fmt.Fprintf(&b, "\n### `%s`\n\n````\n%s\n````\n", f, strings.TrimRight(clipRunes(forge.Mask(content), renameFileMax), "\n"))
	}
	b.WriteString("\n## The head's new tests, as its reports name them (none of them in the base's)\n\n")
	for i, id := range d.Candidates {
		if i == renameCandidatesMax {
			fmt.Fprintf(&b, "(and %d more)\n", len(d.Candidates)-i)
			break
		}
		fmt.Fprintf(&b, "- `%s`\n", id)
	}
	diff, err := git(repo, append([]string{"diff", "--no-color", mb, "HEAD", "--", "."}, excludes(rec)...)...)
	if err != nil {
		diff = "(the diff could not be read: " + err.Error() + ")"
	}
	fmt.Fprintf(&b, "\n## The change, beside the frozen test files (the build and runner files among it)\n\n```diff\n%s\n```\n",
		strings.TrimRight(clipRunes(forge.Mask(diff), renameDiffMax), "\n"))
	return b.String()
}

// excludes are pathspecs leaving the record's files out of a diff.
func excludes(rec record) []string {
	var out []string
	for _, f := range rec.files() {
		out = append(out, ":(exclude,literal)"+f)
	}
	return out
}

// renamedQuestion is what the judge is asked.
const renamedQuestion = "The tests below were written before the code and frozen: whoever builds may not change them. " +
	"The head's test reports do not hold them under the name read from their file (or, one named by its file alone, any test of that file). " +
	"Did each still run on the head under another name the reports give (a display name, a parameter, the runner's own naming), " +
	"or did the change stop it running (a build file, a filter, a skip, an exclusion)?"

// ask keeps the proof and writes the judge's question, for the engine to
// put and pre to run again with its answer.
func (d *renamed) ask(runDir string) error {
	dir := filepath.Join(runDir, "in", "judge", renamedKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o644); err != nil {
		return err
	}
	m := d.Facts + "\n## How to answer\n\nNo read is served: answer from these facts. Your answer is one note: " +
		"yes, each ran under another name, naming it; no, a test was dropped, quoting the change's line that drops it; or unsure, when you cannot tell:\n\n" +
		"- note: \"yes: `restoresOverlay` is reported as `SettingsTest.restores the overlay`, its display name.\"\n" +
		"- note: \"no: the build file excludes `*restoresOverlay*` from the test task.\"\n" +
		"- note: \"unsure: the reports name no test like it, and the change drops none that I can see.\"\n"
	q, _ := json.Marshal(map[string]any{"question": renamedQuestion, "material": m, "reads": true}) // JSON, which YAML reads; reads: so unsure is an answer
	return os.WriteFile(filepath.Join(dir, "question.yaml"), q, 0o644)
}

// renameAnswered is the proof settled by the judge's answer, pre run again
// (WORKLINE_JUDGED=answered); false when no question was kept.
func renameAnswered(runDir string) (verdict.Verdict, bool, error) {
	dir := filepath.Join(runDir, "in", "judge", renamedKey)
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if os.IsNotExist(err) {
		return verdict.Verdict{}, false, nil
	} else if err != nil {
		return verdict.Verdict{}, false, err
	}
	var d renamed
	if err := json.Unmarshal(data, &d); err != nil {
		return verdict.Verdict{}, false, fmt.Errorf("the green proof kept for its judge cannot be read: %w", err)
	}
	var a redAnswer
	data, err = os.ReadFile(filepath.Join(dir, "answer.yaml"))
	if err == nil {
		err = yaml.Unmarshal(data, &a)
	}
	who := fmt.Sprintf("independence: %s, %s", a.Level, a.Model)
	switch {
	case err != nil:
		return d.unproven("the judge gave no answer"), true, nil
	case a.Error != "":
		return d.unproven("the judge did not answer: " + a.Error), true, nil
	case len(a.Reads) > 0:
		return d.unproven("the judge asked to read, and no read is served"), true, nil
	case a.Unsure:
		return d.unproven("the judge cannot tell" + said(a.Why)), true, nil
	case a.Yes == nil:
		return d.unproven("the judge said neither yes, no nor unsure"), true, nil
	case !*a.Yes:
		v := d.Verdict
		v.Findings = clone(v.Findings)
		for _, t := range d.Missing {
			v.Findings = append(v.Findings, missingFinding(t, d.Runs, a.Why+" ("+who+")"))
		}
		return settled(v), true, nil
	}
	v := d.Verdict
	v.Findings = clone(v.Findings)
	for _, t := range d.Missing {
		v.Findings = append(v.Findings, verdict.Finding{Rule: "frozen-test-judged", Level: "info", Where: t,
			Message: fmt.Sprintf("not in the head's reports under its name, and the judge says it ran under another: %s (%s)", a.Why, who)})
	}
	return v, true, nil
}

// greenEnd ends pre with v. A frozen test the judge said ran under
// another name, or did not settle, is written on merge request mr too, in
// the tester's one comment there (principle 14), applied though the run
// blocks; the verdict is post's to give (saidPost).
func greenEnd(runDir string, mr int, v verdict.Verdict) int {
	var lines []string
	for _, f := range v.Findings {
		if f.Rule == FrozenUnproven || f.Rule == "frozen-test-judged" {
			lines = append(lines, fmt.Sprintf("- `%s` (%s): %s", f.Where, f.Rule, f.Message))
		}
	}
	if len(lines) == 0 || mr <= 0 {
		return final(runDir, v)
	}
	body := "**The tester's green proof**: " + v.Summary + ".\n\nA test it froze is not in the head's reports under its name:\n\n" + strings.Join(lines, "\n") + "\n"
	said := []intent.Intention{{Kind: "comment", Value: map[string]any{"merge-request": mr, "sticky": "green", "on-block": true, "body": body}}}
	if err := intent.Write(filepath.Join(runDir, "in", "fallback.yaml"), said); err != nil {
		return fail(err)
	}
	v.Final = true
	data, err := json.Marshal(v)
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "in", saidFile), data, 0o644); err != nil {
		return fail(err)
	}
	return 0
}

// unproven is the proof with each missing test marked: the judge did not
// tell whether it ran under another name or was dropped. Said on the pull
// request, never a pass in silence, never a block on a guess.
func (d *renamed) unproven(why string) verdict.Verdict {
	v := d.Verdict
	v.Findings = clone(v.Findings)
	for _, t := range d.Missing {
		v.Findings = append(v.Findings, verdict.Finding{Rule: FrozenUnproven, Level: "warn", Where: t,
			Message: fmt.Sprintf("a test the tester froze is not in the head's reports under its name, and the head holds %d new test%s it may run as; "+
				"whether it ran under another name or the change dropped it is not told (%s): a person reads the change's build and runner files",
				len(d.Candidates), plural(len(d.Candidates)), why)})
	}
	v.Summary += ", a frozen test unproven"
	return v
}
