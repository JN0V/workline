package tester

// The judge reads a broken build the fast path does not pass (ADR-0065,
// change 1; ADR-0045): post asks it, as pre asks the reviewer's judges
// (out/judge/red-proof/question.yaml), the engine puts the question and
// runs post again with its answer. A yes is red; a no, build-broken, its
// line quoted; unsure, past its reads, failed or capped, red unproven:
// the Draft opens with it written, the developer starts, and the green
// proof on the code settles it. With no agent at all, a person reads it.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JN0V/workline/internal/builtin/reviewer"
	"github.com/JN0V/workline/internal/intent"
	judgment "github.com/JN0V/workline/internal/judge"
	"github.com/JN0V/workline/internal/verdict"
	"go.yaml.in/yaml/v3"
)

// doubt is a build broken on the base past the fast path: what the judge
// reads, the compiler's lines quoted back, and where.
type doubt struct {
	Facts string   `json:"facts"`
	Lines []string `json:"lines"`
	Where string   `json:"where"`
	Base  string   `json:"base"` // the commit the judge's reads are served at
	errs  []buildError
}

// RedUnproven is the rule of the finding a red proof the judge did not
// settle carries: written on the Draft, and on the pull request the
// developer opens from it.
const RedUnproven = "red-unproven"

// lines is the compiler's lines, quoted on one line.
func (d *doubt) lines() string {
	return clipRunes(sample(d.Lines), quoteMax)
}

// waiting is the verdict on a person's tests with no agent to judge
// them: red's, unproven, for a person to read (ADR-0065).
func (d *doubt) waiting(red verdict.Verdict) *verdict.Verdict {
	v := red
	v.Status, v.Summary = verdict.Human, red.Summary+", red unproven: a person reads it"
	v.Findings = append(clone(red.Findings), verdict.Finding{Rule: RedUnproven, Where: d.Where,
		Message: "the build broke on the base with the tests otherwise than by a name the base lacks and the issue names, and no agent judges it (" + d.lines() +
			"): a person tells whether each test fails only because what the change adds does not exist yet"})
	return &v
}

// broken is the verdict a judge's no gives: the tests' own to mend.
func (d *doubt) broken(why, who string) *verdict.Verdict {
	return &verdict.Verdict{Status: verdict.Block, Summary: "the build broke on the base with the change's tests, not by what the issue adds",
		Findings: []verdict.Finding{{Rule: "build-broken", Where: d.Where,
			Message: fmt.Sprintf("the judge: %s (%s; %s): a test must fail by its assertion or by what the issue adds, "+
				"never a syntax error, a wrong import, a name the issue never asks for, or a wrong call on what the base has", why, who, d.lines())}}}
}

// unproven is red's verdict, marked: the judge did not settle it.
func (d *doubt) unproven(red verdict.Verdict, why string) *verdict.Verdict {
	v := red
	v.Findings = append(clone(red.Findings), verdict.Finding{Rule: RedUnproven, Level: "warn", Where: d.Where,
		Message: "the build broke on the base with the tests, and the judge did not tell it is for the right reason (" + why +
			"); its lines: " + d.lines() + ". The developer starts; the green proof on the code settles it"})
	v.Summary += ", red unproven"
	return &v
}

func clone(f []verdict.Finding) []verdict.Finding { return append([]verdict.Finding(nil), f...) }

// redProof is the folder post asks the judge in, and keeps the proof in
// until it answers.
const redProof = "red-proof"

// judging is what post keeps while the judge reads: the verdict and the
// tests a yes gives, the doubt, and each round of reads.
type judging struct {
	Red    verdict.Verdict   `json:"red"`
	Added  []string          `json:"added"`
	Blobs  map[string]string `json:"blobs"`
	Doubt  doubt             `json:"doubt"`
	Cause  string            `json:"cause"` // a test file: a name is looked for near it first
	Rounds []judgeRound      `json:"rounds,omitempty"`
}

