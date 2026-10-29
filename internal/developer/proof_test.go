package developer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokensCap(t *testing.T) {
	run := t.TempDir()
	if err := os.MkdirAll(filepath.Join(run, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := `{"agent":"cmd","tokens-in":1500,"tokens-out":200,"seconds":1}` + "\n" + `{"agent":"cmd","task":"judge","tokens-in":300,"seconds":1}` + "\n"
	if err := os.WriteFile(filepath.Join(run, "out", "calls.jsonl"), []byte(calls), 0o644); err != nil {
		t.Fatal(err)
	}
	if c := tokensCap(run, 3000); c != nil {
		t.Errorf("2000 of 3000: want no cap, got %+v", c)
	}
	if c := tokensCap(run, 0); c != nil {
		t.Errorf("no cap set: want none, got %+v", c)
	}
	c := tokensCap(run, 2000)
	if c == nil || c.Spent != 2000 || c.Allows != 2000 || c.Setting != "ai-max-tokens" {
		t.Errorf("2000 of 2000: want the cap reached, as the engine counts it, got %+v", c)
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
