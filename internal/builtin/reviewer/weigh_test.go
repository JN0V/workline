package reviewer

import "testing"

// The tester's answer reads back as it is written, its reason without
// the marker the forge adds after it (#1303); one before titles too.
func TestNoneNeededReadsTesterAnswer(t *testing.T) {
	body := TesterAnswer("a/b.py:31", "Important: no test for a comma amount — nothing runs 20,00.", "https://x/1#7", "**None needed**: the comma amount keeps today's behaviour.") + "\n\n<!-- workline:sticky=tester/tester-answer-7 -->"
	a, ok := NoneNeeded(body)
	if !ok || a.Where != "a/b.py:31" || a.Title != "no test for a comma amount" || a.Reason != "the comma amount keeps today's behaviour." {
		t.Fatalf("NoneNeeded = %+v, %v", a, ok)
	}
	if _, ok := NoneNeeded(TesterAnswer("a/b.py:31", "", "https://x/1#7", "**Added** in `t.py`: a test.")); ok {
		t.Fatal("a test added read as none needed")
	}
	old := "**The tester** answers a test finding of this review on `a/b.py:31`, sent to its Draft ([the thread](https://x/1#7)): **None needed**: why."
	if a, ok := NoneNeeded(old); !ok || a.Where != "a/b.py:31" || a.Title != "" || a.Reason != "why." {
		t.Fatalf("an answer before titles: %+v, %v", a, ok)
	}
}

// An answer is matched to its finding by its title on the same file, its
// line moved; one with no title, by its line alone. A verdict on one
// finding does not settle another on the same line.
func TestAnswerToAndVerdictsByTitle(t *testing.T) {
	answers := []Answer{{Where: "p.py:31", Title: "no test for a comma amount", Reason: "r"}}
	if r, ok := answerTo(answers, "p.py:33", "Important: no test for a comma amount"); !ok || r != "r" {
		t.Fatal("an answer missed once its finding's line moved")
	}
	if _, ok := answerTo(answers, "p.py:31", "Important: no test for a negative amount"); ok {
		t.Fatal("an answer read for another finding on its line")
	}
	if _, ok := answerTo([]Answer{{Where: "p.py:31", Reason: "r"}}, "p.py:31", "Important: anything"); !ok {
		t.Fatal("an answer before titles missed on its line")
	}
	st := state{Answers: answers, Record: Record{Weighed: []Weighed{{Lens: "tests", Where: "p.py:31", Title: "no test for a comma amount", Answer: answerKey("r"), Holds: true}}}}
	if !st.settles("tests", "p.py:40", "Important: no test for a comma amount — why") {
		t.Fatal("a verdict that holds missed once its finding's line moved")
	}
	if st.settles("tests", "p.py:31", "Important: no test for a negative amount") {
		t.Fatal("a verdict settles another finding on its line")
	}
	st.Answers = nil
	if st.settles("tests", "p.py:31", "Important: no test for a comma amount") {
		t.Fatal("a verdict settles a finding with no answer left")
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
