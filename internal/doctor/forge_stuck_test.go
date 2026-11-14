package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Doctor not held: given a stuck forge command, when workline doctor runs,
// then it ends within its probe timeout (20 s) plus 2 s.
func TestDoctorNotHeldByStuckForgeCommand(t *testing.T) {
	repo := t.TempDir()
	body := "#!/bin/sh\nsleep 30\ntrue\n"
	if err := os.WriteFile(filepath.Join(repo, "forge.sh"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	cs := forgeChecks(t, repo, "cmd:./forge.sh")
	elapsed := time.Since(start)
	if elapsed > 22*time.Second {
		t.Fatalf("the doctor took %s on a stuck forge command, want within 22s", elapsed)
	}
	if len(cs) != 1 || cs[0].Rule != "forge-command-failing" {
		t.Fatalf("want one forge-command-failing check, got %v", cs)
	}
}
