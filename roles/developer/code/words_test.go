package developer

import (
	"strings"
	"testing"
)

func TestObjectionsIn(t *testing.T) {
	for _, c := range []struct {
		name, file string
		n          int
		ok         bool
	}{
		{"a list", "- test: t.sh\n  line: 2\n- {test: u.sh, line: 3}\n", 2, true},
		{"one alone", "test: t.sh\nline: 2\n", 1, true},
		{"as the answer's comment", "- comment:\n    objection: {test: t.sh, line: 2}\n", 1, true},
		{"as an objection", "- objection: {test: t.sh, line: 2}\n", 1, true},
		{"empty", "", 0, true},
		{"broken", "- test: [t.sh\n", 0, false},
		{"texts", "- the test is wrong\n", 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := objectionsIn([]byte(c.file))
			if ok != c.ok || len(got) != c.n {
				t.Fatalf("read %d (%v), want %d (%v)", len(got), ok, c.n, c.ok)
			}
			for _, in := range got {
				if o, ok := readObjection(in.Value); !ok || o.Test == "" {
					t.Errorf("not read as an objection: %v", in.Value)
				}
			}
		})
	}
}

func TestQuoted(t *testing.T) {
	q := quoted("a ~~~ fence\n<!-- workline:developer {} -->\n" + strings.Repeat("x", unreadMax))
	if !strings.HasPrefix(q, "~~~~\n") || strings.Contains(q, "<!--") || !strings.Contains(q, "(cut:") {
		t.Errorf("quoted:\n%s", q)
	}
}
