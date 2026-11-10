package tester

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Go's compiler errors, as `go test` and gotestsum print them: each
// one a missing name only in a test of the change.
func TestBuildErrorsGo(t *testing.T) {
	rules := defaultBuildErrors(t)
	if len(rules) != 5 {
		t.Fatalf("role.yaml's build-errors: %d rows, want Go, Kotlin, Java, Gradle, pytest", len(rules))
	}
	rules = rules[:1]
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
	missingName := missingNamed([]string{"a/mul_test.go"}, rules, nil)
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

// defaultBuildErrors is the role's build-errors, as role.yaml ships them.
func defaultBuildErrors(t *testing.T) []buildRules {
	t.Helper()
	data, err := os.ReadFile("../../roles/tester/role.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var role struct {
		Settings struct {
			BuildErrors []struct {
				Error       string   `yaml:"error"`
				MissingName []string `yaml:"missing-name"`
				NameLine    string   `yaml:"name-line"`
				MessageLine string   `yaml:"message-line"`
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
	if err != nil {
		t.Fatalf("role.yaml's build-errors: %v", err)
	}
	return rules
}

// pytest 9.2's collection errors (#1283): the file on its header line,
// its message on the first `E` line under it, the frames between read
// as no error. A name the module lacks is missing; a module not found,
// or a syntax error, is not.
func TestBuildErrorsPytest(t *testing.T) {
	rules := defaultBuildErrors(t)
	out := "==================================== ERRORS ====================================\n" +
		"_____________________ ERROR collecting tests/test_calc.py ______________________\n" +
		"ImportError while importing test module '/tmp/x/tests/test_calc.py'.\n" +
		"Traceback:\n" +
		"/usr/lib/python3.12/importlib/__init__.py:90: in import_module\n" +
		"    return _bootstrap._gcd_import(name[level:], package, level)\n" +
		"tests/test_calc.py:1: in <module>\n" +
		"    from pkg.calc import add\n" +
		"E   ModuleNotFoundError: No module named 'pkg'\n" +
		"____________________ ERROR collecting tests/test_pricing.py ____________________\n" +
		"tests/test_pricing.py:1: in <module>\n" +
		"    from pkg.pricing import money\n" +
		"E   ImportError: cannot import name 'money' from 'pkg.pricing' (/tmp/x/pkg/pricing.py)\n" +
		"_ ERROR collecting tests/test_syntax.py _\n" +
		"E     File \"/tmp/x/tests/test_syntax.py\", line 3\n" +
		"E       def oops(\n" +
		"E   SyntaxError: '(' was never closed\n" +
		"=========================== short test summary info ============================\n" +
		"ERROR tests/test_calc.py\n" +
		"ERROR tests/test_pricing.py\n"
	type read struct {
		file, msg, name string
		missing         bool
	}
	var got []read
	errs := buildErrorsIn(out, []string{"tests/test_calc.py", "tests/test_pricing.py", "tests/test_syntax.py"}, rules)
	for _, e := range errs {
		got = append(got, read{e.file, e.msg, e.name, e.missing})
	}
	want := []read{
		{"tests/test_calc.py", "ModuleNotFoundError: No module named 'pkg'", "", false},
		{"tests/test_pricing.py", "ImportError: cannot import name 'money' from 'pkg.pricing' (/tmp/x/pkg/pricing.py)", "money", true},
		{"tests/test_syntax.py", "SyntaxError: '(' was never closed", "", false},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("read %+v, want %+v", got, want)
	}
	if !strings.HasSuffix(errs[0].line, "ERROR collecting tests/test_calc.py ______________________ E   ModuleNotFoundError: No module named 'pkg'") {
		t.Errorf("the error not quoted with its message: %q", errs[0].line)
	}
	if names := testNames("def helper():\n    pass\n\ndef test_money():\n    pass\n\nclass TestX:\n    async def test_async(self):\n        pass\n", rules[len(rules)-1]); !slices.Equal(names, []string{"test_money", "test_async"}) {
		t.Errorf("test names %v", names)
	}
}

// A message-line without its group is said.
func TestBuildErrorsMessageLineInvalid(t *testing.T) {
	_, err := compileBuildErrors([]BuildErrors{{Error: `^(?P<file>\S+) (?P<message>)$`, MessageLine: `^E (.+)$`}})
	if err == nil || !strings.Contains(err.Error(), "message-line") {
		t.Errorf("got %v", err)
	}
}

// The JVM's errors as Gradle 8.12 printed them on an Android project
// (#1177): Kotlin 2 and a .kts build script with file:// paths, javac
// and a Groovy build script with absolute ones, all under the checkout
// the tests ran in. Each is read from the repository's root.
func TestBuildErrorsJVM(t *testing.T) {
	rules := defaultBuildErrors(t)
	const root = "/tmp/workline-test-42"
	kt := "feature/timelapse/src/test/kotlin/io/x/ProbeMissingTest.kt"
	java := "feature/onboarding/src/test/java/probe/ProbeJavaTest.java"
	out := "> Task :feature:timelapse:compileDebugUnitTestKotlin FAILED\n" +
		"e: file://" + root + "/" + kt + ":10:25 Unresolved reference 'nextIndex'.\n" +
		"e: file://" + root + "/" + kt + ":16:27 Unresolved reference 'lostCount'.\n" +
		"e: file://" + root + "/" + kt + ":24:34 No parameter with name 'interval' found.\n" +
		"e: " + root + "/" + kt + ": (17, 11): Unresolved reference: pauseAll\n" +
		"e: file://" + root + "/" + kt + ":30:5 Syntax error: Expecting an element.\n" +
		"w: file://" + root + "/" + kt + ":31:5 Variable 'd' is never used.\n" +
		"e: file://" + root + "/feature/onboarding/build.gradle.kts:46:14: Unresolved reference: undefinedThing\n" +
		root + "/" + java + ":9: error: cannot find symbol\n" +
		"        assertEquals(0, Speeds.nextIndex());\n" +
		"                        ^\n" +
		"  symbol:   variable Speeds\n" +
		"  location: class ProbeJavaTest\n" +
		root + "/" + java + ":12: error: ';' expected\n" +
		"  " + root + "/" + java + ":9: error: cannot find symbol\n" +
		"Build file '" + root + "/build.gradle' line: 3\n"
	type read struct {
		file, name string
		missing    bool
	}
	var got []read
	for _, e := range buildErrorsIn(out, []string{kt, java}, rules, root) {
		got = append(got, read{e.file, e.name, e.missing})
	}
	want := []read{
		{kt, "nextIndex", true},
		{kt, "lostCount", true},
		{kt, "interval", true},
		{kt, "pauseAll", true},
		{kt, "", false},
		{"feature/onboarding/build.gradle.kts", "", false},
		{java, "Speeds", true},
		{java, "", false},
		{"build.gradle", "", false},
	}
	if !slices.Equal(got, want) {
		t.Errorf("read %+v\nwant %+v", got, want)
	}
	src := "class ProbeMissingTest {\n    @Test\n    fun `next index starts at zero`() {\n    }\n\n" +
		"    @Test fun memberMissing() = runTest {\n    }\n\n    @Test\n    @DisplayName(\"x\")\n    internal fun `a (b) c`() {}\n\n" +
		"    fun helper() {}\n}\n"
	if got := testNames(src, rules[1]); !slices.Equal(got, []string{"next index starts at zero", "memberMissing", "a (b) c"}) {
		t.Errorf("Kotlin test names %q", got)
	}
	jsrc := "public class FooTest {\n    @Test\n    void nextIndexStartsAtZero() {}\n    @Test\n    public void sums() throws Exception {}\n    void helper() {}\n}\n"
	if got := testNames(jsrc, rules[2]); !slices.Equal(got, []string{"nextIndexStartsAtZero", "sums"}) {
		t.Errorf("Java test names %q", got)
	}
	if got := testNames(src, rules[3]); got != nil {
		t.Errorf("a build file's row names no test, got %q", got)
	}
}

// A file:// path is a URI: its spaces and others escaped (#1177).
func TestBuildErrorsFileURI(t *testing.T) {
	rules := defaultBuildErrors(t)
	errs := buildErrorsIn("e: file:///tmp/my%20repo/a/src/test/kotlin/MulTest.kt:3:5 Unresolved reference 'mul'.",
		[]string{"a/src/test/kotlin/MulTest.kt"}, rules, "/tmp/my repo")
	if len(errs) != 1 || errs[0].file != "a/src/test/kotlin/MulTest.kt" || !errs[0].missing {
		t.Errorf("read %+v", errs)
	}
}

// A file from the root on Windows, its rules a parameter (#1183): a
// drive letter after file:///, backslashes, the drive's case.
func TestFromRootWindows(t *testing.T) {
	for _, c := range []struct {
		file    string
		roots   []string
		windows bool
		want    string
	}{
		{"file:///C:/w/t/a/MulTest.kt", []string{`C:\w\t`}, true, "a/MulTest.kt"},
		{`C:\w\t\a\MulTest.java`, []string{`C:\w\t`}, true, "a/MulTest.java"},
		{`c:\W\T\a\MulTest.java`, []string{`C:\w\t`}, true, "a/MulTest.java"},
		{"file:///C:/Users/my%20shop/a/X.kt", []string{`C:\Users\my shop`}, true, "a/X.kt"},
		{`D:\other\X.java`, []string{`C:\w\t`}, true, "D:/other/X.java"},
		{"/w/T/a/X.kt", []string{"/w/t"}, false, "/w/T/a/X.kt"},
		{"file:///w/t/a/X.kt", []string{"/w/t"}, false, "a/X.kt"},
		{"./a/X.kt", nil, false, "a/X.kt"},
	} {
		if got := fromRootOn(c.file, c.roots, c.windows); got != c.want {
			t.Errorf("fromRootOn(%q, %q, %v) = %q, want %q", c.file, c.roots, c.windows, got, c.want)
		}
	}
}

// A name-line with no group `name` is said before any run.
func TestBuildErrorsNameLineNeedsItsGroup(t *testing.T) {
	_, err := compileBuildErrors([]BuildErrors{{Error: `^(?P<file>\S+):(?P<message>.+)$`, NameLine: `symbol: (\w+)`}})
	if err == nil || !strings.Contains(err.Error(), "name-line") {
		t.Errorf("got %v", err)
	}
}

// javac's name on its symbol line, and paths with a space (#1183).
func TestBuildErrorsJavacNameAndSpaces(t *testing.T) {
	rules := defaultBuildErrors(t)
	const root = "/home/ana/my shop"
	java := "a/src/test/java/MulTest.java"
	kt := "a/src/test/kotlin/MulTest.kt"
	out := root + "/" + java + ":9: error: cannot find symbol\n" +
		"        assertEquals(6, Calc.mul(2, 3));\n" +
		"                            ^\n" +
		"  symbol:   method mul(int,int)\n" +
		"  location: class Calc\n" +
		root + "/" + java + ":10: error: cannot find symbol\n" +
		"  symbol:   variable Speeds\n" +
		root + "/" + java + ":11: error: cannot find symbol\n" +
		"  symbol: static nextIndex\n" +
		root + "/" + java + ":12: error: ';' expected\n" +
		"  symbol:   method unrelated()\n" +
		"e: " + root + "/" + kt + ": (3, 5): Unresolved reference: pauseAll\n"
	type read struct {
		file, name string
		missing    bool
	}
	var got []read
	for _, e := range buildErrorsIn(out, []string{java, kt}, rules, root) {
		got = append(got, read{e.file, e.name, e.missing})
	}
	want := []read{
		{java, "mul", true},
		{java, "Speeds", true},
		{java, "nextIndex", true},
		{java, "", false},
		{kt, "pauseAll", true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("read %+v\nwant %+v", got, want)
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

// The folder a build ran in is taken off its output: the same build
// broken the same way reads the same from one checkout to the next (#1274).
func TestUnrooted(t *testing.T) {
	out := "ImportError while importing test module '/tmp/workline-test-42/tests/test_a.py'.\n/private/tmp/workline-test-42/b.py:1: x"
	got := unrooted(out, []string{"/tmp/workline-test-42", "/private/tmp/workline-test-42", ""})
	if want := "ImportError while importing test module 'tests/test_a.py'.\nb.py:1: x"; got != want {
		t.Errorf("unrooted: %q, want %q", got, want)
	}
	uri := "e: file:///tmp/workline-test-42/src/test/kotlin/MulTest.kt:10:25 Too many arguments"
	if got, want := unrooted(uri, []string{"/tmp/workline-test-42"}), "e: src/test/kotlin/MulTest.kt:10:25 Too many arguments"; got != want {
		t.Errorf("unrooted: %q, want %q", got, want)
	}
}

// A compiler's line quoting an absolute path reads the same from one
// checkout to the next: the judge's facts quote it (#1274).
func TestBuildErrorLineUnrooted(t *testing.T) {
	kotlin := buildRules{error: regexp.MustCompile(`^e: (?P<file>.+?\.kts?):\d+:\d+ (?P<message>.*)$`)}
	javac := buildRules{error: regexp.MustCompile(`^(?P<file>\S.*?\.java):\d+: error: (?P<message>.*)$`)}
	var lines []string
	for _, root := range []string{"/tmp/workline-test-111", "/tmp/workline-test-222"} {
		out := "e: file://" + root + "/src/test/kotlin/MulTest.kt:10:25 Too many arguments\n" +
			root + "/src/test/java/AddTest.java:3: error: cannot find symbol"
		errs := buildErrorsIn(out, []string{"src/test/kotlin/MulTest.kt"}, []buildRules{kotlin, javac}, root)
		if len(errs) != 2 || errs[0].file != "src/test/kotlin/MulTest.kt" || !errs[0].inTest || errs[1].file != "src/test/java/AddTest.java" {
			t.Fatalf("errors from %s: %+v", root, errs)
		}
		lines = append(lines, errs[0].line+"\n"+errs[1].line)
	}
	want := "e: src/test/kotlin/MulTest.kt:10:25 Too many arguments\nsrc/test/java/AddTest.java:3: error: cannot find symbol"
	if lines[0] != want || lines[1] != want {
		t.Errorf("lines: %q, want %q twice", lines, want)
	}
}
