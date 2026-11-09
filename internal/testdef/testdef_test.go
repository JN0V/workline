package testdef

import "testing"

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
		"spec/cart_spec.rb":               false, // generic: no spec folder, a pack's
		"test_mul.py":                     false, // generic miss: pytest's root test_ file
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
	for p, want := range map[string]bool{"examples/a.go": true, "Tools/x.py": true, "tests/a.py": true, "src/a.go": false, "src/testing/a.go": false} {
		if got := d.Aside(p); got != want {
			t.Errorf("Aside(%q) = %v, want %v", p, got, want)
		}
	}
}
