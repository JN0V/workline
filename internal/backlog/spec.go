package backlog

// A spec read before it is built, on the forge (ADR-0020, #128): the
// reviewer reads an issue the product owner refined and keeps what it found
// in one comment on that issue; the product owner's `ready` reads it back
// and holds the issue while an important finding is open. This file is the
// contract both roles share: the comment's marker and the record it hides.

import (
	"fmt"
	"net/url"
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
	In      []string // the sections their causes lie in, by heading
	Stopped bool     // the rounds spent: put to a person, not read again
}

// The record names a section by its heading in the body, as the issue's
// state does, read by meaning (ADR-0039). Each heading is escaped as a
// URL's query escapes a value: words, spaces, a comma or a "-->" read back
// whole, and a heading of letters only is written as it always was, so a
// record written before reads the same.
var specRecord = regexp.MustCompile(`<!-- workline:spec-review body=([0-9a-f]*) round=(\d+) open=(\d+) in=([A-Za-z0-9%+._~,-]*)( stopped)? -->`)

// String is the record as the comment hides it.
func (r SpecReview) String() string {
	stopped := ""
	if r.Stopped {
		stopped = " stopped"
	}
	in := make([]string, len(r.In))
	for i, h := range r.In {
		in[i] = url.QueryEscape(h)
	}
	return fmt.Sprintf("<!-- workline:spec-review body=%s round=%d open=%d in=%s%s -->", r.Body, r.Round, r.Open, strings.Join(in, ","), stopped)
}

// ReadSpecReview reads the reviewer's record from an issue's comments: nil
// when it has none. The last comment carrying the marker is read (GitLab:
// one another token wrote is left behind). One that does not read is an
// error (#302), never taken for none: read as none, the spec was read
// again at round 1, night after night, in silence.
func ReadSpecReview(comments []string) (*SpecReview, error) {
	for i := len(comments) - 1; i >= 0; i-- {
		if !strings.Contains(comments[i], SpecMarker) {
			continue
		}
		m := specRecord.FindStringSubmatch(comments[i])
		if m == nil {
			return nil, UnreadRecord(comments[i], specOpening)
		}
		r := &SpecReview{Body: m[1], Stopped: m[5] != ""}
		r.Round, _ = strconv.Atoi(m[2])
		r.Open, _ = strconv.Atoi(m[3])
		for _, h := range strings.Split(m[4], ",") {
			name, err := url.QueryUnescape(h)
			if err != nil {
				return nil, UnreadRecord(comments[i], specOpening)
			}
			if name != "" {
				r.In = append(r.In, name)
			}
		}
		return r, nil
	}
	return nil, nil
}

// specOpening opens the reviewer's record in its comment on a spec.
const specOpening = "<!-- workline:spec-review"

// UnreadRecord says why a record the engine hid in a text does not read
// (#302): the line holding its opening, quoted as code so a summary shows
// it, or that the text holds none.
func UnreadRecord(text, opening string) error {
	for _, l := range strings.Split(text, "\n") {
		if i := strings.Index(l, opening); i >= 0 {
			line := []rune(strings.ReplaceAll(strings.TrimSpace(l[i:]), "`", "'"))
			if len(line) > 120 {
				line = append(line[:119], '…')
			}
			return fmt.Errorf("`%s` does not read", string(line))
		}
	}
	return fmt.Errorf("no `%s` line in it", strings.TrimPrefix(opening, "<!-- "))
}

// SpecRecordUnread is the rule said of an issue whose reviewer's record
// does not read (#302).
const SpecRecordUnread = "spec-record-unread"

// SpecUnread is what is said of an issue whose reviewer's record does not
// read: the record, why, and what becomes of the issue — held for a
// person, never read again in silence, which loops an agent call a night.
func SpecUnread(err error) string {
	return "the record hidden in the reviewer's comment on its spec does not read (" + err.Error() + "): not read again, and held from ready, " +
		"until a person deletes that comment — the spec then read anew, at round 1 — or accepts the issue (" + LabelAccepted + ")"
}

// SpecHold says why an issue's spec holds it from ready, "" when nothing
// does: rule and why, as a dropped act says them. A person's label
// `workline:accepted` is their yes to the issue as it reads: it lifts the
// hold, whatever the reviewer found.
func SpecHold(is forge.Issue, comments []string) (rule, why string) {
	if Accepted(is) {
		return "", ""
	}
	r, err := ReadSpecReview(comments)
	switch {
	case err != nil:
		return SpecRecordUnread, SpecUnread(err)
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
	r, _ := ReadSpecReview(comments) // one that does not read clears nothing (SpecHold says it)
	return r, r != nil && !r.Stopped && r.Open == 0 && r.Body == BodyDigest(is.Body)
}

// SpecOpen is the reviewer's record when it holds important findings open
// on the body as it is, for the product owner to answer; nil otherwise.
func SpecOpen(is forge.Issue, comments []string) *SpecReview {
	r, _ := ReadSpecReview(comments) // one that does not read holds nothing to answer
	if r == nil || r.Stopped || r.Open == 0 || r.Body != BodyDigest(is.Body) {
		return nil
	}
	return r
}

// SectionAt is the section of a body a line lies in, by its `## ` (or
// `### `) heading above it; "" above the first, or in the body's footer.
func SectionAt(body string, line int) string {
	name := ""
	if main, _ := work.Footer(body); line > strings.Count(main, "\n")+1 {
		return ""
	}
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
	if name == DescriptionName {
		return DescriptionDigest(body)
	}
	return BodyDigest(work.ByMeaning(body)[work.MeaningOf(name)].Text)
}

// Own are the sections of a body still the role's own — a draft no person
// accepted, or a text as the role wrote it (the state's `wrote`) —: those
// it rewrites to answer a person's comment on its proposal (ADR-0038). A
// person's section, edited or deleted, is never among them.
func Own(is forge.Issue, st *State) []string {
	have := work.ByMeaning(is.Body)
	var out []string
	for _, m := range layout() {
		s, ok := have[m]
		switch {
		case !ok || strings.TrimSpace(s.Text) == "":
		case IsDraft(is.Body, m, st) && !Accepted(is), mine(st, m, s.Text):
			out = append(out, s.Heading)
		}
	}
	return out
}

// mine says whether text is as the role wrote a section of meaning m (the
// state's `wrote`).
func mine(st *State, m work.Meaning, text string) bool {
	_, wrote := st.wrote(m)
	return wrote != "" && wrote == BodyDigest(text)
}

// Revisable are the sections of a body the role may rewrite to answer the
// reviewer's findings (#128): those the findings lie in that are still the
// role's own — a draft no person accepted, or a text as the role wrote it
// (the state's `wrote`). A person's section is theirs: never rewritten.
func Revisable(is forge.Issue, st *State, r *SpecReview) []string {
	if r == nil {
		return nil
	}
	have := work.ByMeaning(is.Body)
	var out []string
	for _, name := range r.In {
		m := work.MeaningOf(name)
		s, ok := have[m]
		switch {
		case !ok || !slices.Contains(work.Required, m) || work.Has(out, m):
		case IsDraft(is.Body, m, st) && !Accepted(is), mine(st, m, s.Text):
			out = append(out, name)
		}
	}
	return out
}
