package workline

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const simplicityPage = "docs/research/simplicity-lens-measured.md"

var (
	pageHeading = regexp.MustCompile(`^#{1,6}\s`)
	pageRef     = regexp.MustCompile(`(?:#|/pull/)(\d+)`)
)

// pageSection returns the text under the first heading of the page that
// matches title, up to the next heading of the same or a higher level.
func pageSection(t *testing.T, text string, title *regexp.Regexp) string {
	t.Helper()
	var out []string
	level := 0
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
		}
		if !fenced && pageHeading.MatchString(trimmed) {
			n := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			if level > 0 && n <= level {
				break
			}
			if level == 0 && title.MatchString(trimmed) {
				level = n
			}
			continue
		}
		if level > 0 {
			out = append(out, line)
		}
	}
	if level == 0 {
		t.Fatalf("%s has no heading matching %s", simplicityPage, title)
	}
	return strings.Join(out, "\n")
}

// pageEvidence counts the distinct pull requests a section names and the
// lines of it quoted (a block quote).
func pageEvidence(section string) (pulls, quotes int) {
	seen := map[string]bool{}
	for _, m := range pageRef.FindAllStringSubmatch(section, -1) {
		seen[m[1]] = true
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			quotes++
		}
	}
	return len(seen), quotes
}

// Given past pull requests the maintainer marks over-built, When the lens
// reads each, Then the page shows how many it flagged, each finding quoted.
func TestSimplicityLensMeasuredOverBuiltCaught(t *testing.T) {
	text := readText(t, simplicityPage)
	section := pageSection(t, text, regexp.MustCompile(`(?i)over[- ]?built`))
	m := regexp.MustCompile(`(\d+)\s+of\s+(\d+)`).FindStringSubmatch(section)
	if m == nil {
		t.Fatal(`the over-built section does not say how many it flagged, as "N of M"`)
	}
	flagged, _ := strconv.Atoi(m[1])
	total, _ := strconv.Atoi(m[2])
	if flagged > total {
		t.Errorf("the over-built section flags %d of %d: more than it read", flagged, total)
	}
	pulls, quotes := pageEvidence(section)
	if pulls < total {
		t.Errorf("the over-built section names %d pull requests, it says it read %d", pulls, total)
	}
	if quotes < flagged {
		t.Errorf("the over-built section quotes %d lines for %d flagged pull requests: each finding must be quoted", quotes, flagged)
	}
}

// Given past pull requests the maintainer marks plain, When the lens reads
// each, Then the page shows the important findings it raised on them, each
// quoted.
func TestSimplicityLensMeasuredPlainLeftAlone(t *testing.T) {
	text := readText(t, simplicityPage)
	section := pageSection(t, text, regexp.MustCompile(`(?i)\bplain\b`))
	m := regexp.MustCompile(`(?i)\b(\d+|no)\s+important\b`).FindStringSubmatch(section)
	if m == nil {
		t.Fatal(`the plain section does not say how many important findings the lens raised, as "N important findings"`)
	}
	raised := 0
	if strings.ToLower(m[1]) != "no" {
		raised, _ = strconv.Atoi(m[1])
	}
	pulls, quotes := pageEvidence(section)
	if pulls == 0 {
		t.Error("the plain section names no pull request")
	}
	if quotes < raised {
		t.Errorf("the plain section quotes %d lines for %d important findings: each finding must be quoted", quotes, raised)
	}
}

// Given the over-built and the plain counts, When the maintainer reads the
// page, Then it says whether the lens may block, and at what bar.
func TestSimplicityLensMeasuredDecidedOnNumbers(t *testing.T) {
	text := oneLine(readText(t, simplicityPage))
	verdict := regexp.MustCompile(`(?i)(\b(may|might|can|could|should|must|shall|does|do)( not|n't)? block|cannot block|\b(stays?|remains?|kept) advisory)`)
	if !verdict.MatchString(text) {
		t.Error(`the page does not say whether the lens may block ("may block", "may not block", "stays advisory")`)
	}
	bar := regexp.MustCompile(`(?i)\bbar\b[^.]*(\d+(\.\d+)?\s*%|\d+\s+(important\s+)?(findings?|of\s+\d+))|(\d+(\.\d+)?\s*%|\d+\s+(important\s+)?(findings?|of\s+\d+))[^.]*\bbar\b`)
	if !bar.MatchString(text) {
		t.Error(`the page does not give the bar the decision is made at, a number next to the word "bar"`)
	}
}
