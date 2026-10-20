package inspector

import (
	"strings"
	"testing"
)

func TestRecordReadsBackWhatTheCommentHides(t *testing.T) {
	r := Record{}.with(Triaged{Fingerprint: "a1", Check: "lint", Rule: "errcheck", Where: "x.go:3", At: "abc", Matters: "it fails --> here <b>", FalsePositive: true, Why: "a -- b", Order: 2})
	r = r.with(Triaged{Fingerprint: "b2", Check: "lint", Where: "y.go:1", At: "abc", Matters: "m"})
	text := r.lines(nil, 1<<20)
	if strings.Contains(text, "-->"+" here") || strings.Count(text, "-->") != 2 || strings.Contains(text, "false-positive") {
		t.Fatalf("a text of the agent's ends the comment hiding it, or a dash is left: %s", text)
	}
	back, err := readRecord("before\n" + text + "\nafter")
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Triaged) != 2 || back.Triaged[0] != r.Triaged[0] || back.Triaged[1] != r.Triaged[1] {
		t.Fatalf("read back %+v, want %+v", back, r)
	}
}

func TestRecordCutIsAnError(t *testing.T) {
	if _, err := readRecord("<!-- workline:triaged {\"fingerprint\":\"a\""); err == nil {
		t.Fatal("a line of the record cut short read as none")
	}
	if _, err := readRecord("<!-- workline:triaged {nope} -->"); err == nil {
		t.Fatal("a line that does not read read as none")
	}
}

func TestRecordDropsTheOldestGoneFirst(t *testing.T) {
	var r Record
	for _, fp := range []string{"old", "kept", "new"} {
		r = r.with(Triaged{Fingerprint: fp, Check: "c", Where: "w", At: "h", Matters: strings.Repeat("m", 100)})
	}
	one := len(Record{Triaged: r.Triaged[:1]}.lines(nil, 1<<20))
	text := r.lines(map[string]bool{"kept": true, "new": true}, 2*one+10)
	if strings.Contains(text, `"old"`) || !strings.Contains(text, `"kept"`) || !strings.Contains(text, `"new"`) {
		t.Fatalf("past its room, the record keeps the findings at the head and drops the oldest gone: %s", text)
	}
}

func TestOrderedTakesTheAgentsThenTheOldPlaces(t *testing.T) {
	r := Record{Triaged: []Triaged{{Fingerprint: "a", Order: 1}, {Fingerprint: "b", Order: 2}, {Fingerprint: "c"}, {Fingerprint: "gone", Order: 1}}}
	found := []Found{{Fingerprint: "a"}, {Fingerprint: "b"}, {Fingerprint: "c"}}
	got := r.ordered(found, map[string]int{"c": 1, "a": 2})
	want := map[string]int{"c": 1, "a": 2, "b": 3, "gone": 1}
	for _, tr := range got.Triaged {
		if tr.Order != want[tr.Fingerprint] {
			t.Fatalf("%s placed %d, want %d: %+v", tr.Fingerprint, tr.Order, want[tr.Fingerprint], got.Triaged)
		}
	}
}
