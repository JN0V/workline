package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoutePrePush(t *testing.T) {
	for _, c := range []struct {
		name, before string
		changed      bool
		steps        string
		keeps        string // a text the file must still hold
	}{
		{name: "no file", changed: true, steps: "committer documentalist"},
		{name: "no routing", before: "ai: claude # mine\n", changed: true, steps: "committer documentalist", keeps: "ai: claude # mine\n"},
		{name: "routing without pre-push", before: "routing:\n  events: {merge-request: [committer]}\n", changed: true, steps: "committer documentalist", keeps: "merge-request"},
		{name: "pre-push chosen", before: "routing:\n  events: {pre-push: [committer]}\n", steps: "committer", keeps: "pre-push: [committer]}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := t.TempDir()
			file := filepath.Join(repo, ".workline", "config.yaml")
			if c.before != "" {
				os.MkdirAll(filepath.Dir(file), 0o755)
				os.WriteFile(file, []byte(c.before), 0o644)
			}
			steps, changed, err := RoutePrePush(repo)
			if err != nil {
				t.Fatal(err)
			}
			if changed != c.changed || strings.Join(steps, " ") != c.steps {
				t.Errorf("got %v %v, want %v %q", steps, changed, c.changed, c.steps)
			}
			data, _ := os.ReadFile(file)
			if !strings.Contains(string(data), c.keeps) {
				t.Errorf("lost %q:\n%s", c.keeps, data)
			}
			cfg, err := Load(repo)
			if err != nil {
				t.Fatalf("the file no longer loads: %v\n%s", err, data)
			}
			if strings.Join(cfg.Events["pre-push"], " ") != c.steps {
				t.Errorf("routing reads pre-push as %v:\n%s", cfg.Events["pre-push"], data)
			}
		})
	}
}
