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
// The simulated gh answers as GitHub does: a thread's comments, a hundred
// at most, the first ones.
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
	var first []node
	for i := 1; i <= 100; i++ {
		n := node{ID: i, Body: fmt.Sprintf("comment %d", i), URL: fmt.Sprintf("https://github.com/o/r/pull/1#discussion_r%d", i), Created: "2026-01-01T00:00:00Z", Assoc: "MEMBER"}
		n.Author.Type, n.Author.Login = "User", "someone"
		first = append(first, n)
	}
	thread := map[string]any{"id": "PRRT_1", "isResolved": false, "path": "a_test.go", "line": 3,
		"comments": map[string]any{"nodes": first}}
	data, err := json.Marshal(thread)
	if err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(bin, "page.json")
	if err := os.WriteFile(page, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncat " + page + "\n"
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
