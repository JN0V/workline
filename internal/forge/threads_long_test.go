package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Long thread, last reply read:
// Given an objection's thread of 150 comments whose last is the tester's
// own reply, When the tester answers the Draft's objections, Then it posts
// nothing in that thread. The tester can post nothing only if it sees that
// last reply: the GitHub forge's Threads reads a thread of 250 comments,
// more than two pages of 100, and returns all 250 in their order.
func TestLongThreadLastReplyRead(t *testing.T) {
	const total = 250
	var nodes []string
	for i := 1; i <= total; i++ {
		nodes = append(nodes, fmt.Sprintf(`{"databaseId":%d,"body":"c%d","url":"u%d","createdAt":"2026-01-01T00:00:00Z","authorAssociation":"NONE","author":{"__typename":"User","login":"u"}}`, i, i, i))
	}
	build := func(n int) string {
		return `{"isResolved":false,"path":"a.go","line":3,"comments":{"nodes":[` + strings.Join(nodes[:n], ",") + `]}}`
	}
	bin := t.TempDir()
	// A forge that gives a hundred comments to a query asking only the
	// first hundred, and the rest to a query that reads on.
	script := "#!/bin/sh\ncase \"$*\" in\n*'comments(first: 100)'*) cat <<'EOF'\n" + build(100) + "\nEOF\n;;\n*) cat <<'EOF'\n" + build(total) + "\nEOF\n;;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := (&github{repo: t.TempDir()}).Threads(7)
	if err != nil || len(got) != 1 {
		t.Fatalf("Threads: %+v, %v", got, err)
	}
	cs := got[0].Comments
	if len(cs) != total {
		t.Fatalf("%d comments read, want %d", len(cs), total)
	}
	for i, c := range cs {
		if want := fmt.Sprintf("c%d", i+1); c.Body != want {
			t.Fatalf("comment %d is %q, want %q", i, c.Body, want)
		}
	}
}
