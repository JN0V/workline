package routing

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Empty routing: Given a config with `routing:` empty (also `routing: ~` and
// `routing: {events:}`), When a reviewer is added, Then the config stays valid
// and holds the reviewer.
func TestEmptyRouting(t *testing.T) {
	for _, before := range []string{
		"routing:\n",
		"routing: ~\n",
		"routing: {events:}\n",
	} {
		t.Run(strings.TrimSpace(before), func(t *testing.T) {
			repo := t.TempDir()
			file := filepath.Join(repo, ".workline", "config.yaml")
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(before), 0o644); err != nil {
				t.Fatal(err)
			}
			_, changed, err := AddReviewer(repo)
			data, _ := os.ReadFile(file)
			if err != nil {
				t.Fatalf("error: %v\n%s", err, data)
			}
			if !changed {
				t.Errorf("changed is false:\n%s", data)
			}
			var doc any
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("the file no longer parses: %v\n%s", err, data)
			}
			cfg, err := Load(repo)
			if err != nil {
				t.Fatalf("the file no longer loads: %v\n%s", err, data)
			}
			if !slices.Contains(cfg.Events["merge-request"], "reviewer") {
				t.Errorf("merge-request reads %v, want reviewer in it:\n%s", cfg.Events["merge-request"], data)
			}
		})
	}
}

// Scalar routing: Given `routing: 3` (or `routing: {events: 3}`), When a
// reviewer is added, Then it is refused with a message naming routing and
// ending "reviewer not added", and the file is untouched.
func TestScalarRouting(t *testing.T) {
	for _, before := range []string{
		"routing: 3\n",
		"routing: {events: 3}\n",
	} {
		t.Run(strings.TrimSpace(before), func(t *testing.T) {
			repo := t.TempDir()
			file := filepath.Join(repo, ".workline", "config.yaml")
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(before), 0o644); err != nil {
				t.Fatal(err)
			}
			_, changed, err := AddReviewer(repo)
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), "routing") || !strings.HasSuffix(err.Error(), "reviewer not added") {
				t.Errorf("message %q", err)
			}
			if changed {
				t.Error("changed is true")
			}
			data, _ := os.ReadFile(file)
			if string(data) != before {
				t.Errorf("the file was changed:\n%s", data)
			}
		})
	}
}
