package backlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A quote is found at the lines that hold it, not from an earlier line that
// only shares its first word (workline's BACKLOG.md, imported on a copy:
// an item's issue began with the end of the item before).
func TestLocate(t *testing.T) {
	repo := t.TempDir()
	text := "- **Releases.** Left: the sandbox,\n  then doctor - and init.\n- **Install.** Done: releases.\n  Left: a reusable action.\n"
	os.WriteFile(filepath.Join(repo, "BACKLOG.md"), []byte(text), 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull) // the machine's hooks stay out
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, c := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=T", "-c", "user.email=t@x.invalid", "commit", "-qm", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, c...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	from, to, got, ok := Locate(repo, "BACKLOG.md", "- **Install.** Done: releases. Left: a reusable action.")
	if !ok || from != 3 || to != 4 || got != "- **Install.** Done: releases.\n  Left: a reusable action." {
		t.Fatalf("Locate = %d, %d, %q, %v", from, to, got, ok)
	}
}

// A parent's task list is added once, after its text, and rewritten in
// place when a resumed split lists its children again.
func TestListChildren(t *testing.T) {
	body := ListChildren("Two needs.\n\n## Scope\n\nsrc.", []int{10})
	want := "Two needs.\n\n## Scope\n\nsrc.\n\n## Sub-issues\n\n- [ ] #10"
	if body != want {
		t.Fatalf("first list = %q", body)
	}
	if again := ListChildren(body, []int{10, 11}); again != want+"\n- [ ] #11" {
		t.Fatalf("list rewritten = %q", again)
	}
}

// Two children are one when their titles differ only in case or spaces;
// distinct titles, or one title under two parents, are not.
func TestSplitKey(t *testing.T) {
	same := SplitKey(9, "Keep the last row")
	for _, title := range []string{"Keep the last row", "keep the LAST row", "  Keep  the last\trow "} {
		if SplitKey(9, title) != same {
			t.Errorf("SplitKey(9, %q) differs from %q's", title, "Keep the last row")
		}
	}
	if SplitKey(9, "Quote commas") == same || SplitKey(10, "Keep the last row") == same {
		t.Error("another title, or another parent, gives the same key")
	}
}

// The text proposed to an outsider is read back from the engine's own block,
// whatever fences the agent's words hold; what the comment shows holds no
// fence of the agent's either.
func TestProposalReadBack(t *testing.T) {
	c := Proposal{Do: "refine", Added: []string{"Need", "Scope"}, Sources: []string{"src/a.go"},
		Why:       "It says:\n```yaml\nneed: not this\n```",
		Need:      "Rows kept.\n```\nstray\n```",
		Scope:     "src/a.go",
		Questions: "Which one? ```yaml\nscope: nor this\n```"}
	body := ProposalComment("zed", c, "product-owner") + "\n\n" + ProposalMarker("product-owner", 1)
	p := LastProposal([]string{body}, "product-owner")
	if p == nil || p.Scope != "src/a.go" || !strings.HasPrefix(p.Need, "Rows kept.") || strings.Contains(p.Need, "```") {
		t.Fatalf("read back %+v from:\n%s", p, body)
	}
	if n := strings.Count(body, "```"); n != 2 {
		t.Errorf("%d fences in the comment, want the engine's two:\n%s", n, body)
	}
}

// A text is cut into its questions: a list's items each one, the options
// of one question kept together, a sentence before it left out.
func TestQuestions(t *testing.T) {
	for text, want := range map[string]string{
		"- Do X\n- Since when?":                        "since when?",
		"Which do you mean:\n- the CLI or\n- the API?": "which do you mean: - the cli or - the api?",
		"Thanks. Is it\n1) fast or\n2) slow?":          "is it 1) fast or 2) slow?",
		"E.g. which file? Does a.go fail?":             "e.g. which file?|does a.go fail?",
	} {
		if got := strings.Join(questions(text), "|"); got != want {
			t.Errorf("questions(%q) = %q, want %q", text, got, want)
		}
	}
}

