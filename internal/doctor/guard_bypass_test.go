package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bypassWithGh runs bypassGuard against a stubbed gh whose answers are script.
func bypassWithGh(t *testing.T, script string) []Check {
	t.Helper()
	isolate(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	r := &Report{}
	bypassGuard(r, Options{Repo: t.TempDir(), Forge: "github"}, "")
	return r.Checks
}

// No ruleset: Given a default branch with no ruleset, When I run the
// doctor with --forge github, Then it prints a note that no ruleset
// protects the branch.
func TestBypassGuardNoRuleset(t *testing.T) {
	cs := bypassWithGh(t, "case \"$*\" in\n*rules/branches*) ;;\n*default_branch*) echo main ;;\n*) exit 1 ;;\nesac\n")
	if len(cs) != 1 || cs[0].Level != Note || !strings.Contains(cs[0].Message, "no ruleset") {
		t.Fatalf("want one note saying no ruleset protects main, got %+v", cs)
	}
	if !strings.Contains(cs[0].Message, "main") {
		t.Errorf("the note names the branch: %+v", cs[0])
	}
}

// One unreadable ruleset: Given one ruleset that allows a bypass and one
// that cannot be read, When I run the doctor, Then it prints the warning
// for the first and the note for the second.
func TestBypassGuardOneUnreadable(t *testing.T) {
	script := "case \"$*\" in\n" +
		"*rules/branches*) printf '7\\n8\\n' ;;\n" +
		"*rulesets/7*) echo '{\"id\":7,\"name\":\"first\",\"url\":\"\",\"can-bypass\":\"pull_requests_only\"}' ;;\n" +
		"*default_branch*) echo main ;;\n*) exit 1 ;;\nesac\n"
	cs := bypassWithGh(t, script)
	var warned, noted bool
	for _, c := range cs {
		if c.Rule == "bypass-allowed" && c.Level == Warn && strings.Contains(c.Message, `"first"`) {
			warned = true
		}
		if c.Rule == "bypass-unread" && c.Level == Note {
			noted = true
		}
	}
	if !warned || !noted {
		t.Fatalf("want the warning for the first and the note for the second, got %+v", cs)
	}
}

// Emergency command: Given a person locked out of main, When they read the
// emergency, Then step 1 gives a gh command to put them on the bypass list.
func TestEmergencyStepOneGivesGhCommand(t *testing.T) {
	data, err := os.ReadFile("../../docs/merge-policy-no-bypass.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	_, rest, ok := strings.Cut(doc, "## The emergency")
	if !ok {
		t.Fatal("no emergency section")
	}
	step, _, ok := strings.Cut(rest, "2. **Merge the fix**")
	if !ok {
		t.Fatal("no step 2 in the emergency")
	}
	if !strings.Contains(step, "gh api") || !strings.Contains(step, "bypass_actors") {
		t.Errorf("step 1 gives no gh api command with bypass_actors:\n%s", step)
	}
}
