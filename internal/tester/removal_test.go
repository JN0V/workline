package tester

import (
	"strings"
	"testing"
)

const goBase = `package calc

import (
	"strings"
	"testing"
)

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Fatal("2 + 3")
	}
}

func TestAddAgain(t *testing.T) {
	if Add(1, 1) != 2 {
		t.Fatal("1 + 1")
	}
}

func TestTrim(t *testing.T) {
	if strings.TrimSpace(" a ") != "a" {
		t.Fatal("trim")
	}
}
`

const shBase = "# covers: 1\n. ./lib.sh\n[ \"$(add 2 3)\" = 5 ]\n[ \"$(add 1 1)\" = 2 ]\n"

func TestKeepsBase(t *testing.T) {
	cut := func(s string, parts ...string) string {
		for _, p := range parts {
			s = strings.Replace(s, p, "", 1)
		}
		return s
	}
	for _, c := range []struct {
		name, file, now, was string
		isNew, ok           bool
	}{
		{"a whole Go test removed", "calc_test.go", cut(goBase, "func TestAddAgain(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"1 + 1\")\n\t}\n}\n"), goBase, false, true},
		{"a Go test and its import removed", "calc_test.go", cut(goBase, "\t\"strings\"\n", "func TestTrim(t *testing.T) {\n\tif strings.TrimSpace(\" a \") != \"a\" {\n\t\tt.Fatal(\"trim\")\n\t}\n}\n"), goBase, false, true},
		{"two Go tests merged by deleting a header", "calc_test.go", cut(goBase, "}\n\nfunc TestAddAgain(t *testing.T) {\n"), goBase, false, false},
		{"a Go test rewritten", "calc_test.go", strings.Replace(goBase, "Add(2, 3) != 5", "Add(2, 3) != 5 || Add(1, 1) != 2", 1), goBase, false, false},
		{"Go tests reordered", "calc_test.go", strings.Replace(cut(goBase, "func TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"2 + 3\")\n\t}\n}\n"), "func TestTrim", "func TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"2 + 3\")\n\t}\n}\n\nfunc TestTrim", 1), goBase, false, true},
		{"a Go file with CRLF line ends", "calc_test.go", strings.ReplaceAll(goBase, "\n", "\r\n"), goBase, false, true},
		{"a Go file that no longer parses", "calc_test.go", cut(goBase, "func TestAdd(t *testing.T) {\n"), goBase, false, false},
		{"a new file", "new_test.go", goBase, "", true, false},
		{"a shell line removed", "test_add.sh", cut(shBase, "[ \"$(add 1 1)\" = 2 ]\n"), shBase, false, true},
		{"a shell line written", "test_add.sh", shBase + "[ \"$(add 0 0)\" = 0 ]\n", shBase, false, false},
		{"shell lines reordered", "test_add.sh", "# covers: 1\n. ./lib.sh\n[ \"$(add 1 1)\" = 2 ]\n[ \"$(add 2 3)\" = 5 ]\n", shBase, false, false},
		{"a shell file with CRLF line ends", "test_add.sh", strings.ReplaceAll(shBase, "\n", "\r\n"), shBase, false, true},
	} {
		if what, ok := keepsBase(c.file, c.now, c.was, c.isNew); ok != c.ok {
			t.Errorf("%s: keepsBase = %v (%s), want %v", c.name, ok, what, c.ok)
		}
	}
}
