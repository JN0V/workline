package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A check takes fifty annotations a write: created with the first fifty,
// completed and neutral, then updated with the rest, fifty at a time
// (#412).
func TestGitHubAnnotateInBatches(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	script := "#!/bin/sh\necho \"$*\" >> " + log + "\ncat >> " + log + "\necho >> " + log + "\ncase \"$*\" in\n" +
		"*POST*) printf '%s\\n' '{\"id\":7,\"html_url\":\"https://github.com/o/r/runs/7\"}' ;;\n" +
		"*) echo '{}' ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	var notes []Annotation
	for i := 1; i <= 120; i++ {
		notes = append(notes, Annotation{Path: "docs/a.md", Line: i, Level: "notice", Message: "m"})
	}
	url, err := g.Annotate("abc123", FindingsCheck, "the summary", notes)
	calls, _ := os.ReadFile(log)
	if err != nil || url != "https://github.com/o/r/runs/7" {
		t.Fatalf("Annotate: %q, %v; calls:\n%s", url, err, calls)
	}
	got := string(calls)
	for _, want := range []string{`"head_sha":"abc123"`, `"conclusion":"neutral"`, `"name":"workline findings"`, `"title":"120 findings"`} {
		if !strings.Contains(got, want) {
			t.Errorf("the check lacks %s:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "PATCH repos/{owner}/{repo}/check-runs/7"); n != 2 {
		t.Errorf("%d updates, want 2 (120 annotations, 50 a write):\n%s", n, got)
	}
	if n := strings.Count(got, `"start_line"`); n != 120 {
		t.Errorf("%d annotations sent, want 120", n)
	}
}

// A check refused for the token's rights says the right to give.
func TestGitHubAnnotateRefused(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncat > /dev/null\necho 'gh: Resource not accessible by integration (HTTP 403)' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	_, err := g.Annotate("abc123", FindingsCheck, "s", []Annotation{{Path: "a.md", Line: 1, Level: "notice", Message: "m"}})
	if !Refused(err) || !strings.Contains(err.Error(), "Checks: read and write") {
		t.Errorf("Annotate: %v; want a refusal naming the right", err)
	}
}

// Whether a branch requires every conversation resolved: its rulesets
// first, which any token reads, then its classic protection, which only
// an admin's does; what could not be read is said, never taken for no.
func TestGitHubConversations(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		"*default_branch*) echo main ;;\n" +
		"*rules/branches*) echo \"$RULES\" ;;\n" +
		"*/protection*) [ -z \"$PROT_ERR\" ] || { echo \"$PROT_ERR\" >&2; exit 1; }; echo \"$PROT\" ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	for _, c := range []struct {
		name, rules, prot, protErr string
		required                   bool
		unread                     string
	}{
		{"a ruleset", "true", "", "", true, ""},
		{"classic protection", "false", "true", "", true, ""},
		{"neither", "false", "false", "", false, ""},
		{"not protected", "false", "", "gh: Branch not protected (HTTP 404)", false, ""},
		{"protection unread", "false", "", "gh: Resource not accessible by integration (HTTP 403)", false, "only an admin's token reads"},
		{"protection hidden", "false", "", "gh: Not Found (HTTP 404)", false, "only an admin's token reads"},
	} {
		t.Setenv("RULES", c.rules)
		t.Setenv("PROT", c.prot)
		t.Setenv("PROT_ERR", c.protErr)
		got, err := g.Conversations("")
		if err != nil || got.Branch != "main" || got.Required != c.required || (c.unread == "") != (got.Unread == "") || !strings.Contains(got.Unread, c.unread) {
			t.Errorf("%s: %+v, %v; want required %v, unread %q", c.name, got, err, c.required, c.unread)
		}
	}
}
