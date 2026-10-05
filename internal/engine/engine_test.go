package engine

import "testing"

func TestAskAgain(t *testing.T) {
	for _, c := range []struct {
		promoteAfter int
		want         string // tier of each attempt after the first, until it stops
	}{
		{0, ""},
		{1, "standard"},
		{2, "light standard"},
		{3, "light light standard"},
	} {
		got, tier := "", "light"
		for failures := 1; ; failures++ {
			retry, next := askAgain(failures, c.promoteAfter, tier)
			if !retry {
				break
			}
			tier = next
			got += " " + tier
		}
		if got != " "+c.want && !(c.want == "" && got == "") {
			t.Errorf("promote-after %d: attempts after the first on %q, want %q", c.promoteAfter, got, c.want)
		}
	}
}

// An answer broken in one claim is read claim by claim; a claim holding
// code pasted with its tabs, or a text cut by ` #`, is mended as the whole
// answer would be (agent.Mend), never counted as unreadable.
func TestClaimsOneByOneMended(t *testing.T) {
	answer := "- claim:\n    quote: |\n\tif ok {\n    why: covered by #20 too\n" +
		"- claim: {quote: \"a\"\n" +
		"- claim:\n    quote: |\n        x\n}\n" + // a line at the start of the line: the item broken, not cut
		"- claim:\n    why: see #3 below\n```\nThat is all.\n"
	read, broken, mended := claimsOneByOne(answer)
	if read != 2 || len(broken) != 2 || len(mended) != 2 || mended[1][0] != '2' {
		t.Fatalf("read %d, broken %v, mended %q", read, broken, mended)
	}
	claims := claimsRead(answer)
	c := claims[0].Value.(map[string]any)
	if c["quote"] != "\tif ok {\n" || c["why"] != "covered by #20 too" {
		t.Errorf("the claim was not read as written: %q", c)
	}
	if c := claims[1].Value.(map[string]any); c["why"] != "see #3 below" {
		t.Errorf("the claim before the fence was not read as written: %q", c)
	}
	if b := broken[1].Value.(map[string]any)["unreadable"]; b != "- claim:\n    quote: |\n        x\n}" {
		t.Errorf("the broken claim is not kept whole: %q", b)
	}
	// A line saying no count is said as it is.
	if got := mendedTogether([]string{"2 blocks", "a block", "1 text", "1 blocks", "a block"}); len(got) != 3 || got[0] != "3 blocks" || got[1] != "a block" {
		t.Errorf("mended together: %q", got)
	}
}