type judgeRound struct {
	Asked   []judgment.Read `json:"asked"`
	Served  string          `json:"served"`
	Lines   int             `json:"lines"`
	Refused []string        `json:"refused,omitempty"`
}

// redAnswer is the judge's answer as the engine wrote it (answer.yaml).
type redAnswer struct {
	Yes                  *bool
	Unsure               bool
	Why, Model, Level    string
	Error, Judge, Author string
	Reads                []judgment.Read
}

// proveJudged is proveOnBase, its doubt put to the judge: asked, post
// waits for the answer (out/judge/red-proof), and runs again with it.
func proveJudged(runDir, repo, base string, patches []intent.Intention, s Settings) (v *verdict.Verdict, newTests []string, blobs map[string]string, asked bool, err error) {
	dir := filepath.Join(runDir, "out", "judge", redProof)
	if os.Getenv("WORKLINE_JUDGED") == "answered" {
		// The proof kept while the judge read: lost, the judge is not
		// asked again (the engine would, up to its rounds), the run fails.
		var j judging
		data, err := os.ReadFile(filepath.Join(dir, "state.json"))
		if err == nil {
			err = json.Unmarshal(data, &j)
		}
		if err != nil {
			return nil, nil, nil, false, fmt.Errorf("the red proof kept for its judge cannot be read, the judge not asked again: %w", err)
		}
		return j.answered(dir, repo, s)
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, nil, nil, false, err
	}
	p, blobs, err := proveOnBase(repo, base, patches, s)
	if err != nil || p.doubt == nil || p.v.Status != verdict.Pass || len(p.added) == 0 {
		return p.v, p.added, blobs, false, err
	}
	if ai := os.Getenv("WORKLINE_AI"); ai == "" || ai == "none" {
		return p.doubt.waiting(*p.v), p.added, blobs, false, nil
	}
	j := judging{Red: *p.v, Added: p.added, Blobs: blobs, Doubt: *p.doubt}
	for _, pt := range patches {
		if files := intent.PatchFiles(pt.Value); len(files) > 0 && j.Cause == "" {
			j.Cause = files[0]
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, nil, false, err
	}
	if err := j.ask(dir, s); err != nil {
		return nil, nil, nil, false, err
	}
	return &verdict.Verdict{Status: verdict.Block, Summary: "the red proof waits for its judge"}, nil, nil, true, nil
}

// answered reads the judge's answer: its reads served and asked again
// while it may read; else its verdict.
func (j judging) answered(dir, repo string, s Settings) (*verdict.Verdict, []string, map[string]string, bool, error) {
	var a redAnswer
	data, err := os.ReadFile(filepath.Join(dir, "answer.yaml"))
	if err == nil {
		err = yaml.Unmarshal(data, &a)
	}
	d := &j.Doubt
	who := fmt.Sprintf("independence: %s, %s", a.Level, a.Model)
	switch {
	case err != nil:
		return d.unproven(j.Red, "the judge gave no answer: "+err.Error()), j.Added, j.Blobs, false, nil
	case a.Error != "":
		return d.unproven(j.Red, "the judge did not answer: "+a.Error), j.Added, j.Blobs, false, nil
	case len(a.Reads) > 0:
		rounds, lines := j.left(s)
		if rounds <= 0 || lines <= 0 {
			return d.unproven(j.Red, "the judge asked to read past judge-read-rounds or judge-read-lines-max"+said(a.Why)), j.Added, j.Blobs, false, nil
		}
		served, n, refused := reviewer.ServeReads(repo, d.Base, j.Cause, a.Reads, lines, s.JudgeReadMax)
		j.Rounds = append(j.Rounds, judgeRound{Asked: a.Reads, Served: served, Lines: n, Refused: refused})
		if err := os.Rename(filepath.Join(dir, "answer.yaml"), filepath.Join(dir, fmt.Sprintf("answer-%d.yaml", len(j.Rounds)))); err != nil {
			return nil, nil, nil, false, err
		}
		if err := j.ask(dir, s); err != nil {
			return nil, nil, nil, false, err
		}
		return &verdict.Verdict{Status: verdict.Block, Summary: "the red proof waits for its judge"}, nil, nil, true, nil
	case a.Unsure:
		return d.unproven(j.Red, "the judge cannot tell"+said(a.Why)), j.Added, j.Blobs, false, nil
	case a.Yes == nil:
		return d.unproven(j.Red, "the judge said neither yes, no nor unsure"), j.Added, j.Blobs, false, nil
	case !*a.Yes:
		return d.broken(a.Why, who), nil, j.Blobs, false, nil
	}
	v := j.Red
	v.Findings = append(clone(j.Red.Findings), verdict.Finding{Rule: "red-judged", Level: "info", Where: d.Where,
		Message: fmt.Sprintf("the build broke on the base with the tests, and the judge says it is for the right reason: %s (%s)", a.Why, who)})
	return &v, j.Added, j.Blobs, false, nil
}

func said(why string) string {
	if why == "" {
		return ""
	}
	return ": " + why
}

// left is what the judge may still read: rounds, and lines.
func (j judging) left(s Settings) (rounds, lines int) {
	used := 0
	for _, r := range j.Rounds {
		used += r.Lines
	}
	return s.JudgeReadRounds - len(j.Rounds), s.JudgeReadMax - used
}

// ask keeps what post found and writes the judge's question.
func (j judging) ask(dir string, s Settings) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o644); err != nil {
		return err
	}
	q, _ := json.Marshal(map[string]any{"question": redQuestion, "material": j.material(s), "reads": true}) // JSON, which YAML reads
	return os.WriteFile(filepath.Join(dir, "question.yaml"), q, 0o644)
}

