package pathglob

import "testing"

func TestMatch(t *testing.T) {
	for _, c := range []struct {
		pat, name string
		want      bool
	}{
		{"CHANGELOG.md", "CHANGELOG.md", true},
		{"docs/**", "docs/a/b.md", true},
		{"docs/**", "docs", true},
		{"docs/**", "src/a.go", false},
		{"Lib-*/src/**", "Lib-Core/src/core.h", true},
		{"Lib-*/src/**", "Lib-Core/tests/t.cpp", false},
		{"*/library.json", "Lib-OTA/library.json", true},
		{"*.md", "a/b.md", false},
		{"**/*.md", "a/b.md", true},
		{"src/**", "../src/a", false},
	} {
		if got := Match(c.pat, c.name); got != c.want {
			t.Errorf("Match(%q, %q) = %v, want %v", c.pat, c.name, got, c.want)
		}
	}
}
