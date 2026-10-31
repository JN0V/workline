package routing

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/role"
)

// configured is a repository holding config as its .workline/config.yaml.
func configured(t *testing.T, config string) string {
	t.Helper()
	repo := t.TempDir()
	if config == "" {
		return repo
	}
	if err := os.MkdirAll(filepath.Join(repo, ".workline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".workline", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

func shipped(t *testing.T, config string) *Config {
	t.Helper()
	cfg, err := Load(configured(t, config))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// Each row of the shipped table: the label says the state, the state wakes
// its role where the project routes it on issue, and names the next.
func TestStatesEveryRow(t *testing.T) {
	cfg := shipped(t, "routing: {events: {issue: [tester, developer]}}\n")
	for _, c := range []struct {
		label, state, role, next string
	}{
		{"workline:blocked", "blocked", "", ""},
		{"workline:developing", "doing", "", ""},
		{"workline:ready", "ready", "developer", "doing"},
		{"workline:needs-tests", "tests", "tester", "ready"},
		{"workline:to-refine", "refine", "", "tests"}, // the product owner, not routed on issue
	} {
		s, role, why := cfg.Wakes([]string{"bug", c.label}, "issue")
		if s.Name != c.state || role != c.role || s.Next != c.next {
			t.Errorf("%s: state %q wakes %q, next %q (%s); want %q, %q, %q", c.label, s.Name, role, s.Next, why, c.state, c.role, c.next)
		}
		if role == "" && why == "" {
			t.Errorf("%s wakes nobody and says not why", c.label)
		}
	}
	if s, role, why := cfg.Wakes([]string{"bug"}, "issue"); s.Name != "" || role != "" || !strings.Contains(why, "no state") {
		t.Errorf("an issue with no state's label: %q %q %q", s.Name, role, why)
	}
}

// Two labels: the table's order settles it, Blocked and the claim first.
func TestStatesOrderSettles(t *testing.T) {
	cfg := shipped(t, "")
	for labels, want := range map[string]string{
		"workline:ready workline:developing":  "doing",
		"workline:ready workline:blocked":     "blocked",
		"workline:to-refine workline:ready":   "ready",
		"workline:needs-tests workline:ready": "ready",
	} {
		if s, _ := cfg.StateOf(strings.Fields(labels)); s.Name != want {
			t.Errorf("%s: %q, want %q", labels, s.Name, want)
		}
	}
}

// A state is a role's only where the project routes it: the shipped
// line routes no role on issue.
func TestStatesWakeOnlyRouted(t *testing.T) {
	cfg := shipped(t, "")
	if _, role, why := cfg.Wakes([]string{"workline:ready"}, "issue"); role != "" || !strings.Contains(why, "does not route on issue") {
		t.Errorf("ready, nothing routed: %q (%s)", role, why)
	}
}

// A move puts the state's label on and every other state's off, one call.
func TestStatesMoveExclusive(t *testing.T) {
	cfg := shipped(t, "")
	add, remove, err := cfg.Move("ready")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(add, []string{"workline:ready"}) || len(remove) != len(cfg.States)-1 || slices.Contains(remove, "workline:ready") {
		t.Errorf("move to ready: +%v -%v", add, remove)
	}
	if _, _, err := cfg.Move("done"); err == nil {
		t.Error("a move to no state")
	}
}

// A project's table replaces the shipped one; one the line cannot read is
// refused as a config error.
func TestStatesProjectTable(t *testing.T) {
	cfg := shipped(t, "routing: {states: [{name: a, label: x}, {name: b, label: y, role: tester, next: a}], max-hops: 3}\n")
	if len(cfg.States) != 2 || cfg.MaxHops != 3 {
		t.Errorf("the project's table: %+v, max-hops %d", cfg.States, cfg.MaxHops)
	}
	if shipped(t, "").MaxHops != 8 {
		t.Error("the shipped max-hops is not 8")
	}
	for _, bad := range []string{
		"routing: {states: [{name: a}]}\n",
		"routing: {states: [{name: a, label: x}, {name: a, label: y}]}\n",
		"routing: {states: [{name: a, label: x}, {name: b, label: x}]}\n",
		"routing: {states: [{name: a, label: x, next: z}]}\n",
	} {
		if _, err := Load(configured(t, bad)); !role.IsConfigError(err) {
			t.Errorf("%s: %v, want a config error", bad, err)
		}
	}
}
