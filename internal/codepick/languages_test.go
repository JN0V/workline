package codepick

import (
	"slices"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/testdef"
)

// The words not searched are those of the project's own languages
// (#1109): `override`, `suspend`, `companion` in Kotlin, never searched;
// `open` a name in Python, a keyword in Kotlin.
func TestKeywordsByLanguage(t *testing.T) {
	kotlin, _ := New(repo(t, map[string]string{"src/main/kotlin/shop/Cart.kt": "class Cart {\n    fun total(): Long = 0\n}\n"}), "")
	if got := kotlin.codeNames("`override fun total()`, `suspend fun clear()` in its `companion object`; `open` it"); !slices.Equal(got, []string{"total", "clear"}) {
		t.Errorf("Kotlin: searched %v, want [total clear]", got)
	}
	python, _ := New(repo(t, map[string]string{"shop/cart.py": "def open(path):\n    pass\n"}), "")
	if got := python.codeNames("`open()` fails on `None`; `def` it again"); !slices.Equal(got, []string{"open"}) {
		t.Errorf("Python: searched %v, want [open]", got)
	}
	// `echo`, `export`, `source` are shell commands, as a script's own
	// function is one (#1189): searched; `local`, `then` never.
	shell, _ := New(repo(t, map[string]string{"bin/run.sh": "source ./env.sh\nexport PATH\necho ok\n"}), "")
	if got := shell.codeNames("`source` drops `export PATH`, `echo` says nothing; `local` and `then`"); !slices.Equal(got, []string{"source", "export", "echo"}) {
		t.Errorf("shell: searched %v, want [source export echo]", got)
	}
}

// The files beside a project's own code (#1109) are the project's test
// definition's, and its examples' and tools' (ADR-0065): a Gradle source
// set is a test by the project's test-paths, never by the engine's; a
// folder whose name only holds "test" is code.
func TestAsideByTheTestDefinition(t *testing.T) {
	gradle := testdef.From([]string{"**/src/test/**", "**/src/androidTest/**", "**/src/*Test/**", "**/src/testFixtures/**"}, nil)
	for f, want := range map[string][2]bool{ // generic, gradle
		"tests/a.py":                            {true, false},
		"test/test_ui.cpp":                      {true, false},
		"examples/a.go":                         {true, true},
		"internal/x/testdata/a.go":              {true, true},
		"src/test/kotlin/shop/CartTest.kt":      {true, true},
		"app/src/androidTest/java/a/UiTest.kt":  {false, true},
		"shared/src/commonTest/kotlin/a/T.kt":   {false, true},
		"lib/src/testFixtures/kotlin/a/Fake.kt": {false, true},
		"web/src/cart.test.ts":                  {true, false},
		"src/main/kotlin/shop/Cart.kt":          {false, false},
		"src/attestation/Check.kt":              {false, false},
		"contest/a.py":                          {false, false},
		"src/testing/helpers.go":                {false, false}, // a project's own test helpers are code (#1189)
		"src/testutil/fixtures.go":              {false, false},
	} {
		if got := (testdef.Def{}).Aside(f); got != want[0] {
			t.Errorf("generic: Aside(%q) = %v, want %v", f, got, want[0])
		}
		if got := gradle.Aside(f); got != want[1] {
			t.Errorf("gradle: Aside(%q) = %v, want %v", f, got, want[1])
		}
	}
}

