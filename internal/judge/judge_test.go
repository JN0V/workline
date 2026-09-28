package judge

import "testing"

func TestIndependence(t *testing.T) {
	author := []string{"claude-haiku-4-5"}
	for spec, want := range map[string]string{"cmd:x": "provider", "claude:opus": "model", "claude:haiku": "context"} {
		answered := map[string]string{"cmd:x": "any", "claude:opus": "claude-opus-5-5", "claude:haiku": "claude-haiku-4-5"}[spec]
		if got := Independence(spec, answered, "claude", author); got != want {
			t.Errorf("%s answered by %s: independence %s, want %s", spec, answered, got, want)
		}
	}
}

func TestPick(t *testing.T) {
	for author, want := range map[string]string{"claude-opus-5-5": "claude:sonnet", "claude-sonnet-5": "claude:opus", "claude-haiku-4-5": "claude:opus"} {
		if got := Pick("claude", author); got != want {
			t.Errorf("Pick after %s = %s, want %s", author, got, want)
		}
	}
	if got := Pick("cmd:my-agent", "any"); got != "cmd:my-agent" {
		t.Errorf("another agent judges itself, in a context of its own: got %s", got)
	}
}

func TestVerdict(t *testing.T) {
	for note, want := range map[string]string{
		`yes: same meaning`:            "",
		`- note: no: 'judged' is gone`: "'judged' is gone",
		`- note: "No: it narrows it."`: "it narrows it.",
	} {
		yes, why, err := Verdict(note)
		if err != nil || (want == "") != yes || (!yes && why != want) {
			t.Errorf("Verdict(%q) = %v %q %v; want %q", note, yes, why, err, want)
		}
	}
	if _, _, err := Verdict("The rewrite drops a word"); err == nil {
		t.Error("a note without yes or no is no verdict")
	}
}
