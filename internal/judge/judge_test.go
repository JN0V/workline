package judge

import (
	"reflect"
	"testing"
)

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
		`note: "no: as it reads now"`:  "as it reads now", // not a list's item: seen live
		`note: "yes: it is the value`:  "",
	} {
		yes, why, err := Verdict(note)
		if err != nil || (want == "") != yes || (!yes && why != want) {
			t.Errorf("Verdict(%q) = %v %q %v; want %q", note, yes, why, err, want)
		}
	}
	if _, _, err := Verdict("The rewrite drops a word"); err == nil {
		t.Error("a note without yes or no is no verdict")
	}
	if _, _, err := Verdict("unsure: cannot tell"); err == nil {
		t.Error("unsure is only a reading judge's answer")
	}
}

// A judge that may read (ADR-0045) answers a verdict, its reads, or
// unsure; a no keeps its quote; a verdict wins over reads asked beside it.
func TestAnswerReading(t *testing.T) {
	for answer, want := range map[string]Answer{
		`- note: "yes: it fails"`: {Yes: true, Why: "it fails"},
		"- note: \"no: the guard returns first\"\n- quote: {path: a.go, text: \"if x == nil { return }\"}": {Why: "the guard returns first", Quote: &Quote{Path: "a.go", Text: "if x == nil { return }"}},
		`- note: "unsure: no caller shown"`: {Unsure: true, Why: "no caller shown"},
		"- read: {path: a.go, lines: 10-40, why: the caller}\n- read: {name: Decide}\n- read: {grep: total}": {Reads: []Read{{Path: "a.go", Lines: "10-40", Why: "the caller"}, {Name: "Decide"}, {Grep: "total"}}},
		"- note: \"unsure: I need the caller\"\n- read: {path: a.go, lines: 12}":                             {Reads: []Read{{Path: "a.go", Lines: "12"}}, Why: "I need the caller"},
		"- read: {name: Decide}\n- note: \"yes: it fails\"":                                                  {Yes: true, Why: "it fails"},
		// No YAML as a whole: each item read alone, the unquoted note too.
		"- note: no: the guard: it returns first\n- quote: {path: a.go, text: \"if x == nil {\"}": {Why: "the guard: it returns first", Quote: &Quote{Path: "a.go", Text: "if x == nil {"}},
		"- note: I need: the caller\n- read: {path: a.go, lines: 12}":                             {Reads: []Read{{Path: "a.go", Lines: "12"}}},
	} {
		got, err := answerReading(Answer{}, []byte(answer))
		if err != nil || got.Yes != want.Yes || got.Unsure != want.Unsure || got.Why != want.Why ||
			!reflect.DeepEqual(got.Reads, want.Reads) || !reflect.DeepEqual(got.Quote, want.Quote) {
			t.Errorf("%s:\n got %+v %v\nwant %+v", answer, got, err, want)
		}
	}
	for _, answer := range []string{`- note: "maybe"`, `- read: {why: nothing named}`, `[]`} {
		if _, err := answerReading(Answer{}, []byte(answer)); err == nil {
			t.Errorf("%s: no verdict, no read: an error", answer)
		}
	}
	if s := (Read{Path: "a.go", Lines: "1-9"}).String() + ", " + (Read{Name: "sum"}).String() + ", " + (Read{Grep: "total"}).String(); s != "a.go:1-9, `sum`, grep `total`" {
		t.Errorf("reads said as %q", s)
	}
}
