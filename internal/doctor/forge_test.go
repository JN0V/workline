package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func forgeChecks(t *testing.T, repo, configured string) []Check {
	t.Helper()
	r := &Report{}
	forgeCommand(r, Options{Repo: repo}, configured)
	return r.Checks
}

func TestForgeCommandMissingWarns(t *testing.T) {
	cs := forgeChecks(t, t.TempDir(), "cmd:workline-no-such-forge")
	if len(cs) != 1 {
		t.Fatalf("got %d checks, want 1: %v", len(cs), cs)
	}
	c := cs[0]
	if c.Level != Warn || c.Rule != "forge-command-missing" {
		t.Fatalf("got %s %s, want a warning forge-command-missing", c.Level, c.Rule)
	}
	if !strings.Contains(c.Message, "workline-no-such-forge") || !strings.Contains(c.Fix, "workline-no-such-forge") {
		t.Fatalf("the warning names the command and a fix: %+v", c)
	}
}

func TestForgeCommandReachableAndWhyNot(t *testing.T) {
	repo := t.TempDir()
	write := func(body string) {
		p := filepath.Join(repo, "forgejo.sh")
		if err := os.WriteFile(p, []byte("#!/bin/sh\ncat >/dev/null\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(`echo '{}'`)
	if c := forgeChecks(t, repo, "cmd:./forgejo.sh")[0]; c.Level != OK || c.Rule != "forge-command" {
		t.Fatalf("want reachable ok, got %+v", c)
	}
	write(`echo '{"error":"FORGEJO_URL is not set"}'`)
	c := forgeChecks(t, repo, "cmd:./forgejo.sh")[0]
	if c.Level != Warn || c.Rule != "forge-command-failing" || !strings.Contains(c.Message, "FORGEJO_URL") || c.Fix == "" {
		t.Fatalf("want a warning saying why, got %+v", c)
	}
	write(`echo 'no token' >&2; exit 1`)
	c = forgeChecks(t, repo, "cmd:./forgejo.sh")[0]
	if c.Level != Warn || !strings.Contains(c.Message, "no token") {
		t.Fatalf("want a warning saying why, got %+v", c)
	}
}

// forgeRuleChecks runs the whole doctor on a repository whose config says
// forge, with the --forge override, and keeps the forge command's checks.
func forgeRuleChecks(t *testing.T, repo, configured, override string) []Check {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "forge: " + configured + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []Check
	for _, c := range Run(Options{Repo: repo, Forge: override}).Checks {
		if strings.HasPrefix(c.Rule, "forge-command") {
			got = append(got, c)
		}
	}
	return got
}

func TestRunChecksTheConfiguredForgeCommand(t *testing.T) {
	cs := forgeRuleChecks(t, t.TempDir(), "cmd:workline-no-such-forge", "")
	if len(cs) != 1 || cs[0].Rule != "forge-command-missing" || cs[0].Level != Warn {
		t.Fatalf("want one forge-command-missing warning, got %v", cs)
	}
}

func TestRunForgeOverrideTakesPrecedence(t *testing.T) {
	repo := t.TempDir()
	body := "#!/bin/sh\ncat >/dev/null\necho '{}'\n"
	if err := os.WriteFile(filepath.Join(repo, "forgejo.sh"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	// The configuration names a missing command, the override a working one.
	cs := forgeRuleChecks(t, repo, "cmd:workline-no-such-forge", "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Rule != "forge-command" || cs[0].Level != OK {
		t.Fatalf("want the override's ok check, got %v", cs)
	}
	// The configuration names GitHub, the override a command.
	cs = forgeRuleChecks(t, repo, "github", "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Rule != "forge-command" {
		t.Fatalf("want the override checked over github, got %v", cs)
	}
}

func TestForgeCommandLeavesOtherForges(t *testing.T) {
	for _, f := range []string{"", "github", "gitlab"} {
		if cs := forgeChecks(t, t.TempDir(), f); len(cs) != 0 {
			t.Fatalf("forge %q: want no check, got %v", f, cs)
		}
	}
}
