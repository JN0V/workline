package engine

import "testing"

// Pasted list line
// Given a claim that holds a line `- foo`,
// When the engine reads the answer,
// Then the claim is read whole.
func TestPastedListLine(t *testing.T) {
	answer := "- claim: |\n    first\n- foo\n    last\n"
	items := answerItems(answer)
	if len(items) != 1 {
		t.Fatalf("answerItems gave %d items, want 1 (the claim whole): %q", len(items), items)
	}
	if items[0] != answer {
		t.Errorf("item = %q, want the claim whole: %q", items[0], answer)
	}
}

// Real items
// Given two items that each start with `- claim:`,
// When it reads them,
// Then they are two items.
func TestRealItems(t *testing.T) {
	answer := "- claim: one\n- claim: two\n"
	items := answerItems(answer)
	if len(items) != 2 {
		t.Fatalf("answerItems gave %d items, want 2: %q", len(items), items)
	}
	if items[0] != "- claim: one\n" || items[1] != "- claim: two\n" {
		t.Errorf("items = %q, want the two claims apart", items)
	}
}
