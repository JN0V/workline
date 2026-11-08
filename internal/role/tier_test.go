package role

import "testing"

// A size tier multiplies the size targets alone, int or float64 as YAML
// and JSON read them; 0, no cap, stays 0; the rest is left as it is.
func TestTiered(t *testing.T) {
	in := map[string]any{"ai-max-tokens": 300000, "code-lines-max": float64(1500), "diff-lines-max": 0, "ai-max-seconds": 600, "ai-max-turns": 40}
	got := Tiered(in, 2)
	want := map[string]any{"ai-max-tokens": 600000, "code-lines-max": float64(3000), "diff-lines-max": 0, "ai-max-seconds": 600, "ai-max-turns": 40}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v (%T), want %v (%T)", k, got[k], got[k], v, v)
		}
	}
	if _, ok := got["code-bytes-max"]; ok {
		t.Error("a setting not set was added")
	}
	if in["ai-max-tokens"] != 300000 {
		t.Error("the settings given were changed")
	}
	if same := Tiered(in, 1); same["ai-max-tokens"] != 300000 {
		t.Error("factor 1 changed a setting")
	}
}
