package tester

import (
	"os"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Go's compiler errors, as `go test` and gotestsum print them: each
// one a missing name only in a test of the change.
func TestBuildErrorsGo(t *testing.T) {
	data, err := os.ReadFile("../../roles/tester/role.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var role struct {
		Settings struct {
			BuildErrors []struct {
				Error       string   `yaml:"error"`
				MissingName []string `yaml:"missing-name"`
				TestName    string   `yaml:"test-name"`
			} `yaml:"build-errors"`
		} `yaml:"settings"`
	}
	if err := yaml.Unmarshal(data, &role); err != nil {
		t.Fatal(err)
	}
	var list []BuildErrors
	for _, b := range role.Settings.BuildErrors {
		list = append(list, BuildErrors(b))
	}
	rules, err := compileBuildErrors(list)
	if err != nil || len(rules) != 1 {
		t.Fatalf("role.yaml's build-errors: %v, %d", err, len(rules))
	}
	out := "# example.com/m/a [example.com/m/a.test]\n" +
		"a/mul_test.go:6:5: undefined: Mul\n" +
		"./a/mul_test.go:7:9: undefined: tester.Scale\n" +
		"a/mul_test.go:8:4: v.Scale undefined (type *T has no field or method Scale)\n" +
		"a/mul_test.go:9:12: unknown field Rate in struct literal of type T\n" +
		"a/mul_test.go:10:2: syntax error: unexpected newline\n" +
		"a/a.go:3:1: undefined: Gone\n" +
		"    mul_test.go:12: expected 6, got 0\n" +
		"FAIL\texample.com/m/a [build failed]\n"
	var missing, other []string
	for _, e := range buildErrorsIn(out, []string{"a/mul_test.go"}, rules) {
		if e.missing {
			missing = append(missing, e.line)
		} else {
			other = append(other, e.file)
		}
	}
	if len(missing) != 4 || !slices.Equal(other, []string{"a/mul_test.go", "a/a.go"}) {
		t.Errorf("missing names %q, other errors in %q", missing, other)
	}
	src := "package a\n\nfunc TestMulTwoByThree(t *testing.T) {}\nfunc helper(t *testing.T) {}\nfunc TestByZero(_ *testing.T) {}\n"
	if got := testNames(src, rules[0]); !slices.Equal(got, []string{"TestMulTwoByThree", "TestByZero"}) {
		t.Errorf("test names %q", got)
	}
}
