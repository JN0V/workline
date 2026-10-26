package inspector

import (
	"fmt"
	"strings"
	"testing"
)

func TestRecordReadsBackWhatTheCommentHides(t *testing.T) {
	r := Record{}.with(Triaged{Fingerprint: "a1", Check: "lint", Rule: "errcheck", Where: "x.go:3", At: "abc", Matters: "it fails --> here <b>", FalsePositive: true, Why: "a -- b", Order: 2})
	r = r.with(Triaged{Fingerprint: "b2", Check: "lint", Where: "y.go:1", At: "abc", Matters: "m"})
	text, _ := r.lines(nil, 1<<20)
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
	first, _ := Record{Triaged: r.Triaged[:1]}.lines(nil, 1<<20)
	one := len(first)
	text, left := r.lines(map[string]bool{"kept": true, "new": true}, 2*one+10)
	if strings.Contains(text, `"old"`) || !strings.Contains(text, `"kept"`) || !strings.Contains(text, `"new"`) || left != 0 {
		t.Fatalf("past its room, the record keeps the findings at the head and drops the oldest gone (%d left): %s", left, text)
	}
	// Past its room with the findings at the head alone: the oldest of
	// them go too, counted, asked again.
	text, left = r.lines(map[string]bool{"old": true, "kept": true, "new": true}, one+10)
	if left != 2 || !strings.Contains(text, `"new"`) || strings.Count(text, "workline:triaged") != 1 {
		t.Fatalf("past its room, %d of the findings at the head left out, want the 2 oldest: %s", left, text)
	}
}

func TestCommentStaysWithinTheForgesSize(t *testing.T) {
	run := Run{Head: "abc"}
	for i := range 80 {
		fp := strings.Repeat(string(rune('a'+i%26)), 3) + string(rune('0'+i/26))
		run.Found = append(run.Found, Found{Fingerprint: fp, Check: "lint", Path: "x.go", Line: i + 1})
		run.Record = run.Record.with(Triaged{Fingerprint: fp, Check: "lint", Where: "x.go", At: "abc",
			Matters: strings.Repeat("m", textMax), Fix: strings.Repeat("f", textMax), FalsePositive: true, Why: strings.Repeat("w", textMax), Order: i + 1})
	}
	body := comment(run, current(run))
	if len(body) > commentMax || !strings.Contains(body, "55 more triaged, not listed here") || !strings.Contains(body, "triages not kept in this comment, past its size") {
		t.Fatalf("a comment of %d characters, over %d, or not saying what it left out", len(body), commentMax)
	}
}

func TestMergeCommentKeepsTheRunAppliedSince(t *testing.T) {
	triaged := func(fp, at string) Triaged {
		return Triaged{Fingerprint: fp, Check: "lint", Where: fp + ".go:1", At: at, Matters: "m " + at}
	}
	note := func(r Record) string {
		text, _ := r.lines(nil, 1<<20)
		return "the triage" + text + "\n\n" + marker
	}
	was := Record{}.with(triaged("old", "h0"))
	read, _ := was.lines(nil, 1<<20)
	// This run: the old one read, "both" and "ours" triaged at its head.
	run := Run{Head: "h2", Found: []Found{{Fingerprint: "both"}, {Fingerprint: "ours"}}, Record: was.with(triaged("both", "h2")).with(triaged("ours", "h2"))}
	body := comment(run, current(run))
	// Applied since: the old one, and "both" and "theirs" triaged at h1.
	now := was.with(triaged("both", "h1")).with(triaged("theirs", "h1"))

	if got := MergeComment(body, read, note(was)); got != body {
		t.Fatalf("none applied since this run read, the body changed: %s", got)
	}
	// One this run read and left out past the size, only placed anew by a
	// run applied since: this run's to leave.
	trimmed := run
	trimmed.Record = Record{}.with(triaged("both", "h2")).with(triaged("ours", "h2"))
	left := comment(trimmed, current(trimmed))
	reranked := was
	reranked.Triaged = []Triaged{triaged("old", "h0")}
	reranked.Triaged[0].Order = 3
	if got := MergeComment(left, read, note(reranked)); got != left {
		t.Fatalf("a finding this run read, only placed anew since, was merged back: %s", got)
	}
	if got := MergeComment(body, read, "<!-- workline:triaged {nope} -->"+marker); got != body {
		t.Fatalf("a record that does not read was merged: %s", got)
	}
	got := MergeComment(body, read, note(now))
	merged, err := readRecord(got)
	if err != nil || !strings.HasPrefix(got, body[:strings.Index(body, recordOpening)]) {
		t.Fatalf("the merged comment does not read, or its triage changed (%v): %s", err, got)
	}
	want := []Triaged{triaged("theirs", "h1"), triaged("old", "h0"), triaged("both", "h2"), triaged("ours", "h2")}
	if len(merged.Triaged) != len(want) {
		t.Fatalf("merged %+v, want %+v", merged.Triaged, want)
	}
	for i, w := range want {
		if g := merged.Triaged[i]; g.Fingerprint != w.Fingerprint || g.At != w.At {
			t.Fatalf("merged %+v, want the run's applied since first, then this run's own, its triage of one both made kept: %+v", merged.Triaged, want)
		}
	}
	// Past the comment's size: the run's applied since go, never this
	// run's own.
	big := now
	for i := range 200 {
		big = big.with(Triaged{Fingerprint: fmt.Sprintf("theirs%03d", i), Check: "lint", Where: "x.go", At: "h1", Matters: strings.Repeat("m", textMax), Why: strings.Repeat("w", textMax)})
	}
	got = MergeComment(body, read, note(big))
	merged, err = readRecord(got)
	if err != nil || len(got) > commentMax || len(merged.Triaged) >= len(big.Triaged) {
		t.Fatalf("a merged comment of %d characters, over %d, or none of the others' left out (%v)", len(got), commentMax, err)
	}
	for _, fp := range []string{"old", "both", "ours"} {
		if _, ok := merged.find(fp); !ok {
			t.Fatalf("past its size, the merge left out this run's own %q", fp)
		}
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
