package role

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One key, one file: given a setting in both the guarded config and the
// tuning file, when the engine loads the project's config, then it
// refuses it, naming both files.
func TestLoadProjectConfigRefusesKeyInBothFiles(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".workline")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	both := "roles:\n  tester:\n    settings:\n      suite-max-seconds: %s\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(strings.Replace(both, "%s", "600", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tuning.yaml"), []byte(strings.Replace(both, "%s", "900", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProjectConfig(repo)
	if err == nil {
		t.Fatal("a setting in both files is loaded: want it refused")
	}
	if !IsConfigError(err) {
		t.Errorf("not a config error: %v", err)
	}
	for _, want := range []string{"suite-max-seconds", "config.yaml", "tuning.yaml"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}
