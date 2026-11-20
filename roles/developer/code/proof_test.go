package developer

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/kit/devmark"
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
	want := []devmark.Cap{{Setting: "ai-max-seconds", Spent: 4, Allows: 3}, {Setting: "ai-max-turns", Spent: 7, Allows: 7}, {Setting: "ai-max-tokens", Spent: 2000, Allows: 2000}}
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
	if c := agentCaps(u, Settings{MaxTurns: 100}); !slices.Equal(c, []devmark.Cap{{Setting: "ai-max-turns", Spent: 100, Allows: 100}}) {
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
	if u := calls(t, `{"agent":"cmd","failed":"agent answered without valid proposals"}`, `{"agent":"cmd","turns":2}`); u.failed != "" {
		t.Errorf("asked again, then answered: want not failed, got %q", u.failed)
	}
	if u := calls(t, `{"agent":"cmd","time-cut":true}`); u.failed != "" || !u.timeCut {
		t.Errorf("a time cut: want cut, not failed; got %+v", u)
	}
}

func TestMergeRequestReadyOnlyGreenUncapped(t *testing.T) {
	green := &devmark.Proof{Result: devmark.Green, Command: "make test", Passed: 4}
	for _, tc := range []struct {
		name  string
		a     devmark.Act
		draft bool
	}{
		{"green", devmark.Act{Issue: 7, Proof: green}, false},
		{"green, a cap", devmark.Act{Issue: 7, Proof: green, Caps: []devmark.Cap{{Setting: "ai-max-tokens", Spent: 9, Allows: 5}}}, true},
		{"red", devmark.Act{Issue: 7, Proof: &devmark.Proof{Result: devmark.Red, Red: []string{"t.a"}}}, true},
		{"unproven", devmark.Act{Issue: 7, Proof: &devmark.Proof{Result: devmark.Unproven, Why: "no report"}}, true},
		{"green, its agent failed", devmark.Act{Issue: 7, Proof: green, AgentFailed: "it gave up"}, true},
	} {
		mr := mergeRequest(tc.a, "Add it", Subjects{})
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
