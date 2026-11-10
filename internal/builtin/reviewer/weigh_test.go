package reviewer

import "testing"

// The tester's answer reads back as it is written, its reason without
// the marker the forge adds after it (#1303).
func TestNoneNeededReadsTesterAnswer(t *testing.T) {
	body := TesterAnswer("a/b.py:31", "https://x/1#7", "**None needed**: the comma amount keeps today's behaviour.") + "\n\n<!-- workline:sticky=tester/tester-answer-7 -->"
	where, reason, ok := NoneNeeded(body)
	if !ok || where != "a/b.py:31" || reason != "the comma amount keeps today's behaviour." {
		t.Fatalf("NoneNeeded = %q, %q, %v", where, reason, ok)
	}
	if _, _, ok := NoneNeeded(TesterAnswer("a/b.py:31", "https://x/1#7", "**Added** in `t.py`: a test.")); ok {
		t.Fatal("a test added read as none needed")
	}
}

// A record keeps the answers weighed, and a record before them reads.
func TestRecordKeepsWeighed(t *testing.T) {
	r := Record{Reviewed: []string{"aa"}, Runs: 1, Weighed: []Weighed{{Lens: "tests", Where: "a.go:3", Answer: answerKey("x  y"), Holds: true, Why: "a test-free line"}}}
	got, err := parseRecord(r.String())
	if err != nil || len(got.Weighed) != 1 || got.Weighed[0] != r.Weighed[0] {
		t.Fatalf("parseRecord = %+v, %v", got.Weighed, err)
	}
	if answerKey("x y") != r.Weighed[0].Answer {
		t.Fatal("a reason keyed by its spaces")
	}
	old, err := parseRecord("<!-- workline:record reviewed=aa runs=1 -->")
	if err != nil || old.Weighed != nil {
		t.Fatalf("an older record: %+v, %v", old, err)
	}
}
