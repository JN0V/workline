package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Long thread, last reply read: Given a review thread of 250 comments, more
// than two pages of 100, on GitHub, When the forge lists the threads, Then
// the thread holds all 250 comments in their order, the last one included.
// The simulated gh answers per query, as GitHub does: the first hundred
// comments with a cursor, then the hundred after the cursor it is given,
// and so on to the end.
func TestGitHubThreadsLongThreadLastReplyRead(t *testing.T) {
	bin := t.TempDir()
	type node struct {
		ID      int    `json:"databaseId"`
		Body    string `json:"body"`
		URL     string `json:"url"`
		Created string `json:"createdAt"`
		Assoc   string `json:"authorAssociation"`
		Author  struct {
			Type  string `json:"__typename"`
			Login string `json:"login"`
		} `json:"author"`
	}
	writePage := func(name string, from, to int, next string) {
		var nodes []node
		for i := from; i <= to; i++ {
			n := node{ID: i, Body: fmt.Sprintf("comment %d", i), URL: fmt.Sprintf("https://github.com/o/r/pull/1#discussion_r%d", i), Created: "2026-01-01T00:00:00Z", Assoc: "MEMBER"}
			n.Author.Type, n.Author.Login = "User", "someone"
			nodes = append(nodes, n)
		}
		info := map[string]any{"hasNextPage": next != "", "endCursor": nil}
		if next != "" {
			info["endCursor"] = next
		}
		thread := map[string]any{"id": "PRRT_1", "isResolved": false, "path": "a_test.go", "line": 3,
			"comments": map[string]any{"nodes": nodes, "pageInfo": info}}
		data, err := json.Marshal(thread)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bin, name), append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePage("page1.json", 1, 100, "CURSOR100")
	writePage("page2.json", 101, 200, "CURSOR200")
	writePage("page3.json", 201, 250, "")
	calls := filepath.Join(bin, "calls")
	script := "#!/bin/sh\n" +
		"echo x >> " + calls + "\n" +
		"if [ \"$(wc -l < " + calls + ")\" -gt 20 ]; then exit 1; fi\n" +
		"case \"$*\" in\n" +
		"*CURSOR200*) cat " + filepath.Join(bin, "page3.json") + " ;;\n" +
		"*CURSOR100*) cat " + filepath.Join(bin, "page2.json") + " ;;\n" +
		"*) cat " + filepath.Join(bin, "page1.json") + " ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := &github{repo: t.TempDir()}
	threads, err := g.Threads(1)
	if err != nil || len(threads) != 1 {
		t.Fatalf("Threads: %d threads, %v", len(threads), err)
	}
	got := threads[0].Comments
	if len(got) != 250 {
		t.Fatalf("the thread holds %d comments, want 250: its last reply is not read", len(got))
	}
	for i, c := range got {
		if c.ID != fmt.Sprint(i+1) {
			t.Fatalf("comment %d has id %s, want %d", i, c.ID, i+1)
		}
	}
}
