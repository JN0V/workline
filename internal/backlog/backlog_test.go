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

// A question asked in an earlier comment is told whatever leads it there —
// the reporter named, the sentence before it, spaces and case aside.
func TestAskedBefore(t *testing.T) {
	comments := []string{
		"@ann, to refine this issue: Which rows are lost: the last one, or any? Since when?\n\n" + AskMarker("product-owner", 1),
		"Not sure.",
	}
	for q, want := range map[string]string{
		"Which rows are lost: the last one, or any?":    "asked-before",
		"  which ROWS are lost:\nthe last one, or any?": "asked-before",
		"Is it every export? Since when?":               "asked-before",
		"Which export?":                                 "",
		"Or any?":                                       "", // a question that only ends an earlier one is a new one
		"When?":                                         "", // nor one that ends the lead's
	} {
		c := Proposal{Do: "ask", Questions: q}
		if rule, _ := conversation("product-owner", comments, &c); rule != want {
			t.Errorf("%q: rule %q, want %q", q, rule, want)
		}
	}
	if e := ReadExchange(append(comments, "@ann, still: Which export?\n\n"+AskMarker("product-owner", 2)), "product-owner"); e.Rounds != 2 || e.Answered {
		t.Errorf("exchange = %+v, want two rounds, the last not answered", e)
	}
}
