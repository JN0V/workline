package documentalist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/intent"
)

// New page among user pages:
// Given a condense patch whose new page sits among user pages and has a
// paragraph over the budget,
// When the judge checks it,
// Then the patch is refused, naming `paragraph-too-long` and the page.
func TestNewPageAmongUserPages(t *testing.T) {
	repo := t.TempDir()
	long := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty."
	doc := "# A\n\nIntro.\n\n## Moved\n\n" + long + "\n"
	for p, c := range map[string]string{"docs/dev/a.md": doc} {
		full := filepath.Join(repo, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	diff := "diff --git a/docs/dev/a.md b/docs/dev/a.md\n" +
		"--- a/docs/dev/a.md\n+++ b/docs/dev/a.md\n" +
		"@@ -5,3 +5,1 @@\n" +
		"-## Moved\n-\n-" + long + "\n" +
		"+See [moved](../user/moved.md).\n" +
		"diff --git a/docs/user/moved.md b/docs/user/moved.md\n" +
		"new file mode 100644\n" +
		"--- /dev/null\n+++ b/docs/user/moved.md\n" +
		"@@ -0,0 +1,3 @@\n" +
		"+## Moved\n+\n+" + long + "\n"
	s := Settings{
		Docs: []string{"docs/"},
		Reader: Reader{
			Pages:          []string{"docs/user/"},
			ParagraphWords: 10,
			CellWords:      10,
		},
	}
	c := &condenseTask{Doc: "docs/dev/a.md", Keys: []string{"doc-too-long docs/dev/a.md"}}
	intents := []intent.Intention{{Kind: "patch", Value: diff}}
	found, _, err := judgeCondense(repo, s, c, intents, nil)
	if err != nil {
		t.Fatal(err)
	}
	named := false
	for _, f := range found {
		if strings.Contains(f.Rule+" "+f.Message, "paragraph-too-long") && strings.Contains(f.Where, "docs/user/moved.md") {
			named = true
		}
	}
	if !named {
		t.Fatalf("want a refusal naming paragraph-too-long and docs/user/moved.md, got %+v", found)
	}
}