// A question asked in an earlier comment is told whatever leads it there —
// the reporter named, the sentence before it, spaces and case aside.
func TestAskedBefore(t *testing.T) {
	comments := []string{
		"@ann, to refine this issue: Which rows are lost: the last one, or any? Since when? Does a.go fail? Is it v1.3? Which file?\n\n" + AskMarker("product-owner", 1),
		"Not sure.",
	}
	for q, want := range map[string]string{
		"Which rows are lost: the last one, or any?":    "asked-before",
		"  which ROWS are lost:\nthe last one, or any?": "asked-before",
		"Is it every export? Since when?":               "", // asked without "Since when?"
		"Which export?":                                 "",
		"Or any?":                                       "", // a question that only ends an earlier one is a new one
		"When?":                                         "", // nor one that ends the lead's
		"Does b.go fail?":                               "", // a dot in a name is no sentence's end
		"Does a.go fail?":                               "asked-before",
		"Thanks! Since when?":                           "asked-before", // a sentence ends at "! "
		"Is it v1.2?":                                   "",             // a version's dot cuts nothing
		"Is it v1.3?":                                   "asked-before",
		"- Since when?":                                 "asked-before", // a list's bullet aside, each form
		"* Since when?":                                 "asked-before",
		"• Since when?":                                 "asked-before",
		"1. Since when?":                                "asked-before",
		"2) Since when?":                                "asked-before",
		"Okay. - Since when?":                           "asked-before", // a bullet after a sentence cut
		"E.g. which file?":                              "",             // "e.g." ends no sentence: not cut to "which file?"
		"Okay. Which file?":                             "asked-before",
		"- Do X\n- Since when?":                         "asked-before", // a list's item before it, on its own line
		"Which file:\n- a.go or\n- which file?":         "",             // options of one question, not items: not cut
	} {
		c := Proposal{Do: "ask", Questions: q}
		if rule, _, _ := conversation("product-owner", comments, &c); rule != want {
			t.Errorf("%q: rule %q, want %q", q, rule, want)
		}
	}
	// Only the question asked before is left out, what leads it kept; an
	// ask with none left is dropped, a refine keeps its drafts.
	for _, tc := range []struct{ do, q, kept, rule string }{
		{"ask", "Since when? Which export?", "Which export?", ""},
		{"ask", "Thanks. Since when? Which export?", "Thanks. Which export?", ""},
		{"ask", "Which export?\n- Since when?", "Which export?", ""},
		{"ask", "Since when? Does a.go fail?", "Since when? Does a.go fail?", "asked-before"},
		{"refine", "Since when? Does a.go fail?", "", ""},
		{"refine", "Which export?", "Which export?", ""},
	} {
		c := Proposal{Do: tc.do, ToReporter: tc.do == "refine", Questions: tc.q}
		rule, _, left := conversation("product-owner", comments, &c)
		if rule != tc.rule {
			t.Errorf("%s %q: rule %q, want %q", tc.do, tc.q, rule, tc.rule)
		}
		if rule == "" && (c.Questions != tc.kept || (tc.kept != tc.q) != (len(left) > 0)) {
			t.Errorf("%s %q: kept %q (left out %q), want %q", tc.do, tc.q, c.Questions, left, tc.kept)
		}
	}
	if e := ReadExchange(append(comments, "@ann, still: Which export?\n\n"+AskMarker("product-owner", 2)), "product-owner"); e.Rounds != 2 || e.Answered {
		t.Errorf("exchange = %+v, want two rounds, the last not answered", e)
	}
	// A person quoting the engine's comment, its marker inside, answers it;
	// it is not a round of the engine's.
	quoted := "> " + strings.ReplaceAll(comments[0], "\n", "\n> ") + "\n\nSince Monday."
	if e := ReadExchange([]string{comments[0], quoted}, "product-owner"); e.Rounds != 1 || !e.Answered {
		t.Errorf("exchange with a quoted ask = %+v, want one round, answered", e)
	}
}
