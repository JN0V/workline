package testdef

import (
	"testing"

	"github.com/JN0V/workline/internal/packs"
)

// The generic definition: test folders and test names, never a setting
// file opening on a dot, never a word that only holds the letters.
func TestGenericTest(t *testing.T) {
	d := Def{}
	for p, want := range map[string]bool{
		"mul_test.go":                     true,
		"web/mul.spec.ts":                 true,
		"web/mul.test.js":                 true,
		"tests/helpers.rs":                true,
		"tests/conftest.py":               true,
		"app/src/test/kotlin/a/Fake.kt":   true,
		"spec/cart_spec.rb":               true,
		"test_mul.py":                     true,
		"web/__tests__/cart.js":           true,
		"Sources/Tests/CartTests.swift":   true,
		"spec/support/helpers.rb":         false, // generic: no spec folder, a pack's
		"conftest.py":                     false, // a pack's
		"src/CartTest.kt":                 false, // a pack's
		"app/src/androidTest/kotlin/A.kt": false,
		"latest.go":                       false,
		"docs/specification.md":           false,
		".env.test":                       false,
		".env.test.local":                 false,
		"src/attestation.go":              false,
	} {
		if got := d.Test(p); got != want {
			t.Errorf("Test(%q) = %v, want %v", p, got, want)
		}
	}
}

// The project's test-paths win; its homes map a test's folder onto its
// code's.
func TestProjectDefinitionWins(t *testing.T) {
	d := From([]string{"**/spec/**", "**/src/test/**"}, []Home{{Tests: "**/src/test/**", Code: "**/src/main/**"}})
	if !d.Test("spec/cart_spec.rb") || d.Test("tests/helpers.rs") {
		t.Errorf("test-paths set: spec/ is a test, tests/ is not; got %v, %v", d.Test("spec/cart_spec.rb"), d.Test("tests/helpers.rs"))
	}
	for p, want := range map[string]string{
		"app/src/test/kotlin/tg/x/FakeRepo.kt": "app/src/main/kotlin/tg/x",
		"src/test/A.kt":                        "src/main",
		"pkg/foo/foo_test.go":                  "pkg/foo", // in no home: its own folder
	} {
		if got := d.Home(p); got != want {
			t.Errorf("Home(%q) = %q, want %q", p, got, want)
		}
	}
}

// The generic homes: the folder holding the first test folder.
func TestGenericHome(t *testing.T) {
	for p, want := range map[string]string{
		"Storage/test/a/test_a.cpp":           "Storage",
		"tests/test_x.py":                     ".",
		"feature/x/src/test/kotlin/a/Fake.kt": "feature/x/src",
		"pkg/foo/new_test.go":                 "pkg/foo",
	} {
		if got := (Def{}).Home(p); got != want {
			t.Errorf("Home(%q) = %q, want %q", p, got, want)
		}
	}
}

func TestFromSettings(t *testing.T) {
	d, err := FromSettings(map[string]any{"test-paths": []any{"**/spec/**"}, "test-homes": []any{map[string]any{"tests": "**/spec/**", "code": "**/lib/**"}}})
	if err != nil || !d.PathsSet || !d.HomesSet || d.Home("spec/models/cart_spec.rb") != "lib/models" {
		t.Errorf("FromSettings = %+v, %v", d, err)
	}
	d, err = FromSettings(map[string]any{})
	if err != nil || d.PathsSet || !d.Test("tests/x.py") {
		t.Errorf("nothing set: the generic one; got %+v, %v", d, err)
	}
	if _, err := FromSettings(map[string]any{"test-homes": []any{map[string]any{"tests": "src/test/*", "code": "**/src/main/**"}}}); err == nil {
		t.Errorf("a code with more ** than its tests: no error")
	}
}

func TestAside(t *testing.T) {
	d := Def{}
	for p, want := range map[string]bool{"examples/a.go": true, "Tools/x.py": true, "tests/a.py": true, "src/a.go": false, "src/testing/a.go": false,
		"app/src/androidTest/java/a/UiTest.kt": true, "shared/src/commonTest/kotlin/T.kt": true, "lib/src/testFixtures/kotlin/F.kt": true,
		"app/src/testFreeDebug/java/T.java": true, "Shop.Tests/CartTests.cs": true, "web/src/__mocks__/api.ts": true,
		"src/testFoo/a.go": false, "src/fooTestBar/a.go": false, "src/attestation/Check.kt": false} {
		if got := d.Aside(p); got != want {
			t.Errorf("Aside(%q) = %v, want %v", p, got, want)
		}
	}
}

// Every built-in pack's test-homes can be read, and kotlin-gradle's maps
// a test onto its module's main source set.
func TestBuiltinPackHomes(t *testing.T) {
	for n, p := range packs.Builtin() {
		if err := Check(p.TestHomes); err != nil {
			t.Errorf("pack %s: %v", n, err)
		}
	}
	k := packs.Builtin()["kotlin-gradle"]
	d := From(k.TestPaths, k.TestHomes)
	if got := d.Home("app/src/test/kotlin/a/CartTest.kt"); got != "app/src/main/kotlin/a" {
		t.Errorf("Home = %q", got)
	}
	if !d.Test("app/src/androidTest/kotlin/A.kt") || d.Test("app/src/main/kotlin/A.kt") {
		t.Error("kotlin-gradle test-paths")
	}
}
