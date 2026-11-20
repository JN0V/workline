package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

const twentyWords = "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty."

func readerSettings() Settings {
	return Settings{
		Docs: []string{"docs/**"},
		Reader: Reader{
			Pages:          []string{"docs/user/**"},
			ParagraphWords: 10,
			CellWords:      10,
		},
	}
}

func readerRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for p, c := range files {
		full := filepath.Join(repo, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return repo
}

func patchOf(diff string) []intent.Intention {
	return []intent.Intention{{Kind: "patch", Value: diff}}
}

// Condense: a new page in a subfolder of the doc's own folder passes, though
// its paragraph is over the reader budget: the doc's folder is no user page.
func TestCondenseNewPageWhereDocLives(t *testing.T) {
	doc := "# A\n\nIntro.\n\n## Moved\n\n" + twentyWords + "\n"
	diff := "diff --git a/docs/dev/a.md b/docs/dev/a.md\n" +
		"--- a/docs/dev/a.md\n+++ b/docs/dev/a.md\n" +
		"@@ -5,3 +5,1 @@\n" +
		"-## Moved\n-\n-" + twentyWords + "\n" +
		"+See [moved](a/moved.md).\n" +
		"diff --git a/docs/dev/a/moved.md b/docs/dev/a/moved.md\nnew file mode 100644\n" +
		"--- /dev/null\n+++ b/docs/dev/a/moved.md\n@@ -0,0 +1,3 @@\n+## Moved\n+\n+" + twentyWords + "\n"
	c := &condenseTask{Doc: "docs/dev/a.md", Keys: []string{"doc-too-long docs/dev/a.md"}}
	repo := readerRepo(t, map[string]string{"docs/dev/a.md": doc})
	found, _, err := judgeCondense(repo, readerSettings(), c, patchOf(diff), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		t.Errorf("a page where the doc lives must pass, got %s %s: %s", f.Rule, f.Where, f.Message)
	}
}

// Condense: a patch leaving the user pages' problems as they are passes.
func TestCondenseLeavesUserPagesAlone(t *testing.T) {
	doc := "# A\n\nIntro.\n\n## Moved\n\nMoved text here.\n"
	user := "# U\n\n[a](../dev/a.md)\n\n" + twentyWords + "\n"
	repo := readerRepo(t, map[string]string{"docs/dev/a.md": doc, "docs/user/u.md": user})
	diff := "diff --git a/docs/dev/a.md b/docs/dev/a.md\n" +
		"--- a/docs/dev/a.md\n+++ b/docs/dev/a.md\n" +
		"@@ -5,3 +5,1 @@\n" +
		"-## Moved\n-\n-Moved text here.\n" +
		"+See [moved](a/moved.md).\n" +
		"diff --git a/docs/dev/a/moved.md b/docs/dev/a/moved.md\nnew file mode 100644\n" +
		"--- /dev/null\n+++ b/docs/dev/a/moved.md\n@@ -0,0 +1,3 @@\n+## Moved\n+\n+Moved text here.\n"
	c := &condenseTask{Doc: "docs/dev/a.md", Keys: []string{"doc-too-long docs/dev/a.md"}}
	found, _, err := judgeCondense(repo, readerSettings(), c, patchOf(diff), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		t.Errorf("a patch leaving the user page as it is must pass, got %s %s: %s", f.Rule, f.Where, f.Message)
	}
}

// Dedupe: the doc giving the passage up is a user page; the link left in its
// place is a paragraph over the budget.
func TestDedupeRefusesReaderProblemOnUserPage(t *testing.T) {
	files := map[string]string{
		"docs/user/a.md": "# A\n\nShared passage here.\n",
		"docs/dev/b.md":  "# B\n\nShared passage here.\n",
	}
	d := &dedupeTask{Docs: []string{"docs/user/a.md", "docs/dev/b.md"}, Key: "duplicate x"}
	diff := func(repl string) string {
		return "diff --git a/docs/user/a.md b/docs/user/a.md\n" +
			"--- a/docs/user/a.md\n+++ b/docs/user/a.md\n@@ -3,1 +3,1 @@\n-Shared passage here.\n+" + repl + "\n"
	}
	repo := readerRepo(t, files)
	found, err := judgeDedupe(repo, readerSettings(), d, patchOf(diff("Kept in [b](../dev/b.md). "+twentyWords)), nil)
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, f := range found {
		if f.Rule == "patch-introduces" && f.Where == "docs/user/a.md" && strings.Contains(f.Message, "paragraph-too-long") {
			named = true
		}
	}
	if !named {
		t.Fatalf("want patch-introduces naming paragraph-too-long on docs/user/a.md, got %+v", found)
	}
	found, err = judgeDedupe(repo, readerSettings(), d, patchOf(diff("Kept in [b](../dev/b.md).")), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		t.Errorf("a short link must pass, got %s %s: %s", f.Rule, f.Where, f.Message)
	}
}

// Merge into a card: the card goes into a user page and brings a paragraph
// over the budget.
func TestMergeCardRefusesReaderProblemOnUserPage(t *testing.T) {
	m := &mergeCardTask{Card: "docs/cards/short.md", Cards: []string{"docs/user/target.md"}}
	build := func(text string) (string, []intent.Intention) {
		repo := readerRepo(t, map[string]string{
			"docs/cards/short.md": text + "\n",
			"docs/user/target.md": "# T\n\nOld line.\n",
		})
		diff := "diff --git a/docs/cards/short.md b/docs/cards/short.md\ndeleted file mode 100644\n" +
			"--- a/docs/cards/short.md\n+++ /dev/null\n@@ -1,1 +0,0 @@\n-" + text + "\n" +
			"diff --git a/docs/user/target.md b/docs/user/target.md\n" +
			"--- a/docs/user/target.md\n+++ b/docs/user/target.md\n@@ -3,0 +4,2 @@\n+\n+" + text + "\n"
		return repo, patchOf(diff)
	}
	repo, in := build(twentyWords)
	found, err := judgeMergeCard(repo, readerSettings(), m, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, f := range found {
		if f.Rule == "patch-introduces" && f.Where == "docs/user/target.md" && strings.Contains(f.Message, "paragraph-too-long") {
			named = true
		}
	}
	if !named {
		t.Fatalf("want patch-introduces naming paragraph-too-long on docs/user/target.md, got %+v", found)
	}
	repo, in = build("Short text.")
	found, err = judgeMergeCard(repo, readerSettings(), m, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		t.Errorf("a short card must pass, got %s %s: %s", f.Rule, f.Where, f.Message)
	}
}