// A test's fakes in a Gradle project's `src/test/` are copies beside the
// code, as those in `tests/` (#1109): the code's own definitions first,
// two copies, the rest counted.
func TestFakesInSrcTestAreAside(t *testing.T) {
	for _, dir := range []string{"tests/", "src/test/kotlin/shop/"} {
		_, task := pick(t, repo(t, map[string]string{
			"src/main/kotlin/shop/Cart.kt": "package shop\n\nclass Cart {\n    fun total(): Long {\n        return 1\n    }\n}\n",
			dir + "CartTest.kt":            "package shop\n\nprivate class FakeCart {\n    fun total(): Long {\n        return 2\n    }\n}\n",
			dir + "RatesTest.kt":           "package shop\n\nprivate class FakeRates {\n    fun total(): Long {\n        return 3\n    }\n}\n",
			dir + "OrderTest.kt":           "package shop\n\nprivate class FakeOrder {\n    fun total(): Long {\n        return 4\n    }\n}\n",
		}), Ask{Text: "`override fun total()` is wrong", Budget: 200})
		if want := "`total` (4 definitions, 1 of them not given, most in tests, examples or tools)"; !strings.Contains(task, want) {
			t.Errorf("%s: want %q in:\n%s", dir, want, task)
		}
		if i := strings.Index(task, "return 2"); i >= 0 && i < strings.Index(task, "return 1") {
			t.Errorf("%s: a copy before the code:\n%s", dir, task)
		}
	}
}

// A JVM file uses its package's types with no import (#1109): the files
// of its package whose type it names are one step away, a test's own
// package's too; one it never names, or of another package, is not.
func TestSamePackageOneStepAway(t *testing.T) {
	const pkg = "io/example/tracing/"
	dir := repo(t, map[string]string{
		"feature/src/main/kotlin/" + pkg + "TracingViewModel.kt":   "package io.example.tracing\n\nimport io.example.core.Storage\n\nclass TracingViewModel(val s: Storage) {\n    val state = TracingUiState()\n    fun clear() {\n        s.clear()\n    }\n}\n",
		"feature/src/main/kotlin/" + pkg + "TracingUiState.kt":     "package io.example.tracing\n\ndata class TracingUiState(val count: Int = 0)\n",
		"feature/src/main/kotlin/" + pkg + "Unused.kt":             "package io.example.tracing\n\nclass Unused\n",
		"feature/src/test/kotlin/" + pkg + "FakeStorage.kt":        "package io.example.tracing\n\nclass FakeStorage\n",
		"feature/src/test/kotlin/" + pkg + "TracingTest.kt":        "package io.example.tracing\n\nclass TracingTest {\n    val vm = TracingViewModel(FakeStorage())\n}\n",
		"core/src/main/kotlin/io/example/core/Storage.kt":          "package io.example.core\n\ninterface Storage {\n    fun clear()\n}\n",
		"other/src/main/kotlin/io/example/other/TracingUiState.kt": "package io.example.other\n\nclass TracingUiState\n",
		"java/src/main/java/io/example/shop/Cart.java":             "package io.example.shop;\n\npublic class Cart {\n    Amount total;\n}\n",
		"java/src/main/java/io/example/shop/Amount.java":           "package io.example.shop;\n\npublic record Amount(long cents) {}\n",
	})
	p, err := New(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for from, want := range map[string][]string{
		"feature/src/main/kotlin/" + pkg + "TracingViewModel.kt": {"core/src/main/kotlin/io/example/core/Storage.kt", "feature/src/main/kotlin/" + pkg + "TracingUiState.kt"},
		"feature/src/test/kotlin/" + pkg + "TracingTest.kt":      {"feature/src/test/kotlin/" + pkg + "FakeStorage.kt", "feature/src/main/kotlin/" + pkg + "TracingViewModel.kt"},
		"java/src/main/java/io/example/shop/Cart.java":           {"java/src/main/java/io/example/shop/Amount.java"},
	} {
		var got []string
		for _, l := range p.imports(from, nil) {
			got = append(got, l.file)
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s imports %v, want %v", from, got, want)
		}
	}
	pk, _ := pick(t, dir, Ask{Text: "`TracingViewModel` loses its count", Budget: 200})
	if want := "`feature/src/main/kotlin/" + pkg + "TracingUiState.kt` (in its package)"; !slices.Contains(pk.Linked, want) {
		t.Errorf("linked: %v, want %s", pk.Linked, want)
	}
}
