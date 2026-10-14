package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestOverBudgetSummary(t *testing.T) {
	const head = "the task was over the role's context budget and nobody judged it: a person does"
	if got := overBudgetSummary(""); got != head {
		t.Errorf("empty: got %q", got)
	}
	if got := overBudgetSummary("docs to check"); got != head+" (docs to check)" {
		t.Errorf("said: got %q", got)
	}
}

// A diff in a comment is copied whole and applied: no line of it ends its
// block, and the block holds it as git wrote it, a last blank line of
// context too (#145).
func TestDiffBlock(t *testing.T) {
	plain := "--- a/doc.md\n+++ b/doc.md\n@@ -1,2 +1,2 @@\n Title\n-old\n+new\n"
	if got, want := diffBlock(strings.TrimSuffix(plain, "\n")), "```diff\n"+plain+"```"; got != want {
		t.Errorf("without a backtick, the block changed:\n%s\nwant:\n%s", got, want)
	}
	fenced := "--- a/doc.md\n+++ b/doc.md\n@@ -1,4 +1,5 @@\n ```sh\n tool\n ```\n+```\n \n"
	got := diffBlock(fenced)
	open, rest, _ := strings.Cut(got, "\n")
	fence := strings.TrimSuffix(open, "diff")
	if !strings.HasPrefix(fence, "```") || strings.Trim(fence, "`") != "" {
		t.Fatalf("opens with %q, not a fence and diff", open)
	}
	if inside, ok := strings.CutSuffix(rest, fence); !ok || inside != fenced {
		t.Errorf("the block holds %q, want the diff %q", inside, fenced)
	}
	// CommonMark closes a fence on a line of as many backticks or more,
	// indented three spaces at most.
	for _, l := range strings.Split(fenced, "\n") {
		if bare := strings.TrimLeft(l, " "); len(l)-len(bare) <= 3 && strings.HasPrefix(bare, fence) {
			t.Errorf("line %q of the diff closes the block's fence %q", l, fence)
		}
	}
}

// The diff a comment shows applies whole with `git apply`: a doc changed,
// its hunk ending on a blank line of context, and docs the run created, in
// a folder of their own too; the index is left as it was (#145).
func TestWrittenDiffAppliesWhole(t *testing.T) {
	repo, run, write := gitRepo(t)
	write("doc.md", "# Doc\n\nOne.\nTwo.\nThree.\n\nFour.\n")
	run("add", "doc.md")
	run("commit", "-qm", "docs: start")
	want := map[string]string{
		"doc.md":     "# Doc\n\nOne, changed.\nTwo.\nThree.\n\nFour.\n",
		"new.md":     "# New\n\nSplit off.\n",
		"cards/c.md": "# Card\n",
		"empty.md":   "",
	}
	if err := os.Mkdir(filepath.Join(repo, "cards"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, s := range want {
		write(p, s)
	}
	status := run("status", "--porcelain")
	if diff, err := writtenDiff(runState{Repo: repo}); err != nil || diff != "" {
		t.Errorf("nothing written: diff %q, %v; want none, not the whole tree's", diff, err)
	}
	diff, err := writtenDiff(runState{Repo: repo, Written: []string{"doc.md", "new.md", "cards/c.md", "empty.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if now := run("status", "--porcelain"); now != status {
		t.Errorf("the index changed: %q, was %q", now, status)
	}
	run("checkout", "--", "doc.md")
	for _, p := range []string{"new.md", "cards", "empty.md"} {
		if err := os.RemoveAll(filepath.Join(repo, p)); err != nil {
			t.Fatal(err)
		}
	}
	write("comment.patch", diff)
	run("apply", "comment.patch")
	for p, s := range want {
		if got, err := os.ReadFile(filepath.Join(repo, p)); err != nil || string(got) != s {
			t.Errorf("%s after git apply: %q (%v), want %q", p, got, err, s)
		}
	}
}

// A judge's question pre asks again (ADR-0045) is one answered before pre
// ran whose answer pre took away; past the rounds, it is answered as not
// asked, so pre settles it.
func TestJudgeAskedAgain(t *testing.T) {
	run := t.TempDir()
	for key, answered := range map[string]bool{"01": true, "02": true, "03": false} {
		dir := filepath.Join(run, "in", "judge", key)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "question.yaml"), []byte("{question: q}"), 0o644)
		if answered {
			os.WriteFile(filepath.Join(dir, "answer.yaml"), []byte("yes: true"), 0o644)
		}
	}
	before := judgeAnswered(run)
	if len(before) != 2 {
		t.Fatalf("answered: %v", before)
	}
	os.Remove(filepath.Join(run, "in", "judge", "02", "answer.yaml")) // pre asks 02 again
	again := askedAgain(run, before)
	if len(again) != 1 || !strings.HasSuffix(again[0], filepath.Join("02", "question.yaml")) {
		t.Errorf("asked again: %v; want 02 alone, never 03, which was never answered", again)
	}
	if err := notAsked(again, "not asked: past 10 rounds of the judge"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(run, "in", "judge", "02", "answer.yaml")); !strings.Contains(string(data), "error: 'not asked: past 10 rounds of the judge'") {
		t.Errorf("past the rounds, answered as not asked: %s", data)
	}
	if len(askedAgain(run, judgeAnswered(run))) != 0 {
		t.Error("once answered, nothing is asked again")
	}
}
