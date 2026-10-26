package backlog

import (
	"regexp"
	"strings"
	"unicode"
)

// Each thing is said once in an issue (ADR-0039, amended 2026-10-11): a
// draft of the role's that says again what is above it is not written.
// Measured as near-duplicate text is, by containment over one-word
// shingles (docs/research/issue-formats.md, "A draft that repeats"): the
// share of the draft's words already said above it, common words left
// out. A restated problem is a paraphrase, not a copy: one word, not runs
// of words.

// RepeatedPercentMax is the share of a draft's words already said above it
// past which it is not written, by default (`repeated-percent-max`).
const RepeatedPercentMax = 40

// commonWords are left out of the measure: words any sentence holds, which
// say nothing of what it is about.
var commonWords = func() map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(`a an the and or but of to in on at by for with from as is are was
		were be been it its this that these those not no so then than i me my we our you your he him
		his she her they them their has have had do does did can could will would should may might must
		only just also all any each some such if when while into out up over about who what which there
		here how why one`) {
		out[w] = true
	}
	return out
}()

// markup is what a reader does not read as words: a hidden comment, an
// HTML tag around a fold.
var markup = regexp.MustCompile(`(?s)<!--.*?-->|</?(?:details|summary)[^>]*>`)

// contentWords are the words a text says something with: lowercased runs
// of letters and digits, common words out; code is words too.
func contentWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(markup.ReplaceAllString(text, " ")), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if !commonWords[w] {
			out[w] = true
		}
	}
	return out
}

// Repeated is the share of a draft's words already said in above, in
// percent, rounded: containment of the draft in the text above it. 0 for
// a draft that holds no word.
func Repeated(draft, above string) int {
	words, said := contentWords(draft), contentWords(above)
	if len(words) == 0 {
		return 0
	}
	n := 0
	for w := range words {
		if said[w] {
			n++
		}
	}
	return (200*n + len(words)) / (2 * len(words))
}
