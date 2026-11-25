// Package reviewmark is what the reviewer's words are, as other roles
// write and read them: its name as a project routes it, the tester's
// answer to a test finding of a review, and its reading (ADR-0070 §6 e).
package reviewmark

import (
	"fmt"
	"regexp"
	"strings"
)

// Name is the reviewer's role, as a project routes it.
const Name = "reviewer"

// TesterAnswer is the tester's answer to a test finding of this review,
// found on found and titled title, its thread at url: what it did, text —
// a test added, or **None needed** and why. The title names the finding
// the answer is for, its line moved or another finding on the same line.
func TesterAnswer(found, title, url, text string) string {
	on := ""
	if found != "" {
		on = " on `" + found + "`"
		if t := strings.ReplaceAll(FindingTitle(title), "”", "\""); t != "" {
			on += ", “" + t + "”"
		}
	}
	return fmt.Sprintf("**The tester** answers a test finding of this review%s, sent to its Draft ([the thread](%s)): %s", on, url, text)
}

// FindingTitle is a finding's title as a row, its line or the record
// says it: its severity, its why, what it was grouped with and where it
// was found left out.
func FindingTitle(s string) string {
	s = strings.TrimPrefix(strings.TrimSpace(s), "Important:")
	for _, sep := range []string{" — ", "; also ", " (found "} {
		s, _, _ = strings.Cut(s, sep)
	}
	return strings.TrimSpace(s)
}

// noneNeededWords read a "none needed" answer as TesterAnswer says it:
// the place the review found the finding on, its title, then the reason.
var noneNeededWords = regexp.MustCompile("(?s)\\*\\*The tester\\*\\* answers a test finding of this review on `([^`]+)`(?:, “([^”]*)”)?, sent to its Draft \\(\\[the thread\\]\\([^)]*\\)\\): \\*\\*None needed\\*\\*: (.*)")

// Answer is the tester's answer to a finding: the place the review found
// it on, its title ("" from a tester before titles), and the reason no
// test is needed (NoneNeeded), or the test file it added (Added).
type Answer struct {
	Where  string `json:"where"`
	Title  string `json:"title,omitempty"`
	Reason string `json:"reason"`
}

// addedWords read an "added" answer as TesterAnswer says it: the place
// the review found the finding on, its title, then the test file.
var addedWords = regexp.MustCompile("\\*\\*The tester\\*\\* answers a test finding of this review on `([^`]+)`(?:, “([^”]*)”)?, sent to its Draft \\(\\[the thread\\]\\([^)]*\\)\\): \\*\\*Added\\*\\* in `([^`]+)`")

// Added reads a comment of the tester's on a reviewed pull request
// (TesterAnswer) that answers a finding with a test added: the finding,
// and the test file in Reason. ok is false for any other comment.
func Added(body string) (Answer, bool) {
	m := addedWords.FindStringSubmatch(body)
	if m == nil {
		return Answer{}, false
	}
	return Answer{Where: m[1], Title: m[2], Reason: m[3]}, true
}

// NoneNeeded reads a comment of the tester's on a reviewed pull request
// (TesterAnswer): the finding it answers that no test is needed for, and
// its reason, the markers hidden after it left out. ok is false for any
// other comment.
func NoneNeeded(body string) (Answer, bool) {
	m := noneNeededWords.FindStringSubmatch(body)
	if m == nil {
		return Answer{}, false
	}
	reason, _, _ := strings.Cut(m[3], "<!-- workline:")
	return Answer{Where: m[1], Title: m[2], Reason: strings.TrimSpace(reason)}, true
}
