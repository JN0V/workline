package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// gitRepo is a repository holding files, committed on main.
func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for f, content := range files {
		full := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "init"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return repo
}

// A project whose test command is the tester's setting: run-tests.sh
// writes a JUnit report, one test that passes when lib.sh says "ok".
func provingRepo(t *testing.T, command string) string {
	t.Helper()
	roles, err := filepath.Abs("../../roles")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORKLINE_ROLES_DIR", roles)
	script := `r=$(sh lib.sh)
if [ "$r" = ok ]; then c='<testcase classname="lib" name="ok"/>'; else c='<testcase classname="lib" name="ok"><failure message="no"/></testcase>'; fi
printf '<testsuite name="lib">%s</testsuite>\n' "$c" > "$1"
[ "$r" = ok ]
`
	return gitRepo(t, map[string]string{
		"lib.sh":                "echo no\n",
		"run-tests.sh":          script,
		".workline/config.yaml": "roles:\n  tester:\n    settings:\n      test-command: " + command + "\n",
	})
}

func lib(says string) []intent.Intention {
	return []intent.Intention{{Kind: "patch", Value: map[string]any{"file": "lib.sh", "content": "echo " + says + "\n"}}}
}

func TestProve(t *testing.T) {
	repo := provingRepo(t, `"sh run-tests.sh {report}"`)
	if p, c := prove(repo, "", lib("ok"), Settings{ProofSeconds: 60}); p.Result != Green || p.Passed != 1 || c != nil {
		t.Errorf("green: proof = %+v, cap = %+v; want green, 1 passed, no cap", p, c)
	}
	p, c := prove(repo, "", lib("no"), Settings{ProofSeconds: 60})
	if p.Result != Red || len(p.Red) != 1 || p.Red[0] != "lib.ok" || c != nil {
		t.Errorf("red: proof = %+v, cap = %+v; want red, lib.ok red, no cap", p, c)
	}
	if lines := proofLines(p, nil); !strings.Contains(lines, "0 passed, 1 red") || !strings.Contains(lines, "red: `lib.ok`") {
		t.Errorf("red: Verification lines %q lack what was red", lines)
	}
	if f := proofFindings("#3", p, nil); len(f) != 1 || f[0].Rule != "proof-red" {
		t.Errorf("red: findings = %+v, want one proof-red", f)
	}
}

// No report written: the tests did not run whole, never green.
func TestProveNotRunWhole(t *testing.T) {
	repo := provingRepo(t, `"true"`)
	p, c := prove(repo, "", lib("ok"), Settings{ProofSeconds: 60})
	if p.Result != Unproven || p.Why == "" || c != nil {
		t.Errorf("proof = %+v, cap = %+v; want unproven, why said, no cap", p, c)
	}
	if f := proofFindings("#3", p, nil); len(f) != 1 || f[0].Rule != "proof-not-run" {
		t.Errorf("findings = %+v, want one proof-not-run", f)
	}
}

// A report naming no test proves nothing: unproven, never red.
func TestProveEmptyReportIsUnproven(t *testing.T) {
	repo := provingRepo(t, `"printf '<testsuite name=\"lib\"></testsuite>' > {report}"`)
	p, c := prove(repo, "", lib("ok"), Settings{ProofSeconds: 60})
	if p.Result != Unproven || p.Why != "the report names no test" || c != nil {
		t.Errorf("proof = %+v, cap = %+v; want unproven, the report naming no test", p, c)
	}
}

// Past proof-seconds, the command is stopped: unproven, and the cap said.
func TestProveStoppedPastItsSeconds(t *testing.T) {
	repo := provingRepo(t, `"exec sleep 30"`)
	p, c := prove(repo, "", lib("ok"), Settings{ProofSeconds: 1})
	if p.Result != Unproven || c == nil || c.Setting != "proof-seconds" || c.Allows != 1 {
		t.Fatalf("proof = %+v, cap = %+v; want unproven, the proof-seconds cap", p, c)
	}
	f := proofFindings("#3", p, []Cap{*c})
	if len(f) != 2 || f[1].Rule != "cap-reached" || f[1].Where != "proof-seconds" {
		t.Errorf("findings = %+v, want proof-not-run then cap-reached on proof-seconds", f)
	}
}

