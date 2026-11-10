package tester

import (
	"slices"
	"testing"
)

func TestUncached(t *testing.T) {
	for _, c := range []struct {
		env  []string
		want string
	}{
		{[]string{"HOME=/h"}, "GOFLAGS=-count=1"},
		{[]string{"GOFLAGS=-mod=mod", "HOME=/h"}, "GOFLAGS=-mod=mod -count=1"},
		{[]string{"GOFLAGS=-count=2"}, "GOFLAGS=-count=2"},
	} {
		got := uncached(c.env)
		if n := len(slices.DeleteFunc(slices.Clone(got), func(e string) bool { return e != c.want })); n != 1 {
			t.Errorf("uncached(%q) = %q, want one %q", c.env, got, c.want)
		}
		if slices.Contains(c.env, "HOME=/h") && !slices.Contains(got, "HOME=/h") {
			t.Errorf("uncached(%q) lost HOME: %q", c.env, got)
		}
	}
}

func TestFrozenIn(t *testing.T) {
	r := record{Files: map[string]string{"calc/mul_test.go": "a1", "tests/test_x.py": "b1"}}
	ids := map[string]bool{
		"calc.TestMul": true, "calc.TestMulBig": true, "calc.TestMul/zero": true,
		"tests.test_x.test_y[1]": true, "tests.test_x.test_y_more": true, "rust::mod::test_z": true, "other.TestNew": true,
	}
	for _, c := range []struct {
		test string
		want []string
	}{
		{"calc.TestMulBig", []string{"calc.TestMulBig"}},                            // a report's id: itself
		{"calc/mul_test.go:TestMul", []string{"calc.TestMul", "calc.TestMul/zero"}}, // its subtest too
		{"tests/test_x.py:test_y", []string{"tests.test_x.test_y[1]"}},              // a parameter's mark
		{"calc/mul_test.go:TestDiv", nil},                                           // missing
		{"other/file.go:TestMul", nil},                                              // not a frozen file: no loose match
		{"calc/mul_test.go:TestNew", nil},                                           // another folder's TestNew is not it
		{"rust::mod::test_q", nil},                                                  // a report's id, not there
	} {
		if got := r.frozenIn(c.test, ids); !slices.Equal(got, c.want) {
			t.Errorf("frozenIn(%q) = %q, want %q", c.test, got, c.want)
		}
	}
}

// A test frozen by its file alone (#1293) is met by each test of that
// file, never by another file's.
func TestFrozenInFileAlone(t *testing.T) {
	r := record{Files: map[string]string{"tests/test_mul.c": "a1", "tests/test-div.sh": "b1"}}
	ids := map[string]bool{
		"test_mul.main": true, "tests.test_mul.big": true, "test_mul_more.main": true, "other.test_mul.x": true,
		"tests.test-div": true, "tests.test-div-big": true, "tests.test-div.big.case": true, "test_add.main": true,
	}
	for _, c := range []struct {
		test string
		want []string
	}{
		{"tests/test_mul.c", []string{"test_mul.main", "tests.test_mul.big"}},
		{"tests/test-div.sh", []string{"tests.test-div", "tests.test-div.big.case"}}, // a stem of two words; never test-div-big's
		{"tests/test_sub.c", nil}, // not a frozen file
	} {
		if got := r.frozenIn(c.test, ids); !slices.Equal(got, c.want) {
			t.Errorf("frozenIn(%q) = %q, want %q", c.test, got, c.want)
		}
	}
}

func TestQualifiedBy(t *testing.T) {
	for _, c := range []struct {
		qualifier, file string
		want            bool
	}{
		{"example.com/app/tester", "tester/x_test.go", true},      // the file's package
		{"example.com/app/tester/sub", "tester/x_test.go", false}, // a nested folder's (#1269)
		{"tests", "tests/test_x.sh", true},                        // a report's class: the folder
		{"tests.test_x", "tests/test_x.py", true},                 // a module
		{"tests.test_x.TestK", "tests/test_x.py", true},           // a class in it
		{"test_x", "tests/test_x.py", true},                       // run from inside its folder
		{"tests.sub.test_x", "tests/test_x.py", false},            // a namesake file in a nested folder
		{"com.example.CalcTest", "src/test/java/com/example/CalcTest.java", true},
		{"com.other.CalcTest", "src/test/java/com/example/CalcTest.java", false}, // another package's class
		{"com.example.CalcTest", "src/test/kotlin/CalcTest.kt", true},            // Kotlin: the root package left out
		{"com.example.calc.CalcTest", "src/test/kotlin/calc/CalcTest.kt", true},
		{"Calc.Tests.CalcTests", "tests/Calc.Tests/CalcTests.cs", true}, // .NET: the folder is the namespace
		{"MyApp.Tests.CalcTests", "tests/UnitTests/CalcTests.cs", true}, // a namespace apart from the folders
		{"example.com/app/internal/tester", "internal/tester/x_test.go", true},
		{"example.com/app/cmd/tester", "internal/tester/x_test.go", false}, // a same-named folder elsewhere
		{"tester", "internal/tester/x_test.go", true},                      // a runner's folder, short
		{"pkg.tests.test_x", "tests/test_x.py", false},                     // a same-named folder elsewhere
		{"test_x", "test_x.py", true},                                      // at the root
		{"pkg.test_x", "test_x.py", false},                                 // another folder's namesake
		{"other", "calc/mul_test.go", false},                               // another folder
		{"", "calc/mul_test.go", false},
	} {
		if got := qualifiedBy(c.qualifier, c.file); got != c.want {
			t.Errorf("qualifiedBy(%q, %q) = %v, want %v", c.qualifier, c.file, got, c.want)
		}
	}
}

func TestWithTests(t *testing.T) {
	r := record{Tests: []string{"a.A.one", "b.B.two", "B.kt:three"}}
	got := r.withTests("B.kt", map[string]result{"b.B.two": failed, "base.old": passed}, []string{"b.B.twice"}, nil)
	if want := []string{"a.A.one", "b.B.twice"}; !slices.Equal(got.Tests, want) {
		t.Errorf("withTests = %q, want %q", got.Tests, want)
	}
}

func TestOfFileStemOfWords(t *testing.T) {
	for _, c := range []struct {
		id, file string
		want     bool
	}{
		{"tests.test-mul", "tests/test-mul.sh", true},
		{"tests.test-mul.case one", "tests/test-mul.sh", true},
		{"tests.test-mul-big", "tests/test-mul.sh", false}, // another file's
		{"tests.big-test-mul", "tests/test-mul.sh", false},
		{"tests.test_mul_big", "tests/test_mul.c", false},
	} {
		if got := ofFile(c.id, c.file); got != c.want {
			t.Errorf("ofFile(%q, %q) = %v, want %v", c.id, c.file, got, c.want)
		}
	}
}
