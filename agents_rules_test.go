package workline

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Given an assistant told "merge it", When it reads the rules, Then one
// sentence says it asks the person for their label.
func TestAgentsMergeItAsksForTheLabel(t *testing.T) {
	text := readText(t, "AGENTS.md")
	found := false
	for _, line := range strings.Split(text, "\n") {
		low := strings.ToLower(line)
		if !strings.Contains(low, "merge it") {
			continue
		}
		found = true
		if !strings.HasPrefix(strings.TrimSpace(line), "- ") {
			t.Errorf("the \"merge it\" rule is not a bullet of its own: %q", line)
		}
		if !strings.Contains(low, "ask the person for their label") {
			t.Errorf("the \"merge it\" rule does not say to ask the person for their label: %q", line)
		}
	}
	if !found {
		t.Fatal("AGENTS.md says nothing of \"merge it\"")
	}
}

// Given an assistant told "accept it in my name", When it reads the rules,
// Then it applies the three conditions: that item now, an independent
// analysis, traced.
func TestAgentsAcceptItInMyNameThreeConditions(t *testing.T) {
	text := oneLine(readText(t, "AGENTS.md"))
	_, after, ok := strings.Cut(text, "accept it in my name")
	if !ok {
		t.Fatal(`AGENTS.md does not name "accept it in my name"`)
	}
	for _, want := range []string{"that item, now", "independent analysis", "traced"} {
		if !strings.Contains(after, want) {
			t.Errorf("after \"accept it in my name\", the condition %q is missing", want)
		}
	}
}

// Given the rules, When a reader looks for "independent", Then it says what
// counts: a run other than the change's author.
func TestAgentsSaysWhatIndependentCounts(t *testing.T) {
	text := oneLine(readText(t, "AGENTS.md"))
	if !regexp.MustCompile(`(?i)independent[^.]*a run other than the change's author`).MatchString(text) {
		t.Error(`AGENTS.md does not say that "independent" means a run other than the change's author`)
	}
}

// Given the weekly sample page, When an assistant answers a week for the
// person, Then the page names the same three conditions, or links them.
func TestWeeklySampleAnswerNamesTheThreeConditions(t *testing.T) {
	text := readText(t, "docs/weekly-sample.md")
	_, rest, ok := strings.Cut(text, "**Your answer**")
	if !ok {
		t.Fatal("no Your answer bullet")
	}
	section, _, _ := strings.Cut(rest, "\n- **")
	section = oneLine(section)
	if strings.Contains(section, "AGENTS.md#") {
		return
	}
	for _, want := range []string{"that item, now", "independent analysis", "traced"} {
		if !strings.Contains(section, want) {
			t.Errorf("Your answer does not name %q, nor link the rules", want)
		}
	}
}
