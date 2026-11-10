package tester

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/verdict"
)

// Every answer of the judge ends somewhere (ADR-0065): yes is red, no is
// build-broken with its line, and anything else red unproven, never a
// pass unsaid nor a stuck run.
func TestJudgedEveryAnswerEnds(t *testing.T) {
	s := Settings{JudgeReadRounds: 1, JudgeReadMax: 10}
	red := verdict.Verdict{Status: verdict.Pass, Summary: "1 new test, red on the base (1 not built there, a broken build)"}
	j := judging{Red: red, Added: []string{"T.kt:twoByThree"}, Rounds: []judgeRound{{Lines: 3}},
		Doubt: doubt{Lines: []string{"e: T.kt:9:28 Too many arguments"}, Where: "T.kt"}}
	for _, c := range []struct {
		answer, status, rule, holds string
		tests                       int
	}{
		{"yes: true\nwhy: the signature the issue gives\nlevel: model\nmodel: m\n", verdict.Pass, "red-judged", "the signature the issue gives", 1},
		{"yes: false\nwhy: a helper the issue never asks for\n", verdict.Block, "build-broken", "the judge: a helper the issue never asks for", 0},
		{"unsure: true\nwhy: whether add gains an overload\n", verdict.Pass, RedUnproven, "the judge cannot tell: whether add gains an overload", 1},
		{"error: 'not asked: the run spent its ai-max-tokens'\n", verdict.Pass, RedUnproven, "not asked: the run spent its ai-max-tokens", 1},
		{"reads: [{grep: add}]\nwhy: the base's add\n", verdict.Pass, RedUnproven, "past judge-read-rounds", 1},
		{"why: nothing\n", verdict.Pass, RedUnproven, "neither yes, no nor unsure", 1},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "answer.yaml"), []byte(c.answer), 0o644); err != nil {
			t.Fatal(err)
		}
		v, tests, _, asked, err := j.answered(dir, ".", s)
		if err != nil || asked {
			t.Fatalf("%q: asked %v, err %v", c.answer, asked, err)
		}
		var found *verdict.Finding
		for i, f := range v.Findings {
			if f.Rule == c.rule {
				found = &v.Findings[i]
			}
		}
		if v.Status != c.status || found == nil || !strings.Contains(found.Message, c.holds) || len(tests) != c.tests {
			t.Errorf("%q: %s %v, %d tests; want %s, %s holding %q", c.answer, v.Status, v.Findings, len(tests), c.status, c.rule, c.holds)
		}
		if c.rule == RedUnproven && (!strings.HasSuffix(v.Summary, ", red unproven") || !strings.Contains(found.Message, "Too many arguments")) {
			t.Errorf("%q: unproven not said with its lines: %q, %q", c.answer, v.Summary, found.Message)
		}
	}
}

// With no answer written, the proof is red unproven all the same.
func TestJudgedNoAnswer(t *testing.T) {
	j := judging{Red: verdict.Verdict{Status: verdict.Pass, Summary: "red"}, Doubt: doubt{Lines: []string{"x"}}}
	v, _, _, asked, err := j.answered(t.TempDir(), ".", Settings{})
	if err != nil || asked || unprovenOf(*v) == "" {
		t.Errorf("no answer: %v, asked %v, err %v", v, asked, err)
	}
}

// A person's tests with no agent wait for a person, red's count kept.
func TestDoubtWaiting(t *testing.T) {
	d := doubt{Lines: []string{"m_test.go:6:5: undefined: Mul"}, Where: "m_test.go"}
	v := d.waiting(verdict.Verdict{Status: verdict.Pass, Summary: "1 new test, red on the base"})
	if v.Status != verdict.Human || !strings.HasPrefix(v.Summary, "1 new test, red on the base") || v.Findings[0].Rule != RedUnproven || !strings.Contains(v.Findings[0].Message, "undefined: Mul") {
		t.Errorf("waiting: %+v", v)
	}
}

// The record on the Draft says red unproven, for the developer to read.
func TestRecordSaysRedUnproven(t *testing.T) {
	r := record{Issue: 3, Files: map[string]string{"T.kt": "abc1234"}, RedUnproven: "the judge cannot tell"}
	if b := r.body(); !strings.Contains(b, "**Red unproven**: the judge cannot tell") || !strings.Contains(b, `"red-unproven":"the judge cannot tell"`) {
		t.Errorf("record: %s", b)
	}
}

// A proof lost while the judge read fails the run once, said, never the
// judge asked again round after round.
func TestJudgedLostStateFails(t *testing.T) {
	t.Setenv("WORKLINE_JUDGED", "answered")
	runDir := t.TempDir()
	dir := filepath.Join(runDir, "out", "judge", redProof)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"", "{not json"} {
		if state != "" {
			if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(state), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		v, _, _, asked, err := proveJudged(runDir, ".", "main", nil, Settings{})
		if err == nil || asked || v != nil || !strings.Contains(err.Error(), "the judge not asked again") {
			t.Errorf("state %q: %v, asked %v, err %v", state, v, asked, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "question.yaml")); err == nil {
			t.Errorf("state %q: the judge asked again", state)
		}
	}
}

