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

func TestAgentCaps(t *testing.T) {
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := `{"agent":"cmd","tokens-in":1500,"tokens-out":200,"seconds":1.4,"turns":7}` + "\n" +
		`{"agent":"cmd","task":"judge","tokens-in":300,"seconds":2.4,"turns":2,"time-cut":true}` + "\n"
	if err := os.WriteFile(filepath.Join(run, "out", "calls.jsonl"), []byte(calls), 0o644); err != nil {
		t.Fatal(err)
	}
	u := used(run)
	if u.Tokens != 2000 || u.Turns != 9 || u.Seconds != 4 || !u.timeCut {
		t.Fatalf("used = %+v, want 2000 tokens, 9 turns, 4 s, cut", u)
	}
	if c := agentCaps(u, Settings{MaxTokens: 3000, MaxTurns: 10}); c != nil {
		t.Errorf("under every cap, ai-max-seconds unset: want none, got %+v", c)
	}
	if c := agentCaps(u, Settings{}); c != nil {
		t.Errorf("no cap set: want none, got %+v", c)
	}
	c := agentCaps(u, Settings{MaxTokens: 2000, MaxTurns: 9, MaxSeconds: 3})
	want := []Cap{{"ai-max-seconds", 4, 3}, {"ai-max-turns", 9, 9}, {"ai-max-tokens", 2000, 2000}}
	if !slices.Equal(c, want) {
		t.Errorf("each at or past its cap: want %+v, got %+v", want, c)
	}
	if c := agentCaps(Used{Turns: 0, Tokens: 5}, Settings{MaxTurns: 100}); c != nil {
		t.Errorf("turns not reported: want no cap, got %+v", c)
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
