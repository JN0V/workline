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
		"- claim: {quote: \"a\"\n"
	read, broken, mended := claimsOneByOne(answer)
	if read != 1 || len(broken) != 1 || len(mended) != 2 {
		t.Fatalf("read %d, broken %v, mended %q", read, broken, mended)
	}
	c := claimsRead(answer)[0].Value.(map[string]any)
	if c["quote"] != "\tif ok {\n" || c["why"] != "covered by #20 too" {
		t.Errorf("the claim was not read as written: %q", c)
	}
}
