package backlog

import (
	"strings"
	"testing"
	"time"

	"github.com/JN0V/workline/internal/forge"
)

func TestReadWait(t *testing.T) {
	for _, c := range []struct {
		in   any
		want Delay
		ok   bool
	}{
		{nil, DefaultWait, true},
		{"24h", Delay{D: 24 * time.Hour}, true},
		{"30m", Delay{D: 30 * time.Minute}, true},
		{"2d", Delay{D: 48 * time.Hour}, true},
		{"0", Delay{}, true},
		{0, Delay{}, true},
		{"never", Delay{Never: true}, true},
		{"soon", Delay{}, false},
		{"-1h", Delay{}, false},
		{3, Delay{}, false},
	} {
		got, err := ReadWait(c.in)
		if (err == nil) != c.ok || c.ok && got != c.want {
			t.Errorf("ReadWait(%v) = %v, %v; want %v, ok %v", c.in, got, err, c.want, c.ok)
		}
	}
}

func TestUnansweredCountsFromTheQuestionsComment(t *testing.T) {
	ask := forge.Note{Body: "@ann, to refine this issue: which?\n\n<!-- workline:product-owner/ask -->", Created: "2026-10-01T23:50:00Z"}
	ann := forge.Issue{Author: "ann"}
	s, open := Unanswered([]forge.Note{ask}, "product-owner", ann)
	if !open || s.Round != 1 || !s.Dated || s.Since.Format(time.RFC3339) != "2026-10-01T23:50:00Z" {
		t.Fatalf("Unanswered = %+v, %v", s, open)
	}
	w := Delay{D: 24 * time.Hour}
	if s.Due(w, nil, s.Since.Add(23*time.Hour)) || !s.Due(w, nil, s.Since.Add(24*time.Hour)) {
		t.Error("the delay is not counted from the question's comment")
	}
	if s.Due(w, &State{Assumed: 1}, s.Since.Add(48*time.Hour)) {
		t.Error("a question taken already is due again")
	}
	if _, open := Unanswered([]forge.Note{ask, {Body: "CSV.", Author: "ann"}}, "product-owner", ann); open {
		t.Error("a question answered is still open")
	}
	undated := ask
	undated.Created = ""
	if s, _ := Unanswered([]forge.Note{undated}, "product-owner", ann); s.Due(Delay{}, nil, time.Now()) {
		t.Error("a question the forge does not date timed out")
	}
}

func TestWithAssumptionLeavesThePersonsSections(t *testing.T) {
	body := "## Need\n\nThe plain line.\n\n---\n\n## Scope\n\nsrc/a.go\n\n---\n\n🤖 Generated with [a tool](https://example.com)"
	one := WithAssumption(body, "- Assumption taken on 2026-11-02, x: one.")
	if !strings.HasPrefix(one, "## Need\n\nThe plain line.\n\n---\n\n## Scope\n\nsrc/a.go\n\n## Assumptions\n\n- Assumption taken on 2026-11-02, x: one.") ||
		!strings.HasSuffix(one, "🤖 Generated with [a tool](https://example.com)") {
		t.Fatalf("first assumption:\n%s", one)
	}
	two := WithAssumption(one, "- Assumption taken on 2026-11-03, y: two.")
	if got := Assumptions(two); len(got) != 2 || got[1] != "- Assumption taken on 2026-11-03, y: two." {
		t.Fatalf("second assumption: %q", got)
	}
	answered := WithAnswered(two, "ann", "2026-11-04")
	if got := Assumptions(answered); !strings.HasSuffix(got[0], "Answered by @ann on 2026-11-04: their answer stands over it.") ||
		!strings.Contains(answered, "## Need\n\nThe plain line.") {
		t.Fatalf("answered:\n%s", answered)
	}
	if WithAnswered(answered, "bob", "2026-11-05") != answered {
		t.Error("an answered assumption marked twice")
	}
}
