package reviewer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// loadShipped reads the shipped role's settings as the engine hands them to
// the role, with overrides merged on top.
func loadShipped(t *testing.T, over map[string]any) Settings {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "roles", "reviewer", "role.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Settings map[string]any `yaml:"settings"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for k, v := range over {
		doc.Settings[k] = v
	}
	js, _ := json.Marshal(doc.Settings)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "in", "settings.json"), js, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := settings(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func asked(t *testing.T, s Settings, event string, ready bool) ([]string, bool) {
	t.Helper()
	t.Setenv("WORKLINE_EVENT", event)
	got, facet, err := lenses(t.TempDir(), s, Record{}, map[string]string{"intent": "no issue"}, ready)
	if err != nil {
		t.Fatal(err)
	}
	return got, facet
}

func TestShippedFacetsAtEachMoment(t *testing.T) {
	s := loadShipped(t, nil)
	every := []string{"correctness", "edge-cases", "tests", "claims"} // intent has nothing to read
	if got, f := asked(t, s, "review", false); !slices.Equal(got, every) || f {
		t.Errorf("review: %v %v", got, f)
	}
	if got, f := asked(t, s, "merge-request", false); !slices.Equal(got, []string{"correctness"}) || f {
		t.Errorf("push: %v %v", got, f)
	}
	if got, f := asked(t, s, "merge-request", true); !slices.Equal(got, every) || f {
		t.Errorf("ready: %v %v", got, f)
	}
	if s.LensesPerPush != 1 || s.DiffAlone || s.FacetsByMoment != nil {
		t.Errorf("defaults: %+v", s)
	}
}

func TestFacetsByMomentPushDiffAlone(t *testing.T) {
	s := loadShipped(t, map[string]any{"facets-by-moment": map[string]any{"merge-request-push": []string{"diff-alone"}}})
	every := []string{"correctness", "edge-cases", "tests", "claims"}
	if got, f := asked(t, s, "merge-request", false); len(got) != 0 || !f {
		t.Errorf("push: %v %v, want the diff alone", got, f)
	}
	if got, f := asked(t, s, "review", false); !slices.Equal(got, every) || f {
		t.Errorf("review: %v %v", got, f)
	}
	if got, f := asked(t, s, "merge-request", true); !slices.Equal(got, every) || f {
		t.Errorf("ready: %v %v", got, f)
	}
}

func TestFacetsByMomentRefused(t *testing.T) {
	for want, by := range map[string]map[string][]string{
		"not one of the lenses": {"review": {"nonsense"}},
		"is not a moment":       {"on-friday": {"correctness"}},
		"names no facet":        {"merge-request-push": {}},
	} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "in"), 0o755); err != nil {
			t.Fatal(err)
		}
		js, _ := json.Marshal(map[string]any{"lenses": []string{"correctness"}, "facets-by-moment": by})
		if err := os.WriteFile(filepath.Join(dir, "in", "settings.json"), js, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := settings(dir); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: %v, want %q", by, err, want)
		}
	}
}