// calls writes a run's out/calls.jsonl and reads it back.
func calls(t *testing.T, lines ...string) Used {
	t.Helper()
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(run, "out", "calls.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return used(run)
}

func TestAgentCaps(t *testing.T) {
	u := calls(t, `{"agent":"cmd","tokens-in":1500,"tokens-out":200,"seconds":1.4,"turns":7}`,
		`{"agent":"cmd","task":"judge","tokens-in":300,"seconds":2.4,"turns":2,"time-cut":true}`)
	if u.Tokens != 2000 || u.Turns != 9 || u.mostTurns != 7 || u.Seconds != 4 || !u.timeCut {
		t.Fatalf("used = %+v, want 2000 tokens, 9 turns (7 the most in one call), 4 s, cut", u)
	}
	if c := agentCaps(u, Settings{MaxTokens: 3000, MaxTurns: 10}); c != nil {
		t.Errorf("under every cap, ai-max-seconds unset: want none, got %+v", c)
	}
	if c := agentCaps(u, Settings{}); c != nil {
		t.Errorf("no cap set: want none, got %+v", c)
	}
	c := agentCaps(u, Settings{MaxTokens: 2000, MaxTurns: 7, MaxSeconds: 3})
	want := []Cap{{"ai-max-seconds", 4, 3}, {"ai-max-turns", 7, 7}, {"ai-max-tokens", 2000, 2000}}
	if !slices.Equal(c, want) {
		t.Errorf("each at or past its cap: want %+v, got %+v", want, c)
	}
	if c := agentCaps(Used{Turns: 0, Tokens: 5}, Settings{MaxTurns: 100}); c != nil {
		t.Errorf("turns not reported: want no cap, got %+v", c)
	}
}

// ai-max-turns is told to each call whole (claude's --max-turns): two
// calls under it are not a cap reached, though their sum is past it.
func TestTurnCapIsPerCall(t *testing.T) {
	u := calls(t, `{"agent":"claude","turns":60,"tokens-in":10}`, `{"agent":"claude","turns":60,"tokens-in":10}`)
	if c := agentCaps(u, Settings{MaxTurns: 100}); c != nil {
		t.Errorf("two calls of 60 turns, 100 allowed each: want no cap, got %+v", c)
	}
	if u.Turns != 120 {
		t.Errorf("turns used = %d, want the run's 120 kept for the record", u.Turns)
	}
	u = calls(t, `{"agent":"claude","turns":3,"tokens-in":10}`, `{"agent":"claude","turns":100,"tokens-in":10}`)
	if c := agentCaps(u, Settings{MaxTurns: 100}); !slices.Equal(c, []Cap{{"ai-max-turns", 100, 100}}) {
		t.Errorf("one call at 100 turns: want ai-max-turns 100 of 100, got %+v", c)
	}
}

// A call that reports no turns or tokens is said not checked, even when
// another call did report them; the time held it only when it is set.
func TestUnreported(t *testing.T) {
	both := calls(t, `{"agent":"cmd","turns":4,"tokens-in":10}`)
	if f := unreported("#2", both, Settings{MaxTurns: 100, MaxTokens: 50}); f != nil {
		t.Errorf("both reported: want nothing said, got %+v", f)
	}
	one := calls(t, `{"agent":"cmd","turns":4,"tokens-in":10}`, `{"agent":"cmd"}`)
	f := unreported("#2", one, Settings{MaxTurns: 100, MaxTokens: 50, MaxSeconds: 1200})
	if len(f) != 1 || f[0].Rule != "cap-not-checked" || f[0].Level != "info" ||
		!strings.Contains(f[0].Message, "ai-max-turns nor ai-max-tokens") || !strings.Contains(f[0].Message, "its time held it, ai-max-seconds 1200") {
		t.Errorf("a call reporting nothing: want both said not checked, its time holding it, got %+v", f)
	}
	f = unreported("#2", one, Settings{MaxTurns: 100})
	if len(f) != 1 || strings.Contains(f[0].Message, "its time") || !strings.Contains(f[0].Message, "no cap held it") {
		t.Errorf("no ai-max-seconds: want no time hold claimed, got %+v", f)
	}
	if f := unreported("#2", one, Settings{}); f != nil {
		t.Errorf("no cap set: want nothing said, got %+v", f)
	}
}

// An agent call that failed is read from the run's calls; its time cut
// is a cap, not a failure.
func TestUsedReadsAFailure(t *testing.T) {
	u := calls(t, `{"agent":"cmd","turns":4,"failed":"agent unavailable: the command failed: it gave up"}`)
	if u.failed != "agent unavailable: the command failed: it gave up" {
		t.Errorf("failed = %q, want the call's", u.failed)
	}
	if u := calls(t, `{"agent":"cmd","time-cut":true}`); u.failed != "" || !u.timeCut {
		t.Errorf("a time cut: want cut, not failed; got %+v", u)
	}
}

func TestMergeRequestReadyOnlyGreenUncapped(t *testing.T) {
	green := &Proof{Result: Green, Command: "make test", Passed: 4}
	for _, tc := range []struct {
		name  string
		a     Act
		draft bool
	}{
		{"green", Act{Issue: 7, Proof: green}, false},
		{"green, a cap", Act{Issue: 7, Proof: green, Caps: []Cap{{Setting: "ai-max-tokens", Spent: 9, Allows: 5}}}, true},
		{"red", Act{Issue: 7, Proof: &Proof{Result: Red, Red: []string{"t.a"}}}, true},
		{"unproven", Act{Issue: 7, Proof: &Proof{Result: Unproven, Why: "no report"}}, true},
		{"green, its agent failed", Act{Issue: 7, Proof: green, AgentFailed: "it gave up"}, true},
	} {
		mr := mergeRequest(tc.a, "Add it")
		if mr["draft"] != tc.draft {
			t.Errorf("%s: draft = %v, want %v", tc.name, mr["draft"], tc.draft)
		}
		if body := mr["body"].(string); !strings.HasPrefix(body, "Closes #7\n") || !strings.Contains(body, "## Verification") {
			t.Errorf("%s: body %q lacks Closes #7 or its Verification", tc.name, body)
		}
		if mr["key"] != "7" {
			t.Errorf("%s: key = %v, want 7", tc.name, mr["key"])
		}
	}
}