// redQuestion is what the judge of a broken build is asked.
const redQuestion = "These tests were written for the issue below before its code, and laid on the base, the code as it is. " +
	"Their build broke there. Does each test fail only because what the issue asks for does not exist yet " +
	"(a name, a member, a parameter or a signature the issue adds or changes), and not by a mistake of its own " +
	"(a name the issue never asks for, a wrong call on what the base already has, a syntax error, a wrong import)?"

// material is the facts, what the judge read so far, and how to answer.
func (j judging) material(s Settings) string {
	var b strings.Builder
	b.WriteString(j.Doubt.Facts)
	for i, r := range j.Rounds {
		fmt.Fprintf(&b, "\n## What you read, round %d\n\nAs the base reads at %s.\n%s", i+1, short(j.Doubt.Base), r.Served)
	}
	rounds, lines := j.left(s)
	if rounds > 0 && lines > 0 {
		fmt.Fprintf(&b, "\n## Reading more\n\nYou may read the base, the code as it is at %s, before you answer: %d more round(s) of reads, %d more lines. "+
			"To read, answer with reads instead of a note, each saying why: a file's lines, where a name is defined, or the lines naming a word:\n\n"+
			"- read: {path: \"path/to/file\", lines: \"1-40\", why: \"the signature the base has\"}\n- read: {name: \"restore\", why: \"…\"}\n- read: {grep: \"lostCount\", why: \"…\"}\n\n"+
			"You are then asked again, with what you read.\n", short(j.Doubt.Base), rounds, lines)
	} else {
		b.WriteString("\n## No read is left\n\nYou read what you could (judge-read-rounds, judge-read-lines-max): answer now.\n")
	}
	b.WriteString("\nYour answer is one note: yes; no, quoting the compiler's line that is the test's own mistake; or unsure, when you still cannot tell:\n\n" +
		"- note: \"yes: each error is a call to `restore(count, next)`, the signature the issue gives.\"\n" +
		"- note: \"no: the test calls `stubTwo`, a helper the issue never asks for.\"\n" +
		"- note: \"unsure: whether the issue changes `add` or adds an overload is not said.\"\n")
	return b.String()
}
