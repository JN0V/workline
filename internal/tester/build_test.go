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
	var missing, names, other []string
	for _, e := range buildErrorsIn(out, []string{"a/mul_test.go"}, rules) {
		if e.missing {
			missing = append(missing, e.line)
			names = append(names, e.name)
		} else {
			other = append(other, e.file)
		}
	}
	if len(missing) != 4 || !slices.Equal(other, []string{"a/mul_test.go", "a/a.go"}) {
		t.Errorf("missing names %q, other errors in %q", missing, other)
	}
	if !slices.Equal(names, []string{"Mul", "tester.Scale", "Scale", "Rate"}) {
		t.Errorf("the names read %q", names)
	}
	src := "package a\n\nfunc TestMulTwoByThree(t *testing.T) {}\nfunc helper(t *testing.T) {}\nfunc TestByZero(_ *testing.T) {}\n"
	if got := testNames(src, rules[0]); !slices.Equal(got, []string{"TestMulTwoByThree", "TestByZero"}) {
		t.Errorf("test names %q", got)
	}
	// A lint's finding (#988): golangci-lint's typecheck, the compiler's
	// lines in its message at the package's first file, or the finding
	// itself in a test file.
	missingName := missingNamed([]string{"a/mul_test.go"}, rules)
	for _, c := range []struct {
		path, message string
		want          bool
	}{
		{"a/a.go", ": # example.com/m/a [example.com/m/a.test]\na/mul_test.go:6:5: undefined: Mul", true},
		{"a/a.go", ": # example.com/m/a [example.com/m/a.test]\na/mul_test.go:6:5: undefined: Mul\na/a.go:3:1: undefined: Gone", false},
		{"a/a.go", ": # example.com/m/a\na/mul_test.go:10:2: syntax error: unexpected newline", false},
		{"a/mul_test.go", "undefined: Mul", true},
		{"a/a.go", "undefined: Mul", false},
		{"a/mul_test.go", "Error return value of `os.Remove` is not checked", false},
	} {
		if got := missingName(c.path, c.message); got != c.want {
			t.Errorf("missingNamed(%q, %q) = %v, want %v", c.path, c.message, got, c.want)
		}
	}
}

// A missing name read from a pattern with no group `name`: the first
// quoted word, as g++ quotes it, in a UTF-8 locale or in C (#1111).
func TestBuildErrorsQuotedName(t *testing.T) {
	rules, err := compileBuildErrors([]BuildErrors{{
		Error: `^(?P<file>[^\s:]+\.cpp):\d+:\d+: error: (?P<message>.+)$`,
		MissingName: []string{
			`^[‘'][\w:]+[’'] was not declared in this scope`,
			`^[‘'][\w:]+[’'] is not a member of [‘'][\w:]+[’']`,
			`^[‘'](?:const )?(?:class|struct) [^’']+[’'] has no member named [‘']\w+[’']`,
			`^no matching function`,
		},
		TestName: `(?m)^void\s+(test_\w+)\(`,
	}})
	if err != nil {
		t.Fatal(err)
	}
	out := "t/test_mqtt.cpp:12:5: error: ‘stubSetBrokerLink’ was not declared in this scope\n" +
		"t/test_mqtt.cpp:14:9: error: 'lostCount' is not a member of 'MQTTStatistics'\n" +
		"t/test_mqtt.cpp:15:9: error: ‘const struct MQTTStatistics’ has no member named ‘lostCount’\n" +
		"t/test_mqtt.cpp:16:9: error: no matching function for call to f(int)\n"
	var names []string
	for _, e := range buildErrorsIn(out, []string{"t/test_mqtt.cpp"}, rules) {
		if !e.missing {
			t.Errorf("not read as a missing name: %s", e.line)
		}
		names = append(names, e.name)
	}
	if !slices.Equal(names, []string{"stubSetBrokerLink", "lostCount", "lostCount", ""}) {
		t.Errorf("the names read %q", names)
	}
}

// A name the issue names as a whole word, or its last part (#1111).
func TestNamedIn(t *testing.T) {
	issue := "MQTT: count the lost messages\n\n## Scope\n\n- `MQTTStatistics`: a field `lostCount`.\n- calc.go: Mul beside Add."
	for _, c := range []struct {
		name string
		want bool
	}{
		{"lostCount", true},
		{"MQTTStatistics::lostCount", true},
		{"calc.Mul", true},
		{"Mul", true},
		{"stubSetBrokerLink", false},
		{"lost", true}, // "the lost messages": a word of the issue
		{"Count", false},
		{"Mu", false},
		{"", false},
	} {
		if got := namedIn(c.name, issue); got != c.want {
			t.Errorf("namedIn(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
