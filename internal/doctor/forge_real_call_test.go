package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realCallStub writes a forge command that, like the Forgejo sample, does
// not know `ping` and answers an error for it; it logs each operation it is
// asked, and answers the others with realAnswer (a shell command).
func realCallStub(t *testing.T, realAnswer string) (repo string) {
	t.Helper()
	repo = t.TempDir()
	body := "#!/bin/sh\ncat >/dev/null\n" +
		"echo \"$WORKLINE_FORGE_OPERATION\" >> calls.log\n" +
		"case \"$WORKLINE_FORGE_OPERATION\" in\n" +
		"ping) echo '{\"error\":\"unknown operation ping\"}' ;;\n" +
		"*) " + realAnswer + " ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(repo, "forgejo.sh"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// doctor asks the command forge before saying it is reachable: Given a
// command forge whose command answers, When a person runs `workline
// doctor`, Then it says reachable only after one real call answered.
func TestDoctorAsksTheCommandForgeBeforeSayingItIsReachable(t *testing.T) {
	repo := realCallStub(t, `case "$WORKLINE_FORGE_OPERATION" in *issues*) echo '[]' ;; *) echo '{}' ;; esac`)
	cs := forgeChecks(t, repo, "cmd:./forgejo.sh")
	if len(cs) != 1 || cs[0].Level != OK || cs[0].Rule != "forge-command" {
		t.Fatalf("want one reachable ok check, got %+v", cs)
	}
	log, _ := os.ReadFile(filepath.Join(repo, "calls.log"))
	real := false
	for _, op := range strings.Fields(string(log)) {
		if op != "ping" {
			real = true
		}
	}
	if !real {
		t.Fatalf("reachable was said without one real call; the command was asked only: %q", string(log))
	}
}

// a failing call is reported: Given a command forge whose call fails (wrong
// token), When a person runs `workline doctor`, Then it says the forge does
// not answer and how to fix it.
func TestAFailingCallIsReported(t *testing.T) {
	repo := realCallStub(t, `echo '401 bad token' >&2; exit 1`)
	cs := forgeChecks(t, repo, "cmd:./forgejo.sh")
	if len(cs) != 1 {
		t.Fatalf("want one check, got %+v", cs)
	}
	c := cs[0]
	if c.Level != Warn || c.Rule != "forge-command-failing" {
		t.Fatalf("want a forge-command-failing warning, got %+v", c)
	}
	if !strings.Contains(c.Message, "bad token") || c.Fix == "" {
		t.Fatalf("want the reason and a fix, got %+v", c)
	}
}
