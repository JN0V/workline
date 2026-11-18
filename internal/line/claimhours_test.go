package line

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The claim's hours are the developer's settings, the project's over the
// role's, as the line reads them; none readable, the role's own answer.
func TestClaimHoursReadTheProjectsSettings(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	roles := filepath.Join(filepath.Dir(here), "..", "..", "roles")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "roles: {developer: {settings: {claim-hours: 48}}}\n"
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	tl := newTally()
	tl.rolesDir, tl.repo = roles, repo
	if h := tl.claimHours(); h != 48 {
		t.Errorf("claim-hours set to 48: read %d", h)
	}
	if h := newTally().claimHours(); h != 24 {
		t.Errorf("no roles folder: read %d, want the default 24", h)
	}
}
