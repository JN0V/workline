package documentalist

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JN0V/workline/internal/forge"
)

// A reader branch closed without merging holds its page, unchanged since,
// as a person's decline; one workline closed itself, its comment carrying
// the marker, does not (#592). A forge that cannot be opened, or read, is
// said, and the page held.
func TestReaderDeclined(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("WORKLINE_OPEN_MERGE_REQUESTS", "1")
	repo := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=T", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	page := "# Guide\n\nOne long paragraph.\n"
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "guide.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	run("init", "-q", "-b", "main")
	run("add", ".")
	run("commit", "-qm", "docs: a guide")
	run("checkout", "-qb", "workline/documentalist/reader-docs-guide-md")
	if err := os.WriteFile(filepath.Join(repo, "docs", "guide.md"), []byte("# Guide\n\n- One.\n- Paragraph.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("commit", "-qam", "docs: read docs/guide.md for its reader")
	run("checkout", "-q", "main")

	state := func(comment string) string {
		p := filepath.Join(t.TempDir(), "forge.json")
		data, _ := json.Marshal(forge.FakeState{MergeRequests: []forge.FakeItem{{ID: 1, Branch: "workline/documentalist/reader-docs-guide-md",
			Closed: true, Comments: []forge.FakeComment{{Body: comment}}}}})
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name, forge string
		held        bool
		said        string
	}{
		{"closed by workline", "fake:" + state("Closed by workline's documentalist.\n\n" + forge.Marker("closed")), false, ""},
		{"closed by a person", "fake:" + state("Not what I want."), true, ""},
		{"no forge", "", true, ""},
		{"a forge that cannot be read", "fake:" + filepath.Join(t.TempDir(), "gone.json"), true, "whether workline closed workline/documentalist/reader-docs-guide-md itself could not be read"},
		{"a forge that cannot be opened", "nowhere", true, "unknown forge"},
	}
	for _, c := range cases {
		t.Setenv("WORKLINE_FORGE", c.forge)
		declined, said := readerDeclined(repo, waiting{}, map[string]string{"docs/guide.md": page})
		if held := declined["docs/guide.md"] != ""; held != c.held {
			t.Errorf("%s: held %v, want %v", c.name, held, c.held)
		}
		if c.said == "" && said != "" || !strings.Contains(said, c.said) {
			t.Errorf("%s: said %q, want %q", c.name, said, c.said)
		}
	}
}
