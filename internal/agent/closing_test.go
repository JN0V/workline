package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClosingWords(t *testing.T) {
	kinds := []string{"comment", "note"}
	for _, c := range []struct {
		name, answer, want, mended string
		unread                     bool
	}{
		{"empty", "  ", "[]\n", "", false},
		{"a list", "- note: done\n", "- note: done\n", "", false},
		{"fenced amid prose", "I ran:\n\n```sh\ngo test ./...\n```\n\nThen:\n\n```yaml\n- note: done\n```\nBye.", "- note: done\n", "prose left out", false},
		{"unfenced amid prose", "It is wrong: `x`.\n\n- note: done\n  \n\nThat is all.", "- note: done\n", "prose left out", false},
		{"prose then []", "All green: `mul` added.\n\n[]", "[]\n", "before their `[]`", false},
		{"prose", "Done: `mul` added.\nTests: green.", "[]\n", "", true},
		{"a list of texts", "- I added mul\n- I ran the tests", "[]\n", "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, "out"), 0o755)
			os.WriteFile(filepath.Join(dir, "out", ClosingUnread), []byte("stale"), 0o644)
			got, mended := closingWords(dir, c.answer, kinds)
			if string(got) != c.want {
				t.Errorf("read %q, want %q", got, c.want)
			}
			if said := strings.Join(mended, "; "); !strings.Contains(said, c.mended) || (c.mended == "" && said != "") {
				t.Errorf("mended %q, want %q", said, c.mended)
			}
			data, err := os.ReadFile(filepath.Join(dir, "out", ClosingUnread))
			if c.unread != (err == nil) || (c.unread && string(data) != c.answer) {
				t.Errorf("unread file: %q, %v; want it %v", data, err, c.unread)
			}
		})
	}
}
