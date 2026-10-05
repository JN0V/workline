package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
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

func TestAddReviewer(t *testing.T) {
	for _, c := range []struct {
		name, before string
		changed      bool
		steps        string
		keeps        string
	}{
		{name: "no file", changed: true, steps: "committer documentalist reviewer"},
		{name: "a line of the project's", before: "routing:\n  events: {merge-request: [committer]}\n", changed: true, steps: "committer reviewer"},
		{name: "already there", before: "routing:\n  events: {merge-request: [reviewer]}\n", steps: "reviewer"},
		{name: "other settings kept", before: "ai: claude\n", changed: true, steps: "committer documentalist reviewer", keeps: "ai: claude"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := t.TempDir()
			file := filepath.Join(repo, ".workline", "config.yaml")
			if c.before != "" {
				os.MkdirAll(filepath.Dir(file), 0o755)
				os.WriteFile(file, []byte(c.before), 0o644)
			}
			steps, changed, err := AddReviewer(repo)
			if err != nil {
				t.Fatal(err)
			}
			if changed != c.changed || strings.Join(steps, " ") != c.steps {
				t.Errorf("got %v %v, want %v %q", steps, changed, c.changed, c.steps)
			}
			cfg, err := Load(repo)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(cfg.Events["merge-request"], " "); got != c.steps {
				t.Errorf("the line reads %q, want %q", got, c.steps)
			}
			data, _ := os.ReadFile(file)
			if !strings.Contains(string(data), c.keeps) {
				t.Errorf("lost %q:\n%s", c.keeps, data)
			}
		})
	}
}

func TestSetAutonomy(t *testing.T) {
	for _, c := range []struct {
		name, before string
		changed      bool
		level        string
		fails        bool
		keeps        string
	}{
		{name: "no file", changed: true, level: "cautious"},
		{name: "roles empty", before: "roles:\n", changed: true, level: "cautious"},
		{name: "roles null", before: "roles: ~\nai: claude\n", changed: true, level: "cautious", keeps: "ai: claude"},
		{name: "the role empty", before: "roles:\n  product-owner:\n", changed: true, level: "cautious"},
		{name: "its settings null", before: "roles:\n  product-owner: {settings: null}\n", changed: true, level: "cautious"},
		{name: "other settings kept", before: "roles:\n  product-owner:\n    settings: {issues-per-run: 4}\n", changed: true, level: "cautious", keeps: "issues-per-run: 4"},
		{name: "a level set already", before: "roles:\n  product-owner:\n    settings: {autonomy: enterprising}\n", level: "enterprising"},
		{name: "roles a scalar", before: "roles: all\n", fails: true, keeps: "roles: all"},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo := t.TempDir()
			file := filepath.Join(repo, ".workline", "config.yaml")
			if c.before != "" {
				os.MkdirAll(filepath.Dir(file), 0o755)
				os.WriteFile(file, []byte(c.before), 0o644)
			}
			level, changed, err := SetAutonomy(repo, "cautious")
			data, _ := os.ReadFile(file)
			if c.fails {
				if err == nil {
					t.Errorf("no error; the file:\n%s", data)
				}
				if string(data) != c.before {
					t.Errorf("the file was changed:\n%s", data)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if changed != c.changed || level != c.level {
				t.Errorf("got %q %v, want %q %v", level, changed, c.level, c.changed)
			}
			var doc struct {
				AI    string `yaml:"ai"`
				Roles map[string]struct {
					Settings map[string]any `yaml:"settings"`
				} `yaml:"roles"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("the file no longer reads: %v\n%s", err, data)
			}
			if got := doc.Roles["product-owner"].Settings["autonomy"]; got != c.level {
				t.Errorf("autonomy reads %v, want %s:\n%s", got, c.level, data)
			}
			if !strings.Contains(string(data), c.keeps) {
				t.Errorf("lost %q:\n%s", c.keeps, data)
			}
		})
	}
}
