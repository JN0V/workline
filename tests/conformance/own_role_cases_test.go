package conformance

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Given workline's own line, When each role's cases move to
// `roles/<name>/cases/`, Then the tester's settings map those folders to
// the conformance package, and the count of cases the docs quote reads
// both places, the roles' folders and `tests/conformance/cases`.
func TestOwnRoleCasesRunInTheConformanceSuite(t *testing.T) {
	data, err := os.ReadFile("../../.workline/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Roles map[string]struct {
			Settings map[string]any `yaml:"settings"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	tester, err := yaml.Marshal(cfg.Roles["tester"].Settings)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"roles/*/cases/**", "./tests/conformance"} {
		if !strings.Contains(string(tester), want) {
			t.Errorf("the tester's settings do not hold %q: a case in a role's folder is mapped to no package", want)
		}
	}
	derive, _ := cfg.Roles["documentalist"].Settings["derive"].(map[string]any)
	count, _ := derive["conformance-cases"].(string)
	for _, want := range []string{"roles/*/cases/*.yaml", "tests/conformance/cases/*/*.yaml"} {
		if !strings.Contains(count, want) {
			t.Errorf("the conformance-cases count %q does not read %q", count, want)
		}
	}
}
