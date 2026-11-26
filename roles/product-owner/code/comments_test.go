package productowner

import (
	"regexp"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/backlog"
	"github.com/JN0V/workline/internal/forge"
)

// saidCut matches the task saying a comment was cut.
var saidCut = regexp.MustCompile(`(?i)\bcut\b`)

// longAnswer is a maintainer's comment past bodyMax whose last line settles
// a choice.
func longAnswer() string {
	return strings.Repeat("Some long reasoning goes on and on here. ", 100) + "Decision: the bound is 42."
}

// Given a person's comment of more than 3000 characters that settles a
// choice in its last lines, When the product owner reads the issue, Then
// the whole comment is given to it, and the choice is written into the
// Scope.
func TestLongAnswerGivenWhole(t *testing.T) {
	long := longAnswer()
	if len(long) <= bodyMax {
		t.Fatalf("the fixture holds %d characters, want more than %d", len(long), bodyMax)
	}
	is := forge.Issue{ID: 1364, Title: "Bound", Author: "reporter", Body: "## Scope\n\nA bound.\n"}
	notes := []forge.Note{{Body: long, Author: "maintainer", Insider: true}}
	var b strings.Builder
	writeIssue(&b, "product-owner", 3, is, &backlog.State{}, notes, map[int]bool{}, nil, "")
	got := b.String()
	if !strings.Contains(got, "Decision: the bound is 42.") {
		t.Errorf("the end of a person's long answer is not given:\n%s", got)
	}
	if strings.Contains(got, "[…]") {
		t.Errorf("a person's long answer is cut:\n%s", got)
	}
}

// Given a long answer from a person who counts, When it is given to the
// product owner as a comment among the others, or as the comment that
// brought the issue back, Then it is never cut.
func TestLongAnswerNeverCutInBothPlaces(t *testing.T) {
	long := longAnswer()
	is := forge.Issue{ID: 1364, Title: "Bound", Author: "reporter", Body: "## Scope\n\nA bound.\n"}
	person := forge.Note{Body: long, Author: "maintainer", Insider: true, Created: "2026-11-01T10:00:00Z"}

	// Among the others: after more comments than are kept.
	notes := []forge.Note{person}
	for i := 0; i < 6; i++ {
		notes = append(notes, forge.Note{Body: "ok", Author: "ci[bot]", Bot: true})
	}
	var b strings.Builder
	writeIssue(&b, "product-owner", 3, is, &backlog.State{}, notes, map[int]bool{}, nil, "")
	among := b.String()
	if !strings.Contains(among, "Decision: the bound is 42.") || strings.Contains(among, "[…]") {
		t.Errorf("among the comments, a person's long answer is cut:\n%s", among)
	}

	// As the comment quoted when it wakes the issue: not read before.
	woken := commented(person, is, &backlog.State{}, []forge.Note{person})
	if !strings.Contains(woken, "Decision: the bound is 42.") || strings.Contains(woken, "[…]") {
		t.Errorf("quoted as not read before, a person's long answer is cut:\n%s", woken)
	}

	// As the comment that brought back an issue set aside.
	back := commented(person, is, &backlog.State{Aside: backlog.BodyDigest(is.Body)}, []forge.Note{person})
	if !strings.Contains(back, "Brought back by a comment from @maintainer") {
		t.Fatalf("not the comment that brought the issue back:\n%s", back)
	}
	if !strings.Contains(back, "Decision: the bound is 42.") || strings.Contains(back, "[…]") {
		t.Errorf("as the comment that brought the issue back, a person's long answer is cut:\n%s", back)
	}
}

// Given a comment from a bot or a role longer than 3000 characters, When
// the product owner reads it, Then it is cut and the task says it was cut.
func TestOtherLongCommentsCutAndSaid(t *testing.T) {
	long := strings.Repeat("Build log line after line. ", 150) + "TAILEND"
	is := forge.Issue{ID: 1364, Title: "Bound", Author: "reporter", Body: "## Scope\n\nA bound.\n"}

	// A bot's comment among the issue's comments.
	notes := []forge.Note{{Body: long, Author: "ci[bot]", Bot: true}}
	var b strings.Builder
	writeIssue(&b, "product-owner", 3, is, &backlog.State{}, notes, map[int]bool{}, nil, "")
	got := b.String()
	if strings.Contains(got, "TAILEND") || !strings.Contains(got, " […]") {
		t.Errorf("a bot's long comment is not cut at %d characters:\n%s", bodyMax, got)
	}
	if !saidCut.MatchString(got) {
		t.Errorf("the task does not say a comment was cut:\n%s", got)
	}

	// The reviewer's findings comment, the engine's own.
	findings := backlog.SpecMarker + "\n" + long
	r := &backlog.SpecReview{Round: 1, Open: 1, In: []string{"Scope"}}
	said := specFindings(is, &backlog.State{}, []string{findings}, r)
	if strings.Contains(said, "TAILEND") || !strings.Contains(said, " […]") {
		t.Errorf("the reviewer's long findings are not cut at %d characters:\n%s", bodyMax, said)
	}
	if !saidCut.MatchString(said) {
		t.Errorf("the task does not say the reviewer's findings were cut:\n%s", said)
	}
}
