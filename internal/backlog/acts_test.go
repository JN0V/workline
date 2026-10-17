package backlog

import "testing"

func TestSuggestFromActs(t *testing.T) {
	for _, c := range []struct {
		level        string
		done, undone int
		want         string
	}{
		{Normal, 9, 0, ""},            // too few
		{Normal, 10, 0, Enterprising}, // none undone
		{Normal, 10, 1, ""},           // one in ten: not more
		{Normal, 10, 2, Cautious},     // more than one in ten
		{Enterprising, 10, 0, ""},     // already the most
		{Enterprising, 20, 3, Normal}, // more than one in ten
		{Cautious, 30, 0, ""},         // the report's, from the proposals
		{"unknown", 30, 10, ""},       // a level not read: nothing
	} {
		if got, _ := SuggestFromActs(c.level, c.done, c.undone); got != c.want {
			t.Errorf("%s, %d done, %d undone: %q, want %q", c.level, c.done, c.undone, got, c.want)
		}
	}
}

// An act keeps what the run read (ADR-0046): the commit, the issue's body
// by its digest, the agents and models the run asked — none for an act a
// person accepted, theirs.
func TestActRecordsWhatWasRead(t *testing.T) {
	p := &Plan{config: Config{Level: Normal, Read: Read{At: "d8c4dff", Model: "claude:claude-sonnet-5", Bodies: map[int]string{7: "84aa3a3ca6f7"}}}}
	p.recordDid(Proposal{Do: "sources", Issue: 7, Sources: []string{"src/a.go"}, Quote: &Quote{Issue: 7, Text: "the export"}})
	p.recordDid(Proposal{Do: "ready", Issue: 7, Ticked: "@mia"})
	p.recordDid(Proposal{Do: "ready", Issue: 9})
	d := p.Record.Did
	if len(d) != 3 {
		t.Fatalf("did: %+v", d)
	}
	if d[0].At != "d8c4dff" || d[0].Model != "claude:claude-sonnet-5" || d[0].Body != "84aa3a3ca6f7" {
		t.Errorf("done alone: %+v", d[0])
	}
	if d[1].Model != "" || d[1].At != "d8c4dff" || d[1].Body != "84aa3a3ca6f7" {
		t.Errorf("accepted by a person: %+v, want no model", d[1])
	}
	if d[2].Body != "" || d[2].At != "d8c4dff" {
		t.Errorf("an issue the run did not read: %+v, want no body", d[2])
	}
}
