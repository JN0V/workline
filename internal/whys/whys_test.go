package whys

import (
	"strings"
	"testing"
)

const good = `Five whys
Problem: the test and the code disagree on how mul prints.
1. Why do they disagree? The test checks "6\n" ("printf '%s' $r").
2. Why a newline? The scenario says "Then I get 6",
   silent on printing.
Root cause: the Validation leaves the format open.
Also: the builder did not say which format callers need.
Action (doc): add "Then it prints 6 and a newline" to the scenario.`

func TestParseReadsTheSkillsForm(t *testing.T) {
	a, err := Parse(good)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Whys) != 2 || !strings.HasSuffix(a.Whys[1], "silent on printing.") || a.Kind != "doc" || a.Also == "" {
		t.Fatalf("read %+v", a)
	}
	if md := a.Markdown(); !strings.Contains(md, "**Root cause:**") || !strings.Contains(md, "**Action** (doc)") {
		t.Fatal(md)
	}
}

func TestParseRefusesWhatTheFormLacks(t *testing.T) {
	for name, c := range map[string]struct{ note, says string }{
		"no heading": {strings.TrimPrefix(good, "Five whys\n"), "opens with"},
		"no cause":   {strings.Replace(good, "Root cause:", "Cause:", 1), "lacks Root cause:"},
		"no action":  {strings.Split(good, "\nAction")[0], "Action (<kind>):"},
		"bad kind":   {strings.Replace(good, "(doc)", "(blame)", 1), `"blame"`},
		"order":      {strings.Replace(good, "2. Why", "3. Why", 1), "number them"},
		"too long":   {good + strings.Repeat(" x", MaxChars), "characters"},
		"six whys":   {strings.Replace(good, "Root cause", "3. a\n4. b\n5. c\n6. d\nRoot cause", 1), "five at most"},
	} {
		if _, err := Parse(c.note); err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: %v, want %q", name, err, c.says)
		}
	}
}
