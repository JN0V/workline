package codepick

import (
	"slices"
	"strings"
	"testing"
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

// The folders beside a project's own code, by convention (#1109): a
// Gradle source set (`src/test/`, `src/androidTest/`), `__tests__/`, a
// .NET test project; never a folder whose name only holds "test".
func TestAsideByConvention(t *testing.T) {
	for f, want := range map[string]bool{
		"tests/a.py":                            true,
		"test/test_ui.cpp":                      true,
		"examples/a.go":                         true,
		"internal/x/testdata/a.go":              true,
		"src/test/kotlin/shop/CartTest.kt":      true,
		"app/src/androidTest/java/a/UiTest.kt":  true,
		"lib/src/testFixtures/kotlin/a/Fake.kt": true,
		"shared/src/commonTest/kotlin/a/T.kt":   true,
		"web/src/__tests__/cart.test.ts":        true,
		"web/src/__mocks__/api.ts":              true,
		"Shop.Tests/CartTests.cs":               true,
		"src/main/kotlin/shop/Cart.kt":          false,
		"src/attestation/Check.kt":              false,
		"src/latest/a.go":                       false,
		"internal/tester/write.go":              false,
		"contest/a.py":                          false,
		"app/src/jvmTest/kotlin/a/T.kt":         true,
		"app/src/androidUnitTest/kotlin/a/T.kt": true,
		"app/src/integrationTest/java/a/T.java": true,
		"app/src/watchosArm64Test/kotlin/T.kt":  true,
		"app/src/tvosTest/kotlin/T.kt":          true,
		"app/src/wasmTest/kotlin/T.kt":          true,
		"app/src/testFreeDebug/java/T.java":     true,
		"app/src/androidTestDebug/java/T.java":  true,
		"app/src/main/java/a/Contest.java":      false,
		"src/testing/helpers.go":                false, // a project's own test helpers are code (#1189)
		"src/testutil/fixtures.go":              false,
		"src/testFoo/a.go":                      false,
		"src/fooTestBar/a.go":                   false,
	} {
		if got := aside(f); got != want {
			t.Errorf("aside(%q) = %v, want %v", f, got, want)
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