// The question lets the judge read only while a read is left.
func TestAskReadsOnlyWhenLeft(t *testing.T) {
	for _, c := range []struct {
		s     Settings
		reads bool
	}{
		{Settings{JudgeReadRounds: 2, JudgeReadMax: 400}, true},
		{Settings{JudgeReadRounds: 0, JudgeReadMax: 400}, false},
		{Settings{JudgeReadRounds: 2, JudgeReadMax: 0}, false},
	} {
		dir := t.TempDir()
		if err := (judging{}).ask(dir, c.s); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "question.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		var q struct{ Reads bool }
		if err := json.Unmarshal(data, &q); err != nil {
			t.Fatal(err)
		}
		if q.Reads != c.reads {
			t.Errorf("%+v: reads %v, want %v", c.s, q.Reads, c.reads)
		}
	}
}

// A rewrite red unproven says so, on the record and in the reply; one
// proven keeps what the earlier proof said of the other tests.
func TestRewrittenSaysRedUnproven(t *testing.T) {
	r := record{Issue: 3, Files: map[string]string{"A.kt": "a1", "B.kt": "b1"}}
	unproven := verdict.Verdict{Status: verdict.Pass, Findings: []verdict.Finding{{Rule: RedUnproven, Message: "the judge cannot tell"}}}
	got := r.rewritten("A.kt", "a2", unproven)
	if got.RedUnproven != "the judge cannot tell" || got.Files["A.kt"] != "a2" || got.Files["B.kt"] != "b1" || r.Files["A.kt"] != "a1" {
		t.Errorf("rewritten: %+v (from %+v)", got, r)
	}
	if s := unprovenSaid(unproven); !strings.Contains(s, "**Red unproven**: the judge cannot tell") {
		t.Errorf("reply: %q", s)
	}
	proven := verdict.Verdict{Status: verdict.Pass}
	if again := got.rewritten("A.kt", "a3", proven); again.RedUnproven != "the judge cannot tell" || unprovenSaid(proven) != "" {
		t.Errorf("proven: %+v, %q", again, unprovenSaid(proven))
	}
}

// The judge's last answer is kept by its input in the run (#1274): the
// same build broken the same way gets it again, with no call; another
// input, or a round of reads, keeps nothing.
func TestJudgedAnswerKeptByInput(t *testing.T) {
	runDir := t.TempDir()
	dir := filepath.Join(runDir, "out", "judge", redProof)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := Settings{JudgeReadRounds: 1, JudgeReadMax: 10}
	same := doubt{Facts: "## The build's errors\n\nT.kt:9: Too many arguments", Base: "abc", Lines: []string{"T.kt:9: Too many arguments"}, Where: "T.kt"}
	other := same
	other.Facts += "\nT.kt:12: Unresolved reference: stub"
	j := judging{Red: verdict.Verdict{Status: verdict.Pass}, Added: []string{"T.kt:t"}, Doubt: same, Key: inputKey(&same)}

	// A round of reads is no answer: nothing kept.
	if err := os.WriteFile(filepath.Join(dir, "answer.yaml"), []byte("reads: [{grep: add}]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, asked, err := j.answered(dir, ".", s); err != nil || !asked {
		t.Fatalf("reads: asked %v, err %v", asked, err)
	}
	if _, ok := keptAnswer(runDir, j.Key); ok {
		t.Fatal("a round of reads was kept as an answer")
	}

	if err := os.WriteFile(filepath.Join(dir, "answer.yaml"), []byte("yes: false\nwhy: a helper the issue never asks for\nlevel: model\nmodel: m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, _, _, _, err := j.answered(dir, ".", s); err != nil || v.Status != verdict.Block {
		t.Fatalf("no: %v, err %v", v, err)
	}
	a, ok := keptAnswer(runDir, inputKey(&same))
	if !ok || a.Yes == nil || *a.Yes || a.Why != "a helper the issue never asks for" {
		t.Fatalf("same input: kept %+v, %v", a, ok)
	}
	v, _, _ := j.settle(a, ", its answer to the same build earlier in this run")
	if v.Status != verdict.Block || !strings.Contains(v.Findings[0].Message, "the judge: a helper the issue never asks for") || !strings.Contains(v.Findings[0].Message, "earlier in this run") {
		t.Errorf("kept answer's verdict: %+v", v)
	}
	if _, ok := keptAnswer(runDir, inputKey(&other)); ok {
		t.Error("other errors took the answer given to the first")
	}
	if _, ok := keptAnswer(t.TempDir(), inputKey(&same)); ok {
		t.Error("another run took this run's answer")
	}
}
