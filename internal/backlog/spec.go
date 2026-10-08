package backlog

// A spec read before it is built, on the forge (ADR-0020, #128): the
// reviewer reads an issue the product owner refined and keeps what it found
// in one comment on that issue; the product owner's `ready` reads it back
// and holds the issue while an important finding is open. This file is the
// contract both roles share: the comment's marker and the record it hides.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/JN0V/workline/internal/forge"
	"github.com/JN0V/workline/internal/work"
)

// SpecReviewer is the role whose comment on an issue holds its spec's
// review; SpecKey keys that comment (`sticky=reviewer/spec`).
const (
	SpecReviewer = "reviewer"
	SpecKey      = "spec"
)

// SpecMarker marks the reviewer's comment on an issue's spec.
var SpecMarker = forge.Marker("sticky=" + SpecReviewer + "/" + SpecKey)

// SpecReview is what the reviewer's comment records of the spec it read.
type SpecReview struct {
	Body    string   // the digest of the body it read (BodyDigest); "" when no review was whole
	Round   int      // the reviews in a row that found an important finding open, this one included
	Open    int      // the important findings open, as it read the body
	In      []string // the sections their causes lie in
	Stopped bool     // the rounds spent: put to a person, not read again
}

var specRecord = regexp.MustCompile(`<!-- workline:spec-review body=([0-9a-f]*) round=(\d+) open=(\d+) in=([A-Za-z,]*)( stopped)? -->`)

// String is the record as the comment hides it.
func (r SpecReview) String() string {
	stopped := ""
	if r.Stopped {
		stopped = " stopped"
	}
	return fmt.Sprintf("<!-- workline:spec-review body=%s round=%d open=%d in=%s%s -->", r.Body, r.Round, r.Open, strings.Join(r.In, ","), stopped)
}

// ReadSpecReview reads the reviewer's record from an issue's comments: nil
// when it has none, or one that does not read. The last comment carrying
// the marker is read (GitLab: one another token wrote is left behind).
func ReadSpecReview(comments []string) *SpecReview {
	for i := len(comments) - 1; i >= 0; i-- {
		if !strings.Contains(comments[i], SpecMarker) {
			continue
		}
		m := specRecord.FindStringSubmatch(comments[i])
		if m == nil {
			return nil
		}
		r := &SpecReview{Body: m[1], Stopped: m[5] != ""}
		r.Round, _ = strconv.Atoi(m[2])
		r.Open, _ = strconv.Atoi(m[3])
		if m[4] != "" {
			r.In = strings.Split(m[4], ",")
		}
		return r
	}
	return nil
}

// SpecHold says why an issue's spec holds it from ready, "" when nothing
// does: rule and why, as a dropped act says them. A person's label
// `workline:accepted` is their yes to the issue as it reads: it lifts the
// hold, whatever the reviewer found.
func SpecHold(is forge.Issue, comments []string) (rule, why string) {
	if Accepted(is) {
		return "", ""
	}
	r := ReadSpecReview(comments)
	switch {
	case r == nil:
		return "spec-not-reviewed", "the reviewer has not read its spec yet: ready once it has and no important finding is open, or a person accepts it (" + LabelAccepted + ")"
	case r.Stopped:
		return "spec-rounds-spent", fmt.Sprintf("the reviewer's findings still open after %d rounds: a person's to decide, as the reviewer's comment on it asks (%s, or %s by hand)", r.Round, LabelAccepted, LabelReady)
	case r.Body != BodyDigest(is.Body):
		return "spec-not-reviewed", "its spec changed since the reviewer read it: ready once it reads it again and no important finding is open"
	case r.Open > 0:
		return "spec-findings-open", fmt.Sprintf("%s of the reviewer open on its spec (round %d, in %s): the product owner answers at its next refine", plural(r.Open, "important finding"), r.Round, strings.Join(r.In, ", "))
	}
	return "", ""
}

// SpecCleared says whether the reviewer read the issue's body as it is and
// found no important finding open.
func SpecCleared(is forge.Issue, comments []string) (*SpecReview, bool) {
	r := ReadSpecReview(comments)
	return r, r != nil && !r.Stopped && r.Open == 0 && r.Body == BodyDigest(is.Body)
}

// SpecOpen is the reviewer's record when it holds important findings open
// on the body as it is, for the product owner to answer; nil otherwise.
func SpecOpen(is forge.Issue, comments []string) *SpecReview {
	r := ReadSpecReview(comments)
	if r == nil || r.Stopped || r.Open == 0 || r.Body != BodyDigest(is.Body) {
		return nil
	}
	return r
}

// SectionAt is the section of a body a line lies in, by its `## ` (or
// `### `) heading above it; "" above the first.
func SectionAt(body string, line int) string {
	name := ""
	for i, l := range strings.Split(body, "\n") {
		if i+1 > line {
			break
		}
		if h, ok := work.Heading(l); ok {
			name = h
		}
	}
	return name
}

// SectionDigest is the digest of a section's text, as the role wrote it:
// another text there is a person's.
func SectionDigest(body, name string) string {
	return BodyDigest(work.Sections(body)[name])
}

// Own are the sections of a body still the role's own — a draft no person
// accepted, or a text as the role wrote it (the state's `wrote`) —: those
// it rewrites to answer a person's comment on its proposal (ADR-0038). A
// person's section, edited or deleted, is never among them.
func Own(is forge.Issue, st *State) []string {
	have := work.Sections(is.Body)
	var out []string
	for _, name := range Sections {
		text, ok := have[name]
		switch {
		case !ok || strings.TrimSpace(text) == "":
		case slices.Contains(drafted, name) && strings.Contains(text, DraftMarker) && !Accepted(is):
			out = append(out, name)
		case st != nil && st.Wrote[name] != "" && st.Wrote[name] == BodyDigest(text):
			out = append(out, name)
		}
	}
	return out
}

// Revisable are the sections of a body the role may rewrite to answer the
// reviewer's findings (#128): those the findings lie in that are still the
// role's own — a draft no person accepted, or a text as the role wrote it
// (the state's `wrote`). A person's section is theirs: never rewritten.
func Revisable(is forge.Issue, st *State, r *SpecReview) []string {
	if r == nil {
		return nil
	}
	have := work.Sections(is.Body)
	var out []string
	for _, name := range r.In {
		text, ok := have[name]
		switch {
		case !ok || !slices.Contains(Sections, name) || slices.Contains(out, name):
		case slices.Contains(drafted, name) && strings.Contains(text, DraftMarker) && !Accepted(is):
			out = append(out, name)
		case st != nil && st.Wrote[name] != "" && st.Wrote[name] == BodyDigest(text):
			out = append(out, name)
		}
	}
	return out
}
