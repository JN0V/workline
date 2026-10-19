package role

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A person's own settings are laid last, only those the role lets a
// person set: a value wins, a list adds, the rest is returned, left out.
func TestWithPersonal(t *testing.T) {
	r := &Role{Personal: []string{"rewrite", "internal-codes-allow"}}
	merged := map[string]any{"rewrite": false, "subject-max": 72, "internal-codes-allow": []any{"UTF-8", "ADR-[0-9]+"}}
	got, refused := r.WithPersonal(merged, map[string]any{
		"rewrite": true, "subject-max": 200, "internal-codes-allow": []any{"nRF52", "UTF-8"},
	})
	want := map[string]any{"rewrite": true, "subject-max": 72, "internal-codes-allow": []any{"UTF-8", "ADR-[0-9]+", "nRF52"}}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(refused, []string{"subject-max"}) {
		t.Fatalf("got %v, refused %v", got, refused)
	}
	if merged["rewrite"] != false || len(merged["internal-codes-allow"].([]any)) != 2 {
		t.Fatal("the merged settings are left as they were")
	}
}

func TestPersonalSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if s, err := PersonalSettings("committer"); s != nil || err != nil {
		t.Fatalf("no config, no settings and no error: %v %v", s, err)
	}
	os.MkdirAll(filepath.Join(dir, "workline"), 0o755)
	os.WriteFile(filepath.Join(dir, "workline", "config.yaml"), []byte("ai: claude\nroles: {committer: {settings: {rewrite: true}}}\n"), 0o644)
	if s, err := PersonalSettings("committer"); err != nil || s["rewrite"] != true {
		t.Fatalf("the committer's own: %v %v", s, err)
	}
	if s, err := PersonalSettings("reviewer"); s != nil || err != nil {
		t.Fatalf("a role the config says nothing of: %v %v", s, err)
	}
	os.WriteFile(filepath.Join(dir, "workline", "config.yaml"), []byte("roles: [\n"), 0o644)
	if _, err := PersonalSettings("committer"); err == nil {
		t.Fatal("a config that cannot be read is said")
	}
}
