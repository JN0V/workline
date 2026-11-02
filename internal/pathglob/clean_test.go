package pathglob_test

import (
	"testing"

	"github.com/JN0V/workline/internal/pathglob"
)

func TestMatchCleansPattern(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"./docs/archive/**", "docs/archive/old.md", true},
		{"docs//archive/**", "docs/archive/old.md", true},
		{"./*.md", "README.md", true},
		{"./docs/archive/**", "src/a.go", false},
	}
	for _, tt := range tests {
		if got := pathglob.Match(tt.pattern, tt.name); got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}
