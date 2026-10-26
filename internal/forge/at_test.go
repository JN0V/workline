package forge

import "testing"

// A comment written by a later run is newer; one naming no run, a
// machine's or an older engine's, never is.
func TestAtNewer(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "")
	t.Setenv("CI_PIPELINE_ID", "102")
	b := At("b2")
	t.Setenv("CI_PIPELINE_ID", "101")
	a := At("a1")
	t.Setenv("CI_PIPELINE_ID", "")
	none := At("c3")
	if b != "<!-- workline:at head=b2 run=102 -->" || none != "<!-- workline:at head=c3 -->" {
		t.Fatalf("stamps: %q %q", b, none)
	}
	for _, c := range []struct {
		found, body string
		newer       bool
	}{{b, a, true}, {a, b, false}, {a, a, false}, {none, a, false}, {b, none, false}, {"", a, false}} {
		if got := Newer("text\n"+c.found+"\nrecord", c.body); got != c.newer {
			t.Errorf("Newer(%q, %q) = %v", c.found, c.body, got)
		}
	}
}
